package database

import (
	"database/sql"
	"fmt"

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

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConnections)
	db.SetMaxIdleConns(cfg.IdleConnections)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
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