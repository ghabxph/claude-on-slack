-- Add permission field to sessions table
-- Permission is shared across all child sessions from the root session

ALTER TABLE sessions ADD COLUMN permission VARCHAR(50) NOT NULL DEFAULT 'default';

-- Add index for permission queries (optional, for future performance)
CREATE INDEX idx_sessions_permission ON sessions(permission);

-- Update existing sessions to have default permission
UPDATE sessions SET permission = 'default' WHERE permission IS NULL;