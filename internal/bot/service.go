package bot

import (
	"context"
	"math"
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
	"github.com/ghabxph/claude-on-slack/internal/database"
	"github.com/ghabxph/claude-on-slack/internal/files"
	"github.com/ghabxph/claude-on-slack/internal/logging"
	"github.com/ghabxph/claude-on-slack/internal/notifications"
	"github.com/ghabxph/claude-on-slack/internal/session"
	"github.com/ghabxph/claude-on-slack/internal/version"
)

// Service represents the main bot service
type Service struct {
	config         *config.Config
	logger         *zap.Logger
	dualLogger     *logging.DualLogger
	slackAPI       *slack.Client
	socketClient   *socketmode.Client
	httpServer     *http.Server
	authService    *auth.Service
	sessionManager session.SessionManager
	claudeExecutor *claude.Executor
	fileDownloader *files.Downloader
	fileCleanup    *files.CleanupService
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
	
	// Initialize database with retry logic
	db, err := database.NewDatabase(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	// Use database-backed session manager
	sessionManager := session.NewDatabaseManager(cfg, logger, claudeExecutor, db)

	// Initialize file downloader
	storageDir := "/tmp/claude-slack-images"
	fileDownloader, err := files.NewDownloader(slackAPI, logger, storageDir, cfg.SlackBotToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create file downloader: %w", err)
	}
	fileCleanup := files.NewCleanupService(fileDownloader, logger)

	// Initialize dual logger for centralized error reporting
	dualLogger := logging.NewDualLogger(logger, slackAPI)

	service := &Service{
		config:         cfg,
		logger:         logger,
		dualLogger:     dualLogger,
		slackAPI:       slackAPI,
		socketClient:   socketClient,
		authService:    authService,
		sessionManager: sessionManager,
		claudeExecutor: claudeExecutor,
		fileDownloader: fileDownloader,
		fileCleanup:    fileCleanup,
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
		zap.String("version", version.GetVersion()),
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
	httpServerErrCh := make(chan error, 1)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.startHTTPServer(); err != nil {
			httpServerErrCh <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()
	
	// Check if HTTP server started successfully
	select {
	case err := <-httpServerErrCh:
		return err
	case <-time.After(2 * time.Second):
		s.logger.Info("HTTP server startup check passed")
	}

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

	// Start file cleanup service
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.fileCleanup.Start(ctx)
	}()

	// Start socket mode client (will only work if app is configured for Socket Mode)
	go func() {
		if err := s.socketClient.Run(); err != nil {
			s.logger.Debug("Socket Mode not available or disabled", zap.Error(err))
		}
	}()

	// Send startup notification after successful initialization
	s.sendStartupNotification()

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

		case "file_shared":
			fileEvent, ok := innerEvent.Data.(*slackevents.FileSharedEvent)
			if !ok {
				s.logger.Warn("Failed to type assert file shared event")
				return
			}
			s.handleFileSharedEvent(fileEvent)
		}
	}
}

// handleMessageEvent handles message events
func (s *Service) handleMessageEvent(event *slackevents.MessageEvent) {
	// Ignore bot messages, messages from the bot itself, and messages with empty user ID
	if event.BotID != "" || event.User == s.botUserID || event.User == "" {
		return
	}

	s.logger.Debug("Processing message in allowed channel",
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

// handleFileSharedEvent handles file shared events
func (s *Service) handleFileSharedEvent(event *slackevents.FileSharedEvent) {
	s.logger.Debug("File shared event received", 
		zap.String("fileID", event.FileID))

	// Note: File shared events don't contain user or channel info directly
	// We need to get file info to find where it was shared
	// For now, we'll just log it - the actual file processing happens
	// when the file is shared in a message event with Files field
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
		errCtx := logging.CreateErrorContext(event.Channel, event.User, "message_processor", "authorization")
		return s.logErrorWithTrace(ctx, errCtx, err, "Authorization failed")
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
	// Process file attachments if present
	downloadedFiles := []*files.FileInfo{}
	if len(event.Files) > 0 {
		for _, file := range event.Files {
			// Only process image files
			if s.IsImageMimeType(file.Mimetype) {
				s.logger.Info("Processing image attachment", 
					zap.String("fileID", file.ID), 
					zap.String("filename", file.Name),
					zap.String("mimetype", file.Mimetype))

				fileInfo, err := s.fileDownloader.DownloadFile(file.ID, event.User)
				if err != nil {
					s.logger.Error("Failed to download image", 
						zap.String("fileID", file.ID), 
						zap.Error(err))
					return fmt.Sprintf("❌ Failed to process image %s: %v", file.Name, err)
				}
				downloadedFiles = append(downloadedFiles, fileInfo)
			}
		}
	}

	// Add image references to the text if files were downloaded
	if len(downloadedFiles) > 0 {
		imagePrompts := []string{}
		for _, fileInfo := range downloadedFiles {
			imagePrompts = append(imagePrompts, fmt.Sprintf("Please analyze the image at %s", fileInfo.LocalPath))
		}
		
		if text != "" {
			text = strings.Join(imagePrompts, ". ") + ". " + text
		} else {
			text = strings.Join(imagePrompts, ". ")
		}
	}

	// Schedule cleanup of downloaded files
	defer func() {
		for _, fileInfo := range downloadedFiles {
			go func(path string) {
				time.Sleep(5 * time.Minute) // Wait 5 minutes before cleanup
				s.fileDownloader.CleanupFile(path)
			}(fileInfo.LocalPath)
		}
	}()

	// Get or create session
	userSession, err := s.sessionManager.GetOrCreateSession(event.User, event.Channel)
	if err != nil {
		errCtx := logging.CreateErrorContext(event.Channel, event.User, "message_processor", "create_session")
		return s.logErrorWithTrace(ctx, errCtx, err, "Failed to create session")
	}

	// Check if we should queue this message
	queued, err := s.sessionManager.QueueMessage(userSession.GetID(), text)
	if err != nil {
		s.logger.Error("Failed to check message queue", zap.Error(err))
		errCtx := logging.CreateErrorContext(event.Channel, event.User, "message_processor", "queue_message")
		errCtx.WithSession(userSession.GetID())
		return s.logErrorWithTrace(ctx, errCtx, err, "Failed to process message")
	}

	if queued {
		return "" // Message queued, no response needed yet
	}

	// Check rate limiting
	limited, remaining, err := s.sessionManager.CheckRateLimit(userSession.GetID())
	if err != nil {
		s.logger.Error("Rate limit check failed", zap.Error(err))
		return "❌ Failed to check rate limit"
	}

	if limited {
		return fmt.Sprintf("⏱️ Rate limit exceeded. Try again in %v", remaining.Truncate(time.Second))
	}

	// Mark as processing
	if err := s.sessionManager.SetProcessing(userSession.GetID(), true); err != nil {
		s.logger.Error("Failed to set processing state", zap.Error(err))
		return fmt.Sprintf("❌ Failed to process message: %v", err)
	}
	defer s.sessionManager.SetProcessing(userSession.GetID(), false)

	// Get any queued messages and combine with current message
	queuedMessages, err := s.sessionManager.GetQueuedMessages(userSession.GetID())
	if err != nil {
		s.logger.Error("Failed to get queued messages", zap.Error(err))
		return fmt.Sprintf("❌ Failed to process message: %v", err)
	}

	if len(queuedMessages) > 0 {
		text = strings.Join(append([]string{text}, queuedMessages...), " ")
	}

	// Send "Thinking..." message immediately and capture for deletion
	// Get current mode
	currentMode, err := s.getPermissionModeForChannel(event.Channel, userSession.GetID())
	if err != nil {
		currentMode = config.PermissionModeDefault
	}
	
	// Format Thinking message with Mode, Session, and Working Dir
	thinkingMsg := fmt.Sprintf("🤔 _Thinking..._\n\n_• Mode: `%s`\n• Session: `%s`\n• Working Dir: `%s`_",
		currentMode, userSession.GetID(), userSession.GetCurrentWorkDir())
	
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

	// For database sessions, we handle concurrency differently
	// TODO: Implement database-level session locking if needed

	// Determine Claude session ID based on conversation state
	var claudeSessionID string
	var isNewSession bool
	
	// Check if there are any child sessions (actual Claude conversations)
	latestChildSessionID, err := s.sessionManager.GetLatestChildSessionID(userSession.GetID())
	if err != nil {
		errCtx := logging.CreateErrorContext(event.Channel, event.User, "message_processor", "get_session_info")
		errCtx.WithSession(userSession.GetID())
		return s.logErrorWithTrace(ctx, errCtx, err, "Failed to get session info")
	}
	
	s.logger.Info("Session determination logic", 
		zap.String("bot_session_id", userSession.GetID()),
		zap.String("channel_id", event.Channel),
		zap.String("user_id", event.User),
		zap.Bool("has_child_sessions", latestChildSessionID != nil && *latestChildSessionID != ""))
	
	if latestChildSessionID == nil || *latestChildSessionID == "" {
		// No child sessions = first actual Claude conversation
		claudeSessionID = userSession.GetID()
		isNewSession = true
		s.logger.Info("FIRST MESSAGE - using --session-id", 
			zap.String("bot_session_id", userSession.GetID()),
			zap.String("claude_session_id", claudeSessionID),
			zap.Bool("is_new_session", isNewSession))
	} else {
		// Child sessions exist = resume conversation
		claudeSessionID = *latestChildSessionID
		isNewSession = false
		s.logger.Info("RESUME MESSAGE - using --resume with child session ID", 
			zap.String("bot_session_id", userSession.GetID()),
			zap.String("claude_session_id", claudeSessionID),
			zap.Bool("is_new_session", isNewSession))
	}

	// Get permission mode
	permMode, permErr := s.getPermissionModeForChannel(event.Channel, userSession.GetID())
	if permErr != nil {
		s.logger.Error("Failed to get permission mode", zap.Error(permErr))
		permMode = config.PermissionModeDefault
	}

	// Process with Claude Code CLI
	response, newClaudeSessionID, cost, rawJSON, err := s.claudeExecutor.ProcessClaudeCodeRequest(ctx, text, claudeSessionID, event.User, userSession.GetCurrentWorkDir(), allowedTools, isNewSession, permMode)
	if err != nil {
		s.logger.Error("Claude Code processing failed", zap.Error(err))
		errCtx := logging.CreateErrorContext(event.Channel, event.User, "message_processor", "claude_processing")
		errCtx.WithSession(claudeSessionID)
		return s.logErrorWithTrace(ctx, errCtx, err, "Claude Code processing failed")
	}
	
	// Store the latest response (raw JSON)
	if err := s.sessionManager.UpdateLatestResponse(userSession.GetID(), rawJSON); err != nil {
		s.logger.Error("Failed to update latest response", zap.Error(err))
	}

	// Always store Claude's returned session ID as a child session for future resume operations
	if newClaudeSessionID != "" {
		if dbManager, ok := s.sessionManager.(*session.DatabaseManager); ok {
			if err := dbManager.ProcessClaudeAIResponse(userSession.GetID(), newClaudeSessionID, response); err != nil {
				s.logger.Error("Failed to store Claude AI response as child session", 
					zap.String("bot_session_id", userSession.GetID()),
					zap.String("claude_session_id", newClaudeSessionID),
					zap.Error(err))
			} else {
				s.logger.Debug("Stored Claude AI response as child session", 
					zap.String("bot_session_id", userSession.GetID()),
					zap.String("claude_session_id", newClaudeSessionID),
					zap.String("input_session_id", claudeSessionID))
			}
		}
	}

	// Permission mode persists until explicitly changed

	// Note: Working directory is preserved from the session's configured path
	// Claude Code execution might change directories internally, but the session keeps its base path

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
		zap.String("session_id", userSession.GetID()),
		zap.String("claude_session_id", newClaudeSessionID),
		zap.Float64("cost_usd", cost))

	// Format final response with Mode, Session, Working Dir, and Message Count
	currentMode, getPermErr := s.getPermissionModeForChannel(event.Channel, userSession.GetID())
	if getPermErr != nil {
		currentMode = config.PermissionModeDefault
	}
	
	// Get message count for display
	displayMessageCount, err := s.sessionManager.GetTotalMessageCount(userSession.GetID())
	if err != nil {
		s.logger.Debug("Failed to get message count for display", zap.Error(err))
		displayMessageCount = 0 // fallback to 0
	}
	
	response = fmt.Sprintf("%s\n\n• Mode: _%s_\n• Session: _%s_\n• Working Dir: _%s_\n• Messages: _%d_",
		response, currentMode, newClaudeSessionID, userSession.GetCurrentWorkDir(), displayMessageCount)

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
🚦 Rate Limit: %d/min

Use `+"`sessions`"+` to see your active sessions.`,
		uptime,
		authStats["total_users"],
		sessionStats["active_sessions"],
		sessionStats["total_messages"],
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
		if session.GetChannelID() == event.Channel {
			if err := s.sessionManager.CloseSession(session.GetID()); err != nil {
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
• Auth Enabled: %v`,
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
		// Show current session info and available sessions
		userSession, err := s.sessionManager.GetOrCreateSession(event.User, event.Channel)
		if err != nil {
			errCtx := logging.CreateErrorContext(event.Channel, event.User, "session_command", "get_session_info")
			return s.logErrorWithTrace(ctx, errCtx, err, "Failed to get session info"), err
		}

		currentSessionID := userSession.GetID()
		if currentSessionID == "" {
			currentSessionID = "None (new conversation)"
		}

		// Get list of available sessions
		sessions, err := s.sessionManager.ListAllSessions(10)
		if err != nil {
			s.logger.Error("Failed to list sessions", zap.Error(err))
			// Still continue - this is not a fatal error for the help display
		}

		// Get known paths
		paths, err := s.sessionManager.GetKnownPaths(10)
		if err != nil {
			s.logger.Error("Failed to get known paths", zap.Error(err))
		}

		// Get message count for session info display
		messageCount, err := s.sessionManager.GetTotalMessageCount(userSession.GetID())
		if err != nil {
			messageCount = 0
		}
		
		response := fmt.Sprintf("📋 **Current Session Info**\n\nClaude Session ID: `%s`\nBot Session ID: `%s`\nMessages: %d\n\n**Usage:**\n• `session list` - Show detailed list of all sessions\n• `session <claude-session-id>` - Switch to specific Claude session\n• `session new <path>` - Start new conversation in specific path\n• `session new` - Start new conversation in current directory\n• `session . <path>` - Switch to or create session for specific path",
			currentSessionID, userSession.GetID(), messageCount)

		if len(sessions) > 0 {
			response += "\n\n**Available Sessions:**\n"
			for i, session := range sessions {
				if i >= 5 { // Limit to 5 sessions
					response += "• _... and more_\n"
					break
				}
				response += fmt.Sprintf("• `%s` - %s (%s)\n", 
					session.GetID()[:8], // Show first 8 chars of session ID
					session.GetWorkspaceDir(), 
					session.GetLastActivity().Format("Jan 2 15:04"))
			}
		}

		if len(paths) > 0 {
			response += "\n**Known Paths:**\n"
			for i, path := range paths {
				if i >= 5 { // Limit to 5 paths
					response += "• _... and more_\n"
					break
				}
				response += fmt.Sprintf("• `%s`\n", path)
			}
		}

		return response, nil
	}

	if args[0] == "list" {
		// Show detailed list of all sessions
		response, err := s.handleSessionListCommand(event.User, event.Channel)
		if err != nil {
			errCtx := logging.CreateErrorContext(event.Channel, event.User, "session_command", "list_sessions")
			return s.logErrorWithTrace(ctx, errCtx, err, "Failed to list sessions"), err
		}
		return response, nil
	} else if args[0] == "new" {
		// Handle new session creation with optional path
		var workingDir string
		if len(args) > 1 {
			workingDir = args[1]
		} else {
			workingDir = s.config.WorkingDirectory
		}

		// Create a new session with the specified working directory
		newSession, err := s.sessionManager.CreateSessionWithPath(event.User, event.Channel, workingDir)
		if err != nil {
			s.logger.Error("Failed to create new session", zap.Error(err))
			return "❌ **Error:** Failed to create new session", nil
		}

		return fmt.Sprintf("✅ **New Conversation Started**\n\nSession ID: `%s`\nWorking directory: `%s`\nNext message will start a fresh conversation with Claude.", newSession.GetID(), workingDir), nil
	} else if args[0] == "." {
		// Switch to or create session for specific path
		if len(args) < 2 {
			return "❌ **Usage:** `session . <path>` - Switch to or create session for specific path", nil
		}

		newPath := args[1]
		
		// Find existing sessions for this path
		existingSessions, err := s.sessionManager.GetSessionsByPath(newPath, 5)
		if err != nil {
			s.logger.Error("Failed to get sessions by path", zap.Error(err))
		}

		if len(existingSessions) == 0 {
			// No existing sessions for this path, create a new one
			// For database sessions, no session manipulation needed

			return fmt.Sprintf("✅ **New Session Created for Path**\n\nWorking directory: `%s`\nNext message will start a fresh conversation in this path.", newPath), nil
		} else {
			// Found existing sessions, let user choose
			response := fmt.Sprintf("📋 **Found %d existing session(s) for path:** `%s`\n\n", len(existingSessions), newPath)
			response += "**Available Sessions:**\n"
			
			for i, session := range existingSessions {
				if i >= 3 { // Limit to 3 sessions
					response += "• _... and more_\n"
					break
				}
				response += fmt.Sprintf("• `%s` - Last used: %s\n", 
					session.GetID(), 
					session.GetLastActivity().Format("Jan 2 15:04"))
			}
			
			response += "\n**Usage:**\n"
			response += fmt.Sprintf("• `session %s` - Use most recent session\n", existingSessions[0].GetID()[:8])
			response += fmt.Sprintf("• `session new %s` - Create new session for this path", newPath)
			
			return response, nil
		}
	} else {
		// Switch to specific Claude session ID
		sessionID := args[0]

		// For database sessions, session switching is handled differently
		// Session ID is managed automatically
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
func (s *Service) startHTTPServer() error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc(s.config.HealthCheckPath, s.handleHealth)

	// Slack events endpoint
	mux.HandleFunc("/slack/events", s.handleSlackEvents)

	// Slack slash commands endpoint
	mux.HandleFunc("/slack/commands", s.handleSlashCommands)
	
	// Delete session command endpoint  
	mux.HandleFunc("/slack/delete", s.handleDeleteCommand)

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
		return fmt.Errorf("HTTP server listen error: %w", err)
	}
	
	s.logger.Info("HTTP server stopped gracefully")
	return nil
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
		s.logger.Error("Slack signing secret not configured, rejecting request")
		return false // Fail securely when secret is not configured
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

	if math.Abs(float64(time.Now().Unix()-ts)) > 300 { // 5 minutes
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

	// If no argument or "help", show help/current info with suggestions
	if len(args) == 0 || args[0] == "help" {
		userSession, err := s.sessionManager.GetOrCreateSession(userID, channelID)
		if err != nil {
			errCtx := logging.CreateErrorContext(channelID, userID, "session_slash_command", "get_session_info")
			return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to get session info")
		}

		currentSessionID := userSession.GetID()
		if currentSessionID == "" {
			currentSessionID = "None (new conversation)"
		}

		// Get list of available sessions
		sessions, err := s.sessionManager.ListAllSessions(10)
		if err != nil {
			s.logger.Error("Failed to list sessions", zap.Error(err))
			// Still continue - this is not a fatal error for the help display
		}

		// Get known paths with default suggestion
		paths, err := s.sessionManager.GetKnownPaths(10)
		if err != nil {
			s.logger.Error("Failed to get known paths", zap.Error(err))
		}
		
		// Add default working directory if no paths found
		if len(paths) == 0 {
			paths = []string{s.config.WorkingDirectory}
		}

		// Get message count for session help display
		messageCount, err := s.sessionManager.GetTotalMessageCount(userSession.GetID())
		if err != nil {
			messageCount = 0
		}

		// Get channel state to determine parent and leaf sessions
		parentSessionInfo := "None"
		leafSessionInfo := "None"
		
		// Access the database manager to get channel state
		if dbManager, ok := s.sessionManager.(*session.DatabaseManager); ok {
			channelState, err := dbManager.GetChannelState(channelID)
			if err == nil && channelState != nil {
				// Get parent session info
				if channelState.ActiveSessionID != nil {
					if parentSession, err := dbManager.LoadSessionByID(*channelState.ActiveSessionID); err == nil && parentSession != nil {
						parentSessionInfo = fmt.Sprintf("`%s`", parentSession.SessionID)
					}
				}
				
				// Get leaf session info  
				if channelState.ActiveChildSessionID != nil {
					if leafSession, err := dbManager.GetChildSessionByID(*channelState.ActiveChildSessionID); err == nil && leafSession != nil {
						leafSessionInfo = fmt.Sprintf("`%s`", leafSession.SessionID)
					}
				}
			}
		}
		
		response := fmt.Sprintf("📋 **Session Management Help**\n\n**Current Session:**\n• Parent Session: %s\n• Leaf Session: %s\n• Messages: %d\n\n**Usage:**\n• `/session` - Show this help\n• `/session list` - Show detailed list of all sessions\n• `/session info <uuid>` - Show child conversations for parent session\n• `/session <claude-session-id>` - Switch to specific Claude session\n• `/session new <path>` - Start new conversation in specific path\n• `/session new` - Start new conversation in current directory\n• `/session . <path>` - Switch to or create session for specific path",
			parentSessionInfo, leafSessionInfo, messageCount)

		if len(sessions) > 0 {
			response += "\n\n**Available Sessions:**\n"
			for i, session := range sessions {
				if i >= 5 { // Limit to 5 sessions
					response += "• _... and more_\n"
					break
				}
				response += fmt.Sprintf("• `%s` - %s (%s)\n", 
					session.GetID()[:8], // Show first 8 chars of session ID
					session.GetWorkspaceDir(), 
					session.GetLastActivity().Format("Jan 2 15:04"))
			}
		}

		if len(paths) > 0 {
			response += "\n**Suggested Paths:**\n"
			for i, path := range paths {
				if i >= 5 { // Limit to 5 paths
					response += "• _... and more_\n"
					break
				}
				response += fmt.Sprintf("• `%s`\n", path)
			}
		}

		response += "\n\n**Note:** Each message shows the session ID at the bottom."

		return response
	}

	if args[0] == "list" {
		// Show detailed list of all sessions
		response, err := s.handleSessionListCommand(userID, channelID)
		if err != nil {
			errCtx := logging.CreateErrorContext(channelID, userID, "session_slash_command", "list_sessions")
			return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to list sessions")
		}
		return response
	} else if args[0] == "info" {
		// Show child conversations for a parent session
		if len(args) < 2 {
			return "❌ **Usage:** `/session info <parent-session-uuid>` - Show child conversations for parent session"
		}
		return s.handleSessionInfoCommand(userID, channelID, args[1])
	} else if args[0] == "new" {
		// Handle new session creation with optional path
		var workingDir string
		if len(args) > 1 {
			workingDir = args[1]
		} else {
			workingDir = s.config.WorkingDirectory
		}

		// Create a new session with the specified working directory
		newSession, err := s.sessionManager.CreateSessionWithPath(userID, channelID, workingDir)
		if err != nil {
			s.logger.Error("Failed to create new session", zap.Error(err))
			return "❌ **Error:** Failed to create new session"
		}

		return fmt.Sprintf("✅ **New Conversation Started**\n\nSession ID: `%s`\nWorking directory: `%s`\nNext message will start a fresh conversation with Claude.", newSession.GetID(), workingDir)
	} else if args[0] == "." {
		// Switch to or create session for specific path
		if len(args) < 2 {
			return "❌ **Usage:** `/session . <path>` - Switch to or create session for specific path"
		}

		newPath := args[1]
		
		// Find existing sessions for this path
		existingSessions, err := s.sessionManager.GetSessionsByPath(newPath, 5)
		if err != nil {
			s.logger.Error("Failed to get sessions by path", zap.Error(err))
		}

		if len(existingSessions) == 0 {
			// No existing sessions for this path, create a new one
			// For database sessions, no session manipulation needed

			return fmt.Sprintf("✅ **New Session Created for Path**\n\nWorking directory: `%s`\nNext message will start a fresh conversation in this path.", newPath)
		} else {
			// Found existing sessions, let user choose
			response := fmt.Sprintf("📋 **Found %d existing session(s) for path:** `%s`\n\n", len(existingSessions), newPath)
			response += "**Available Sessions:**\n"
			
			for i, session := range existingSessions {
				if i >= 3 { // Limit to 3 sessions
					response += "• _... and more_\n"
					break
				}
				response += fmt.Sprintf("• `%s` - Last used: %s\n", 
					session.GetID(), 
					session.GetLastActivity().Format("Jan 2 15:04"))
			}
			
			response += "\n**Usage:**\n"
			response += fmt.Sprintf("• `/session %s` - Use most recent session\n", existingSessions[0].GetID()[:8])
			response += fmt.Sprintf("• `/session new %s` - Create new session for this path", newPath)
			
			return response
		}
	} else {
		// Switch to specific Claude session ID
		sessionID := args[0]

		// Validate that the session exists first
		session, err := s.sessionManager.GetSessionBySessionID(sessionID)
		if err != nil {
			errCtx := logging.CreateErrorContext(channelID, userID, "session_switch", "validate_session")
			return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to validate session for switching")
		}

		if session == nil {
			return fmt.Sprintf("❌ **Session not found**\n\nSession `%s` does not exist.", sessionID)
		}

		// Perform the actual session switch
		err = s.sessionManager.SwitchToSessionInChannel(channelID, sessionID)
		if err != nil {
			errCtx := logging.CreateErrorContext(channelID, userID, "session_switch", "update_channel")
			return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to switch session")
		}

		return fmt.Sprintf("✅ **Session Switched**\n\nNow using Claude session: `%s`\n\nNext message will resume this conversation.", sessionID)
	}
}

// handlePermissionSlashCommand handles the /permission slash command
// handleDebugSlashCommand handles the /debug slash command
func (s *Service) handleDebugSlashCommand(userID, channelID string) string {
	// For database sessions, latest response functionality is not yet implemented
	return "❌ Debug response functionality is not available for database sessions yet."
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
	isProcessing := s.sessionManager.IsProcessing(userSession.GetID())
	if !isProcessing {
		return "No active processing to stop.", nil
	}

	// Cancel processing by closing the stop channel
	close(s.stopCh)
	
	// Reinitialize the stop channel for future use
	s.stopCh = make(chan struct{})

	return "✅ Processing stopped.", nil
}

// sendStartupNotification sends a notification to all allowed channels when the bot starts up
func (s *Service) sendStartupNotification() {
	// Use all allowed channels for deployment notifications
	notifyChannels := s.config.AllowedChannels
	
	if len(notifyChannels) == 0 {
		s.logger.Info("No allowed channels configured, skipping startup notification")
		return
	}

	s.logger.Info("Sending startup notification", zap.Strings("channels", notifyChannels))

	// Create notifier
	notifier := notifications.NewDeploymentNotifier(s.slackAPI, notifyChannels, s.logger)

	// Send startup notification in a goroutine to not block startup
	go func() {
		// Wait a few seconds to ensure the bot is fully initialized
		time.Sleep(3 * time.Second)

		changes := []string{
			"Enhanced session management with interactive features",
			"Smart path suggestions based on session history",
			"Improved /session command with session listing",
			"Path-based session switching with /session . <path>",
			"Intelligent session selection for existing paths",
		}

		if err := notifier.NotifyDeployment(changes); err != nil {
			s.logger.Error("Failed to send startup notification", zap.Error(err))
		} else {
			s.logger.Info("Startup notification sent successfully")
		}
	}()
}

// handleSessionListCommand shows a detailed list of all sessions
func (s *Service) handleSessionListCommand(userID, channelID string) (string, error) {
	// Get all sessions (limit to 20 for readability)
	sessions, err := s.sessionManager.ListAllSessions(20)
	if err != nil {
		s.logger.Error("Failed to list sessions", zap.Error(err))
		errCtx := logging.CreateErrorContext(channelID, userID, "session_list", "retrieve_sessions")
		return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to retrieve session list"), err
	}

	if len(sessions) == 0 {
		return "📋 **No Sessions Found**\n\nNo sessions exist yet. Use `/session new` to create your first session.", nil
	}

	// Group sessions by working directory
	sessionsByPath := make(map[string][]session.SessionInfo)
	for _, session := range sessions {
		path := session.GetWorkspaceDir()
		sessionsByPath[path] = append(sessionsByPath[path], session)
	}

	response := fmt.Sprintf("📋 **All Sessions** (%d total)\n\n", len(sessions))

	// Show sessions grouped by path
	pathCount := 0
	for path, pathSessions := range sessionsByPath {
		if pathCount >= 5 { // Limit to 5 paths to avoid overwhelming
			response += fmt.Sprintf("_... and %d more paths_\n", len(sessionsByPath)-pathCount)
			break
		}

		response += fmt.Sprintf("**Path:** `%s` (%d sessions)\n", path, len(pathSessions))
		
		// Show up to 3 sessions per path
		for i, session := range pathSessions {
			if i >= 3 {
				response += fmt.Sprintf("  • _... and %d more sessions_\n", len(pathSessions)-3)
				break
			}
			
			sessionID := session.GetID()
			
			response += fmt.Sprintf("  • `%s` - Last used: %s\n", 
				sessionID,
				session.GetLastActivity().Format("Jan 2 15:04"))
		}
		response += "\n"
		pathCount++
	}

	response += "**Usage:**\n"
	response += "• `/session <session-id>` - Switch to specific session\n" 
	response += "• `/session . <path>` - Switch to or create session for path\n"
	response += "• `/session new <path>` - Create new session for path"

	return response, nil
}

// handleSessionInfoCommand shows child conversations for a parent session
func (s *Service) handleSessionInfoCommand(userID, channelID, parentSessionID string) string {
	// First, get the parent session from the database by session ID
	session, err := s.sessionManager.GetSessionBySessionID(parentSessionID)
	if err != nil {
		errCtx := logging.CreateErrorContext(channelID, userID, "session_info", "get_parent_session")
		return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to get parent session")
	}
	
	if session == nil {
		return "❌ **Parent session ID does not exist**"
	}
	
	// Get the conversation tree (all child sessions)
	children, err := s.sessionManager.GetConversationTree(parentSessionID)
	if err != nil {
		errCtx := logging.CreateErrorContext(channelID, userID, "session_info", "get_conversation_tree")
		return s.logErrorWithTrace(context.Background(), errCtx, err, "Failed to get conversation tree")
	}
	
	// Build response
	response := fmt.Sprintf("📋 **Session Info for: `%s`**\n\n", parentSessionID)
	
	if len(children) == 0 {
		response += "**Child Conversations:** None (new session with no conversations yet)"
	} else {
		response += fmt.Sprintf("**Child Conversations (%d total):**\n", len(children))
		for _, child := range children {
			response += fmt.Sprintf("• `%s` - Created: %s\n", 
				child.SessionID,
				child.CreatedAt.Format("Jan 2 15:04"))
		}
	}
	
	return response
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
		currentMode, err := s.getPermissionModeForChannel(channelID, userSession.GetID())
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

	// Set mode - use channel-based permissions if available
	if channelPermMgr, ok := s.sessionManager.(session.ChannelPermissionManager); ok {
		err = channelPermMgr.SetPermissionModeForChannel(channelID, mode)
	} else {
		err = s.sessionManager.SetPermissionMode(userSession.GetID(), mode)
	}
	
	if err != nil {
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

// getPermissionModeForChannel is a helper that gets permission mode using channel ID when available
func (s *Service) getPermissionModeForChannel(channelID string, fallbackSessionID string) (config.PermissionMode, error) {
	// Use channel-based permissions if available
	if channelPermMgr, ok := s.sessionManager.(session.ChannelPermissionManager); ok {
		return channelPermMgr.GetPermissionModeForChannel(channelID)
	}
	// Fallback to session-based permissions
	return s.sessionManager.GetPermissionMode(fallbackSessionID)
}

// logErrorWithTrace logs an error using the dual logger and returns a user-friendly message
func (s *Service) logErrorWithTrace(ctx context.Context, errCtx *logging.ErrorContext, err error, message string) string {
	// Use dual logger to send to both console and Slack
	s.dualLogger.LogError(ctx, errCtx, err, message)
	
	// Return a simplified message for immediate response
	return fmt.Sprintf("❌ %s: %v", message, err)
}

// IsImageMimeType checks if the given mime type is a supported image format
func (s *Service) IsImageMimeType(mimeType string) bool {
	supportedTypes := []string{
		"image/jpeg",
		"image/png", 
		"image/gif",
		"image/webp",
	}
	
	for _, supported := range supportedTypes {
		if mimeType == supported {
			return true
		}
	}
	return false
}

// handleDeleteCommand handles the /delete slash command
func (s *Service) handleDeleteCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read delete command body", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Verify Slack signature (if configured)
	if s.config.SlackSigningSecret != "" {
		if !s.verifySlackSignature(r.Header, bodyBytes) {
			s.logger.Warn("Invalid Slack signature for delete command")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse form data
	formData, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		s.logger.Error("Failed to parse delete command form data", zap.Error(err))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	text := formData.Get("text")
	userID := formData.Get("user_id")
	channelID := formData.Get("channel_id")

	s.logger.Info("Received delete command",
		zap.String("text", text),
		zap.String("user_id", userID),
		zap.String("channel_id", channelID))

	// Authorize user
	authCtx := &auth.AuthContext{
		UserID:    userID,
		ChannelID: channelID,
		Command:   "/delete",
		Timestamp: time.Now(),
	}

	if err := s.authService.AuthorizeUser(authCtx, auth.PermissionWrite); err != nil {
		response := fmt.Sprintf("❌ Authorization failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"text": response})
		return
	}

	// Process delete command
	response := s.handleDeleteSessionCommand(userID, channelID, text)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": response})
}

// handleDeleteSessionCommand processes the delete session command
func (s *Service) handleDeleteSessionCommand(userID, channelID, text string) string {
	args := strings.Fields(text)
	
	if len(args) == 0 {
		return "❌ **Usage:** `/delete <session-id>` - Delete a specific session"
	}

	sessionID := args[0]
	
	// Try to delete the session
	err := s.sessionManager.DeleteSession(sessionID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("❌ **Session Not Found**\n\nSession `%s` does not exist or may have already been deleted.", sessionID)
		}
		s.logger.Error("Failed to delete session", zap.Error(err))
		return fmt.Sprintf("❌ **Delete Failed**\n\nFailed to delete session `%s`: %v", sessionID, err)
	}

	return fmt.Sprintf("✅ **Session Deleted**\n\nSession `%s` has been successfully deleted along with all its conversation history.", sessionID)
}
