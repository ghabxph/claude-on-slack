-- Migration 007: Add channel-based message queue system
-- This creates a persistent message queue per Slack channel for FIFO processing

-- Channel message queue table: Stores queued messages per channel
CREATE TABLE channel_message_queue (
    id SERIAL PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    message_content TEXT NOT NULL,
    message_order INTEGER NOT NULL,
    queued_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    
    -- Foreign key to slack_channels table
    CONSTRAINT fk_channel_message_queue_channel 
        FOREIGN KEY (channel_id) REFERENCES slack_channels(channel_id) 
        ON DELETE CASCADE,
        
    -- Ensure ordering within channel
    UNIQUE (channel_id, message_order)
);

-- Channel processing state: Tracks if channel is currently processing
CREATE TABLE channel_processing_state (
    id SERIAL PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL UNIQUE,
    is_processing BOOLEAN NOT NULL DEFAULT FALSE,
    processing_started_at TIMESTAMP,
    processing_user_id VARCHAR(255),
    last_activity_at TIMESTAMP DEFAULT NOW(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Foreign key to slack_channels table
    CONSTRAINT fk_channel_processing_channel 
        FOREIGN KEY (channel_id) REFERENCES slack_channels(channel_id) 
        ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_channel_message_queue_channel_order ON channel_message_queue(channel_id, message_order);
CREATE INDEX idx_channel_message_queue_queued_at ON channel_message_queue(queued_at);
CREATE INDEX idx_channel_processing_state_channel ON channel_processing_state(channel_id);

-- Comments for documentation
COMMENT ON TABLE channel_message_queue IS 'FIFO message queue per Slack channel for intelligent message combining';
COMMENT ON TABLE channel_processing_state IS 'Processing state tracking per channel to prevent concurrent message processing';
COMMENT ON COLUMN channel_message_queue.message_order IS 'Sequence number for FIFO ordering within channel';
COMMENT ON COLUMN channel_processing_state.is_processing IS 'True when channel has active Claude processing';