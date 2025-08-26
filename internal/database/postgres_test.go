package database

import (
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/ghabxph/claude-on-slack/internal/config"
)

func TestNewDatabase(t *testing.T) {
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

	// This test requires a running PostgreSQL instance
	// Skip if no database available
	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available for testing: %v", err)
	}
	defer db.Close()

	// Test health check
	if err := db.Health(); err != nil {
		t.Errorf("Database health check failed: %v", err)
	}

	// Test connection status
	if !db.IsConnected() {
		t.Error("Database should be connected")
	}
}

func TestDatabase_Close(t *testing.T) {
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

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Skipf("PostgreSQL not available for testing: %v", err)
	}

	// Close and verify
	if err := db.Close(); err != nil {
		t.Errorf("Database close failed: %v", err)
	}

	// Health should fail after close
	if err := db.Health(); err == nil {
		t.Error("Health check should fail after closing database")
	}
}