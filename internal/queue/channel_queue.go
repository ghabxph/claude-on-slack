package queue

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/database"
)

// ChannelMessageQueue represents a queued message
type ChannelMessageQueue struct {
	ID             int       `json:"id"`
	ChannelID      string    `json:"channel_id"`
	UserID         string    `json:"user_id"`
	MessageContent string    `json:"message_content"`
	MessageOrder   int       `json:"message_order"`
	QueuedAt       time.Time `json:"queued_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// ChannelProcessingState represents channel processing status
type ChannelProcessingState struct {
	ID                   int       `json:"id"`
	ChannelID            string    `json:"channel_id"`
	IsProcessing         bool      `json:"is_processing"`
	ProcessingStartedAt  *time.Time `json:"processing_started_at"`
	ProcessingUserID     *string   `json:"processing_user_id"`
	LastActivityAt       time.Time `json:"last_activity_at"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// ChannelQueueService manages channel-based message queuing
type ChannelQueueService struct {
	db     *database.Database
	logger *zap.Logger
}

// NewChannelQueueService creates a new channel queue service
func NewChannelQueueService(db *database.Database, logger *zap.Logger) *ChannelQueueService {
	return &ChannelQueueService{
		db:     db,
		logger: logger,
	}
}

// QueueMessage adds a message to the channel queue if processing, returns true if queued
func (cqs *ChannelQueueService) QueueMessage(channelID, userID, message string) (bool, error) {
	// Check if channel is processing
	isProcessing, err := cqs.IsChannelProcessing(channelID)
	if err != nil {
		return false, fmt.Errorf("failed to check processing state: %w", err)
	}

	// If not processing, allow immediate processing
	if !isProcessing {
		return false, nil
	}

	// Get next message order for this channel
	nextOrder, err := cqs.getNextMessageOrder(channelID)
	if err != nil {
		return false, fmt.Errorf("failed to get next message order: %w", err)
	}

	// Insert message into queue
	query := `
		INSERT INTO channel_message_queue (channel_id, user_id, message_content, message_order)
		VALUES ($1, $2, $3, $4)
	`
	
	_, err = cqs.db.Exec(query, channelID, userID, message, nextOrder)
	if err != nil {
		return false, fmt.Errorf("failed to queue message: %w", err)
	}

	cqs.logger.Info("Message queued for channel",
		zap.String("channel_id", channelID),
		zap.String("user_id", userID),
		zap.Int("message_order", nextOrder),
		zap.String("message_preview", truncateString(message, 50)))

	return true, nil
}

// SetChannelProcessing sets the processing state for a channel
func (cqs *ChannelQueueService) SetChannelProcessing(channelID, userID string, processing bool) error {
	now := time.Now()
	
	if processing {
		// Start processing
		query := `
			INSERT INTO channel_processing_state (channel_id, is_processing, processing_started_at, processing_user_id, last_activity_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (channel_id) 
			DO UPDATE SET 
				is_processing = $2,
				processing_started_at = $3,
				processing_user_id = $4,
				last_activity_at = $5,
				updated_at = $5
		`
		_, err := cqs.db.Exec(query, channelID, true, now, userID, now)
		if err != nil {
			return fmt.Errorf("failed to set processing state: %w", err)
		}
		
		cqs.logger.Debug("Channel processing started",
			zap.String("channel_id", channelID),
			zap.String("user_id", userID))
	} else {
		// Stop processing
		query := `
			UPDATE channel_processing_state 
			SET is_processing = $2, processing_started_at = NULL, processing_user_id = NULL, last_activity_at = $3, updated_at = $3
			WHERE channel_id = $1
		`
		_, err := cqs.db.Exec(query, channelID, false, now)
		if err != nil {
			return fmt.Errorf("failed to clear processing state: %w", err)
		}
		
		cqs.logger.Debug("Channel processing stopped",
			zap.String("channel_id", channelID))
	}

	return nil
}

// IsChannelProcessing checks if a channel is currently processing
func (cqs *ChannelQueueService) IsChannelProcessing(channelID string) (bool, error) {
	query := `
		SELECT is_processing 
		FROM channel_processing_state 
		WHERE channel_id = $1
	`
	
	var isProcessing bool
	err := cqs.db.QueryRow(query, channelID).Scan(&isProcessing)
	if err != nil {
		if err == sql.ErrNoRows {
			// No record means not processing
			return false, nil
		}
		return false, fmt.Errorf("failed to check processing state: %w", err)
	}
	
	return isProcessing, nil
}

// GetQueuedMessages retrieves and removes all queued messages for a channel (FIFO)
func (cqs *ChannelQueueService) GetQueuedMessages(channelID string) ([]string, error) {
	// Get all queued messages in order
	query := `
		SELECT id, message_content 
		FROM channel_message_queue 
		WHERE channel_id = $1 
		ORDER BY message_order ASC
	`
	
	rows, err := cqs.db.Query(query, channelID)
	if err != nil {
		return nil, fmt.Errorf("failed to query queued messages: %w", err)
	}
	defer rows.Close()

	var messages []string
	var messageIDs []int

	for rows.Next() {
		var id int
		var content string
		if err := rows.Scan(&id, &content); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, content)
		messageIDs = append(messageIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	// Clear the queue after reading (FIFO consumption)
	if len(messageIDs) > 0 {
		err = cqs.clearQueuedMessages(channelID)
		if err != nil {
			cqs.logger.Error("Failed to clear queued messages after reading",
				zap.Error(err),
				zap.String("channel_id", channelID))
			// Don't return error here - messages were read successfully
		} else {
			cqs.logger.Info("Cleared queued messages after reading",
				zap.String("channel_id", channelID),
				zap.Int("message_count", len(messages)))
		}
	}

	return messages, nil
}

// GetQueueCount returns the number of queued messages for a channel
func (cqs *ChannelQueueService) GetQueueCount(channelID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM channel_message_queue 
		WHERE channel_id = $1
	`
	
	var count int
	err := cqs.db.QueryRow(query, channelID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue count: %w", err)
	}
	
	return count, nil
}

// CleanupStaleProcessing cleans up stale processing states (older than timeout)
func (cqs *ChannelQueueService) CleanupStaleProcessing(timeout time.Duration) error {
	cutoff := time.Now().Add(-timeout)
	
	query := `
		UPDATE channel_processing_state 
		SET is_processing = FALSE, processing_started_at = NULL, processing_user_id = NULL, updated_at = NOW()
		WHERE is_processing = TRUE AND processing_started_at < $1
	`
	
	result, err := cqs.db.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup stale processing: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		cqs.logger.Warn("Cleaned up stale processing states",
			zap.Int64("channels_affected", rowsAffected),
			zap.Duration("timeout", timeout))
	}
	
	return nil
}

// CombineMessages intelligently combines multiple messages into one
func (cqs *ChannelQueueService) CombineMessages(currentMessage string, queuedMessages []string) string {
	if len(queuedMessages) == 0 {
		return currentMessage
	}

	allMessages := append([]string{currentMessage}, queuedMessages...)
	
	// Simple combination with clear separation
	combined := strings.Join(allMessages, "\n\n---\n\n")
	
	cqs.logger.Debug("Combined messages",
		zap.Int("total_messages", len(allMessages)),
		zap.String("combined_preview", truncateString(combined, 100)))
	
	return combined
}

// getNextMessageOrder gets the next order number for a channel
func (cqs *ChannelQueueService) getNextMessageOrder(channelID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(message_order), 0) + 1 
		FROM channel_message_queue 
		WHERE channel_id = $1
	`
	
	var nextOrder int
	err := cqs.db.QueryRow(query, channelID).Scan(&nextOrder)
	if err != nil {
		return 0, fmt.Errorf("failed to get next message order: %w", err)
	}
	
	return nextOrder, nil
}

// clearQueuedMessages removes all queued messages for a channel
func (cqs *ChannelQueueService) clearQueuedMessages(channelID string) error {
	query := `DELETE FROM channel_message_queue WHERE channel_id = $1`
	
	_, err := cqs.db.Exec(query, channelID)
	if err != nil {
		return fmt.Errorf("failed to clear queued messages: %w", err)
	}
	
	return nil
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}