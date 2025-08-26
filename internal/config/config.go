package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// PermissionMode defines Claude's permission level
type PermissionMode string

const (
	PermissionModeDefault         PermissionMode = "default"
	PermissionModeAcceptEdits    PermissionMode = "acceptEdits"
	PermissionModeBypassPerms    PermissionMode = "bypassPermissions"
	PermissionModePlan           PermissionMode = "plan"
)

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	URL              string
	Host             string
	Port             int
	Name             string
	User             string
	Password         string
	MaxConnections   int
	IdleConnections  int
	MaxLifetime      time.Duration
}

// Config holds all configuration for the Claude on Slack bot
type Config struct {
	// Slack configuration
	SlackBotToken      string
	SlackAppToken      string
	SlackSigningSecret string

	// Claude Code configuration
	ClaudeCodePath   string
	ClaudeTimeout    time.Duration
	AllowedTools     []string
	DisallowedTools  []string

	// Bot configuration
	BotName         string
	BotDisplayName  string
	CommandPrefix   string
	AllowedChannels []string
	AllowedUsers    []string
	AutoResponseChannels []string  // Channels where bot responds to all messages (no mention needed)

	// Session configuration
	SessionTimeout    time.Duration
	MaxSessionsPerUser int
	SessionCleanupInterval time.Duration

	// Security configuration
	AdminUsers         []string
	RateLimitPerMinute int
	MaxMessageLength   int

	// Logging configuration
	LogLevel    string
	LogFormat   string
	EnableDebug bool

	// Server configuration
	ServerPort int
	ServerHost string
	HealthCheckPath string

	// Working directory for Claude Code
	WorkingDirectory string
	AllowedCommands  []string
	BlockedCommands  []string
	CommandTimeout   time.Duration
	MaxOutputLength  int

	// Database configuration
	Database                DatabaseConfig
	EnableDatabasePersistence bool
	NotificationChannels    []string
	AppVersion              string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Default values
		ClaudeCodePath:         "claude",
		ClaudeTimeout:          time.Minute * 5,
		AllowedTools:           []string{}, // Empty = all tools allowed for full access
		DisallowedTools:        []string{},
		BotName:                "claude-bot",
		BotDisplayName:         "Claude Bot",
		CommandPrefix:          "!claude",
		SessionTimeout:         time.Hour * 2,
		MaxSessionsPerUser:     3,
		SessionCleanupInterval: time.Minute * 15,
		RateLimitPerMinute:     20,
		MaxMessageLength:       4000,
		LogLevel:               "info",
		LogFormat:              "json",
		ServerPort:             8080,
		ServerHost:             "0.0.0.0",
		HealthCheckPath:        "/health",
		WorkingDirectory:       "", // Default to current directory - set in .env
		CommandTimeout:         time.Minute * 5,
		MaxOutputLength:        10000,
		// Database defaults
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            5432,
			Name:            "claude_slack",
			User:            "claude_bot",
			MaxConnections:  10,
			IdleConnections: 2,
			MaxLifetime:     time.Hour,
		},
		EnableDatabasePersistence: false,
		AppVersion:               "2.0.0",
	}

	// Load required environment variables
	var err error
	
	cfg.SlackBotToken = getEnvRequired("SLACK_BOT_TOKEN")
	if cfg.SlackBotToken == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN is required")
	}

	cfg.SlackAppToken = getEnvRequired("SLACK_APP_TOKEN")
	if cfg.SlackAppToken == "" {
		return nil, fmt.Errorf("SLACK_APP_TOKEN is required")
	}

	cfg.SlackSigningSecret = getEnvRequired("SLACK_SIGNING_SECRET")
	if cfg.SlackSigningSecret == "" {
		return nil, fmt.Errorf("SLACK_SIGNING_SECRET is required")
	}

	// Load optional Claude Code configuration
	if val := os.Getenv("CLAUDE_CODE_PATH"); val != "" {
		cfg.ClaudeCodePath = val
	}

	if val := os.Getenv("ALLOWED_TOOLS"); val != "" {
		cfg.AllowedTools = strings.Split(val, ",")
	}

	if val := os.Getenv("DISALLOWED_TOOLS"); val != "" {
		cfg.DisallowedTools = strings.Split(val, ",")
	}

	if val := os.Getenv("CLAUDE_TIMEOUT"); val != "" {
		cfg.ClaudeTimeout, err = time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid CLAUDE_TIMEOUT: %v", err)
		}
	}

	if val := os.Getenv("BOT_NAME"); val != "" {
		cfg.BotName = val
	}

	if val := os.Getenv("BOT_DISPLAY_NAME"); val != "" {
		cfg.BotDisplayName = val
	}

	if val := os.Getenv("COMMAND_PREFIX"); val != "" {
		cfg.CommandPrefix = val
	}

	if val := os.Getenv("ALLOWED_CHANNELS"); val != "" {
		cfg.AllowedChannels = strings.Split(val, ",")
	}

	if val := os.Getenv("ALLOWED_USERS"); val != "" {
		cfg.AllowedUsers = strings.Split(val, ",")
	}

	if val := os.Getenv("AUTO_RESPONSE_CHANNELS"); val != "" {
		cfg.AutoResponseChannels = strings.Split(val, ",")
	}

	if val := os.Getenv("SESSION_TIMEOUT"); val != "" {
		cfg.SessionTimeout, err = time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid SESSION_TIMEOUT: %v", err)
		}
	}

	if val := os.Getenv("MAX_SESSIONS_PER_USER"); val != "" {
		cfg.MaxSessionsPerUser, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_SESSIONS_PER_USER: %v", err)
		}
	}

	if val := os.Getenv("SESSION_CLEANUP_INTERVAL"); val != "" {
		cfg.SessionCleanupInterval, err = time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid SESSION_CLEANUP_INTERVAL: %v", err)
		}
	}


	if val := os.Getenv("ADMIN_USERS"); val != "" {
		cfg.AdminUsers = strings.Split(val, ",")
	}

	if val := os.Getenv("RATE_LIMIT_PER_MINUTE"); val != "" {
		cfg.RateLimitPerMinute, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid RATE_LIMIT_PER_MINUTE: %v", err)
		}
	}

	if val := os.Getenv("MAX_MESSAGE_LENGTH"); val != "" {
		cfg.MaxMessageLength, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_MESSAGE_LENGTH: %v", err)
		}
	}

	if val := os.Getenv("LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	if val := os.Getenv("LOG_FORMAT"); val != "" {
		cfg.LogFormat = val
	}

	if val := os.Getenv("ENABLE_DEBUG"); val != "" {
		cfg.EnableDebug, err = strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid ENABLE_DEBUG: %v", err)
		}
	}

	if val := os.Getenv("SERVER_PORT"); val != "" {
		cfg.ServerPort, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %v", err)
		}
	}

	if val := os.Getenv("SERVER_HOST"); val != "" {
		cfg.ServerHost = val
	}

	if val := os.Getenv("HEALTH_CHECK_PATH"); val != "" {
		cfg.HealthCheckPath = val
	}

	if val := os.Getenv("WORKING_DIRECTORY"); val != "" {
		cfg.WorkingDirectory = val
	}

	if val := os.Getenv("ALLOWED_COMMANDS"); val != "" {
		cfg.AllowedCommands = strings.Split(val, ",")
	}

	if val := os.Getenv("BLOCKED_COMMANDS"); val != "" {
		cfg.BlockedCommands = strings.Split(val, ",")
	}

	if val := os.Getenv("COMMAND_TIMEOUT"); val != "" {
		cfg.CommandTimeout, err = time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid COMMAND_TIMEOUT: %v", err)
		}
	}

	if val := os.Getenv("MAX_OUTPUT_LENGTH"); val != "" {
		cfg.MaxOutputLength, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid MAX_OUTPUT_LENGTH: %v", err)
		}
	}

	// Database configuration
	if val := os.Getenv("DATABASE_URL"); val != "" {
		cfg.Database.URL = val
	}

	if val := os.Getenv("DB_HOST"); val != "" {
		cfg.Database.Host = val
	}

	if val := os.Getenv("DB_PORT"); val != "" {
		cfg.Database.Port, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_PORT: %v", err)
		}
	}

	if val := os.Getenv("DB_NAME"); val != "" {
		cfg.Database.Name = val
	}

	if val := os.Getenv("DB_USER"); val != "" {
		cfg.Database.User = val
	}

	if val := os.Getenv("DB_PASSWORD"); val != "" {
		cfg.Database.Password = val
	}

	if val := os.Getenv("DB_MAX_CONNECTIONS"); val != "" {
		cfg.Database.MaxConnections, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MAX_CONNECTIONS: %v", err)
		}
	}

	if val := os.Getenv("DB_IDLE_CONNECTIONS"); val != "" {
		cfg.Database.IdleConnections, err = strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_IDLE_CONNECTIONS: %v", err)
		}
	}

	if val := os.Getenv("DB_MAX_LIFETIME"); val != "" {
		cfg.Database.MaxLifetime, err = time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("invalid DB_MAX_LIFETIME: %v", err)
		}
	}

	if val := os.Getenv("ENABLE_DATABASE_PERSISTENCE"); val != "" {
		cfg.EnableDatabasePersistence, err = strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("invalid ENABLE_DATABASE_PERSISTENCE: %v", err)
		}
	}

	if val := os.Getenv("SLACK_NOTIFICATION_CHANNELS"); val != "" {
		cfg.NotificationChannels = strings.Split(val, ",")
	}

	if val := os.Getenv("APP_VERSION"); val != "" {
		cfg.AppVersion = val
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.SlackBotToken == "" {
		return fmt.Errorf("slack bot token is required")
	}
	if c.SlackAppToken == "" {
		return fmt.Errorf("slack app token is required")
	}
	if c.SlackSigningSecret == "" {
		return fmt.Errorf("slack signing secret is required")
	}
	if c.ClaudeCodePath == "" {
		return fmt.Errorf("claude code path is required")
	}
	if c.SessionTimeout <= 0 {
		return fmt.Errorf("session timeout must be positive")
	}
	if c.MaxSessionsPerUser <= 0 {
		return fmt.Errorf("max sessions per user must be positive")
	}
	if c.RateLimitPerMinute <= 0 {
		return fmt.Errorf("rate limit per minute must be positive")
	}
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}
	return nil
}

// IsUserAllowed checks if a user is allowed to use the bot
func (c *Config) IsUserAllowed(userID string) bool {
	if len(c.AllowedUsers) == 0 {
		return true // Allow all users if no restriction is set
	}
	
	for _, allowedUser := range c.AllowedUsers {
		if allowedUser == userID {
			return true
		}
	}
	return false
}

// IsChannelAllowed checks if a channel is allowed for bot usage
func (c *Config) IsChannelAllowed(channelID string) bool {
	if len(c.AllowedChannels) == 0 {
		return true // Allow all channels if no restriction is set
	}
	
	for _, allowedChannel := range c.AllowedChannels {
		if allowedChannel == channelID {
			return true
		}
	}
	return false
}

// IsAutoResponseChannel checks if a channel should get automatic responses (no mention needed)
func (c *Config) IsAutoResponseChannel(channelID string) bool {
	for _, autoChannel := range c.AutoResponseChannels {
		if autoChannel == channelID {
			return true
		}
	}
	return false
}

// IsUserAdmin checks if a user is an admin
func (c *Config) IsUserAdmin(userID string) bool {
	for _, adminUser := range c.AdminUsers {
		if adminUser == userID {
			return true
		}
	}
	return false
}

// IsCommandAllowed checks if a command is allowed
func (c *Config) IsCommandAllowed(command string) bool {
	// Check if command is in blocked list
	for _, blockedCmd := range c.BlockedCommands {
		if strings.Contains(command, blockedCmd) {
			return false
		}
	}
	
	// If allowed commands list is empty, allow all (except blocked)
	if len(c.AllowedCommands) == 0 {
		return true
	}
	
	// Check if command is in allowed list
	for _, allowedCmd := range c.AllowedCommands {
		if strings.Contains(command, allowedCmd) {
			return true
		}
	}
	
	return false
}

// getEnvRequired gets an environment variable and returns error if not set
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return value
}