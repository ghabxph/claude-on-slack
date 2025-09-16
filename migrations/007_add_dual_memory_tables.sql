-- Migration 007: Add dual memory system tables
-- Creates short_term_memory and long_term_memory tables for perfect conversation archival

-- Short-term memory table: Ephemeral, detailed step tracking (cleared after compaction)
CREATE TABLE short_term_memory (
    id SERIAL PRIMARY KEY,
    step_id VARCHAR(255) NOT NULL UNIQUE,
    session_id VARCHAR(255) REFERENCES sessions(session_id) NOT NULL,
    child_session_id INTEGER REFERENCES child_sessions(id),
    
    -- Step Classification
    step_type VARCHAR(50) NOT NULL, -- 'tool_call', 'tool_result', 'thinking', 'response', 'user_input'
    step_order INTEGER NOT NULL,
    
    -- Tool Information (Direct from Claude CLI)
    tool_name VARCHAR(100),
    tool_input JSONB,
    tool_output TEXT,
    tool_status VARCHAR(20), -- 'success', 'error', 'timeout'
    
    -- Context & Content
    thinking_context TEXT,
    content TEXT,
    error_details TEXT,
    
    -- Token Tracking (From Claude CLI Usage)
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cumulative_tokens INTEGER DEFAULT 0,
    
    -- Timing & Metadata
    duration_ms INTEGER,
    metadata JSONB,
    
    created_at TIMESTAMP DEFAULT NOW()
);

-- Long-term memory table: Persistent, compressed conversation summaries
CREATE TABLE long_term_memory (
    id SERIAL PRIMARY KEY,
    memory_id VARCHAR(255) NOT NULL UNIQUE,
    session_id VARCHAR(255) REFERENCES sessions(session_id) NOT NULL,
    
    -- Compaction Information
    compaction_sequence INTEGER NOT NULL, -- 1st, 2nd, 3rd compaction for this session
    original_step_range_start VARCHAR(255), -- First step_id that was compacted
    original_step_range_end VARCHAR(255), -- Last step_id that was compacted
    
    -- Perfect Memory Storage
    complete_step_data JSONB NOT NULL, -- Full JSON array of ALL compacted steps
    conversation_summary TEXT NOT NULL,
    key_outcomes TEXT,
    
    -- Complete Tool Archive
    tools_used JSONB, -- {"Read": [files], "Write": [files], "Bash": [commands]}
    files_accessed TEXT[],
    commands_executed TEXT[],
    errors_encountered JSONB,
    
    -- Perfect Context Preservation
    user_inputs TEXT[],
    ai_responses TEXT[],
    thinking_segments TEXT[],
    successful_operations JSONB,
    failed_operations JSONB,
    
    -- Future Retrieval Metadata
    conversation_topics TEXT[],
    technologies_mentioned TEXT[],
    file_extensions_worked_with TEXT[],
    directory_paths_accessed TEXT[],
    
    -- Compaction Statistics
    original_token_count INTEGER,
    original_step_count INTEGER,
    compacted_at TIMESTAMP DEFAULT NOW(),
    original_timespan_start TIMESTAMP,
    original_timespan_end TIMESTAMP,
    compaction_duration_ms INTEGER
);

-- Add new fields to existing sessions table for token tracking
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS current_token_count INTEGER DEFAULT 0;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS compaction_count INTEGER DEFAULT 0;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS last_compacted_at TIMESTAMP;
ALTER TABLE sessions ADD COLUMN IF NOT EXISTS auto_compact_threshold INTEGER DEFAULT 180000;

-- Indexes for short_term_memory performance
CREATE INDEX idx_stm_session_order ON short_term_memory (session_id, step_order);
CREATE INDEX idx_stm_cumulative_tokens ON short_term_memory (session_id, cumulative_tokens);
CREATE INDEX idx_stm_tool_type ON short_term_memory (tool_name, step_type);
CREATE INDEX idx_stm_created ON short_term_memory (created_at DESC);

-- Indexes for long_term_memory future retrieval
CREATE INDEX idx_ltm_session_sequence ON long_term_memory (session_id, compaction_sequence);
CREATE INDEX idx_ltm_topics ON long_term_memory USING GIN (conversation_topics);
CREATE INDEX idx_ltm_technologies ON long_term_memory USING GIN (technologies_mentioned);
CREATE INDEX idx_ltm_file_types ON long_term_memory USING GIN (file_extensions_worked_with);
CREATE INDEX idx_ltm_timespan ON long_term_memory (original_timespan_start, original_timespan_end);
CREATE INDEX idx_ltm_compacted ON long_term_memory (compacted_at DESC);

-- Add comments for clarity
COMMENT ON TABLE short_term_memory IS 'Ephemeral storage for detailed conversation steps, cleared after compaction';
COMMENT ON TABLE long_term_memory IS 'Persistent archive of compacted conversation segments with perfect memory preservation';
COMMENT ON COLUMN short_term_memory.step_id IS 'Unique identifier for each individual step/action';
COMMENT ON COLUMN short_term_memory.cumulative_tokens IS 'Running total of tokens for session, used for compaction threshold';
COMMENT ON COLUMN long_term_memory.memory_id IS 'Unique identifier for each compacted conversation segment';
COMMENT ON COLUMN long_term_memory.complete_step_data IS 'Full JSON archive of all compacted steps with original step_ids preserved';