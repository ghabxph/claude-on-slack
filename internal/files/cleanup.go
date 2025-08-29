package files

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// CleanupService manages periodic cleanup of downloaded files
type CleanupService struct {
	downloader *Downloader
	logger     *zap.Logger
	interval   time.Duration
	maxAge     time.Duration
	stopCh     chan struct{}
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(downloader *Downloader, logger *zap.Logger) *CleanupService {
	return &CleanupService{
		downloader: downloader,
		logger:     logger,
		interval:   30 * time.Minute, // Run every 30 minutes
		maxAge:     2 * time.Hour,    // Clean files older than 2 hours
		stopCh:     make(chan struct{}),
	}
}

// Start begins the cleanup service
func (c *CleanupService) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.logger.Info("Starting file cleanup service",
		zap.Duration("interval", c.interval),
		zap.Duration("maxAge", c.maxAge))

	// Run initial cleanup
	c.runCleanup()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping file cleanup service")
			return
		case <-c.stopCh:
			c.logger.Info("Stopping file cleanup service")
			return
		case <-ticker.C:
			c.runCleanup()
		}
	}
}

// Stop stops the cleanup service
func (c *CleanupService) Stop() {
	close(c.stopCh)
}

// runCleanup performs the actual cleanup
func (c *CleanupService) runCleanup() {
	if err := c.downloader.CleanupOldFiles(c.maxAge); err != nil {
		c.logger.Error("Failed to cleanup old files", zap.Error(err))
	}
}