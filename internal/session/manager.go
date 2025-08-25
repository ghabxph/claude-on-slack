package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/config"
	"github.com/ghabxph/claude-on-slack/internal/claude"
)

// MessageQueue tracks queued messages while processing
type MessageQueue struct {
	Messages     []string  `json:"messages"`      // Queued messages
	LastUpdate   time.Time `json:"last_update"`   // Time of last message
	IsProcessing bool      `json:"is_processing"` // Whether Claude is processing
}

// Session represents a conversation session
type Session struct {
	ID            string                 `json:"id"`
	UserID        string                 `json:"user_id"`
	ChannelID     string                 `json:"channel_id"`
	CreatedAt     time.Time              `json:"created_at"`
	LastActivity  time.Time              `json:"last_activity"`
	MessageCount  int                    `json:"message_count"`
	WorkspaceDir    string                 `json:"workspace_dir"`
	CurrentWorkDir  string                 `json:"current_work_dir"` // Current working directory from Claude
	History         []claude.Message       `json:"history"`
	Context       map[string]interface{} `json:"context"`
	IsActive      bool                   `json:"is_active"`
	TokensUsed    int                    `json:"tokens_used"`
	RateLimitInfo *RateLimitInfo         `json:"rate_limit_info"`
	ExecutionMutex  sync.Mutex             `json:"-"` // Prevents concurrent executions within same session
	ClaudeSessionID string                 `json:"claude_session_id"` // Current Claude Code session ID
	MessageQueue    *MessageQueue          `json:"message_queue"`     // Queue for combining messages
	PermissionMode  config.PermissionMode  `json:"permission_mode"`   // Current Claude permission mode
	LatestResponse  string                 `json:"latest_response"`   // Latest raw JSON response from Claude
}

// RateLimitInfo tracks rate limiting for a session
type RateLimitInfo struct {
	RequestCount    int       `json:"request_count"`
	LastRequestTime time.Time `json:"last_request_time"`
	WindowStart     time.Time `json:"window_start"`
	IsLimited       bool      `json:"is_limited"`
	LimitUntil      time.Time `json:"limit_until"`
}

// Manager handles session management
type Manager struct {
	config     *config.Config
	logger     *zap.Logger
	sessions   map[string]*Session
	userSessions map[string][]*Session
	mu         sync.RWMutex
	executor   *claude.Executor
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewManager creates a new session manager
func NewManager(cfg *config.Config, logger *zap.Logger, executor *claude.Executor) *Manager {
	m := &Manager{
		config:       cfg,
		logger:       logger,
		sessions:     make(map[string]*Session),
		userSessions: make(map[string][]*Session),
		executor:     executor,
		stopCh:       make(chan struct{}),
	}

	// Start cleanup routine
	m.startCleanupRoutine()

	return m
}

// CreateSession creates a new session for a user
func (m *Manager) CreateSession(userID, channelID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user has reached max sessions limit
	if userSessions := m.userSessions[userID]; len(userSessions) >= m.config.MaxSessionsPerUser {
		// Remove oldest inactive session
		var oldestSession *Session
		var oldestIndex int
		for i, session := range userSessions {
			if !session.IsActive && (oldestSession == nil || session.LastActivity.Before(oldestSession.LastActivity)) {
				oldestSession = session
				oldestIndex = i
			}
		}

		if oldestSession != nil {
			m.logger.Info("Removing oldest session due to limit",
				zap.String("user_id", userID),
				zap.String("old_session_id", oldestSession.ID))
			
			// Remove from sessions map
			delete(m.sessions, oldestSession.ID)
			
			// Remove from user sessions slice
			m.userSessions[userID] = append(userSessions[:oldestIndex], userSessions[oldestIndex+1:]...)
			
			// Cleanup workspace
			if oldestSession.WorkspaceDir != "" {
				go func(workspaceDir string) {
					if err := m.executor.CleanupWorkspace(workspaceDir); err != nil {
						m.logger.Error("Failed to cleanup workspace", zap.Error(err))
					}
				}(oldestSession.WorkspaceDir)
			}
		} else {
			return nil, fmt.Errorf("user %s has reached maximum number of active sessions (%d)", userID, m.config.MaxSessionsPerUser)
		}
	}

	// Generate session ID
	sessionID := uuid.New().String()

	// Create workspace
	workspaceDir, err := m.executor.CreateWorkspace(userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create session
	session := &Session{
		ID:           sessionID,
		UserID:       userID,
		ChannelID:    channelID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		WorkspaceDir: workspaceDir,
		History:      make([]claude.Message, 0),
		Context:      make(map[string]interface{}),
		IsActive:     true,
		RateLimitInfo: &RateLimitInfo{
			WindowStart: time.Now(),
		},
		MessageQueue: &MessageQueue{
			Messages: make([]string, 0),
		},
		PermissionMode: config.PermissionModeDefault,
		LatestResponse: "",
	}

	// Store session
	m.sessions[sessionID] = session
	m.userSessions[userID] = append(m.userSessions[userID], session)

	m.logger.Info("Created new session",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("channel_id", channelID),
		zap.String("workspace", workspaceDir))

	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

// GetActiveSessionsForUser returns all active sessions for a user
func (m *Manager) GetActiveSessionsForUser(userID string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	userSessions := m.userSessions[userID]
	activeSessions := make([]*Session, 0)

	for _, session := range userSessions {
		if session.IsActive {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions
}

// GetOrCreateSession gets an existing active session or creates a new one
func (m *Manager) GetOrCreateSession(userID, channelID string) (*Session, error) {
	// Check for existing active session in the same channel
	activeSessions := m.GetActiveSessionsForUser(userID)
	for _, session := range activeSessions {
		if session.ChannelID == channelID && session.IsActive {
			return session, nil
		}
	}

	// Create new session
	return m.CreateSession(userID, channelID)
}

// UpdateSessionActivity updates the last activity time for a session
func (m *Manager) UpdateSessionActivity(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.LastActivity = time.Now()
	return nil
}

// AddMessageToSession adds a message to session history
func (m *Manager) AddMessageToSession(sessionID string, message claude.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.History = append(session.History, message)
	session.MessageCount++
	session.LastActivity = time.Now()

	// Limit history size to prevent memory issues
	maxHistorySize := 50 // Keep last 50 messages
	if len(session.History) > maxHistorySize {
		session.History = session.History[len(session.History)-maxHistorySize:]
	}

	m.logger.Debug("Added message to session",
		zap.String("session_id", sessionID),
		zap.String("role", message.Role),
		zap.Int("history_length", len(session.History)))

	return nil
}

// CloseSession closes a session and cleans up resources
func (m *Manager) CloseSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	// Mark as inactive
	session.IsActive = false

	// Remove from maps
	delete(m.sessions, sessionID)

	// Remove from user sessions
	userSessions := m.userSessions[session.UserID]
	for i, s := range userSessions {
		if s.ID == sessionID {
			m.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
			break
		}
	}

	m.logger.Info("Closed session",
		zap.String("session_id", sessionID),
		zap.String("user_id", session.UserID),
		zap.Int("message_count", session.MessageCount),
		zap.Duration("duration", time.Since(session.CreatedAt)))

	// Cleanup workspace in background
	if session.WorkspaceDir != "" {
		go func(workspaceDir string) {
			if err := m.executor.CleanupWorkspace(workspaceDir); err != nil {
				m.logger.Error("Failed to cleanup workspace", zap.Error(err))
			}
		}(session.WorkspaceDir)
	}

	return nil
}

// CheckRateLimit checks if a user/session is rate limited
func (m *Manager) CheckRateLimit(sessionID string) (bool, time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false, 0, fmt.Errorf("session %s not found", sessionID)
	}

	rateLimitInfo := session.RateLimitInfo
	now := time.Now()

	// Check if currently rate limited
	if rateLimitInfo.IsLimited && now.Before(rateLimitInfo.LimitUntil) {
		remaining := rateLimitInfo.LimitUntil.Sub(now)
		return true, remaining, nil
	}

	// Reset rate limit if window has passed
	if now.Sub(rateLimitInfo.WindowStart) > time.Minute {
		rateLimitInfo.RequestCount = 0
		rateLimitInfo.WindowStart = now
		rateLimitInfo.IsLimited = false
	}

	// Check if exceeding rate limit
	if rateLimitInfo.RequestCount >= m.config.RateLimitPerMinute {
		rateLimitInfo.IsLimited = true
		rateLimitInfo.LimitUntil = now.Add(time.Minute)
		return true, time.Minute, nil
	}

	// Increment request count
	rateLimitInfo.RequestCount++
	rateLimitInfo.LastRequestTime = now

	return false, 0, nil
}

// GetSessionStats returns statistics about sessions
func (m *Manager) GetSessionStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalSessions := len(m.sessions)
	activeSessions := 0
	totalUsers := len(m.userSessions)
	totalMessages := 0

	for _, session := range m.sessions {
		if session.IsActive {
			activeSessions++
		}
		totalMessages += session.MessageCount
	}

	return map[string]interface{}{
		"total_sessions":  totalSessions,
		"active_sessions": activeSessions,
		"total_users":     totalUsers,
		"total_messages":  totalMessages,
	}
}

// startCleanupRoutine starts the background cleanup routine
func (m *Manager) startCleanupRoutine() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		ticker := time.NewTicker(m.config.SessionCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.cleanupExpiredSessions()
			case <-m.stopCh:
				return
			}
		}
	}()
}

// cleanupExpiredSessions removes expired sessions
func (m *Manager) cleanupExpiredSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	expiredSessions := make([]*Session, 0)

	for sessionID, session := range m.sessions {
		if now.Sub(session.LastActivity) > m.config.SessionTimeout {
			expiredSessions = append(expiredSessions, session)
			delete(m.sessions, sessionID)

			// Remove from user sessions
			userSessions := m.userSessions[session.UserID]
			for i, s := range userSessions {
				if s.ID == sessionID {
					m.userSessions[session.UserID] = append(userSessions[:i], userSessions[i+1:]...)
					break
				}
			}
		}
	}

	if len(expiredSessions) > 0 {
		m.logger.Info("Cleaned up expired sessions",
			zap.Int("count", len(expiredSessions)))

		// Cleanup workspaces in background
		for _, session := range expiredSessions {
			if session.WorkspaceDir != "" {
				go func(workspaceDir string) {
					if err := m.executor.CleanupWorkspace(workspaceDir); err != nil {
						m.logger.Error("Failed to cleanup workspace", zap.Error(err))
					}
				}(session.WorkspaceDir)
			}
		}
	}
}

// Stop stops the session manager and cleanup routines
func (m *Manager) Stop() {
	close(m.stopCh)
	m.wg.Wait()

	m.logger.Info("Session manager stopped")
}

// ListUserSessions returns formatted list of user sessions
func (m *Manager) ListUserSessions(userID string) string {
	activeSessions := m.GetActiveSessionsForUser(userID)
	
	if len(activeSessions) == 0 {
		return "No active sessions found."
	}

	result := fmt.Sprintf("Active sessions (%d):\n", len(activeSessions))
	for i, session := range activeSessions {
		duration := time.Since(session.CreatedAt).Truncate(time.Second)
		result += fmt.Sprintf("%d. Session %s (Channel: <#%s>, Duration: %v, Messages: %d)\n",
			i+1, session.ID[:8], session.ChannelID, duration, session.MessageCount)
	}

	return result
}

// QueueMessage adds a message to the queue if processing, or returns false if ready to process
func (m *Manager) QueueMessage(sessionID string, message string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false, fmt.Errorf("session %s not found", sessionID)
	}

	if session.MessageQueue == nil {
		session.MessageQueue = &MessageQueue{
			Messages: make([]string, 0),
		}
	}

	// If processing, queue the message
	if session.MessageQueue.IsProcessing {
		session.MessageQueue.Messages = append(session.MessageQueue.Messages, message)
		session.MessageQueue.LastUpdate = time.Now()
		return true, nil
	}

	// Not processing, ready to handle message
	return false, nil
}

// SetProcessing marks a session as processing or not
func (m *Manager) SetProcessing(sessionID string, processing bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	if session.MessageQueue == nil {
		session.MessageQueue = &MessageQueue{
			Messages: make([]string, 0),
		}
	}

	session.MessageQueue.IsProcessing = processing
	return nil
}

// GetQueuedMessages gets and clears the message queue
func (m *Manager) GetQueuedMessages(sessionID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	if session.MessageQueue == nil || len(session.MessageQueue.Messages) == 0 {
		return nil, nil
	}

	messages := session.MessageQueue.Messages
	session.MessageQueue.Messages = make([]string, 0)
	return messages, nil
}

// UpdateCurrentWorkDir updates the current working directory
func (m *Manager) UpdateCurrentWorkDir(sessionID string, workDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.CurrentWorkDir = workDir
	return nil
}

// SetPermissionMode sets the permission mode for a session
func (m *Manager) SetPermissionMode(sessionID string, mode config.PermissionMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.PermissionMode = mode
	return nil
}

// UpdateLatestResponse updates the latest response for a session
func (m *Manager) UpdateLatestResponse(sessionID string, response string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.LatestResponse = response
	return nil
}

// IsProcessing checks if a session is currently processing
func (m *Manager) IsProcessing(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return false
	}

	if session.MessageQueue == nil {
		return false
	}

	return session.MessageQueue.IsProcessing
}

// GetPermissionMode gets the permission mode for a session
func (m *Manager) GetPermissionMode(sessionID string) (config.PermissionMode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	if session.PermissionMode == "" {
		return config.PermissionModeDefault, nil
	}
	return session.PermissionMode, nil
}