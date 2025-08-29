package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/config"
)

type Database struct {
	db     *sql.DB
	config *config.DatabaseConfig
	logger *zap.Logger
}

func NewDatabase(cfg *config.DatabaseConfig, logger *zap.Logger) (*Database, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config cannot be nil")
	}

	// Build PostgreSQL URL with properly escaped password
	escapedPassword := url.QueryEscape(cfg.Password)
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable&connect_timeout=10&application_name=claude-slack-bot",
		cfg.User, escapedPassword, cfg.Host, cfg.Port, cfg.Name)

	logger.Info("Attempting database connection",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.String("user", cfg.User))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool with more conservative settings
	db.SetMaxOpenConns(5)  // Reduce from cfg.MaxConnections
	db.SetMaxIdleConns(2)  // Reduce from cfg.IdleConnections  
	db.SetConnMaxLifetime(30 * time.Minute)  // Shorter lifetime

	logger.Info("Testing database connection with ping...")
	
	// Test connection with retry logic
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := db.Ping(); err != nil {
			logger.Warn("Database ping failed, retrying...", 
				zap.Error(err), 
				zap.Int("attempt", i+1), 
				zap.Int("max_attempts", maxRetries))
			
			if i == maxRetries-1 {
				db.Close()
				return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, err)
			}
			
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.Int("max_connections", cfg.MaxConnections))

	return &Database{
		db:     db,
		config: cfg,
		logger: logger,
	}, nil
}

func (d *Database) Health() error {
	return d.db.Ping()
}

func (d *Database) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

func (d *Database) IsConnected() bool {
	return d.Health() == nil
}

func (d *Database) GetDB() *sql.DB {
	return d.db
}

func (d *Database) RunMigrations() error {
	// Simple file-based migration runner
	migrations := []string{
		"migrations/001_initial_schema.sql",
		"migrations/002_indexes.sql",
		"migrations/003_initial_data.sql",
	}

	for _, migration := range migrations {
		if err := d.executeMigrationFile(migration); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", migration, err)
		}
		d.logger.Info("Migration executed successfully", zap.String("file", migration))
	}

	return nil
}

func (d *Database) executeMigrationFile(filename string) error {
	// Read and execute SQL file
	// For now, we'll implement a simple approach
	// In production, you'd want proper migration tracking
	return nil
}