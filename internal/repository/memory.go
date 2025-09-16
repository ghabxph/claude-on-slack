package repository

import (
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/ghabxph/claude-on-slack/internal/database"
)

// ShortTermMemory represents a single step in the conversation
type ShortTermMemory struct {
	ID               int       `db:"id"`
	StepID           string    `db:"step_id"`
	SessionID        string    `db:"session_id"`
	ChildSessionID   *int      `db:"child_session_id"`
	StepType         string    `db:"step_type"`
	StepOrder        int       `db:"step_order"`
	ToolName         *string   `db:"tool_name"`
	ToolInput        *string   `db:"tool_input"`        // JSONB stored as string
	ToolOutput       *string   `db:"tool_output"`
	ToolStatus       *string   `db:"tool_status"`
	ThinkingContext  *string   `db:"thinking_context"`
	Content          *string   `db:"content"`
	ErrorDetails     *string   `db:"error_details"`
	InputTokens      int       `db:"input_tokens"`
	OutputTokens     int       `db:"output_tokens"`
	CumulativeTokens int       `db:"cumulative_tokens"`
	DurationMS       *int      `db:"duration_ms"`
	Metadata         *string   `db:"metadata"`          // JSONB stored as string
	CreatedAt        time.Time `db:"created_at"`
}

// LongTermMemory represents a compacted conversation segment
type LongTermMemory struct {
	ID                      int       `db:"id"`
	MemoryID                string    `db:"memory_id"`
	SessionID               string    `db:"session_id"`
	CompactionSequence      int       `db:"compaction_sequence"`
	OriginalStepRangeStart  *string   `db:"original_step_range_start"`
	OriginalStepRangeEnd    *string   `db:"original_step_range_end"`
	CompleteStepData        string    `db:"complete_step_data"`      // JSONB stored as string
	ConversationSummary     string    `db:"conversation_summary"`
	KeyOutcomes             *string   `db:"key_outcomes"`
	ToolsUsed               *string   `db:"tools_used"`              // JSONB stored as string
	FilesAccessed           []string  `db:"files_accessed"`          // TEXT[] array
	CommandsExecuted        []string  `db:"commands_executed"`       // TEXT[] array
	ErrorsEncountered       *string   `db:"errors_encountered"`      // JSONB stored as string
	UserInputs              []string  `db:"user_inputs"`             // TEXT[] array
	AIResponses             []string  `db:"ai_responses"`            // TEXT[] array
	ThinkingSegments        []string  `db:"thinking_segments"`       // TEXT[] array
	SuccessfulOperations    *string   `db:"successful_operations"`   // JSONB stored as string
	FailedOperations        *string   `db:"failed_operations"`       // JSONB stored as string
	ConversationTopics      []string  `db:"conversation_topics"`     // TEXT[] array
	TechnologiesMentioned   []string  `db:"technologies_mentioned"`  // TEXT[] array
	FileExtensionsWorked    []string  `db:"file_extensions_worked_with"` // TEXT[] array
	DirectoryPathsAccessed  []string  `db:"directory_paths_accessed"` // TEXT[] array
	OriginalTokenCount      *int      `db:"original_token_count"`
	OriginalStepCount       *int      `db:"original_step_count"`
	CompactedAt             time.Time `db:"compacted_at"`
	OriginalTimespanStart   *time.Time `db:"original_timespan_start"`
	OriginalTimespanEnd     *time.Time `db:"original_timespan_end"`
	CompactionDurationMS    *int      `db:"compaction_duration_ms"`
}

// MemoryRepository handles short-term and long-term memory operations
type MemoryRepository struct {
	db     *database.Database
	logger *zap.Logger
}

// NewMemoryRepository creates a new memory repository
func NewMemoryRepository(db *database.Database, logger *zap.Logger) *MemoryRepository {
	return &MemoryRepository{
		db:     db,
		logger: logger,
	}
}

// CreateStep inserts a new step into short-term memory
func (r *MemoryRepository) CreateStep(step *ShortTermMemory) error {
	query := `
		INSERT INTO short_term_memory (
			step_id, session_id, child_session_id, step_type, step_order,
			tool_name, tool_input, tool_output, tool_status, thinking_context,
			content, error_details, input_tokens, output_tokens, cumulative_tokens,
			duration_ms, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW())
		RETURNING id`

	err := r.db.GetDB().QueryRow(query,
		step.StepID, step.SessionID, step.ChildSessionID, step.StepType, step.StepOrder,
		step.ToolName, step.ToolInput, step.ToolOutput, step.ToolStatus, step.ThinkingContext,
		step.Content, step.ErrorDetails, step.InputTokens, step.OutputTokens, step.CumulativeTokens,
		step.DurationMS, step.Metadata).Scan(&step.ID)

	if err != nil {
		return fmt.Errorf("failed to create step: %w", err)
	}

	r.logger.Debug("Step created in short-term memory",
		zap.String("step_id", step.StepID),
		zap.String("session_id", step.SessionID),
		zap.Int("step_order", step.StepOrder),
		zap.String("tool_name", stringPtrToString(step.ToolName)))

	return nil
}

// GetAllStepsForSession retrieves all steps for a session in order
func (r *MemoryRepository) GetAllStepsForSession(sessionID string) ([]*ShortTermMemory, error) {
	query := `
		SELECT id, step_id, session_id, child_session_id, step_type, step_order,
			   tool_name, tool_input, tool_output, tool_status, thinking_context,
			   content, error_details, input_tokens, output_tokens, cumulative_tokens,
			   duration_ms, metadata, created_at
		FROM short_term_memory 
		WHERE session_id = $1 
		ORDER BY step_order ASC`

	rows, err := r.db.GetDB().Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get steps for session: %w", err)
	}
	defer rows.Close()

	var steps []*ShortTermMemory
	for rows.Next() {
		step := &ShortTermMemory{}
		err := rows.Scan(
			&step.ID, &step.StepID, &step.SessionID, &step.ChildSessionID, &step.StepType, &step.StepOrder,
			&step.ToolName, &step.ToolInput, &step.ToolOutput, &step.ToolStatus, &step.ThinkingContext,
			&step.Content, &step.ErrorDetails, &step.InputTokens, &step.OutputTokens, &step.CumulativeTokens,
			&step.DurationMS, &step.Metadata, &step.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan step: %w", err)
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// GetCumulativeTokens gets the current cumulative token count for a session
func (r *MemoryRepository) GetCumulativeTokens(sessionID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(cumulative_tokens), 0) 
		FROM short_term_memory 
		WHERE session_id = $1`

	var cumulativeTokens int
	err := r.db.GetDB().QueryRow(query, sessionID).Scan(&cumulativeTokens)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No steps yet
		}
		return 0, fmt.Errorf("failed to get cumulative tokens: %w", err)
	}

	return cumulativeTokens, nil
}

// GetNextStepOrder gets the next step order number for a session
func (r *MemoryRepository) GetNextStepOrder(sessionID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(step_order), 0) + 1 
		FROM short_term_memory 
		WHERE session_id = $1`

	var nextOrder int
	err := r.db.GetDB().QueryRow(query, sessionID).Scan(&nextOrder)
	if err != nil {
		if err == sql.ErrNoRows {
			return 1, nil // First step
		}
		return 0, fmt.Errorf("failed to get next step order: %w", err)
	}

	return nextOrder, nil
}

// ClearAllStepsForSession removes all short-term memory for a session (used after compaction)
func (r *MemoryRepository) ClearAllStepsForSession(sessionID string) error {
	query := `DELETE FROM short_term_memory WHERE session_id = $1`

	result, err := r.db.GetDB().Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to clear steps for session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	r.logger.Info("Cleared short-term memory for session",
		zap.String("session_id", sessionID),
		zap.Int64("steps_cleared", rowsAffected))

	return nil
}

// UpdateCumulativeTokens updates the cumulative token count for a specific step
func (r *MemoryRepository) UpdateCumulativeTokens(stepID string, cumulativeTokens int) error {
	query := `
		UPDATE short_term_memory 
		SET cumulative_tokens = $1 
		WHERE step_id = $2`

	_, err := r.db.GetDB().Exec(query, cumulativeTokens, stepID)
	if err != nil {
		return fmt.Errorf("failed to update cumulative tokens: %w", err)
	}

	return nil
}

// GetStepByID retrieves a specific step by its step_id
func (r *MemoryRepository) GetStepByID(stepID string) (*ShortTermMemory, error) {
	query := `
		SELECT id, step_id, session_id, child_session_id, step_type, step_order,
			   tool_name, tool_input, tool_output, tool_status, thinking_context,
			   content, error_details, input_tokens, output_tokens, cumulative_tokens,
			   duration_ms, metadata, created_at
		FROM short_term_memory 
		WHERE step_id = $1`

	step := &ShortTermMemory{}
	err := r.db.GetDB().QueryRow(query, stepID).Scan(
		&step.ID, &step.StepID, &step.SessionID, &step.ChildSessionID, &step.StepType, &step.StepOrder,
		&step.ToolName, &step.ToolInput, &step.ToolOutput, &step.ToolStatus, &step.ThinkingContext,
		&step.Content, &step.ErrorDetails, &step.InputTokens, &step.OutputTokens, &step.CumulativeTokens,
		&step.DurationMS, &step.Metadata, &step.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Step not found
		}
		return nil, fmt.Errorf("failed to get step by ID: %w", err)
	}

	return step, nil
}

// Helper function to convert string pointer to string for logging
func stringPtrToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}