package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/config"
)


// Executor handles Claude Code CLI execution
type Executor struct {
	config        *config.Config
	logger        *zap.Logger
	claudeCodePath string
}

// ClaudeCodeResponse represents the response from Claude Code CLI
type ClaudeCodeResponse struct {
	Type         string      `json:"type"`
	Subtype      string      `json:"subtype"`
	IsError      bool        `json:"is_error"`
	Result       string      `json:"result"`
	SessionID    string      `json:"session_id"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	Usage        ClaudeUsage `json:"usage"`
	Error        string      `json:"error,omitempty"`
	LatestResponse string    `json:"-"` // Raw JSON response
}

// ClaudeUsage represents token usage information
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Message represents a conversation message
type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// CommandResult represents the result of command execution
type CommandResult struct {
	Command    string        `json:"command"`
	Output     string        `json:"output"`
	Error      string        `json:"error"`
	ExitCode   int           `json:"exit_code"`
	Duration   time.Duration `json:"duration"`
	Timestamp  time.Time     `json:"timestamp"`
}

// NewExecutor creates a new Claude Code executor
func NewExecutor(cfg *config.Config, logger *zap.Logger) (*Executor, error) {
	// Detect Claude Code CLI path
	claudePath := "claude"
	if envPath := os.Getenv("CLAUDE_CODE_PATH"); envPath != "" {
		claudePath = envPath
	}
	
	// Validate that Claude Code CLI is available
	if _, err := exec.LookPath(claudePath); err != nil {
		return nil, fmt.Errorf("claude code CLI not found in PATH: %w", err)
	}
	
	// Test Claude Code CLI
	cmd := exec.Command(claudePath, "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude code CLI not responding: %w", err)
	}
	
	logger.Info("Claude Code CLI detected", zap.String("path", claudePath))
	
	return &Executor{
		config:        cfg,
		logger:        logger,
		claudeCodePath: claudePath,
	}, nil
}

// ExecuteClaudeCode executes a request using Claude Code CLI
func (e *Executor) ExecuteClaudeCode(ctx context.Context, userMessage string, sessionID string, workingDir string, allowedTools []string, isNewSession bool, permissionMode config.PermissionMode) (*ClaudeCodeResponse, error) {
	// Prepare Claude Code CLI arguments
	args := []string{
		"--print",
		"--output-format", "json",
		"--model", "sonnet",
	}
	
	// Add session flag based on whether it's a new session or continuation
	if sessionID != "" {
		if isNewSession {
			args = append(args, "--session-id", sessionID)
		} else {
			args = append(args, "--resume", sessionID)
		}
	}
	
	// Add allowed tools if specified (empty means all tools available)
	if len(allowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(allowedTools, ","))
	}
	// If allowedTools is empty, don't add --allowedTools flag = Claude Code uses all tools
	
	// Add permission mode
	args = append(args, "--permission-mode", string(permissionMode))
	
	// Add system prompt for Slack bot context
	systemPrompt := "You are Claude Code running in a Slack bot environment. Be helpful, concise, and format responses appropriately for Slack."
	args = append(args, "--append-system-prompt", systemPrompt)
	
	// Create command with timeout
	cmd := exec.CommandContext(ctx, e.claudeCodePath, args...)
	cmd.Dir = workingDir
	
	// Set up stdin with user message
	cmd.Stdin = strings.NewReader(userMessage)
	
	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	e.logger.Info("Executing Claude Code CLI",
		zap.String("session_id", sessionID),
		zap.String("working_dir", workingDir),
		zap.Strings("allowed_tools", allowedTools))
	
	// Execute command
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	
	if err != nil {
		e.logger.Error("Claude Code CLI execution failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()),
			zap.Duration("duration", duration))
		return nil, fmt.Errorf("claude code execution failed: %w", err)
	}
	
	// Parse JSON response
	var response ClaudeCodeResponse
	responseBytes := stdout.Bytes()
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		e.logger.Error("Failed to parse Claude Code response",
			zap.Error(err),
			zap.String("stdout", stdout.String()))
		return nil, fmt.Errorf("failed to parse Claude Code response: %w", err)
	}
	
	// Save raw response
	response.LatestResponse = string(responseBytes)
	
	// Check for errors in response
	if response.IsError {
		e.logger.Error("Claude Code returned error",
			zap.String("error", response.Error))
		return nil, fmt.Errorf("claude code error: %s", response.Error)
	}
	
	e.logger.Debug("Claude Code execution successful",
		zap.String("session_id", response.SessionID),
		zap.Float64("cost_usd", response.TotalCostUSD),
		zap.Int("input_tokens", response.Usage.InputTokens),
		zap.Int("output_tokens", response.Usage.OutputTokens),
		zap.Duration("duration", duration))
	
	return &response, nil
}

// ExecuteCommand executes a system command with safety checks
func (e *Executor) ExecuteCommand(ctx context.Context, command string, workingDir string) (*CommandResult, error) {
	result := &CommandResult{
		Command:   command,
		Timestamp: time.Now(),
	}

	// Security check: validate command
	if !e.config.IsCommandAllowed(command) {
		result.Error = "Command not allowed"
		result.ExitCode = 1
		e.logger.Warn("Blocked command execution", zap.String("command", command))
		return result, fmt.Errorf("command not allowed: %s", command)
	}

	// Create working directory if it doesn't exist
	if workingDir == "" {
		workingDir = e.config.WorkingDirectory
	}

	if err := os.MkdirAll(workingDir, 0755); err != nil {
		result.Error = fmt.Sprintf("Failed to create working directory: %v", err)
		result.ExitCode = 1
		e.logger.Error("Failed to create working directory", zap.Error(err), zap.String("dir", workingDir))
		return result, fmt.Errorf("failed to create working directory: %w", err)
	}

	// Create context with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, e.config.CommandTimeout)
	defer cancel()

	start := time.Now()

	// Parse command - handle shell commands properly
	var cmd *exec.Cmd
	if strings.Contains(command, "|") || strings.Contains(command, "&&") || 
		strings.Contains(command, "||") || strings.Contains(command, ";") {
		// Complex shell command
		cmd = exec.CommandContext(cmdCtx, "bash", "-c", command)
	} else {
		// Simple command - split by spaces
		parts := strings.Fields(command)
		if len(parts) == 0 {
			result.Error = "Empty command"
			result.ExitCode = 1
			return result, fmt.Errorf("empty command")
		}
		cmd = exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	}

	cmd.Dir = workingDir

	// Set up environment
	cmd.Env = append(os.Environ(),
		"CLAUDE_SESSION=true",
		"CLAUDE_BOT=true",
	)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.logger.Info("Executing command",
		zap.String("command", command),
		zap.String("working_dir", workingDir))

	// Execute command
	err := cmd.Run()
	result.Duration = time.Since(start)

	// Process output
	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// Limit output length
	if len(stdoutStr) > e.config.MaxOutputLength {
		stdoutStr = stdoutStr[:e.config.MaxOutputLength] + "\n... (output truncated)"
	}
	if len(stderrStr) > e.config.MaxOutputLength {
		stderrStr = stderrStr[:e.config.MaxOutputLength] + "\n... (output truncated)"
	}

	result.Output = stdoutStr
	if stderrStr != "" {
		if result.Output != "" {
			result.Output += "\n--- STDERR ---\n"
		}
		result.Output += stderrStr
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
		result.Error = err.Error()

		// Don't treat non-zero exit codes as errors for logging
		if result.ExitCode != 0 {
			e.logger.Debug("Command completed with non-zero exit code",
				zap.String("command", command),
				zap.Int("exit_code", result.ExitCode),
				zap.Duration("duration", result.Duration))
		}
	} else {
		result.ExitCode = 0
		e.logger.Debug("Command completed successfully",
			zap.String("command", command),
			zap.Duration("duration", result.Duration))
	}

	return result, nil
}

// GetClaudeCodeSystemPrompt returns the system prompt for Claude Code
func (e *Executor) GetClaudeCodeSystemPrompt() string {
	return `You are Claude Code, Anthropic's official CLI for Claude. You are an agent for Claude Code, running in a Slack bot environment.

Your capabilities:
- Execute shell commands and scripts
- Read and write files
- Analyze code and provide solutions
- Help with development tasks
- Search through codebases
- Debug issues and provide explanations

Guidelines:
- Always be helpful, accurate, and safe
- When executing commands, explain what you're doing
- If a command might be dangerous, ask for confirmation
- Provide clear explanations of your actions
- Format code blocks properly for Slack
- Keep responses concise but informative
- If you need to run multiple commands, explain your plan first

Security considerations:
- Never execute commands that could harm the system
- Don't access sensitive files without permission
- Ask before making significant changes
- Validate user requests for safety

Working directory: ` + e.config.WorkingDirectory + `
Available commands are filtered for security.`
}

// ProcessClaudeCodeRequest processes a request using Claude Code CLI
func (e *Executor) ProcessClaudeCodeRequest(ctx context.Context, userMessage string, sessionID string, userID string, allowedTools []string, isNewSession bool, permissionMode config.PermissionMode) (string, string, float64, string, error) {
	// Use configured working directory instead of isolated workspace for full system access
	workingDir := e.config.WorkingDirectory
	if workingDir == "" {
		// Default to user's home directory for full access
		homeDir, err := os.UserHomeDir()
		if err == nil {
			workingDir = homeDir
		} else {
			workingDir = "." // Fallback to current directory
		}
	}
	
	// Ensure working directory exists
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		e.logger.Error("Failed to create working directory", zap.Error(err))
		return "", "", 0, "", fmt.Errorf("failed to create working directory: %w", err)
	}

	e.logger.Info("Processing Claude Code request",
		zap.String("user_id", userID),
		zap.String("session_id", sessionID),
		zap.String("working_dir", workingDir))

	// Execute Claude Code CLI
	response, err := e.ExecuteClaudeCode(ctx, userMessage, sessionID, workingDir, allowedTools, isNewSession, permissionMode)
	if err != nil {
		e.logger.Error("Failed to execute Claude Code", zap.Error(err))
		return "", "", 0, "", fmt.Errorf("failed to execute Claude Code: %w", err)
	}

	e.logger.Info("Claude Code request completed",
		zap.String("user_id", userID),
		zap.String("session_id", response.SessionID),
		zap.Float64("cost_usd", response.TotalCostUSD),
		zap.Int("input_tokens", response.Usage.InputTokens),
		zap.Int("output_tokens", response.Usage.OutputTokens))

	return response.Result, response.SessionID, response.TotalCostUSD, response.LatestResponse, nil
}

// CreateWorkspace creates a dedicated workspace directory for a user session
func (e *Executor) CreateWorkspace(userID, sessionID string) (string, error) {
	workspaceDir := filepath.Join(e.config.WorkingDirectory, "sessions", userID, sessionID)
	
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		e.logger.Error("Failed to create workspace", 
			zap.Error(err), 
			zap.String("workspace", workspaceDir))
		return "", fmt.Errorf("failed to create workspace: %w", err)
	}

	e.logger.Info("Created workspace", 
		zap.String("workspace", workspaceDir),
		zap.String("user_id", userID),
		zap.String("session_id", sessionID))

	return workspaceDir, nil
}

// CleanupWorkspace removes a workspace directory
func (e *Executor) CleanupWorkspace(workspaceDir string) error {
	if workspaceDir == "" || !strings.Contains(workspaceDir, e.config.WorkingDirectory) {
		return fmt.Errorf("invalid workspace directory")
	}

	if err := os.RemoveAll(workspaceDir); err != nil {
		e.logger.Error("Failed to cleanup workspace", zap.Error(err), zap.String("workspace", workspaceDir))
		return fmt.Errorf("failed to cleanup workspace: %w", err)
	}

	e.logger.Info("Cleaned up workspace", zap.String("workspace", workspaceDir))
	return nil
}