package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/database"
)

type Session struct {
	ID               int       `db:"id"`
	SessionID        string    `db:"session_id"`
	WorkingDirectory string    `db:"working_directory"`
	SystemUser       string    `db:"system_user"`
	UserPrompt       *string   `db:"user_prompt"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}

type ChildSession struct {
	ID                int       `db:"id"`
	SessionID         string    `db:"session_id"`
	PreviousSessionID *int      `db:"previous_session_id"`
	RootParentID      int       `db:"root_parent_id"`
	AIResponse        *string   `db:"ai_response"`
	UserPrompt        *string   `db:"user_prompt"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type SlackChannel struct {
	ID                    int       `db:"id"`
	ChannelID             string    `db:"channel_id"`
	ActiveSessionID       *int      `db:"active_session_id"`
	ActiveChildSessionID  *int      `db:"active_child_session_id"`
	CreatedAt             time.Time `db:"created_at"`
	UpdatedAt             time.Time `db:"updated_at"`
}

type SessionRepository struct {
	db     *database.Database
	logger *zap.Logger
}

func NewSessionRepository(db *database.Database, logger *zap.Logger) *SessionRepository {
	return &SessionRepository{
		db:     db,
		logger: logger,
	}
}

// CreateSession inserts a new root session
func (r *SessionRepository) CreateSession(session *Session) error {
	query := `
		INSERT INTO sessions (session_id, working_directory, system_user, user_prompt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id`

	err := r.db.GetDB().QueryRow(query, session.SessionID, session.WorkingDirectory, 
		session.SystemUser, session.UserPrompt).Scan(&session.ID)
	
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	r.logger.Debug("Session created", 
		zap.String("session_id", session.SessionID),
		zap.Int("id", session.ID))

	return nil
}

// CreateChildSession inserts a new child session in the conversation
func (r *SessionRepository) CreateChildSession(childSession *ChildSession) error {
	query := `
		INSERT INTO child_sessions (session_id, previous_session_id, root_parent_id, 
			ai_response, user_prompt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id`

	err := r.db.GetDB().QueryRow(query, childSession.SessionID, childSession.PreviousSessionID,
		childSession.RootParentID, childSession.AIResponse, childSession.UserPrompt).Scan(&childSession.ID)

	if err != nil {
		return fmt.Errorf("failed to create child session: %w", err)
	}

	r.logger.Debug("Child session created",
		zap.String("session_id", childSession.SessionID),
		zap.Int("id", childSession.ID),
		zap.Int("root_parent_id", childSession.RootParentID))

	return nil
}

// GetConversationTree loads entire conversation tree for O(1) memory processing
func (r *SessionRepository) GetConversationTree(rootParentID int) ([]*ChildSession, error) {
	query := `SELECT * FROM child_sessions WHERE root_parent_id = $1 ORDER BY id`
	
	rows, err := r.db.GetDB().Query(query, rootParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load conversation tree: %w", err)
	}
	defer rows.Close()

	var children []*ChildSession
	for rows.Next() {
		child := &ChildSession{}
		err := rows.Scan(&child.ID, &child.SessionID, &child.PreviousSessionID,
			&child.RootParentID, &child.AIResponse, &child.UserPrompt,
			&child.CreatedAt, &child.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan child session: %w", err)
		}
		children = append(children, child)
	}

	return children, nil
}

// GetSessionBySessionID retrieves a root session by its session ID
func (r *SessionRepository) GetSessionBySessionID(sessionID string) (*Session, error) {
	query := `SELECT * FROM sessions WHERE session_id = $1`
	
	session := &Session{}
	err := r.db.GetDB().QueryRow(query, sessionID).Scan(
		&session.ID, &session.SessionID, &session.WorkingDirectory,
		&session.SystemUser, &session.UserPrompt, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// FindLeafChild finds the latest child session (conversation endpoint)
func (r *SessionRepository) FindLeafChild(rootParentID int) (*ChildSession, error) {
	query := `SELECT * FROM child_sessions WHERE root_parent_id = $1 ORDER BY id DESC LIMIT 1`
	
	child := &ChildSession{}
	err := r.db.GetDB().QueryRow(query, rootParentID).Scan(
		&child.ID, &child.SessionID, &child.PreviousSessionID,
		&child.RootParentID, &child.AIResponse, &child.UserPrompt,
		&child.CreatedAt, &child.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No children found
		}
		return nil, fmt.Errorf("failed to find leaf child: %w", err)
	}

	return child, nil
}

// UpdateSessionUserPrompt updates the user prompt for a root session
func (r *SessionRepository) UpdateSessionUserPrompt(sessionID string, prompt string) error {
	query := `UPDATE sessions SET user_prompt = $1, updated_at = NOW() WHERE session_id = $2`
	
	_, err := r.db.GetDB().Exec(query, prompt, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session user prompt: %w", err)
	}

	return nil
}

// UpdateChildUserPrompt updates the user prompt for a child session
func (r *SessionRepository) UpdateChildUserPrompt(childID int, prompt string) error {
	query := `UPDATE child_sessions SET user_prompt = $1, updated_at = NOW() WHERE id = $2`
	
	_, err := r.db.GetDB().Exec(query, prompt, childID)
	if err != nil {
		return fmt.Errorf("failed to update child user prompt: %w", err)
	}

	return nil
}

// GetChannelState retrieves the active session state for a Slack channel
func (r *SessionRepository) GetChannelState(channelID string) (*SlackChannel, error) {
	query := `SELECT * FROM slack_channels WHERE channel_id = $1`
	
	channel := &SlackChannel{}
	err := r.db.GetDB().QueryRow(query, channelID).Scan(
		&channel.ID, &channel.ChannelID, &channel.ActiveSessionID,
		&channel.ActiveChildSessionID, &channel.CreatedAt, &channel.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Channel not found
		}
		return nil, fmt.Errorf("failed to get channel state: %w", err)
	}

	return channel, nil
}

// UpdateChannelState updates the active session for a Slack channel
func (r *SessionRepository) UpdateChannelState(channelID string, activeSessionID, activeChildSessionID *int) error {
	// First check if channel exists
	existingChannel, err := r.GetChannelState(channelID)
	if err != nil {
		return err
	}

	if existingChannel == nil {
		// Create new channel state
		query := `INSERT INTO slack_channels (channel_id, active_session_id, active_child_session_id, created_at, updated_at)
				  VALUES ($1, $2, $3, NOW(), NOW())`
		_, err = r.db.GetDB().Exec(query, channelID, activeSessionID, activeChildSessionID)
		if err != nil {
			return fmt.Errorf("failed to create channel state: %w", err)
		}
	} else {
		// Update existing channel state
		query := `UPDATE slack_channels 
				  SET active_session_id = $1, active_child_session_id = $2, updated_at = NOW()
				  WHERE channel_id = $3`
		_, err = r.db.GetDB().Exec(query, activeSessionID, activeChildSessionID, channelID)
		if err != nil {
			return fmt.Errorf("failed to update channel state: %w", err)
		}
	}

	return nil
}

// ListAllSessions returns all sessions with their paths, ordered by most recent
func (r *SessionRepository) ListAllSessions(limit int) ([]*Session, error) {
	query := `SELECT * FROM sessions ORDER BY updated_at DESC LIMIT $1`
	
	rows, err := r.db.GetDB().Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(&session.ID, &session.SessionID, &session.WorkingDirectory,
			&session.SystemUser, &session.UserPrompt, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetUniqueWorkingDirectories returns unique working directories from all sessions
func (r *SessionRepository) GetUniqueWorkingDirectories(limit int) ([]string, error) {
	query := `SELECT DISTINCT working_directory FROM sessions ORDER BY working_directory LIMIT $1`
	
	rows, err := r.db.GetDB().Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique working directories: %w", err)
	}
	defer rows.Close()

	var directories []string
	for rows.Next() {
		var dir string
		err := rows.Scan(&dir)
		if err != nil {
			return nil, fmt.Errorf("failed to scan directory: %w", err)
		}
		directories = append(directories, dir)
	}

	return directories, nil
}

// GetSessionByID retrieves a session by database ID
func (r *SessionRepository) GetSessionByID(id int) (*Session, error) {
	query := `SELECT * FROM sessions WHERE id = $1`
	
	session := &Session{}
	err := r.db.GetDB().QueryRow(query, id).Scan(
		&session.ID, &session.SessionID, &session.WorkingDirectory,
		&session.SystemUser, &session.UserPrompt, &session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get session by ID: %w", err)
	}

	return session, nil
}

// GetSessionsByWorkingDirectory returns sessions that match a specific working directory
func (r *SessionRepository) GetSessionsByWorkingDirectory(workingDir string, limit int) ([]*Session, error) {
	query := `SELECT * FROM sessions WHERE working_directory = $1 ORDER BY updated_at DESC LIMIT $2`
	
	rows, err := r.db.GetDB().Query(query, workingDir, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by working directory: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(&session.ID, &session.SessionID, &session.WorkingDirectory,
			&session.SystemUser, &session.UserPrompt, &session.CreatedAt, &session.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}
// CountMessagesInConversationTree counts total messages (both user and AI) in the conversation tree
func (r *SessionRepository) CountMessagesInConversationTree(rootParentID int) (int, error) {
	// Count child sessions (each child session represents a user-AI exchange)
	// Each child session has both user prompt and AI response, so it counts as 2 messages
	query := `SELECT COUNT(*) FROM child_sessions WHERE root_parent_id = $1`
	
	var childCount int
	err := r.db.GetDB().QueryRow(query, rootParentID).Scan(&childCount)
	if err != nil {
		return 0, fmt.Errorf("failed to count child sessions: %w", err)
	}
	
	// Each child session represents a user-AI exchange (2 messages)
	// Plus the initial user prompt from the root session (1 message)
	totalMessages := (childCount * 2) + 1
	
	r.logger.Debug("Counted messages in conversation tree",
		zap.Int("root_parent_id", rootParentID),
		zap.Int("child_sessions", childCount),
		zap.Int("total_messages", totalMessages))
	
	return totalMessages, nil
}

// DeleteSession deletes a session and all its associated child sessions
func (r *SessionRepository) DeleteSession(sessionID string) error {
	// First, get the session to get its ID for deleting child sessions
	session, err := r.GetSessionBySessionID(sessionID)
	if err != nil {
		return fmt.Errorf("failed to find session to delete: %w", err)
	}

	// Start transaction
	tx, err := r.db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all child sessions first
	deleteChildQuery := `DELETE FROM child_sessions WHERE root_parent_id = $1`
	_, err = tx.Exec(deleteChildQuery, session.ID)
	if err != nil {
		return fmt.Errorf("failed to delete child sessions: %w", err)
	}

	// Clear any channel state pointing to this session
	clearChannelQuery := `UPDATE slack_channels SET active_session_id = NULL WHERE active_session_id = $1`
	_, err = tx.Exec(clearChannelQuery, session.ID)
	if err != nil {
		return fmt.Errorf("failed to clear channel state: %w", err)
	}

	// Delete the parent session
	deleteSessionQuery := `DELETE FROM sessions WHERE session_id = $1`
	_, err = tx.Exec(deleteSessionQuery, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Deleted session and its child sessions",
		zap.String("session_id", sessionID),
		zap.Int("db_id", session.ID))

	return nil
}
