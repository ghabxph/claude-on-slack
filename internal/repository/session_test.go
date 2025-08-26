package repository

import (
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/ghabxph/claude-on-slack/internal/config"
	"github.com/ghabxph/claude-on-slack/internal/database"
)

func setupTestDB(t *testing.T) *database.Database {
	logger := zaptest.NewLogger(t)
	
	cfg := &config.DatabaseConfig{
		Host:            "localhost",
		Port:            5432,
		Name:            "claude_slack_test",
		User:            "postgres",
		Password:        "test",
		MaxConnections:  5,
		IdleConnections: 1,
		MaxLifetime:     time.Hour,
	}

	db, err := database.NewDatabase(cfg, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available for testing: %v", err)
	}

	return db
}

func TestSessionRepository_CreateSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zaptest.NewLogger(t)
	repo := NewSessionRepository(db, logger)

	session := &Session{
		SessionID:        "test-session-123",
		WorkingDirectory: "/tmp/test",
		SystemUser:       "testuser",
	}

	err := repo.CreateSession(session)
	if err != nil {
		t.Errorf("Failed to create session: %v", err)
	}

	if session.ID == 0 {
		t.Error("Session ID should be set after creation")
	}
}

func TestSessionRepository_GetSessionBySessionID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zaptest.NewLogger(t)
	repo := NewSessionRepository(db, logger)

	// Create test session
	session := &Session{
		SessionID:        "test-session-456",
		WorkingDirectory: "/tmp/test2",
		SystemUser:       "testuser2",
	}

	err := repo.CreateSession(session)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Retrieve session
	retrieved, err := repo.GetSessionBySessionID("test-session-456")
	if err != nil {
		t.Errorf("Failed to get session: %v", err)
	}

	if retrieved == nil {
		t.Error("Retrieved session should not be nil")
	}

	if retrieved.SessionID != "test-session-456" {
		t.Errorf("Expected session ID test-session-456, got %s", retrieved.SessionID)
	}
}