-- Performance indexes for claude-on-slack database
-- Optimizes queries for O(1) session lookup performance

-- Index for session lookup by session_id
CREATE INDEX idx_sessions_session_id ON sessions(session_id);

-- Index for child session conversation tree loading
CREATE INDEX idx_child_sessions_root_parent_id ON child_sessions(root_parent_id);

-- Index for child session lookup by session_id
CREATE INDEX idx_child_sessions_session_id ON child_sessions(session_id);

-- Index for slack channel lookup
CREATE INDEX idx_slack_channels_channel_id ON slack_channels(channel_id);