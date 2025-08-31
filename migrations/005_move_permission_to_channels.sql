-- Migration 005: Move permission field from sessions table to slack_channels table
-- This makes permission channel-scoped rather than session-scoped for better UX

-- Add permission field to slack_channels table
ALTER TABLE slack_channels ADD COLUMN permission VARCHAR(50) NOT NULL DEFAULT 'default';

-- Remove permission field from sessions table
ALTER TABLE sessions DROP COLUMN permission;

-- Add comment for clarity
COMMENT ON COLUMN slack_channels.permission IS 'Permission mode for this channel (default, acceptEdits, bypassPermissions, plan)';