package bot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/auth"
	"github.com/ghabxph/claude-on-slack/internal/claude"
	"github.com/ghabxph/claude-on-slack/internal/config"
	"github.com/ghabxph/claude-on-slack/internal/session"
)

// Service represents the main bot service
type Service struct {
	config         *config.Config
	logger         *zap.Logger
	slackAPI       *slack.Client
	socketClient   *socketmode.Client
	httpServer     *http.Server
	authService    *auth.Service
	sessionManager *session.Manager
	claudeExecutor *claude.Executor
	stopCh         chan struct{}
	wg             sync.WaitGroup
	botUserID      string
	startTime      time.Time
}

// CommandHandler represents a command handler function
type CommandHandler func(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error)

// commandRegistry holds all registered commands
var commandRegistry = make(map[string]CommandHandler)

// NewService creates a new bot service
func NewService(cfg *config.Config, logger *zap.Logger) (*Service, error) {
	// Initialize Slack clients
	slackAPI := slack.New(cfg.SlackBotToken, slack.OptionDebug(cfg.EnableDebug), slack.OptionAppLevelToken(cfg.SlackAppToken))
	socketClient := socketmode.New(slackAPI, socketmode.OptionDebug(cfg.EnableDebug))

	// Initialize other services
	authService := auth.NewService(cfg, logger)
	claudeExecutor, err := claude.NewExecutor(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude executor: %w", err)
	}
	sessionManager := session.NewManager(cfg, logger, claudeExecutor)

	service := &Service{
		config:         cfg,
		logger:         logger,
		slackAPI:       slackAPI,
		socketClient:   socketClient,
		authService:    authService,
		sessionManager: sessionManager,
		claudeExecutor: claudeExecutor,
		stopCh:         make(chan struct{}),
		startTime:      time.Now(),
	}

	// Register built-in commands
	service.registerCommands()

	return service, nil
}

// Start starts the bot service
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("Starting Claude on Slack bot",
		zap.String("bot_name", s.config.BotName),
		zap.String("command_prefix", s.config.CommandPrefix))

	// Get bot user info
	authResp, err := s.slackAPI.AuthTest()
	if err != nil {
		return fmt.Errorf("failed to authenticate with Slack: %w", err)
	}
	s.botUserID = authResp.UserID

	s.logger.Info("Bot authenticated",
		zap.String("bot_user_id", s.botUserID),
		zap.String("team", authResp.Team),
		zap.String("user", authResp.User))

	// Set bot presence to online
	err = s.slackAPI.SetUserPresence("auto")
	if err != nil {
		s.logger.Debug("Failed to set bot presence", zap.Error(err))
	} else {
		s.logger.Info("Bot presence set to online")
	}

	// Start HTTP server for Events API
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.startHTTPServer()
	}()

	// Start event handling for Socket Mode
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.handleEvents()
	}()

	// Start periodic cleanup
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.periodicCleanup()
	}()

	// Start socket mode client (will only work if app is configured for Socket Mode)
	go func() {
		if err := s.socketClient.Run(); err != nil {
			s.logger.Debug("Socket Mode not available or disabled", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the bot service
func (s *Service) Stop() {
	s.logger.Info("Stopping Claude on Slack bot")

	close(s.stopCh)

	// Stop HTTP server
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}

	s.wg.Wait()

	if s.sessionManager != nil {
		s.sessionManager.Stop()
	}

	s.logger.Info("Bot stopped successfully")
}

// handleEvents handles incoming Slack events
func (s *Service) handleEvents() {
	for {
		select {
		case envelope := <-s.socketClient.Events:
			switch envelope.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := envelope.Data.(slackevents.EventsAPIEvent)
				if !ok {
					s.logger.Warn("Failed to type assert events API event")
					continue
				}
				s.handleEventsAPIEvent(&eventsAPIEvent)
				s.socketClient.Ack(*envelope.Request)

			case socketmode.EventTypeSlashCommand:
				slashCommand, ok := envelope.Data.(slack.SlashCommand)
				if !ok {
					s.logger.Warn("Failed to type assert slash command")
					continue
				}
				s.handleSlashCommand(&slashCommand)
				s.socketClient.Ack(*envelope.Request)

			case socketmode.EventTypeInteractive:
				callback, ok := envelope.Data.(slack.InteractionCallback)
				if !ok {
					s.logger.Warn("Failed to type assert interaction callback")
					continue
				}
				s.handleInteractiveEvent(&callback)
				s.socketClient.Ack(*envelope.Request)

			default:
				s.logger.Debug("Received unhandled event", zap.String("type", string(envelope.Type)))
			}

		case <-s.stopCh:
			return
		}
	}
}

// handleEventsAPIEvent handles Events API events
func (s *Service) handleEventsAPIEvent(event *slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch innerEvent.Type {
		case "message":
			messageEvent, ok := innerEvent.Data.(*slackevents.MessageEvent)
			if !ok {
				s.logger.Warn("Failed to type assert message event")
				return
			}
			s.handleMessageEvent(messageEvent)

		case "app_mention":
			mentionEvent, ok := innerEvent.Data.(*slackevents.AppMentionEvent)
			if !ok {
				s.logger.Warn("Failed to type assert mention event")
				return
			}
			s.handleMentionEvent(mentionEvent)
		}
	}
}

// handleMessageEvent handles message events
func (s *Service) handleMessageEvent(event *slackevents.MessageEvent) {
	// Ignore bot messages, messages from the bot itself, and messages with empty user ID
	if event.BotID != "" || event.User == s.botUserID || event.User == "" {
		return
	}

	// Check if message starts with command prefix or mentions the bot
	text := strings.TrimSpace(event.Text)
	isMentioned := strings.HasPrefix(text, s.config.CommandPrefix) || strings.Contains(text, fmt.Sprintf("<@%s>", s.botUserID))
	isAutoResponseChannel := s.config.IsAutoResponseChannel(event.Channel)

	// Respond if bot is mentioned OR if it's an auto-response channel
	if !isMentioned && !isAutoResponseChannel {
		return
	}

	s.logger.Debug("Processing message",
		zap.String("user_id", event.User),
		zap.String("channel_id", event.Channel),
		zap.String("text", event.Text))

	ctx := context.Background()
	response := s.processMessage(ctx, event)

	if response != "" {
		s.sendResponse(event.Channel, response)
	}
}

// handleMentionEvent handles app mention events
func (s *Service) handleMentionEvent(event *slackevents.AppMentionEvent) {
	if event.BotID != "" || event.User == s.botUserID {
		return
	}

	// Convert mention event to message event format
	messageEvent := &slackevents.MessageEvent{
		Type:        "message",
		User:        event.User,
		Text:        event.Text,
		TimeStamp:   event.TimeStamp,
		Channel:     event.Channel,
		ChannelType: "channel", // Default since AppMentionEvent doesn't have ChannelType
	}

	s.handleMessageEvent(messageEvent)
}

// handleSlashCommand handles slash commands
func (s *Service) handleSlashCommand(command *slack.SlashCommand) {
	ctx := context.Background()
	response := s.processSlashCommand(ctx, command)

	if response != "" {
		s.sendResponse(command.ChannelID, response)
	}
}

// handleInteractiveEvent handles interactive events (buttons, modals, etc.)
func (s *Service) handleInteractiveEvent(callback *slack.InteractionCallback) {
	s.logger.Debug("Received interactive event",
		zap.String("type", string(callback.Type)),
		zap.String("user_id", callback.User.ID))

	// Handle different interaction types
	switch callback.Type {
	case slack.InteractionTypeBlockActions:
		s.handleBlockActions(callback)
	case slack.InteractionTypeShortcut:
		s.handleShortcut(callback)
	default:
		s.logger.Debug("Unhandled interaction type", zap.String("type", string(callback.Type)))
	}
}

// processMessage processes incoming messages
func (s *Service) processMessage(ctx context.Context, event *slackevents.MessageEvent) string {
	// Create auth context
	authCtx := &auth.AuthContext{
		UserID:    event.User,
		ChannelID: event.Channel,
		Timestamp: time.Now(),
	}

	// Check authorization
	if err := s.authService.AuthorizeUser(authCtx, auth.PermissionRead); err != nil {
		s.logger.Warn("Authorization failed", zap.Error(err))
		return fmt.Sprintf("❌ Authorization failed: %v", err)
	}

	// Parse message
	text := strings.TrimSpace(event.Text)

	// Remove bot mention if present
	mentionPattern := fmt.Sprintf("<@%s>", s.botUserID)
	text = strings.ReplaceAll(text, mentionPattern, "")
	text = strings.TrimSpace(text)

	// Remove command prefix if present
	if strings.HasPrefix(text, s.config.CommandPrefix) {
		text = strings.TrimPrefix(text, s.config.CommandPrefix)
		text = strings.TrimSpace(text)
	}

	// Check if it's a specific bot command (help, status, etc.)
	if strings.HasPrefix(text, "help") && len(strings.Fields(text)) == 1 {
		return s.getHelpMessage()
	}
	if strings.HasPrefix(text, "status") && len(strings.Fields(text)) == 1 {
		response, _ := s.handleStatusCommand(ctx, event, []string{})
		return response
	}
	if strings.HasPrefix(text, "version") && len(strings.Fields(text)) == 1 {
		response, _ := s.handleVersionCommand(ctx, event, []string{})
		return response
	}

	// Process everything else as Claude conversation (natural language)
	return s.processClaudeMessage(ctx, event, text)
}

// processCommand processes bot commands
func (s *Service) processCommand(ctx context.Context, event *slackevents.MessageEvent, text string) string {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return s.getHelpMessage()
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	s.logger.Info("Processing command",
		zap.String("command", command),
		zap.Strings("args", args),
		zap.String("user_id", event.User))

	// Check if command exists
	handler, exists := commandRegistry[command]
	if !exists {
		return fmt.Sprintf("❌ Unknown command: `%s`. Type `help` for available commands.", command)
	}

	// Execute command
	response, err := handler(ctx, event, args)
	if err != nil {
		s.logger.Error("Command execution failed", zap.Error(err))
		return fmt.Sprintf("❌ Command failed: %v", err)
	}

	return response
}

// processClaudeMessage processes Claude conversation messages
func (s *Service) processClaudeMessage(ctx context.Context, event *slackevents.MessageEvent, text string) string {
	// Get or create session
	userSession, err := s.sessionManager.GetOrCreateSession(event.User, event.Channel)
	if err != nil {
		s.logger.Error("Failed to get session", zap.Error(err))
		return fmt.Sprintf("❌ Failed to create session: %v", err)
	}

	// Check if we should queue this message
	queued, err := s.sessionManager.QueueMessage(userSession.ID, text)
	if err != nil {
		s.logger.Error("Failed to check message queue", zap.Error(err))
		return fmt.Sprintf("❌ Failed to process message: %v", err)
	}

	if queued {
		return "" // Message queued, no response needed yet
	}

	// Check rate limiting
	limited, remaining, err := s.sessionManager.CheckRateLimit(userSession.ID)
	if err != nil {
		s.logger.Error("Rate limit check failed", zap.Error(err))
		return "❌ Failed to check rate limit"
	}

	if limited {
		return fmt.Sprintf("⏱️ Rate limit exceeded. Try again in %v", remaining.Truncate(time.Second))
	}

	// Mark as processing
	if err := s.sessionManager.SetProcessing(userSession.ID, true); err != nil {
		s.logger.Error("Failed to set processing state", zap.Error(err))
		return fmt.Sprintf("❌ Failed to process message: %v", err)
	}
	defer s.sessionManager.SetProcessing(userSession.ID, false)

	// Get any queued messages and combine with current message
	queuedMessages, err := s.sessionManager.GetQueuedMessages(userSession.ID)
	if err != nil {
		s.logger.Error("Failed to get queued messages", zap.Error(err))
		return fmt.Sprintf("❌ Failed to process message: %v", err)
	}

	if len(queuedMessages) > 0 {
		text = strings.Join(append([]string{text}, queuedMessages...), " ")
	}

	// Send "Thinking..." message immediately and capture for deletion
	// Get current mode
	currentMode, err := s.sessionManager.GetPermissionMode(userSession.ID)
	if err != nil {
		currentMode = config.PermissionModeDefault
	}
	
	// Format Thinking message with Mode, Session, and Working Dir
	thinkingMsg := fmt.Sprintf("🤔 Thinking... (Mode: `%s`, Session: `%s`, Working Dir: `%s`)",
		currentMode, userSession.ClaudeSessionID, userSession.CurrentWorkDir)
	
	_, thinkingTimestamp, err := s.slackAPI.PostMessage(event.Channel, slack.MsgOptionText(thinkingMsg, false))
	if err != nil {
		s.logger.Error("Failed to send thinking message", zap.Error(err))
		thinkingTimestamp = "" // Ensure it's empty if posting failed
	}

	// Get allowed tools for this user
	// Empty AllowedTools means all tools are allowed (full system access)
	allowedTools := s.config.AllowedTools

	// If no tools specified (empty array), allow all tools by passing empty array to Claude Code
	// Claude Code will use all available tools when no --allowedTools is specified
	if len(allowedTools) == 0 {
		allowedTools = []string{} // Empty means all tools
	} else {
		// Filter out disallowed tools if specific tools are configured
		filteredTools := []string{}
		for _, tool := range allowedTools {
			isDisallowed := false
			for _, disallowed := range s.config.DisallowedTools {
				if tool == disallowed {
					isDisallowed = true
					break
				}
			}
			if !isDisallowed {
				filteredTools = append(filteredTools, tool)
			}
		}
		allowedTools = filteredTools
	}

	// Lock session to prevent concurrent executions
	userSession.ExecutionMutex.Lock()
	defer userSession.ExecutionMutex.Unlock()

	// Determine if this is a new Claude session or continuation
	isNewSession := userSession.ClaudeSessionID == ""
	claudeSessionID := userSession.ClaudeSessionID
	if isNewSession {
		claudeSessionID = userSession.ID // Use our session ID for new Claude sessions
	}

	// Get permission mode
	permMode, err := s.sessionManager.GetPermissionMode(userSession.ID)
	if err != nil {
		s.logger.Error("Failed to get permission mode", zap.Error(err))
		permMode = config.PermissionModeDefault
	}

	// Process with Claude Code CLI
	response, newClaudeSessionID, cost, err := s.claudeExecutor.ProcessClaudeCodeRequest(ctx, text, claudeSessionID, event.User, allowedTools, isNewSession, permMode)
	if err != nil {
		s.logger.Error("Claude Code processing failed", zap.Error(err))
		return fmt.Sprintf("❌ Claude Code processing failed: %v", err)
	}
	
	// Store the latest response
	if err := s.sessionManager.UpdateLatestResponse(userSession.ID, string(response)); err != nil {
		s.logger.Error("Failed to update latest response", zap.Error(err))
	}

	// Update the Claude session ID for future requests
	userSession.ClaudeSessionID = newClaudeSessionID

	// Permission mode persists until explicitly changed

	// Update the current working directory from Claude's execution context
	// For now, we'll use the configured working directory since Claude might have changed directories
	if err := s.sessionManager.UpdateCurrentWorkDir(userSession.ID, s.config.WorkingDirectory); err != nil {
		s.logger.Debug("Failed to update working directory", zap.Error(err))
		// Non-fatal error, continue processing
	}

	// Delete the "Thinking..." message now that we have the response
	if thinkingTimestamp != "" {
		_, _, deleteErr := s.slackAPI.DeleteMessage(event.Channel, thinkingTimestamp)
		if deleteErr != nil {
			s.logger.Debug("Failed to delete thinking message", zap.Error(deleteErr))
		}
	}

	// Log cost for monitoring
	s.logger.Info("Claude Code request completed",
		zap.String("user_id", event.User),
		zap.String("session_id", userSession.ID),
		zap.String("claude_session_id", newClaudeSessionID),
		zap.Float64("cost_usd", cost))

	// Format final response with Mode, Session, and Working Dir
	currentMode, getPermErr := s.sessionManager.GetPermissionMode(userSession.ID)
	if getPermErr != nil {
		currentMode = config.PermissionModeDefault
	}
	
	response = fmt.Sprintf("%s\n\n_Mode: `%s`, Session: `%s`, Working Dir: `%s`_",
		response, currentMode, newClaudeSessionID, userSession.CurrentWorkDir)

	return response
}

// processSlashCommand processes slash commands
func (s *Service) processSlashCommand(ctx context.Context, command *slack.SlashCommand) string {
	authCtx := &auth.AuthContext{
		UserID:    command.UserID,
		ChannelID: command.ChannelID,
		Command:   command.Command,
		Timestamp: time.Now(),
	}

	if err := s.authService.AuthorizeUser(authCtx, auth.PermissionRead); err != nil {
		return fmt.Sprintf("❌ Authorization failed: %v", err)
	}

	return s.processCommand(ctx, &slackevents.MessageEvent{
		User:    command.UserID,
		Channel: command.ChannelID,
		Text:    command.Text,
	}, command.Text)
}

// sendResponse sends a response message to a channel
func (s *Service) sendResponse(channelID, message string) {
	// Split long messages
	messages := s.splitMessage(message, s.config.MaxMessageLength)

	for _, msg := range messages {
		_, _, err := s.slackAPI.PostMessage(channelID,
			slack.MsgOptionText(msg, false),
			slack.MsgOptionAsUser(true))

		if err != nil {
			s.logger.Error("Failed to send message", zap.Error(err))
		}
	}
}

// splitMessage splits long messages into smaller chunks
func (s *Service) splitMessage(message string, maxLength int) []string {
	if len(message) <= maxLength {
		return []string{message}
	}

	var messages []string
	words := strings.Split(message, " ")
	var currentMessage strings.Builder

	for _, word := range words {
		if currentMessage.Len()+len(word)+1 > maxLength {
			if currentMessage.Len() > 0 {
				messages = append(messages, currentMessage.String())
				currentMessage.Reset()
			}
		}

		if currentMessage.Len() > 0 {
			currentMessage.WriteString(" ")
		}
		currentMessage.WriteString(word)
	}

	if currentMessage.Len() > 0 {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

// handleBlockActions handles block actions from interactive components
func (s *Service) handleBlockActions(callback *slack.InteractionCallback) {
	for _, action := range callback.ActionCallback.BlockActions {
		s.logger.Debug("Block action",
			zap.String("action_id", action.ActionID),
			zap.String("value", action.Value))
	}
}

// handleShortcut handles shortcuts
func (s *Service) handleShortcut(callback *slack.InteractionCallback) {
	s.logger.Debug("Shortcut",
		zap.String("callback_id", callback.CallbackID))
}

// periodicCleanup performs periodic cleanup tasks
func (s *Service) periodicCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.authService.CleanupExpiredEntries()
			s.logger.Debug("Performed periodic cleanup")
		case <-s.stopCh:
			return
		}
	}
}

// registerCommands registers built-in commands
func (s *Service) registerCommands() {
	commandRegistry["help"] = s.handleHelpCommand
	commandRegistry["status"] = s.handleStatusCommand
	commandRegistry["sessions"] = s.handleSessionsCommand
	commandRegistry["close"] = s.handleCloseSessionCommand
	commandRegistry["stats"] = s.handleStatsCommand
	commandRegistry["version"] = s.handleVersionCommand
	commandRegistry["session"] = s.handleSetSessionCommand
	// Debug command is handled through slash commands only
	commandRegistry["stop"] = s.handleStopCommand
}

// Command handlers
func (s *Service) handleHelpCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	return s.getHelpMessage(), nil
}

func (s *Service) handleStatusCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	uptime := time.Since(s.startTime).Truncate(time.Second)
	sessionStats := s.sessionManager.GetSessionStats()
	authStats := s.authService.GetStats()

	return fmt.Sprintf(`📊 *Bot Status*

🟢 Status: Running
⏰ Uptime: %v
👥 Total Users: %v
🎯 Active Sessions: %v
📝 Total Messages: %v
🔒 Auth Enabled: %v
🚦 Rate Limit: %d/min

Use `+"`sessions`"+` to see your active sessions.`,
		uptime,
		authStats["total_users"],
		sessionStats["active_sessions"],
		sessionStats["total_messages"],
		authStats["auth_enabled"],
		s.config.RateLimitPerMinute), nil
}

func (s *Service) handleSessionsCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	return s.sessionManager.ListUserSessions(event.User), nil
}

func (s *Service) handleCloseSessionCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	sessions := s.sessionManager.GetActiveSessionsForUser(event.User)
	if len(sessions) == 0 {
		return "No active sessions to close.", nil
	}

	// Close all sessions for the user in this channel
	closed := 0
	for _, session := range sessions {
		if session.ChannelID == event.Channel {
			if err := s.sessionManager.CloseSession(session.ID); err != nil {
				s.logger.Error("Failed to close session", zap.Error(err))
			} else {
				closed++
			}
		}
	}

	if closed == 0 {
		return "No active sessions found in this channel.", nil
	}

	return fmt.Sprintf("✅ Closed %d session(s) in this channel.", closed), nil
}

func (s *Service) handleStatsCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	// Check if user is admin
	if !s.authService.IsUserAdmin(event.User) {
		return "❌ This command requires admin privileges.", fmt.Errorf("insufficient permissions")
	}

	sessionStats := s.sessionManager.GetSessionStats()
	authStats := s.authService.GetStats()

	return fmt.Sprintf(`📈 *Detailed Statistics*

**Sessions:**
• Total: %v
• Active: %v
• Messages: %v

**Users:**
• Total: %v
• Admins: %v
• Banned: %v

**Channels:**
• Total: %v

**System:**
• Uptime: %v
• Auth: %v`,
		sessionStats["total_sessions"],
		sessionStats["active_sessions"],
		sessionStats["total_messages"],
		authStats["total_users"],
		authStats["admin_users"],
		authStats["banned_users"],
		authStats["total_channels"],
		time.Since(s.startTime).Truncate(time.Second),
		authStats["auth_enabled"]), nil
}

func (s *Service) handleVersionCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	return fmt.Sprintf(`🤖 *%s*

Version: 1.0.0
Claude Model: %s
Working Directory: %s
Command Prefix: %s

Built with ❤️ for Slack`,
		s.config.BotDisplayName,
		"claude-code-cli", // Using Claude Code CLI instead of specific model
		s.config.WorkingDirectory,
		s.config.CommandPrefix), nil
}

func (s *Service) handleSetSessionCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	if len(args) == 0 {
		// Show current session info
		userSession, err := s.sessionManager.GetOrCreateSession(event.User, event.Channel)
		if err != nil {
			return "❌ Failed to get session info", err
		}

		currentSessionID := userSession.ClaudeSessionID
		if currentSessionID == "" {
			currentSessionID = "None (new conversation)"
		}

		return fmt.Sprintf("📋 **Current Session Info**\n\nClaude Session ID: `%s`\nBot Session ID: `%s`\nMessages: %d\n\n**Usage:**\n• `session <claude-session-id>` - Switch to specific Claude session\n• `session new` - Start a new conversation",
			currentSessionID, userSession.ID, userSession.MessageCount), nil
	}

	sessionID := args[0]

	// Get current session
	userSession, err := s.sessionManager.GetOrCreateSession(event.User, event.Channel)
	if err != nil {
		return "❌ Failed to get session", err
	}

	if sessionID == "new" {
		// Clear Claude session ID to start fresh
		userSession.ClaudeSessionID = ""
		return "✅ **New Conversation Started**\n\nNext message will start a fresh conversation with Claude.", nil
	} else {
		// Set specific Claude session ID
		userSession.ClaudeSessionID = sessionID
		return fmt.Sprintf("✅ **Session Switched**\n\nNow using Claude session: `%s`\n\nNext message will resume this conversation.", sessionID), nil
	}
}

// getHelpMessage returns the help message
func (s *Service) getHelpMessage() string {
	return fmt.Sprintf(`🤖 *%s Help*

**Commands:**
• `+"`help`"+` - Show this help message
• `+"`status`"+` - Show bot status
• `+"`sessions`"+` - List your active sessions
• `+"`session`"+` - Show current Claude session ID
• `+"`session <id>`"+` - Switch to specific Claude session
• `+"`session new`"+` - Start a new conversation
• `+"`close`"+` - Close session in this channel
• `+"`stats`"+` - Show statistics (admin only)
• `+"`version`"+` - Show bot version

**Usage:**
• Direct message: Just type your message
• Channel: Use `+"`%s <message>`"+` or mention @%s
• Ask Claude anything about code, files, or development tasks

**Examples:**
• `+"`%s help me debug this Python script`"+`
• `+"`%s list files in /tmp`"+`
• `+"`%s explain this error message`"+`

Type any message to start a conversation with!`,
		s.config.BotDisplayName,
		s.config.CommandPrefix,
		s.config.BotDisplayName,
		s.config.CommandPrefix,
		s.config.CommandPrefix,
		s.config.CommandPrefix)
}

// startHTTPServer starts the HTTP server for Events API
func (s *Service) startHTTPServer() {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc(s.config.HealthCheckPath, s.handleHealth)

	// Slack events endpoint
	mux.HandleFunc("/slack/events", s.handleSlackEvents)

	// Slack slash commands endpoint
	mux.HandleFunc("/slack/commands", s.handleSlashCommands)

	// Metrics endpoint (basic)
	mux.HandleFunc("/metrics", s.handleMetrics)

	// Version endpoint
	mux.HandleFunc("/version", s.handleVersion)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.ServerHost, s.config.ServerPort),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("Starting HTTP server",
		zap.String("addr", s.httpServer.Addr),
		zap.String("health_path", s.config.HealthCheckPath))

	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		s.logger.Error("HTTP server error", zap.Error(err))
	}
}

// handleSlackEvents handles the /slack/events endpoint for Events API
func (s *Service) handleSlackEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read and verify the request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify Slack signature
	if !s.verifySlackSignature(r.Header, body) {
		s.logger.Warn("Invalid Slack signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the event
	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
	if err != nil {
		s.logger.Error("Failed to parse Slack event", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	s.logger.Debug("Received Slack event", zap.String("type", eventsAPIEvent.Type))

	// Handle different event types
	switch eventsAPIEvent.Type {
	case slackevents.URLVerification:
		// Respond to URL verification challenge
		var challenge slackevents.ChallengeResponse
		if err := json.Unmarshal(body, &challenge); err != nil {
			s.logger.Error("Failed to unmarshal challenge", zap.Error(err))
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge.Challenge))
		s.logger.Info("Responded to URL verification challenge")
		return

	case slackevents.CallbackEvent:
		// Handle callback events asynchronously
		go s.handleEventsAPIEvent(&eventsAPIEvent)

		// Acknowledge immediately
		w.WriteHeader(http.StatusOK)
		return

	default:
		s.logger.Debug("Unhandled event type", zap.String("type", eventsAPIEvent.Type))
		w.WriteHeader(http.StatusOK)
		return
	}
}

// verifySlackSignature verifies the Slack request signature
func (s *Service) verifySlackSignature(headers http.Header, body []byte) bool {
	if s.config.SlackSigningSecret == "" {
		s.logger.Warn("Slack signing secret not configured, skipping signature verification")
		return true // Skip verification if no secret configured
	}

	timestamp := headers.Get("X-Slack-Request-Timestamp")
	signature := headers.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return false
	}

	// Check timestamp to prevent replay attacks
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}

	if time.Now().Unix()-ts > 300 { // 5 minutes
		return false
	}

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(s.config.SlackSigningSecret))
	mac.Write([]byte(fmt.Sprintf("v0:%s:", timestamp)))
	mac.Write(body)
	expectedSignature := "v0=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// handleHealth handles health check requests
func (s *Service) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":      "healthy",
		"uptime":      time.Since(s.startTime).String(),
		"bot_user_id": s.botUserID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleMetrics handles metrics requests
func (s *Service) handleMetrics(w http.ResponseWriter, r *http.Request) {
	sessionStats := s.sessionManager.GetSessionStats()
	authStats := s.authService.GetStats()

	metrics := map[string]interface{}{
		"uptime_seconds":  time.Since(s.startTime).Seconds(),
		"total_sessions":  sessionStats["total_sessions"],
		"active_sessions": sessionStats["active_sessions"],
		"total_messages":  sessionStats["total_messages"],
		"total_users":     authStats["total_users"],
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleVersion handles version requests
func (s *Service) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	version := map[string]interface{}{
		"app":          "claude-on-slack",
		"version":      "1.0.0",
		"bot_name":     s.config.BotDisplayName,
		"claude_model": "claude-code-cli",
		"working_dir":  s.config.WorkingDirectory,
		"uptime":       time.Since(s.startTime).String(),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(version)
}

// handleSlashCommands handles Slack slash commands
func (s *Service) handleSlashCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body first for signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Verify Slack signature (if configured)
	if s.config.SlackSigningSecret != "" {
		if !s.verifySlackSignature(r.Header, bodyBytes) {
			s.logger.Warn("Invalid Slack signature for slash command")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse form data from the body we just read
	formData, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		s.logger.Error("Failed to parse slash command form data", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Extract slash command data from parsed form
	command := formData.Get("command")
	text := formData.Get("text")
	userID := formData.Get("user_id")
	channelID := formData.Get("channel_id")

	s.logger.Info("Received slash command",
		zap.String("command", command),
		zap.String("text", text),
		zap.String("user_id", userID),
		zap.String("channel_id", channelID))

	// Handle the slash command
	var response string
	switch command {
	case "/session":
		response = s.handleSessionSlashCommand(userID, channelID, text)
	case "/permission":
		response = s.handlePermissionSlashCommand(userID, channelID, text)
	case "/debug":
		response = s.handleDebugSlashCommand(userID, channelID)
	case "/stop":
		response, _ = s.handleStopCommand(context.Background(), &slackevents.MessageEvent{User: userID, Channel: channelID}, nil)
	default:
		response = fmt.Sprintf("Unknown command: %s", command)
	}

	// Send response back to Slack
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	slackResponse := map[string]string{
		"response_type": "ephemeral", // Only visible to the user who ran the command
		"text":          response,
	}

	json.NewEncoder(w).Encode(slackResponse)
}

// handleSessionSlashCommand handles the /session slash command
func (s *Service) handleSessionSlashCommand(userID, channelID, text string) string {
	// Create auth context
	authCtx := &auth.AuthContext{
		UserID:    userID,
		ChannelID: channelID,
		Command:   "/session",
		Timestamp: time.Now(),
	}

	// Check authorization
	if err := s.authService.AuthorizeUser(authCtx, auth.PermissionRead); err != nil {
		s.logger.Warn("Authorization failed for slash command", zap.Error(err))
		return fmt.Sprintf("❌ Authorization failed: %v", err)
	}

	args := strings.Fields(text)

	// If no argument or "help", show help/current info
	if len(args) == 0 || args[0] == "help" {
		userSession, err := s.sessionManager.GetOrCreateSession(userID, channelID)
		if err != nil {
			return "❌ Failed to get session info"
		}

		currentSessionID := userSession.ClaudeSessionID
		if currentSessionID == "" {
			currentSessionID = "None (new conversation)"
		}

		return fmt.Sprintf("📋 **Session Management Help**\n\n**Current Session:**\n• Claude Session ID: `%s`\n• Bot Session ID: `%s`\n• Messages: %d\n\n**Usage:**\n• `/session` - Show this help\n• `/session <claude-session-id>` - Switch to specific Claude session\n• `/session new` - Start a new conversation\n• `/session help` - Show this help\n\n**Note:** Each message shows the session ID at the bottom.",
			currentSessionID, userSession.ID, userSession.MessageCount)
	}

	sessionID := args[0]

	// Get current session
	userSession, err := s.sessionManager.GetOrCreateSession(userID, channelID)
	if err != nil {
		return "❌ Failed to get session"
	}

	if sessionID == "new" {
		// Clear Claude session ID to start fresh
		userSession.ClaudeSessionID = ""
		return "✅ **New Conversation Started**\n\nNext message will start a fresh conversation with Claude."
	} else {
		// Set specific Claude session ID
		userSession.ClaudeSessionID = sessionID
		return fmt.Sprintf("✅ **Session Switched**\n\nNow using Claude session: `%s`\n\nNext message will resume this conversation.", sessionID)
	}
}

// handlePermissionSlashCommand handles the /permission slash command
// handleDebugSlashCommand handles the /debug slash command
func (s *Service) handleDebugSlashCommand(userID, channelID string) string {
	// Get session
	userSession, err := s.sessionManager.GetOrCreateSession(userID, channelID)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get session: %v", err)
	}

	// Check if there's a latest response
	if userSession.LatestResponse == "" {
		return "❌ No Claude response available. Try sending a message first."
	}

	// Pretty print the JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(userSession.LatestResponse), "", "  "); err != nil {
		return fmt.Sprintf("❌ Failed to format JSON: %v", err)
	}

	return fmt.Sprintf("```json\n%s\n```", prettyJSON.String())
}

// handleStopCommand handles the /stop command to force-stop current processing
func (s *Service) handleStopCommand(ctx context.Context, event *slackevents.MessageEvent, args []string) (string, error) {
	// Check if user is admin
	if !s.authService.IsUserAdmin(event.User) {
		return "❌ This command requires admin privileges.", fmt.Errorf("insufficient permissions")
	}

	// Get session
	userSession, err := s.sessionManager.GetOrCreateSession(event.User, event.Channel)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get session: %v", err), err
	}

	// Check if session is processing
	isProcessing := s.sessionManager.IsProcessing(userSession.ID)
	if !isProcessing {
		return "No active processing to stop.", nil
	}

	// Cancel processing by closing the stop channel
	close(s.stopCh)
	
	// Reinitialize the stop channel for future use
	s.stopCh = make(chan struct{})

	return "✅ Processing stopped.", nil
}

func (s *Service) handlePermissionSlashCommand(userID, channelID, text string) string {
	// Get session
	userSession, err := s.sessionManager.GetOrCreateSession(userID, channelID)
	if err != nil {
		return fmt.Sprintf("❌ Failed to get session: %v", err)
	}

	args := strings.Fields(text)

	// If no argument or "help", show help
	if len(args) == 0 || args[0] == "help" {
		currentMode, err := s.sessionManager.GetPermissionMode(userSession.ID)
		if err != nil {
			currentMode = "default" // fallback
		}

		return fmt.Sprintf("📋 **Permission Mode Help**\n\n**Current Mode:** `%s`\n\n**Available Modes:**\n• `default` - Standard permissions with user prompts\n• `acceptEdits` - Automatically accept file edits\n• `bypassPermissions` - Bypass all permission checks\n• `plan` - Planning mode, won't execute actions\n\n**Usage:**\n• `/permission` - Show this help\n• `/permission <mode>` - Set permission mode\n• `/permission help` - Show this help", currentMode)
	}

	// Get the permission mode argument
	modeStr := args[0]

	// Validate mode
	mode := config.PermissionMode(modeStr)
	switch mode {
	case config.PermissionModeDefault,
		config.PermissionModeAcceptEdits,
		config.PermissionModeBypassPerms,
		config.PermissionModePlan:
		// Valid mode
	default:
		return "❌ **Invalid Permission Mode**\n\nAvailable modes:\n• `default`\n• `acceptEdits`\n• `bypassPermissions`\n• `plan`\n\nUse `/permission help` for more info."
	}

	// Set mode
	if err := s.sessionManager.SetPermissionMode(userSession.ID, mode); err != nil {
		return fmt.Sprintf("❌ Failed to set permission mode: %v", err)
	}

	var description string
	switch mode {
	case config.PermissionModeDefault:
		description = "Standard permissions with user prompts"
	case config.PermissionModeAcceptEdits:
		description = "Automatically accept file edits"
	case config.PermissionModeBypassPerms:
		description = "Bypass all permission checks"
	case config.PermissionModePlan:
		description = "Planning mode, won't execute actions"
	}

	return fmt.Sprintf("✅ **Permission Mode Set**\n\nMode: `%s`\nDescription: %s", mode, description)
}
