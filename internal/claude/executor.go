package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/config"
)

// Executor handles Claude API communication and command execution
type Executor struct {
	config     *config.Config
	logger     *zap.Logger
	httpClient *http.Client
}

// Message represents a message in the conversation
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest represents the request to Claude API
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
	System    string    `json:"system,omitempty"`
}

// ClaudeResponse represents the response from Claude API
type ClaudeResponse struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Role         string                 `json:"role"`
	Content      []ClaudeContentBlock   `json:"content"`
	Model        string                 `json:"model"`
	StopReason   string                 `json:"stop_reason"`
	StopSequence string                 `json:"stop_sequence"`
	Usage        ClaudeUsage           `json:"usage"`
	Error        *ClaudeError          `json:"error,omitempty"`
}

// ClaudeContentBlock represents a content block in Claude's response
type ClaudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeUsage represents token usage information
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeError represents an error from Claude API
type ClaudeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
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

// NewExecutor creates a new Claude executor
func NewExecutor(cfg *config.Config, logger *zap.Logger) *Executor {
	return &Executor{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: cfg.ClaudeTimeout,
		},
	}
}

// ExecuteClaudeRequest sends a request to Claude API
func (e *Executor) ExecuteClaudeRequest(ctx context.Context, messages []Message, systemPrompt string) (*ClaudeResponse, error) {
	request := ClaudeRequest{
		Model:     e.config.ClaudeModel,
		MaxTokens: e.config.ClaudeMaxTokens,
		Messages:  messages,
		System:    systemPrompt,
	}

	reqBody, err := json.Marshal(request)
	if err != nil {
		e.logger.Error("Failed to marshal request", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.config.ClaudeAPIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		e.logger.Error("Failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.config.ClaudeAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	e.logger.Debug("Sending request to Claude API",
		zap.String("model", request.Model),
		zap.Int("max_tokens", request.MaxTokens),
		zap.Int("message_count", len(messages)))

	resp, err := e.httpClient.Do(req)
	if err != nil {
		e.logger.Error("Failed to send request", zap.Error(err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		e.logger.Error("Failed to read response", zap.Error(err))
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		e.logger.Error("Failed to unmarshal response", zap.Error(err), zap.String("body", string(respBody)))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		e.logger.Error("Claude API returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.Any("error", claudeResp.Error))
		return nil, fmt.Errorf("claude API error: %s", claudeResp.Error.Message)
	}

	if claudeResp.Error != nil {
		e.logger.Error("Claude returned error in response", zap.Any("error", claudeResp.Error))
		return nil, fmt.Errorf("claude error: %s", claudeResp.Error.Message)
	}

	e.logger.Debug("Received response from Claude",
		zap.String("id", claudeResp.ID),
		zap.String("stop_reason", claudeResp.StopReason),
		zap.Int("input_tokens", claudeResp.Usage.InputTokens),
		zap.Int("output_tokens", claudeResp.Usage.OutputTokens))

	return &claudeResp, nil
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
	if strings.Contains(command, "|") || strings.Contains(command, "&&") || strings.Contains(command, "||") || strings.Contains(command, ";") {
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

// ProcessClaudeCodeRequest processes a Claude Code request with command execution capabilities
func (e *Executor) ProcessClaudeCodeRequest(ctx context.Context, userMessage string, conversationHistory []Message, userID string) (string, error) {
	// Add system prompt
	systemPrompt := e.GetClaudeCodeSystemPrompt()

	// Prepare messages
	messages := make([]Message, 0, len(conversationHistory)+1)
	messages = append(messages, conversationHistory...)
	messages = append(messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	e.logger.Info("Processing Claude Code request",
		zap.String("user_id", userID),
		zap.Int("history_length", len(conversationHistory)))

	// Send request to Claude
	response, err := e.ExecuteClaudeRequest(ctx, messages, systemPrompt)
	if err != nil {
		e.logger.Error("Failed to get Claude response", zap.Error(err))
		return "", fmt.Errorf("failed to get Claude response: %w", err)
	}

	// Extract text from response
	var responseText strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			responseText.WriteString(block.Text)
		}
	}

	result := responseText.String()

	// Check if Claude wants to execute commands (basic detection)
	if strings.Contains(result, "```bash") || strings.Contains(result, "```sh") || strings.Contains(result, "```shell") {
		e.logger.Info("Claude response contains shell commands",
			zap.String("user_id", userID))
		
		// For now, just return the response
		// In a more advanced implementation, we could parse and execute the commands
		// But that would require more sophisticated parsing and safety checks
	}

	return result, nil
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