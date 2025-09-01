# Channel-Based Message Queue Integration Plan

## Overview

This document outlines the implementation of channel-based message queuing to replace the current session-based queuing system. The new system provides FIFO (First In, First Out) message processing per Slack channel with intelligent message combining.

## Key Improvements

### Current Limitations (Session-Based)
- ❌ **Multiple users, separate queues**: Each session has its own queue
- ❌ **No persistence**: Queues lost on service restart  
- ❌ **Memory-based**: Not scalable or reliable
- ❌ **Session-dependent**: Queue tied to specific session, not channel

### New Benefits (Channel-Based)
- ✅ **One queue per channel**: All users in a channel share the same queue
- ✅ **Persistent storage**: Database-backed, survives restarts
- ✅ **FIFO processing**: Messages processed in order received
- ✅ **Intelligent combining**: Multiple queued messages combined into one request

## Database Schema Changes

### New Tables Created

```sql
-- Migration 007: Channel message queue tables
CREATE TABLE channel_message_queue (
    id SERIAL PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    message_content TEXT NOT NULL,
    message_order INTEGER NOT NULL,  -- FIFO ordering
    queued_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (channel_id, message_order)
);

CREATE TABLE channel_processing_state (
    id SERIAL PRIMARY KEY,
    channel_id VARCHAR(255) NOT NULL UNIQUE,
    is_processing BOOLEAN NOT NULL DEFAULT FALSE,
    processing_started_at TIMESTAMP,
    processing_user_id VARCHAR(255),
    last_activity_at TIMESTAMP DEFAULT NOW()
);
```

## Implementation Components

### 1. ChannelQueueService (`internal/queue/channel_queue.go`)

**Core Methods:**
- `QueueMessage(channelID, userID, message)` - Adds message to queue if channel is processing
- `SetChannelProcessing(channelID, userID, processing)` - Sets channel processing state
- `IsChannelProcessing(channelID)` - Checks if channel is busy
- `GetQueuedMessages(channelID)` - Retrieves and clears queued messages (FIFO)
- `CombineMessages(current, queued)` - Intelligently combines messages

**Key Features:**
- Thread-safe database operations
- Automatic message ordering (FIFO)
- Queue cleanup after reading
- Stale processing cleanup
- Detailed logging

### 2. Bot Service Integration Changes

The bot service needs the following modifications:

#### A. Add ChannelQueueService to Service struct
```go
type Service struct {
    // ... existing fields ...
    channelQueue   *queue.ChannelQueueService  // 🆕 NEW
}
```

#### B. Initialize in NewService()
```go
// Initialize channel queue service
channelQueue := queue.NewChannelQueueService(db, logger)
```

#### C. Replace session-based queuing in message processing
```go
// OLD: Session-based queuing
queued, err := s.sessionManager.QueueMessage(userSession.GetID(), text)

// NEW: Channel-based queuing  
queued, err := s.channelQueue.QueueMessage(event.Channel, event.User, text)
```

#### D. Update message processing workflow
```go
// Set channel as processing
err = s.channelQueue.SetChannelProcessing(event.Channel, event.User, true)

// Process message with Claude
response, err := s.processWithClaude(...)

// Get any queued messages and combine
queuedMessages, err := s.channelQueue.GetQueuedMessages(event.Channel)
if len(queuedMessages) > 0 {
    combinedMessage := s.channelQueue.CombineMessages(text, queuedMessages)
    // Re-process with combined message
}

// Clear processing state
err = s.channelQueue.SetChannelProcessing(event.Channel, event.User, false)
```

## Message Flow Comparison

### Current Flow (Session-Based)
```
User A (Channel #dev): "Help me debug" 
  → Session A Queue: ["Help me debug"]
  → Process immediately

User B (Channel #dev): "What's the error?"
  → Session B Queue: ["What's the error?"] 
  → Process immediately (separate session)
```
**Problem**: Two separate processing threads for same channel

### New Flow (Channel-Based)
```
User A (Channel #dev): "Help me debug"
  → Channel #dev Processing: TRUE
  → Process immediately

User B (Channel #dev): "What's the error?"  
  → Channel #dev Queue: ["What's the error?"]
  → Wait for User A's processing to complete

User A's processing completes:
  → Channel #dev Processing: FALSE
  → Get queued messages: ["What's the error?"]
  → Combine: "Help me debug\n\n---\n\nWhat's the error?"
  → Process combined message
```
**Benefit**: Sequential processing with intelligent message combining

## Implementation Steps

### Phase 1: Database Setup ✅ COMPLETED
- [x] Create migration 007 with new tables
- [x] Add indexes for performance
- [x] Add foreign key constraints

### Phase 2: Core Service ✅ COMPLETED  
- [x] Implement ChannelQueueService
- [x] Add all required methods
- [x] Include error handling and logging

### Phase 3: Bot Service Integration (PENDING)
- [ ] Add queue import to bot service
- [ ] Add channelQueue field to Service struct
- [ ] Initialize channelQueue in NewService()
- [ ] Replace session.QueueMessage() calls
- [ ] Update message processing workflow
- [ ] Add cleanup for stale processing states

### Phase 4: Testing & Validation (PENDING)
- [ ] Test with multiple users in same channel
- [ ] Verify FIFO message ordering
- [ ] Test message combining logic
- [ ] Verify persistence across restarts
- [ ] Performance testing with high message volume

## Configuration Changes

### Environment Variables (Optional)
```env
# Message queue settings
CHANNEL_QUEUE_STALE_TIMEOUT=300s  # Clean up stale processing after 5 minutes
CHANNEL_QUEUE_MAX_COMBINE=10      # Maximum messages to combine
```

### Database Migrations
Run the new migration:
```bash
# Apply migration 007
docker-compose exec postgres psql -U claude_bot -d claude_slack -f /host_migrations/007_add_channel_message_queue.sql
```

## Expected User Experience

### Before (Session-Based)
1. Multiple users send messages simultaneously
2. Each gets separate processing threads
3. Responses may be inconsistent or overlapping
4. No message combining

### After (Channel-Based)  
1. User A sends message → Processing starts
2. User B sends message → Gets queued automatically
3. User C sends message → Also gets queued  
4. When A's processing completes → B & C messages combined
5. Combined message processed as one comprehensive request
6. All users get coherent, context-aware response

## Benefits Summary

1. **Better UX**: No overlapping responses, coherent conversation flow
2. **Resource Efficiency**: One processing thread per channel vs per session
3. **Intelligent Combining**: Multiple messages become one comprehensive request
4. **Persistence**: Queue survives service restarts
5. **Scalability**: Database-backed, handles high message volume
6. **Debugging**: Better logging and state tracking

## Risk Mitigation

1. **Stale Processing States**: Automatic cleanup after timeout
2. **Database Performance**: Proper indexing on channel_id and message_order
3. **Message Loss**: Persistent storage prevents queue loss
4. **Concurrency**: Database-level unique constraints prevent race conditions

## Ready for Implementation

All core components are implemented and ready for integration:
- ✅ Database schema designed and created
- ✅ ChannelQueueService fully implemented  
- ✅ Integration plan documented
- 🔄 **Next Step**: Integrate with bot service (requires file modifications)

The system is designed to be backward compatible and can be enabled/disabled via configuration if needed.