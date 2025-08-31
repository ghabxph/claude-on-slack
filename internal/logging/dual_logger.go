package logging

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// DualLogger provides centralized error logging to both console and Slack
type DualLogger struct {
	zapLogger *zap.Logger
	slackAPI  *slack.Client
}

// ErrorContext contains context information for error logging
type ErrorContext struct {
	ChannelID     string
	UserID        string
	Component     string
	Operation     string
	SessionID     string
}

// NewDualLogger creates a new dual logger instance
func NewDualLogger(zapLogger *zap.Logger, slackAPI *slack.Client) *DualLogger {
	return &DualLogger{
		zapLogger: zapLogger,
		slackAPI:  slackAPI,
	}
}

// LogError logs an error to both console and Slack channel
func (dl *DualLogger) LogError(ctx context.Context, errCtx *ErrorContext, err error, message string) {
	// Always log to console first
	dl.logToConsole(errCtx, err, message)
	
	// If we have a channel ID, also send to Slack
	if errCtx.ChannelID != "" {
		dl.logToSlack(ctx, errCtx, err, message)
	}
}

// LogErrorf logs a formatted error message to both console and Slack
func (dl *DualLogger) LogErrorf(ctx context.Context, errCtx *ErrorContext, err error, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	dl.LogError(ctx, errCtx, err, message)
}

// logToConsole logs detailed error information to console
func (dl *DualLogger) logToConsole(errCtx *ErrorContext, err error, message string) {
	stack := string(debug.Stack())
	
	fields := []zap.Field{
		zap.String("component", errCtx.Component),
		zap.String("operation", errCtx.Operation),
		zap.String("channel_id", errCtx.ChannelID),
		zap.String("user_id", errCtx.UserID),
		zap.String("session_id", errCtx.SessionID),
		zap.Error(err),
		zap.String("stack_trace", stack),
	}
	
	dl.zapLogger.Error(message, fields...)
}

// logToSlack sends error information to the Slack channel
func (dl *DualLogger) logToSlack(ctx context.Context, errCtx *ErrorContext, err error, message string) {
	// Create a user-friendly error message for Slack
	slackMessage := dl.formatSlackMessage(errCtx, err, message)
	
	// Send ephemeral message (only visible to the user who triggered the error)
	_, err = dl.slackAPI.PostEphemeral(
		errCtx.ChannelID,
		errCtx.UserID,
		slack.MsgOptionText(slackMessage, false),
		slack.MsgOptionAsUser(false),
	)
	
	// If posting to Slack fails, log it to console but don't create an infinite loop
	if err != nil {
		dl.zapLogger.Error("Failed to post error message to Slack",
			zap.String("channel_id", errCtx.ChannelID),
			zap.Error(err))
	}
}

// formatSlackMessage creates a user-friendly error message for Slack
func (dl *DualLogger) formatSlackMessage(errCtx *ErrorContext, err error, message string) string {
	// Get simplified stack trace for location info
	stack := string(debug.Stack())
	location := dl.extractLocation(stack)
	
	// Create timestamp for this error
	timestamp := time.Now().Format("15:04:05")
	
	// Format the message
	var parts []string
	parts = append(parts, fmt.Sprintf("🚨 **Error in %s** [%s]", errCtx.Component, timestamp))
	parts = append(parts, fmt.Sprintf("**Operation**: %s", errCtx.Operation))
	parts = append(parts, fmt.Sprintf("**Message**: %s", message))
	parts = append(parts, fmt.Sprintf("**Error**: %v", err))
	
	if location != "unknown" {
		parts = append(parts, fmt.Sprintf("**Location**: %s", location))
	}
	
	if errCtx.SessionID != "" {
		parts = append(parts, fmt.Sprintf("**Session**: %s", errCtx.SessionID))
	}
	
	parts = append(parts, "")
	parts = append(parts, "_This error has been automatically logged for debugging._")
	
	return strings.Join(parts, "\n")
}

// extractLocation extracts the relevant location from stack trace
func (dl *DualLogger) extractLocation(stack string) string {
	stackLines := strings.Split(stack, "\n")
	
	for i, line := range stackLines {
		if strings.Contains(line, "claude-on-slack/internal/") && !strings.Contains(line, "logging/dual_logger.go") {
			location := strings.TrimSpace(line)
			// Add line number info if available
			if i+1 < len(stackLines) {
				nextLine := strings.TrimSpace(stackLines[i+1])
				if strings.Contains(nextLine, ":") {
					parts := strings.Split(nextLine, ":")
					if len(parts) >= 2 {
						location = fmt.Sprintf("%s:%s", location, parts[1])
					}
				}
			}
			return location
		}
	}
	
	return "unknown"
}

// CreateErrorContext creates an ErrorContext from common parameters
func CreateErrorContext(channelID, userID, component, operation string) *ErrorContext {
	return &ErrorContext{
		ChannelID: channelID,
		UserID:    userID,
		Component: component,
		Operation: operation,
	}
}

// WithSession adds session information to an ErrorContext
func (ec *ErrorContext) WithSession(sessionID string) *ErrorContext {
	ec.SessionID = sessionID
	return ec
}