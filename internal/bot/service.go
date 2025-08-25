package bot

import (
	"context"
	"fmt"
	"regexp"
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
	slackAPI := slack.New(cfg.SlackBotToken, slack.OptionDebug(cfg.EnableDebug))
	socketClient := socketmode.New(slackAPI, socketmode.OptionDebug(cfg.EnableDebug), socketmode.OptionAppToken(cfg.SlackAppToken))

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

	// Start event handling
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

	// Start socket mode client
	return s.socketClient.Run()
}

// Stop stops the bot service
func (s *Service) Stop() {
	s.logger.Info("Stopping Claude on Slack bot")
	
	close(s.stopCh)
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
		case slackevents.Message:
			messageEvent, ok := innerEvent.Data.(*slackevents.MessageEvent)
			if !ok {
				s.logger.Warn("Failed to type assert message event")
				return
			}
			s.handleMessageEvent(messageEvent)

		case slackevents.AppMention:
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
	// Ignore bot messages and messages from the bot itself
	if event.BotID != "" || event.User == s.botUserID {
		return
	}

	// Check if message starts with command prefix or mentions the bot
	text := strings.TrimSpace(event.Text)
	if !strings.HasPrefix(text, s.config.CommandPrefix) && !strings.Contains(text, fmt.Sprintf("<@%s>", s.botUserID)) {
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
		ChannelType: event.ChannelType,
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

	// Check if it's a command
	if strings.HasPrefix(text, "/") || strings.Contains(text, " ") && !strings.HasPrefix(text, " ") {
		return s.processCommand(ctx, event, text)
	}

	// Process as Claude conversation
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

	// Check rate limiting
	limited, remaining, err := s.sessionManager.CheckRateLimit(userSession.ID)
	if err != nil {
		s.logger.Error("Rate limit check failed", zap.Error(err))
		return "❌ Failed to check rate limit"
	}

	if limited {
		return fmt.Sprintf("⏱️ Rate limit exceeded. Try again in %v", remaining.Truncate(time.Second))
	}

	// Send typing indicator
	s.slackAPI.PostMessage(event.Channel, slack.MsgOptionText("🤔 Thinking...", false))

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

	// Process with Claude Code CLI
	response, cost, err := s.claudeExecutor.ProcessClaudeCodeRequest(ctx, text, userSession.ID, event.User, allowedTools)
	if err != nil {
		s.logger.Error("Claude Code processing failed", zap.Error(err))
		return fmt.Sprintf("❌ Claude Code processing failed: %v", err)
	}

	// Log cost for monitoring
	s.logger.Info("Claude Code request completed",
		zap.String("user_id", event.User),
		zap.String("session_id", userSession.ID),
		zap.Float64("cost_usd", cost))


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

Use \`sessions\` to see your active sessions.`, 
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
		s.config.ClaudeModel,
		s.config.WorkingDirectory,
		s.config.CommandPrefix), nil
}

// getHelpMessage returns the help message
func (s *Service) getHelpMessage() string {
	return fmt.Sprintf(`🤖 *%s Help*

**Commands:**
• \`help\` - Show this help message
• \`status\` - Show bot status
• \`sessions\` - List your active sessions
• \`close\` - Close session in this channel
• \`stats\` - Show statistics (admin only)
• \`version\` - Show bot version

**Usage:**
• Direct message: Just type your message
• Channel: Use \`%s <message>\` or mention @%s
• Ask Claude anything about code, files, or development tasks

**Examples:**
• \`%s help me debug this Python script\`
• \`%s list files in /tmp\`
• \`%s explain this error message\`

Type any message to start a conversation with!`,
		s.config.BotDisplayName,
		s.config.CommandPrefix,
		s.config.BotDisplayName,
		s.config.CommandPrefix,
		s.config.CommandPrefix,
		s.config.CommandPrefix)
}