package files

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"go.uber.org/zap"
)

// Downloader handles downloading files from Slack
type Downloader struct {
	client    *slack.Client
	logger    *zap.Logger
	storageDir string
	token     string
}

// FileInfo represents downloaded file information
type FileInfo struct {
	LocalPath     string
	OriginalName  string
	MimeType      string
	Size          int64
	DownloadedAt  time.Time
}

// NewDownloader creates a new file downloader
func NewDownloader(client *slack.Client, logger *zap.Logger, storageDir string, token string) (*Downloader, error) {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &Downloader{
		client:     client,
		logger:     logger,
		storageDir: storageDir,
		token:      token,
	}, nil
}

// DownloadFile downloads a file from Slack and returns local file info
func (d *Downloader) DownloadFile(fileID string, userID string) (*FileInfo, error) {
	// Get file info from Slack API
	file, _, _, err := d.client.GetFileInfo(fileID, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if it's an image file
	if !d.isImageFile(file.Mimetype) {
		return nil, fmt.Errorf("file is not a supported image type: %s", file.Mimetype)
	}

	// Check file size (limit to 50MB)
	if file.Size > 50*1024*1024 {
		return nil, fmt.Errorf("file too large: %d bytes (max 50MB)", file.Size)
	}

	// Generate local filename
	timestamp := time.Now().Unix()
	extension := d.getFileExtension(file.Name, file.Mimetype)
	localFilename := fmt.Sprintf("%s_%d_%s%s", userID, timestamp, d.sanitizeFilename(file.Name), extension)
	localPath := filepath.Join(d.storageDir, localFilename)

	// Download the file
	err = d.downloadToFile(file.URLPrivateDownload, localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	d.logger.Info("Downloaded file",
		zap.String("fileID", fileID),
		zap.String("localPath", localPath),
		zap.String("mimeType", file.Mimetype),
		zap.Int("size", file.Size))

	return &FileInfo{
		LocalPath:    localPath,
		OriginalName: file.Name,
		MimeType:     file.Mimetype,
		Size:         int64(file.Size),
		DownloadedAt: time.Now(),
	}, nil
}

// isImageFile checks if the mime type is a supported image format
func (d *Downloader) isImageFile(mimeType string) bool {
	supportedTypes := []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, supported := range supportedTypes {
		if mimeType == supported {
			return true
		}
	}
	return false
}

// getFileExtension returns the appropriate file extension
func (d *Downloader) getFileExtension(filename, mimeType string) string {
	// Try to get extension from filename first
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}

	// Fall back to mime type
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}

// sanitizeFilename removes potentially dangerous characters from filename
func (d *Downloader) sanitizeFilename(filename string) string {
	// Remove extension for sanitization
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Replace dangerous characters
	dangerous := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", " "}
	result := name
	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "_")
	}
	
	// Limit length
	if len(result) > 50 {
		result = result[:50]
	}
	
	return result
}

// downloadToFile downloads from URL to local file
func (d *Downloader) downloadToFile(url, localPath string) error {
	// Create HTTP request with Slack bot token
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+d.token)

	// Make the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create the local file
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy data
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		// Clean up partial file
		os.Remove(localPath)
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// CleanupFile removes a downloaded file
func (d *Downloader) CleanupFile(localPath string) error {
	if err := os.Remove(localPath); err != nil {
		d.logger.Warn("Failed to cleanup file", zap.String("path", localPath), zap.Error(err))
		return err
	}
	d.logger.Debug("Cleaned up file", zap.String("path", localPath))
	return nil
}

// CleanupOldFiles removes files older than the specified duration
func (d *Downloader) CleanupOldFiles(maxAge time.Duration) error {
	entries, err := os.ReadDir(d.storageDir)
	if err != nil {
		return fmt.Errorf("failed to read storage directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(d.storageDir, entry.Name())
			if err := os.Remove(path); err == nil {
				cleaned++
				d.logger.Debug("Cleaned up old file", zap.String("path", path))
			}
		}
	}

	if cleaned > 0 {
		d.logger.Info("Cleaned up old files", zap.Int("count", cleaned))
	}

	return nil
}