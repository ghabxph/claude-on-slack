package compaction

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/claude"
	"github.com/ghabxph/claude-on-slack/internal/config"
	"github.com/ghabxph/claude-on-slack/internal/repository"
)

// Service handles conversation compaction
type Service struct {
	memoryRepo     *repository.MemoryRepository
	sessionRepo    *repository.SessionRepository
	claudeExecutor *claude.Executor
	config         *config.Config
	logger         *zap.Logger
}

// CompactionResult represents the result of a compaction operation
type CompactionResult struct {
	MemoryID          string
	OriginalStepCount int
	OriginalTokenCount int
	CompactionDurationMS int64
	Summary           string
	KeyOutcomes       string
}

// NewService creates a new compaction service
func NewService(
	memoryRepo *repository.MemoryRepository,
	sessionRepo *repository.SessionRepository,
	claudeExecutor *claude.Executor,
	config *config.Config,
	logger *zap.Logger,
) *Service {
	return &Service{
		memoryRepo:     memoryRepo,
		sessionRepo:    sessionRepo,
		claudeExecutor: claudeExecutor,
		config:         config,
		logger:         logger,
	}
}

// ShouldCompact checks if a session should be compacted based on configured thresholds
func (s *Service) ShouldCompact(sessionID string, cumulativeTokens int) (bool, string) {
	// Check if auto-compaction is enabled
	if !s.config.AutoCompactEnabled {
		return false, "auto-compaction disabled"
	}

	// Use configured threshold
	threshold := s.config.AutoCompactThreshold

	// Check token threshold
	if cumulativeTokens >= threshold {
		return true, fmt.Sprintf("token threshold exceeded (%d >= %d)", cumulativeTokens, threshold)
	}

	return false, "thresholds not met"
}

// CompactSession performs compaction for a session
func (s *Service) CompactSession(ctx context.Context, sessionID string) (*CompactionResult, error) {
	startTime := time.Now()
	
	s.logger.Info("Starting session compaction",
		zap.String("session_id", sessionID))

	// Get all steps for the session
	steps, err := s.memoryRepo.GetAllStepsForSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps for compaction: %w", err)
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("no steps found for session %s", sessionID)
	}

	// Get current cumulative tokens
	cumulativeTokens, err := s.memoryRepo.GetCumulativeTokens(sessionID)
	if err != nil {
		s.logger.Warn("Failed to get cumulative tokens", zap.Error(err))
		cumulativeTokens = 0
	}

	// Generate memory ID
	memoryID := uuid.New().String()

	// Convert steps to JSON for complete preservation
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize steps: %w", err)
	}

	// Generate conversation summary using Claude
	summary, keyOutcomes, err := s.generateSummary(ctx, steps)
	if err != nil {
		s.logger.Warn("Failed to generate summary, using fallback", zap.Error(err))
		summary = s.generateFallbackSummary(steps)
		keyOutcomes = "Summary generation failed - using fallback"
	}

	// Extract metadata from steps
	metadata := s.extractMetadata(steps)

	// Get next compaction sequence number
	compactionSequence := 1 // For now, just increment - could be more sophisticated

	// Create long-term memory record
	longTermMemory := &repository.LongTermMemory{
		MemoryID:                memoryID,
		SessionID:               sessionID,
		CompactionSequence:      compactionSequence,
		OriginalStepRangeStart:  &steps[0].StepID,
		OriginalStepRangeEnd:    &steps[len(steps)-1].StepID,
		CompleteStepData:        string(stepsJSON),
		ConversationSummary:     summary,
		KeyOutcomes:             &keyOutcomes,
		ToolsUsed:               &metadata.ToolsUsedJSON,
		FilesAccessed:           metadata.FilesAccessed,
		CommandsExecuted:        metadata.CommandsExecuted,
		ErrorsEncountered:       &metadata.ErrorsJSON,
		UserInputs:              metadata.UserInputs,
		AIResponses:             metadata.AIResponses,
		ThinkingSegments:        metadata.ThinkingSegments,
		SuccessfulOperations:    &metadata.SuccessfulOpsJSON,
		FailedOperations:        &metadata.FailedOpsJSON,
		ConversationTopics:      metadata.Topics,
		TechnologiesMentioned:   metadata.Technologies,
		FileExtensionsWorked:    metadata.FileExtensions,
		DirectoryPathsAccessed:  metadata.DirectoryPaths,
		OriginalTokenCount:      &cumulativeTokens,
		OriginalStepCount:       &[]int{len(steps)}[0],
		OriginalTimespanStart:   &steps[0].CreatedAt,
		OriginalTimespanEnd:     &steps[len(steps)-1].CreatedAt,
		CompactionDurationMS:    &[]int{int(time.Since(startTime).Milliseconds())}[0],
	}

	// Store in long-term memory
	err = s.storeLongTermMemory(longTermMemory)
	if err != nil {
		return nil, fmt.Errorf("failed to store long-term memory: %w", err)
	}

	// Clear short-term memory for this session
	err = s.memoryRepo.ClearAllStepsForSession(sessionID)
	if err != nil {
		s.logger.Error("Failed to clear short-term memory after compaction", 
			zap.String("session_id", sessionID),
			zap.Error(err))
		// Don't fail the compaction if clearing fails - memory is already preserved
	}

	duration := time.Since(startTime)
	
	s.logger.Info("Session compaction completed",
		zap.String("session_id", sessionID),
		zap.String("memory_id", memoryID),
		zap.Int("steps_compacted", len(steps)),
		zap.Int("original_tokens", cumulativeTokens),
		zap.Duration("duration", duration))

	return &CompactionResult{
		MemoryID:             memoryID,
		OriginalStepCount:    len(steps),
		OriginalTokenCount:   cumulativeTokens,
		CompactionDurationMS: duration.Milliseconds(),
		Summary:              summary,
		KeyOutcomes:          keyOutcomes,
	}, nil
}

// Metadata represents extracted metadata from conversation steps
type Metadata struct {
	ToolsUsedJSON      string
	FilesAccessed      []string
	CommandsExecuted   []string
	ErrorsJSON         string
	UserInputs         []string
	AIResponses        []string
	ThinkingSegments   []string
	SuccessfulOpsJSON  string
	FailedOpsJSON      string
	Topics             []string
	Technologies       []string
	FileExtensions     []string
	DirectoryPaths     []string
}

// extractMetadata extracts structured metadata from conversation steps
func (s *Service) extractMetadata(steps []*repository.ShortTermMemory) *Metadata {
	metadata := &Metadata{
		FilesAccessed:    make([]string, 0),
		CommandsExecuted: make([]string, 0),
		UserInputs:       make([]string, 0),
		AIResponses:      make([]string, 0),
		ThinkingSegments: make([]string, 0),
		Topics:           make([]string, 0),
		Technologies:     make([]string, 0),
		FileExtensions:   make([]string, 0),
		DirectoryPaths:   make([]string, 0),
	}

	toolsUsed := make(map[string][]string)
	errors := make([]map[string]interface{}, 0)
	successfulOps := make([]map[string]interface{}, 0)
	failedOps := make([]map[string]interface{}, 0)

	// Process each step
	for _, step := range steps {
		// Extract tool usage
		if step.ToolName != nil {
			toolName := *step.ToolName
			if _, exists := toolsUsed[toolName]; !exists {
				toolsUsed[toolName] = make([]string, 0)
			}
			
			// Add tool input summary
			if step.ToolInput != nil {
				toolsUsed[toolName] = append(toolsUsed[toolName], *step.ToolInput)
			}
		}

		// Extract user inputs and AI responses based on step type
		if step.StepType == "user_input" && step.Content != nil {
			metadata.UserInputs = append(metadata.UserInputs, *step.Content)
		}
		
		if step.StepType == "response" && step.Content != nil {
			metadata.AIResponses = append(metadata.AIResponses, *step.Content)
		}

		if step.StepType == "thinking" && step.ThinkingContext != nil {
			metadata.ThinkingSegments = append(metadata.ThinkingSegments, *step.ThinkingContext)
		}

		// Track errors and operations
		if step.ErrorDetails != nil {
			errorData := map[string]interface{}{
				"step_id": step.StepID,
				"tool":    step.ToolName,
				"error":   *step.ErrorDetails,
				"time":    step.CreatedAt,
			}
			errors = append(errors, errorData)
			failedOps = append(failedOps, errorData)
		} else if step.ToolName != nil {
			successData := map[string]interface{}{
				"step_id": step.StepID,
				"tool":    *step.ToolName,
				"time":    step.CreatedAt,
			}
			successfulOps = append(successfulOps, successData)
		}

		// Extract file paths and commands from tool outputs
		if step.ToolOutput != nil {
			output := *step.ToolOutput
			// Simple extraction - could be more sophisticated
			if step.ToolName != nil {
				switch *step.ToolName {
				case "Read", "Write", "Edit":
					// Extract file paths from tool outputs
					if strings.Contains(output, "/") {
						// Simple file path extraction
						lines := strings.Split(output, "\n")
						for _, line := range lines {
							if strings.Contains(line, "/") && !strings.Contains(line, "http") {
								metadata.FilesAccessed = append(metadata.FilesAccessed, line)
							}
						}
					}
				case "Bash":
					// Extract commands
					if step.ToolInput != nil {
						metadata.CommandsExecuted = append(metadata.CommandsExecuted, *step.ToolInput)
					}
				}
			}
		}
	}

	// Convert maps to JSON
	toolsJSON, _ := json.Marshal(toolsUsed)
	metadata.ToolsUsedJSON = string(toolsJSON)

	errorsJSON, _ := json.Marshal(errors)
	metadata.ErrorsJSON = string(errorsJSON)

	successJSON, _ := json.Marshal(successfulOps)
	metadata.SuccessfulOpsJSON = string(successJSON)

	failedJSON, _ := json.Marshal(failedOps)
	metadata.FailedOpsJSON = string(failedJSON)

	return metadata
}

// generateSummary generates a conversation summary using Claude
func (s *Service) generateSummary(ctx context.Context, steps []*repository.ShortTermMemory) (string, string, error) {
	// Use Claude executor to generate summary
	// For now, return a simple summary - this would be enhanced with actual Claude calls
	summary := fmt.Sprintf("Conversation compacted with %d steps", len(steps))
	keyOutcomes := "Steps compacted and preserved in long-term memory"
	
	return summary, keyOutcomes, nil
}

// generateFallbackSummary creates a basic summary when Claude generation fails
func (s *Service) generateFallbackSummary(steps []*repository.ShortTermMemory) string {
	if len(steps) == 0 {
		return "Empty conversation compacted"
	}

	toolCounts := make(map[string]int)
	var userMsgCount, aiMsgCount int

	for _, step := range steps {
		if step.ToolName != nil {
			toolCounts[*step.ToolName]++
		}
		if step.StepType == "user_input" {
			userMsgCount++
		}
		if step.StepType == "response" {
			aiMsgCount++
		}
	}

	summary := fmt.Sprintf("Compacted conversation: %d total steps, %d user messages, %d AI responses", 
		len(steps), userMsgCount, aiMsgCount)

	if len(toolCounts) > 0 {
		summary += ". Tools used: "
		var tools []string
		for tool, count := range toolCounts {
			tools = append(tools, fmt.Sprintf("%s(%d)", tool, count))
		}
		summary += strings.Join(tools, ", ")
	}

	return summary
}

// buildSummaryPrompt builds a prompt for Claude to summarize the conversation
func (s *Service) buildSummaryPrompt(steps []*repository.ShortTermMemory) string {
	return fmt.Sprintf(`Please summarize this conversation of %d steps, focusing on:
1. Key tasks accomplished
2. Main topics discussed  
3. Files and directories worked with
4. Important outcomes or decisions
5. Any issues encountered

Provide a concise but comprehensive summary.`, len(steps))
}

// storeLongTermMemory stores the compacted memory using the repository
func (s *Service) storeLongTermMemory(memory *repository.LongTermMemory) error {
	return s.memoryRepo.CreateLongTermMemory(memory)
}