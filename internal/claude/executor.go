package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
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
	
	// Add image storage directory for file access
	imageStorageDir := "/tmp/claude-slack-images"
	args = append(args, "--add-dir", imageStorageDir)
	
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
	
	// Log the complete command for debugging
	fullCommand := fmt.Sprintf("echo '%s' | %s %s", userMessage, e.claudeCodePath, strings.Join(args, " "))
	
	e.logger.Info("Executing Claude Code CLI",
		zap.String("session_id", sessionID),
		zap.String("working_dir", workingDir),
		zap.Strings("allowed_tools", allowedTools),
		zap.Strings("args", args),
		zap.Bool("is_new_session", isNewSession),
		zap.String("full_command", fullCommand))
	
	// Execute command
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	
	if err != nil {
		stderrOutput := strings.TrimSpace(stderr.String())
		e.logger.Error("Claude Code CLI execution failed",
			zap.Error(err),
			zap.String("stderr", stderrOutput),
			zap.Duration("duration", duration))
		
		// Create enhanced error message with stderr details and debug info
		debugInfo := map[string]interface{}{
			"session_id":     sessionID,
			"is_new_session": isNewSession,
			"working_dir":    workingDir,
			"args":          args,
			"full_command":  fullCommand,
		}
		enhancedErr := e.createEnhancedError(err, stderrOutput, duration, debugInfo)
		return nil, enhancedErr
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

// createEnhancedError creates a detailed error message with context and troubleshooting information
func (e *Executor) createEnhancedError(originalErr error, stderrOutput string, duration time.Duration, debugInfo map[string]interface{}) error {
	// Parse the original error for specific patterns
	errorType := e.categorizeError(originalErr, stderrOutput)
	
	// Create base error message
	baseMsg := fmt.Sprintf("Claude Code execution failed after %v", duration.Truncate(time.Millisecond))
	
	// Format debug information
	debugMsg := fmt.Sprintf("**Debug Information:**\n• Session ID: `%v`\n• New Session: `%v`\n• Working Dir: `%v`\n• Command: `%v`",
		debugInfo["session_id"], debugInfo["is_new_session"], debugInfo["working_dir"], debugInfo["full_command"])
	
	// Add specific error details based on type
	switch errorType {
	case "permission_denied":
		return fmt.Errorf("%s\n\n🔒 **Permission Denied**\nThe system denied access to required resources.\n\n**Stderr Output:**\n```\n%s\n```\n\n**Troubleshooting:**\n• Check file/directory permissions\n• Verify you have access to the working directory\n• Try running with appropriate privileges", baseMsg, stderrOutput)
	
	case "command_not_found":
		return fmt.Errorf("%s\n\n❌ **Command Not Found**\nA required command or binary was not found.\n\n**Stderr Output:**\n```\n%s\n```\n\n**Troubleshooting:**\n• Check if the required tool is installed\n• Verify PATH environment variable\n• Install missing dependencies", baseMsg, stderrOutput)
	
	case "syntax_error":
		return fmt.Errorf("%s\n\n⚠️ **Syntax Error**\nCode or command syntax is invalid.\n\n**Stderr Output:**\n```\n%s\n```\n\n**Troubleshooting:**\n• Review the code syntax\n• Check for typos in commands\n• Validate file formats", baseMsg, stderrOutput)
	
	case "network_error":
		return fmt.Errorf("%s\n\n🌐 **Network Error**\nNetwork connectivity or timeout issue.\n\n**Stderr Output:**\n```\n%s\n```\n\n**Troubleshooting:**\n• Check internet connection\n• Verify network settings\n• Try again after a moment", baseMsg, stderrOutput)
	
	case "file_not_found":
		return fmt.Errorf("%s\n\n📁 **File Not Found**\nRequired file or directory does not exist.\n\n**Stderr Output:**\n```\n%s\n```\n\n**Troubleshooting:**\n• Check file paths are correct\n• Verify files exist in expected locations\n• Check working directory", baseMsg, stderrOutput)
	
	case "timeout":
		return fmt.Errorf("%s\n\n⏱️ **Operation Timeout**\nThe operation took too long to complete.\n\n**Stderr Output:**\n```\n%s\n```\n\n**Troubleshooting:**\n• Operation may require more time\n• Check system resources\n• Try breaking down into smaller tasks", baseMsg, stderrOutput)
	
	default:
		// Generic error with full stderr output and debug info
		if stderrOutput != "" {
			return fmt.Errorf("%s\n\n🚨 **Execution Error**\nOriginal error: %v\n\n**Stderr Output:**\n```\n%s\n```\n\n%s\n\n**Troubleshooting:**\n• Review the error details above\n• Check system logs for more information\n• Verify all requirements are met", baseMsg, originalErr, stderrOutput, debugMsg)
		} else {
			return fmt.Errorf("%s\n\n🚨 **Execution Error**\nOriginal error: %v\n\n%s\n\n**Troubleshooting:**\n• Check system logs for more information\n• Try running the command manually\n• Verify Claude Code CLI is properly installed", baseMsg, originalErr, debugMsg)
		}
	}
}

// categorizeError analyzes the error and stderr to determine the error type
func (e *Executor) categorizeError(originalErr error, stderrOutput string) string {
	// Convert to lowercase for easier matching
	errorStr := strings.ToLower(originalErr.Error())
	stderrLower := strings.ToLower(stderrOutput)
	
	// Combined text for analysis
	combinedText := errorStr + " " + stderrLower
	
	// Check for permission errors
	if strings.Contains(combinedText, "permission denied") ||
		strings.Contains(combinedText, "access denied") ||
		strings.Contains(combinedText, "operation not permitted") ||
		strings.Contains(combinedText, "insufficient privileges") {
		return "permission_denied"
	}
	
	// Check for command not found errors
	if strings.Contains(combinedText, "command not found") ||
		strings.Contains(combinedText, "no such file or directory") && strings.Contains(combinedText, "/bin/") ||
		strings.Contains(combinedText, "executable file not found") ||
		strings.Contains(combinedText, "not found in path") {
		return "command_not_found"
	}
	
	// Check for syntax errors
	if strings.Contains(combinedText, "syntax error") ||
		strings.Contains(combinedText, "invalid syntax") ||
		strings.Contains(combinedText, "parse error") ||
		strings.Contains(combinedText, "unexpected token") ||
		strings.Contains(combinedText, "invalid character") {
		return "syntax_error"
	}
	
	// Check for network errors
	if strings.Contains(combinedText, "network") ||
		strings.Contains(combinedText, "connection refused") ||
		strings.Contains(combinedText, "timeout") ||
		strings.Contains(combinedText, "dns") ||
		strings.Contains(combinedText, "unreachable") ||
		strings.Contains(combinedText, "connection timed out") {
		return "network_error"
	}
	
	// Check for file not found errors
	if strings.Contains(combinedText, "no such file") ||
		strings.Contains(combinedText, "file not found") ||
		strings.Contains(combinedText, "directory not found") ||
		strings.Contains(combinedText, "cannot find") && (strings.Contains(combinedText, "file") || strings.Contains(combinedText, "directory")) {
		return "file_not_found"
	}
	
	// Check for timeout errors
	if strings.Contains(combinedText, "timeout") ||
		strings.Contains(combinedText, "deadline exceeded") ||
		strings.Contains(combinedText, "context deadline exceeded") ||
		strings.Contains(combinedText, "operation timed out") {
		return "timeout"
	}
	
	return "generic"
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
func (e *Executor) ProcessClaudeCodeRequest(ctx context.Context, userMessage string, sessionID string, userID string, workingDir string, allowedTools []string, isNewSession bool, permissionMode config.PermissionMode) (string, string, float64, string, error) {
	// Use provided working directory, fallback to config if empty
	if workingDir == "" {
		workingDir = e.config.WorkingDirectory
		if workingDir == "" {
			// Default to user's home directory for full access
			homeDir, err := os.UserHomeDir()
			if err == nil {
				workingDir = homeDir
			} else {
				workingDir = "." // Fallback to current directory
			}
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
	// Just use the base working directory - no nested sessions folders
	workspaceDir := e.config.WorkingDirectory
	
	// Ensure directory exists
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		e.logger.Error("Failed to create workspace", 
			zap.Error(err), 
			zap.String("workspace", workspaceDir))
		return "", fmt.Errorf("failed to create workspace: %w", err)
	}

	e.logger.Info("Using workspace", 
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

// ExecuteClaudeSummary executes Claude Code CLI for conversation summarization
// This is a "disposable" call that doesn't require session management
func (e *Executor) ExecuteClaudeSummary(ctx context.Context, conversationText string) (string, error) {
	// Create specialized system prompt for detailed summarization
	systemPrompt := `You are a conversation summarization specialist. Create a detailed, comprehensive summary of this technical conversation between a user and AI assistant. 

Your summary should be optimized for context compression while preserving maximum detail for task continuity. Focus on:

**1. Primary Goals & Tasks**: What the user was trying to accomplish, including specific objectives and requirements
**2. Technical Context**: Code changes, files modified, technologies involved, architectural decisions, and system interactions  
**3. Problem-Solving Journey**: Issues encountered, debugging steps taken, solutions implemented, and approaches tried
**4. Key Decisions**: Important choices made, reasoning behind them, trade-offs considered, and alternative approaches discussed
**5. Current State**: Where the conversation left off, what was completed, what's in progress, and system state
**6. Next Steps**: Logical follow-up actions, pending tasks, known blockers, and recommended approaches

**Critical Requirements:**
- Be extremely detailed - this summary will be used to continue work seamlessly in a new conversation
- Include specific technical details: file paths, function names, error messages, configuration changes
- Preserve context about what worked, what didn't work, and why
- Include relevant code snippets, commands, and configurations mentioned
- Note any important discoveries, insights, or lessons learned
- Use clear, structured formatting for easy parsing

The summary should be comprehensive enough that someone could read it and immediately understand the full context to continue the technical work without missing any important details.`

	// Prepare Claude Code CLI arguments for disposable summarization
	args := []string{
		"--print",
		"--output-format", "json",
		"--model", "sonnet",
		"--session-id", uuid.New().String(), // Disposable session ID
	}

	// Add working directory (use current bot working directory)
	if e.config.WorkingDirectory != "" {
		args = append(args, "--add-dir", e.config.WorkingDirectory)
	}

	// Add system prompt as a separate argument
	args = append(args, "--append-system-prompt", systemPrompt)

	// Prepare the user message (conversation to summarize)
	userMessage := fmt.Sprintf("**CONVERSATION TO SUMMARIZE:**\n\n%s", conversationText)

	// Execute Claude Code CLI
	cmd := exec.CommandContext(ctx, e.claudeCodePath, args...)
	cmd.Dir = e.config.WorkingDirectory

	// Set up stdin with user message (to avoid command line escaping issues)
	cmd.Stdin = strings.NewReader(userMessage)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	e.logger.Info("Executing Claude summarization",
		zap.String("claude_path", e.claudeCodePath),
		zap.String("working_dir", e.config.WorkingDirectory),
		zap.Int("conversation_length", len(conversationText)))

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		e.logger.Error("Claude summarization failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()),
			zap.Duration("duration", duration))

		return "", fmt.Errorf("claude summarization failed after %v: %v\nStderr: %s", 
			duration.Truncate(time.Millisecond), err, stderr.String())
	}

	output := stdout.Bytes()

	// Parse Claude response 
	var response ClaudeCodeResponse
	if err := json.Unmarshal(output, &response); err != nil {
		e.logger.Error("Failed to parse Claude summarization response",
			zap.Error(err),
			zap.String("raw_output", string(output)))
		return "", fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Check for Claude-level errors
	if response.IsError {
		e.logger.Error("Claude returned an error during summarization",
			zap.String("error", response.Error),
			zap.String("result", response.Result))
		return "", fmt.Errorf("claude summarization error: %s", response.Error)
	}

	e.logger.Info("Claude summarization completed",
		zap.Duration("duration", duration),
		zap.Float64("cost_usd", response.TotalCostUSD),
		zap.Int("input_tokens", response.Usage.InputTokens),
		zap.Int("output_tokens", response.Usage.OutputTokens),
		zap.Int("summary_length", len(response.Result)))

	return response.Result, nil
}