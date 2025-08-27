package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/config"
	"github.com/ghabxph/claude-on-slack/internal/claude"
	"github.com/ghabxph/claude-on-slack/internal/repository"
	"github.com/ghabxph/claude-on-slack/internal/database"
)

// DatabaseManager handles database-backed session management
type DatabaseManager struct {
	config     *config.Config
	logger     *zap.Logger
	repository *repository.SessionRepository
	executor   *claude.Executor
	
	// Memory optimization: conversation trees loaded on demand
	conversationTrees map[int][]*repository.ChildSession  // keyed by root_parent_id
	sessionLookup     map[string]*repository.Session       // keyed by session_id for O(1) lookup
	mu               sync.RWMutex
}

// NewDatabaseManager creates a new database-backed session manager
func NewDatabaseManager(cfg *config.Config, logger *zap.Logger, executor *claude.Executor, db *database.Database) *DatabaseManager {
	repo := repository.NewSessionRepository(db, logger)
	
	return &DatabaseManager{
		config:            cfg,
		logger:            logger,
		repository:        repo,
		executor:          executor,
		conversationTrees: make(map[int][]*repository.ChildSession),
		sessionLookup:     make(map[string]*repository.Session),
	}
}

// CreateSession creates a new database-backed session
func (m *DatabaseManager) CreateSession(userID, channelID string) (*repository.Session, error) {
	// Generate session ID
	sessionID := uuid.New().String()

	// Create workspace
	workspaceDir, err := m.executor.CreateWorkspace(userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// Create session in database
	session := &repository.Session{
		SessionID:        sessionID,
		WorkingDirectory: workspaceDir,
		SystemUser:       userID,
		UserPrompt:       nil, // Will be set when user sends first message
	}

	if err := m.repository.CreateSession(session); err != nil {
		// Cleanup workspace if database creation fails
		go func() {
			if cleanupErr := m.executor.CleanupWorkspace(workspaceDir); cleanupErr != nil {
				m.logger.Error("Failed to cleanup workspace after session creation failure", zap.Error(cleanupErr))
			}
		}()
		return nil, fmt.Errorf("failed to create session in database: %w", err)
	}

	// Update channel state to point to new session
	if err := m.repository.UpdateChannelState(channelID, &session.ID, nil); err != nil {
		m.logger.Error("Failed to update channel state", zap.Error(err))
	}

	// Cache in memory for O(1) lookup
	m.mu.Lock()
	m.sessionLookup[sessionID] = session
	m.mu.Unlock()

	m.logger.Info("Created new database session",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.String("channel_id", channelID),
		zap.String("workspace", workspaceDir),
		zap.Int("db_id", session.ID))

	return session, nil
}

// GetOrCreateSession gets existing session for channel or creates new one
func (m *DatabaseManager) GetOrCreateSession(userID, channelID string) (*repository.Session, error) {
	// Check channel state for existing active session
	channelState, err := m.repository.GetChannelState(channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel state: %w", err)
	}

	if channelState != nil && channelState.ActiveSessionID != nil {
		// Load existing session from cache or database
		session, err := m.loadSessionByID(*channelState.ActiveSessionID)
		if err != nil {
			m.logger.Error("Failed to load existing session, creating new one", 
				zap.Error(err), 
				zap.Int("session_id", *channelState.ActiveSessionID))
		} else {
			return session, nil
		}
	}

	// Create new session
	return m.CreateSession(userID, channelID)
}

// LoadConversationTree loads entire conversation tree into memory for O(1) processing
func (m *DatabaseManager) LoadConversationTree(rootParentID int) ([]*repository.ChildSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already cached
	if tree, exists := m.conversationTrees[rootParentID]; exists {
		return tree, nil
	}

	// Load from database
	tree, err := m.repository.GetConversationTree(rootParentID)
	if err != nil {
		return nil, err
	}

	// Cache for future O(1) access
	m.conversationTrees[rootParentID] = tree
	
	m.logger.Debug("Loaded conversation tree", 
		zap.Int("root_parent_id", rootParentID),
		zap.Int("child_count", len(tree)))

	return tree, nil
}

// ProcessUserMessage handles user message with database persistence
func (m *DatabaseManager) ProcessUserMessage(sessionID string, message string) error {
	session, err := m.getSessionBySessionID(sessionID)
	if err != nil {
		return err
	}

	// If this is the first message, update the root session
	if session.UserPrompt == nil {
		if err := m.repository.UpdateSessionUserPrompt(sessionID, message); err != nil {
			return fmt.Errorf("failed to update session user prompt: %w", err)
		}
		session.UserPrompt = &message
		return nil
	}

	// Find leaf child session or create conversation tree
	leafChild, err := m.repository.FindLeafChild(session.ID)
	if err != nil {
		return fmt.Errorf("failed to find leaf child: %w", err)
	}

	if leafChild != nil {
		// Update existing leaf with user prompt
		if err := m.repository.UpdateChildUserPrompt(leafChild.ID, message); err != nil {
			return fmt.Errorf("failed to update child user prompt: %w", err)
		}
	}

	return nil
}

// ProcessAIResponse creates new child session with AI response
func (m *DatabaseManager) ProcessAIResponse(sessionID string, aiResponse string) error {
	session, err := m.getSessionBySessionID(sessionID)
	if err != nil {
		return err
	}

	// Generate new session ID for this response
	newChildSessionID := uuid.New().String()

	// Find current leaf to link as previous
	leafChild, err := m.repository.FindLeafChild(session.ID)
	if err != nil {
		return fmt.Errorf("failed to find leaf child: %w", err)
	}

	var previousSessionID *int
	if leafChild != nil {
		previousSessionID = &leafChild.ID
	}

	// Create new child session
	childSession := &repository.ChildSession{
		SessionID:         newChildSessionID,
		PreviousSessionID: previousSessionID,
		RootParentID:      session.ID,
		AIResponse:        &aiResponse,
		UserPrompt:        nil, // Will be set when user responds
	}

	if err := m.repository.CreateChildSession(childSession); err != nil {
		return fmt.Errorf("failed to create child session: %w", err)
	}

	// Update conversation tree cache
	m.mu.Lock()
	if tree, exists := m.conversationTrees[session.ID]; exists {
		m.conversationTrees[session.ID] = append(tree, childSession)
	}
	m.mu.Unlock()

	m.logger.Debug("Created child session for AI response",
		zap.String("session_id", sessionID),
		zap.String("child_session_id", newChildSessionID),
		zap.Int("root_parent_id", session.ID))

	return nil
}

// getSessionBySessionID retrieves session with caching
func (m *DatabaseManager) getSessionBySessionID(sessionID string) (*repository.Session, error) {
	m.mu.RLock()
	if session, exists := m.sessionLookup[sessionID]; exists {
		m.mu.RUnlock()
		return session, nil
	}
	m.mu.RUnlock()

	// Load from database
	session, err := m.repository.GetSessionBySessionID(sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Cache for future lookups
	m.mu.Lock()
	m.sessionLookup[sessionID] = session
	m.mu.Unlock()

	return session, nil
}

// loadSessionByID loads session by database ID with caching
func (m *DatabaseManager) loadSessionByID(id int) (*repository.Session, error) {
	// Check cache first by iterating (could optimize with reverse lookup map)
	m.mu.RLock()
	for _, session := range m.sessionLookup {
		if session.ID == id {
			m.mu.RUnlock()
			return session, nil
		}
	}
	m.mu.RUnlock()

	// Load from database
	session, err := m.repository.GetSessionByID(id)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("session with ID %d not found", id)
	}

	// Cache for future lookups
	m.mu.Lock()
	m.sessionLookup[session.SessionID] = session
	m.mu.Unlock()

	return session, nil
}

// SwitchToSession handles session switching and branching
func (m *DatabaseManager) SwitchToSession(sessionID string) error {
	session, err := m.getSessionBySessionID(sessionID)
	if err != nil {
		return err
	}

	// Load conversation tree to memory for fast access
	_, err = m.LoadConversationTree(session.ID)
	if err != nil {
		return fmt.Errorf("failed to load conversation tree: %w", err)
	}

	return nil
}

// GetSessionStats returns database-backed session statistics
func (m *DatabaseManager) GetSessionStats() map[string]interface{} {
	// Could implement with database queries for accuracy
	// For now, return cache-based stats
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"cached_sessions":        len(m.sessionLookup),
		"cached_conversation_trees": len(m.conversationTrees),
		"database_backed":        true,
	}
}

// ListAllSessions returns all sessions with pagination (SessionManager interface)
func (m *DatabaseManager) ListAllSessions(limit int) ([]SessionInfo, error) {
	sessions, err := m.repository.ListAllSessions(limit)
	if err != nil {
		return nil, err
	}

	var sessionInfos []SessionInfo
	for _, session := range sessions {
		sessionInfos = append(sessionInfos, &DbSessionInfo{session})
	}

	return sessionInfos, nil
}

// GetKnownPaths returns unique working directories from all sessions (SessionManager interface)
func (m *DatabaseManager) GetKnownPaths(limit int) ([]string, error) {
	return m.repository.GetUniqueWorkingDirectories(limit)
}

// GetSessionsByPath returns sessions for a specific path (database implementation)
func (m *DatabaseManager) GetSessionsByPath(path string, limit int) ([]SessionInfo, error) {
	sessions, err := m.repository.GetSessionsByWorkingDirectory(path, limit)
	if err != nil {
		return nil, err
	}

	var sessionInfos []SessionInfo
	for _, session := range sessions {
		sessionInfos = append(sessionInfos, &DbSessionInfo{session})
	}

	return sessionInfos, nil
}

// DbSessionInfo wraps repository.Session to implement SessionInfo interface
type DbSessionInfo struct {
	*repository.Session
}

// Ensure DbSessionInfo implements SessionInfo
var _ SessionInfo = (*DbSessionInfo)(nil)

// SessionInfo implementation for database Session
func (s *DbSessionInfo) GetID() string                         { return s.SessionID }
func (s *DbSessionInfo) GetUserID() string                     { return s.SystemUser }
func (s *DbSessionInfo) GetChannelID() string                  { return "" } // Not stored in DB session
func (s *DbSessionInfo) GetWorkspaceDir() string               { return s.WorkingDirectory }
func (s *DbSessionInfo) GetCurrentWorkDir() string             { return s.WorkingDirectory }
func (s *DbSessionInfo) GetPermissionMode() config.PermissionMode { return config.PermissionModeDefault } // Default for DB sessions
func (s *DbSessionInfo) GetCreatedAt() time.Time               { return s.CreatedAt }
func (s *DbSessionInfo) GetLastActivity() time.Time            { return s.UpdatedAt }
func (s *DbSessionInfo) IsActive() bool                        { return true } // DB sessions are considered active

// Stop cleanup resources (no background routines in database mode)
func (m *DatabaseManager) Stop() {
	m.logger.Info("Database session manager stopped")
}