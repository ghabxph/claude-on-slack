-- Initial database schema for claude-on-slack PostgreSQL migration
-- Creates the 3-table structure for session management

-- Sessions table: Root conversations
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL UNIQUE,
    working_directory VARCHAR(500) NOT NULL,
    system_user VARCHAR(100) NOT NULL,
    user_prompt TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Child sessions table: Conversation chains
CREATE TABLE child_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    previous_session_id VARCHAR(255),
    root_parent_id INTEGER REFERENCES sessions(id) NOT NULL,
    ai_response TEXT,
    user_prompt TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Slack channels table: Channel state tracking
CREATE TABLE slack_channels (
    id SERIAL PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    active_session_id INTEGER REFERENCES sessions(id),
    active_child_session_id INTEGER REFERENCES child_sessions(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);