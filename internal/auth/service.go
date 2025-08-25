package auth

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/config"
)

// Permission represents a permission level
type Permission int

const (
	PermissionNone Permission = iota
	PermissionRead
	PermissionWrite
	PermissionExecute
	PermissionAdmin
)

// UserInfo represents user information
type UserInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Email       string            `json:"email"`
	TeamID      string            `json:"team_id"`
	IsBot       bool              `json:"is_bot"`
	IsAdmin     bool              `json:"is_admin"`
	Permissions []Permission      `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	LastSeen    time.Time         `json:"last_seen"`
}

// ChannelInfo represents channel information
type ChannelInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	IsPrivate   bool              `json:"is_private"`
	Members     []string          `json:"members"`
	Metadata    map[string]string `json:"metadata"`
	Permissions map[string]Permission `json:"permissions"`
}

// AuthContext represents the context of an authentication request
type AuthContext struct {
	UserID      string    `json:"user_id"`
	ChannelID   string    `json:"channel_id"`
	TeamID      string    `json:"team_id"`
	Command     string    `json:"command"`
	Timestamp   time.Time `json:"timestamp"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	SessionID   string    `json:"session_id"`
}

// Service handles authentication and authorization
type Service struct {
	config         *config.Config
	logger         *zap.Logger
	users          map[string]*UserInfo
	channels       map[string]*ChannelInfo
	bannedUsers    map[string]time.Time
	rateLimitMap   map[string]*RateLimitEntry
	mu             sync.RWMutex
}

// RateLimitEntry tracks rate limiting per user
type RateLimitEntry struct {
	Count     int       `json:"count"`
	Window    time.Time `json:"window"`
	LastReset time.Time `json:"last_reset"`
}

// NewService creates a new authentication service
func NewService(cfg *config.Config, logger *zap.Logger) *Service {
	return &Service{
		config:       cfg,
		logger:       logger,
		users:        make(map[string]*UserInfo),
		channels:     make(map[string]*ChannelInfo),
		bannedUsers:  make(map[string]time.Time),
		rateLimitMap: make(map[string]*RateLimitEntry),
	}
}

// AuthenticateUser authenticates a user
func (s *Service) AuthenticateUser(ctx *AuthContext) (*UserInfo, error) {
	s.mu.RLock()
	user, exists := s.users[ctx.UserID]
	s.mu.RUnlock()

	if !exists {
		// Create new user info
		user = &UserInfo{
			ID:          ctx.UserID,
			TeamID:      ctx.TeamID,
			IsAdmin:     s.config.IsUserAdmin(ctx.UserID),
			Permissions: s.getDefaultPermissions(ctx.UserID),
			Metadata:    make(map[string]string),
			CreatedAt:   time.Now(),
			LastSeen:    time.Now(),
		}

		s.mu.Lock()
		s.users[ctx.UserID] = user
		s.mu.Unlock()

		s.logger.Info("Created new user",
			zap.String("user_id", ctx.UserID),
			zap.Bool("is_admin", user.IsAdmin))
	} else {
		// Update last seen
		s.mu.Lock()
		user.LastSeen = time.Now()
		s.mu.Unlock()
	}

	return user, nil
}

// AuthorizeUser checks if a user is authorized for a specific action
func (s *Service) AuthorizeUser(ctx *AuthContext, requiredPermission Permission) error {
	// Check if authentication is enabled
	if !s.config.EnableAuth {
		s.logger.Debug("Authentication disabled, allowing all requests")
		return nil
	}

	// Check if user is banned
	if s.isUserBanned(ctx.UserID) {
		s.logger.Warn("Blocked banned user", zap.String("user_id", ctx.UserID))
		return fmt.Errorf("user %s is banned", ctx.UserID)
	}

	// Check rate limiting
	if limited, until := s.checkRateLimit(ctx.UserID); limited {
		s.logger.Warn("Rate limited user",
			zap.String("user_id", ctx.UserID),
			zap.Time("until", until))
		return fmt.Errorf("rate limit exceeded, try again in %v", time.Until(until))
	}

	// Authenticate user
	user, err := s.AuthenticateUser(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Check if user is allowed
	if !s.config.IsUserAllowed(ctx.UserID) {
		s.logger.Warn("Blocked unauthorized user", zap.String("user_id", ctx.UserID))
		return fmt.Errorf("user %s is not authorized to use this bot", ctx.UserID)
	}

	// Check channel permissions
	if !s.config.IsChannelAllowed(ctx.ChannelID) {
		s.logger.Warn("Blocked unauthorized channel",
			zap.String("channel_id", ctx.ChannelID),
			zap.String("user_id", ctx.UserID))
		return fmt.Errorf("bot is not authorized in this channel")
	}

	// Check user permissions
	if !s.hasPermission(user, requiredPermission) {
		s.logger.Warn("User lacks required permission",
			zap.String("user_id", ctx.UserID),
			zap.String("required", s.permissionToString(requiredPermission)))
		return fmt.Errorf("insufficient permissions")
	}

	// Check command permissions
	if ctx.Command != "" && !s.config.IsCommandAllowed(ctx.Command) {
		s.logger.Warn("Blocked unauthorized command",
			zap.String("command", ctx.Command),
			zap.String("user_id", ctx.UserID))
		return fmt.Errorf("command not allowed: %s", ctx.Command)
	}

	s.logger.Debug("Authorization successful",
		zap.String("user_id", ctx.UserID),
		zap.String("channel_id", ctx.ChannelID),
		zap.String("permission", s.permissionToString(requiredPermission)))

	return nil
}

// IsUserAdmin checks if a user is an admin
func (s *Service) IsUserAdmin(userID string) bool {
	s.mu.RLock()
	user, exists := s.users[userID]
	s.mu.RUnlock()

	if exists {
		return user.IsAdmin
	}

	return s.config.IsUserAdmin(userID)
}

// BanUser bans a user for a specific duration
func (s *Service) BanUser(userID string, duration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	until := time.Now().Add(duration)
	s.bannedUsers[userID] = until

	s.logger.Info("Banned user",
		zap.String("user_id", userID),
		zap.Duration("duration", duration),
		zap.Time("until", until))

	return nil
}

// UnbanUser removes a ban from a user
func (s *Service) UnbanUser(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.bannedUsers, userID)

	s.logger.Info("Unbanned user", zap.String("user_id", userID))

	return nil
}

// GetUserInfo returns user information
func (s *Service) GetUserInfo(userID string) (*UserInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	return user, nil
}

// UpdateUserPermissions updates user permissions
func (s *Service) UpdateUserPermissions(userID string, permissions []Permission) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	user.Permissions = permissions

	s.logger.Info("Updated user permissions",
		zap.String("user_id", userID),
		zap.Int("permission_count", len(permissions)))

	return nil
}

// GetChannelInfo returns channel information
func (s *Service) GetChannelInfo(channelID string) (*ChannelInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	channel, exists := s.channels[channelID]
	if !exists {
		return nil, fmt.Errorf("channel %s not found", channelID)
	}

	return channel, nil
}

// RegisterChannel registers a channel
func (s *Service) RegisterChannel(channelID, name, channelType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	channel := &ChannelInfo{
		ID:          channelID,
		Name:        name,
		Type:        channelType,
		IsPrivate:   strings.HasPrefix(channelType, "private"),
		Members:     make([]string, 0),
		Metadata:    make(map[string]string),
		Permissions: make(map[string]Permission),
	}

	s.channels[channelID] = channel

	s.logger.Info("Registered channel",
		zap.String("channel_id", channelID),
		zap.String("name", name),
		zap.String("type", channelType))

	return nil
}

// isUserBanned checks if a user is currently banned
func (s *Service) isUserBanned(userID string) bool {
	s.mu.RLock()
	bannedUntil, exists := s.bannedUsers[userID]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(bannedUntil) {
		// Ban expired, remove it
		s.mu.Lock()
		delete(s.bannedUsers, userID)
		s.mu.Unlock()
		return false
	}

	return true
}

// checkRateLimit checks and updates rate limiting for a user
func (s *Service) checkRateLimit(userID string) (bool, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.rateLimitMap[userID]

	if !exists {
		// Create new entry
		s.rateLimitMap[userID] = &RateLimitEntry{
			Count:     1,
			Window:    now,
			LastReset: now,
		}
		return false, time.Time{}
	}

	// Reset counter if window has passed
	if now.Sub(entry.Window) > time.Minute {
		entry.Count = 1
		entry.Window = now
		entry.LastReset = now
		return false, time.Time{}
	}

	// Check if rate limit exceeded
	if entry.Count >= s.config.RateLimitPerMinute {
		nextWindow := entry.Window.Add(time.Minute)
		return true, nextWindow
	}

	// Increment counter
	entry.Count++
	return false, time.Time{}
}

// getDefaultPermissions returns default permissions for a user
func (s *Service) getDefaultPermissions(userID string) []Permission {
	permissions := []Permission{PermissionRead}

	if s.config.IsUserAdmin(userID) {
		permissions = []Permission{
			PermissionRead,
			PermissionWrite,
			PermissionExecute,
			PermissionAdmin,
		}
	} else if s.config.IsUserAllowed(userID) {
		permissions = []Permission{
			PermissionRead,
			PermissionWrite,
			PermissionExecute,
		}
	}

	return permissions
}

// hasPermission checks if a user has a specific permission
func (s *Service) hasPermission(user *UserInfo, required Permission) bool {
	for _, permission := range user.Permissions {
		if permission >= required {
			return true
		}
	}
	return false
}

// permissionToString converts permission to string
func (s *Service) permissionToString(permission Permission) string {
	switch permission {
	case PermissionNone:
		return "none"
	case PermissionRead:
		return "read"
	case PermissionWrite:
		return "write"
	case PermissionExecute:
		return "execute"
	case PermissionAdmin:
		return "admin"
	default:
		return "unknown"
	}
}

// GetStats returns authentication statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalUsers := len(s.users)
	adminUsers := 0
	bannedUsers := 0
	activeBans := 0
	totalChannels := len(s.channels)
	
	now := time.Now()
	for _, user := range s.users {
		if user.IsAdmin {
			adminUsers++
		}
	}

	for _, bannedUntil := range s.bannedUsers {
		bannedUsers++
		if now.Before(bannedUntil) {
			activeBans++
		}
	}

	return map[string]interface{}{
		"total_users":    totalUsers,
		"admin_users":    adminUsers,
		"banned_users":   bannedUsers,
		"active_bans":    activeBans,
		"total_channels": totalChannels,
		"auth_enabled":   s.config.EnableAuth,
	}
}

// CleanupExpiredEntries removes expired bans and rate limit entries
func (s *Service) CleanupExpiredEntries() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	
	// Clean up expired bans
	for userID, bannedUntil := range s.bannedUsers {
		if now.After(bannedUntil) {
			delete(s.bannedUsers, userID)
		}
	}

	// Clean up old rate limit entries
	for userID, entry := range s.rateLimitMap {
		if now.Sub(entry.LastReset) > time.Hour {
			delete(s.rateLimitMap, userID)
		}
	}
}

// ValidateSlackSignature validates Slack request signature (placeholder)
func (s *Service) ValidateSlackSignature(timestamp, signature, body string) bool {
	// TODO: Implement actual Slack signature validation
	// This is a placeholder implementation
	return true
}