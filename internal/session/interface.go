package session

import (
	"time"

	"github.com/ghabxph/claude-on-slack/internal/config"
	"github.com/ghabxph/claude-on-slack/internal/claude"
)

// SessionManager interface defines the contract for session management
type SessionManager interface {
	// Session lifecycle
	CreateSession(userID, channelID string) (SessionInfo, error)
	CreateSessionWithPath(userID, channelID, workingDir string) (SessionInfo, error)
	GetOrCreateSession(userID, channelID string) (SessionInfo, error)
	CloseSession(sessionID string) error
	DeleteSession(sessionID string) error

	// Session operations
	UpdateSessionActivity(sessionID string) error
	AddMessageToSession(sessionID string, message claude.Message) error
	CheckRateLimit(sessionID string) (bool, time.Duration, error)
	GetLatestChildSessionID(sessionID string) (*string, error)

	// Permission and state management
	SetPermissionMode(sessionID string, mode config.PermissionMode) error
	GetPermissionMode(sessionID string) (config.PermissionMode, error)
	UpdateLatestResponse(sessionID string, response string) error
	UpdateCurrentWorkDir(sessionID string, workDir string) error

	// Message queuing
	QueueMessage(sessionID string, message string) (bool, error)
	SetProcessing(sessionID string, processing bool) error
	GetQueuedMessages(sessionID string) ([]string, error)
	IsProcessing(sessionID string) bool

	// User and statistics
	GetActiveSessionsForUser(userID string) []SessionInfo
	ListUserSessions(userID string) string
	GetSessionStats() map[string]interface{}
	GetTotalMessageCount(sessionID string) (int, error)

	// Session listing and paths (for enhanced /session command)
	ListAllSessions(limit int) ([]SessionInfo, error)
	GetKnownPaths(limit int) ([]string, error)
	GetSessionsByPath(path string, limit int) ([]SessionInfo, error)

	// Lifecycle
	Stop()
}

// ChannelPermissionManager is an optional extension interface for channel-based permissions
type ChannelPermissionManager interface {
	SetPermissionModeForChannel(channelID string, mode config.PermissionMode) error
	GetPermissionModeForChannel(channelID string) (config.PermissionMode, error)
}

// SessionInfo provides a common interface for session data
type SessionInfo interface {
	GetID() string
	GetUserID() string
	GetChannelID() string
	GetWorkspaceDir() string
	GetCurrentWorkDir() string
	GetPermissionMode() config.PermissionMode
	GetCreatedAt() time.Time
	GetLastActivity() time.Time
	IsActive() bool
}

// Ensure current Session implements SessionInfo
var _ SessionInfo = (*Session)(nil)

// SessionInfo implementation for current Session
func (s *Session) GetID() string                         { return s.ID }
func (s *Session) GetUserID() string                     { return s.UserID }
func (s *Session) GetChannelID() string                  { return s.ChannelID }
func (s *Session) GetWorkspaceDir() string               { return s.WorkspaceDir }
func (s *Session) GetCurrentWorkDir() string             { return s.CurrentWorkDir }
func (s *Session) GetPermissionMode() config.PermissionMode { return s.PermissionMode }
func (s *Session) GetCreatedAt() time.Time               { return s.CreatedAt }
func (s *Session) GetLastActivity() time.Time            { return s.LastActivity }
func (s *Session) IsActive() bool                        { return s.Active }