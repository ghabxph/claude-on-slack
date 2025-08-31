-- Migration 006: Add summary field to child_sessions table
-- Summary will contain brief description of the conversation exchange

ALTER TABLE child_sessions ADD COLUMN summary TEXT;

-- Add comment for clarity
COMMENT ON COLUMN child_sessions.summary IS 'Brief summary of the conversation exchange (nullable)';