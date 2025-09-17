# Changelog

All notable changes to claude-on-slack will be documented in this file.

## [2.10.0] - 2025-09-17

### Added - Intelligent Auto-Compaction System 🤖🗜️
- **Automatic Memory Optimization**: Conversations now automatically compress when reaching 90% of token capacity (162k of 180k tokens)
- **Seamless Background Processing**: Auto-compaction runs asynchronously without interrupting active conversations
- **Smart Trigger System**: Intelligent threshold detection with configurable compaction levels (critical/recommended/consider/none)
- **Zero-Interruption Operation**: Users continue working while memory optimization happens in the background
- **Automatic Session Cleanup**: Short-term memory automatically cleared after successful compaction to long-term storage

### Enhanced - Reusable Compaction Engine 🔄⚡
- **Unified Architecture**: Single compaction system handles both manual (`/compact now`) and automatic triggers
- **Modular Design**: Core compaction methods (`GetCompactionStatus`, `PerformSessionCompaction`, `TriggerAutoCompaction`) reusable across contexts
- **Consistent Error Handling**: Unified error reporting and logging for both manual and automatic operations
- **Comprehensive Monitoring**: Detailed logging with trigger source identification (manual/automatic) for debugging and analytics
- **Flexible Integration**: Easy to call compaction programmatically from any part of the system

### Enhanced - Advanced Memory Analytics 📊🔍
- **Detailed Status Reporting**: Enhanced `/compact status` with token usage percentages, archived data counts, and recommendation levels
- **Threshold Visualization**: Clear display of current usage vs. thresholds with color-coded status indicators
- **Historical Tracking**: Archive timestamps and compaction sequence numbers for complete memory history
- **Capacity Planning**: Shows remaining token capacity and projected compaction timing
- **Smart Recommendations**: Context-aware suggestions based on current memory usage patterns

### Enhanced - Administrative Controls 🎛️⚙️
- **Improved Manual Commands**: 
  - `/compact status` - Enhanced memory analytics with detailed recommendations
  - `/compact now` - Streamlined immediate compaction with real-time notifications
  - `/compact threshold <tokens>` - Configurable auto-compaction triggers (50k-500k range)
  - `/compact help` - Comprehensive command documentation
- **Admin-Only Safety**: All compaction commands require admin privileges to prevent accidental system-wide impacts
- **Real-Time Feedback**: Asynchronous notifications show compaction progress and completion status
- **Flexible Thresholds**: Per-session threshold configuration (future: global settings)

### Technical Improvements 🔧💾
- **Memory-Efficient Processing**: Optimized token counting and step retrieval for better performance
- **Robust Error Recovery**: Graceful handling of compaction failures with detailed error reporting
- **PostgreSQL Integration**: Enhanced long-term memory storage with optimized indexing for fast retrieval
- **Session Isolation**: Each session's compaction operates independently without affecting others
- **Context Preservation**: Complete conversation history maintained through compression cycles

### Performance & Reliability 🚀✅
- **Non-Blocking Operations**: All compaction operations run asynchronously to maintain responsiveness
- **Fail-Safe Design**: Compaction failures don't affect ongoing conversations or system stability
- **Memory Pressure Relief**: Automatic optimization prevents token limit issues in long conversations
- **Scalable Architecture**: Handles unlimited conversation lengths through intelligent memory management
- **Production Ready**: Comprehensive error handling and monitoring for enterprise deployment

### Documentation & Monitoring 📋🔍
- **Comprehensive Logging**: Detailed logs for compaction triggers, operations, and results with performance metrics
- **Status Transparency**: Users can monitor memory usage and compaction history through commands
- **Admin Insights**: Complete visibility into automatic compaction patterns and system behavior
- **Future Planning**: Foundation for advanced features like conversation search, context retrieval, and analytics

## [2.9.0] - 2025-09-17

### Added - Dual-Memory System Foundation 🧠💾
- **Short-Term Memory**: Ephemeral step-by-step conversation tracking with automatic compaction at 180k token threshold
- **Long-Term Memory**: Perfect conversation archive system preserving complete context for future intelligent retrieval
- **Token Monitoring**: Real-time cumulative token tracking per session with warning thresholds at 150k tokens
- **Perfect Memory Architecture**: Complete tool call, input, output, and thinking context preservation in PostgreSQL
- **Feature Flag Protection**: Data collection only active when THINKING_PROCESS feature is enabled

### Technical Infrastructure
- **Database Migration**: Added `short_term_memory` and `long_term_memory` tables with optimized indexes for performance
- **Memory Repository**: Complete CRUD operations for step tracking and session memory management
- **Token Estimation**: Simple character-based token counting (4 chars per token) with configurable thresholds
- **Streaming Integration**: Seamless hook into existing tool call streaming system for zero-impact data collection
- **PostgreSQL Optimization**: Indexed queries for fast step retrieval and cumulative token calculations

### Foundation for Future Features
- **Automatic Compaction**: Infrastructure ready for 200k token limit conversation compression
- **Perfect Context Preservation**: Complete conversation reconstruction capabilities
- **Intelligent Retrieval**: Rich metadata storage for future chunk-based memory access
- **Scalable Architecture**: Handles unlimited conversation length through intelligent archival
- **Token Growth Analysis**: Complete data collection for studying conversation token consumption patterns

## [2.8.0] - 2025-09-16

### Enhanced - Contextual Tool Messages 🧠⚙️
- **Contextual Tool Display**: Tool messages now show the thinking context that led to each tool call, making Claude's reasoning process transparent
- **Intelligent Context Capture**: Automatically captures relevant thinking text before each tool call (truncated to 200 chars for readability)
- **Enhanced User Experience**: Instead of generic "Reading file" messages, see "I need to check the current configuration" → "🔍 Reading config.go..."
- **Smart Text Processing**: Advanced parsing to extract meaningful context while filtering out formatting artifacts
- **Real-Time Reasoning**: Users can now understand not just what Claude is doing, but *why* it's taking each specific action

### Technical Implementation
- **Enhanced ToolCall Structure**: Added `Context` field to capture thinking text before tool execution
- **Streaming Parser Enhancement**: Modified JSON streaming parser to associate thinking content with subsequent tool calls
- **Context Extraction**: Intelligent text processing to extract meaningful context while avoiding repetitive or formatting text
- **Slack Formatting**: Contextual information properly formatted for optimal Slack display with emoji indicators
- **Debug Logging**: Comprehensive logging for context capture and tool call association for troubleshooting

### User Experience Transformation
- **Before**: `🔍 Reading main.go...` (generic)
- **After**: `I need to check the current configuration` → `🔍 Reading main.go...` (contextual)
- **Transparency**: Users understand the reasoning behind each tool call
- **Learning**: Observe Claude's problem-solving approach step-by-step
- **Engagement**: More intuitive and educational AI interaction experience

## [2.7.0] - 2025-09-16

### Added - Real-Time Thinking Process Visibility 🧠✨
- **THINKING_PROCESS Feature Flag**: New `FEATURES=THINKING_PROCESS` environment variable to enable Claude's thinking process visibility
- **Real-Time Tool Execution Display**: See Claude's tool usage in real-time with live message updates (🔍 Reading files, ⚙️ Running commands, ✏️ Writing code)
- **Stream-JSON Integration**: Seamless integration with Claude Code CLI's streaming JSON format for tool execution transparency
- **Backwards Compatible**: Existing users see no changes - feature is opt-in only
- **Slack-Optimized Formatting**: Custom emojis and descriptions for each tool type (Read, Write, Bash, Grep, etc.)
- **Non-Blocking Updates**: Progress messages don't interfere with Claude's execution performance

### Technical Implementation
- **Feature Flag System**: Robust feature detection with `config.IsFeatureEnabled()` method
- **Streaming Parser**: Real-time JSON event parsing with tool call extraction
- **Dual Execution Modes**: Automatic fallback to regular JSON when streaming is disabled
- **Tool Progress Callbacks**: Live callback system for real-time Slack message updates
- **Enhanced Text Collection**: Improved streaming response text aggregation for complete final responses
- **Debug Logging**: Comprehensive tool call callback logging for troubleshooting
- **Error Handling**: Graceful degradation and comprehensive logging for debugging

### User Experience
- **Default Behavior**: `🤔 Thinking...` → Final response (unchanged)  
- **With THINKING_PROCESS**: `🤔 Claude is thinking...` → `🔍 Reading main.go...` → `⚙️ Running go build...` → `✏️ Writing config.go...` → Final response
- **Transparency**: Users can now see exactly what Claude is doing step-by-step
- **Engagement**: More interactive and engaging AI experience with visible "thinking"

### Fixed - Deployment Notification System
- **Corrected Startup Messages**: Fixed hardcoded deployment notifications to reflect v2.7.0 features instead of outdated v2.6.x session management content
- **Dead Code Cleanup**: Removed unused fallback deployment message system that was never executed
- **Single Source of Truth**: Deployment messages now come from one location (`internal/bot/service.go`) eliminating confusion


## [2.6.3] - 2025-08-31

### Enhanced - Markdown Heading Prevention
- **No Markdown Headings**: Added explicit instruction to never use markdown headings (## text) in Slack responses
- **Bold Headings Only**: Enforces use of *bold text:* format instead of markdown headings for better Slack rendering
- **Improved Readability**: Ensures all section headers display properly in Slack interface

## [2.6.2] - 2025-08-31

### Enhanced - Slack Formatting & User Experience  
- **Slack-Compatible Formatting**: Added explicit instructions to format responses for optimal Slack display using Slack syntax instead of markdown
- **Better Visual Presentation**: Ensures proper rendering of bold, italic, code blocks, and lists in Slack interface
- **Conditional Markdown**: Only use markdown formatting when explicitly requested by users

## [2.6.1] - 2025-08-31

### Enhanced - System Prompt & Task Management
- **Improved System Prompt**: Enhanced Claude Code system prompt with comprehensive workflow guidelines for better task planning, research, and execution
- **Better Task Communication**: More verbose explanations since Claude's internal reasoning isn't visible in Slack
- **Code Cleanup**: Removed unused `GetClaudeCodeSystemPrompt()` function

## [2.6.0] - 2025-08-31

### Added - Conversation Summarization Feature
- **New `/summarize` Slash Command**: Generate detailed AI summaries of current conversation sessions
- **Async Processing**: Prevents Slack timeout errors with immediate response and background processing
- **Context Compression**: Specialized system prompt optimized for detailed technical summaries and task continuity  
- **Disposable Sessions**: Uses Claude Code CLI with temporary session IDs to avoid affecting main conversations
- **Smart Conversation Formatting**: Chronological timestamp-based conversation reconstruction for accurate summarization

### Enhanced - User Experience
- **Immediate Feedback**: Returns "Summarizing [session-id]... Please wait." instantly to prevent timeouts
- **Follow-up Delivery**: Sends complete summary as separate message when processing completes
- **Error Handling**: Private error messages using existing dual logger system (console + ephemeral Slack messages)
- **Session Context**: Displays session UUID and conversation count in summary headers
- **Slack Formatting**: Converts markdown formatting to Slack-compatible display format

### Technical - Implementation Details
- **Background Goroutines**: Async processing pattern using `go performAsyncSummarization()`
- **Direct Claude Integration**: New `ExecuteClaudeSummary()` method with specialized prompting
- **Message Pipeline**: Uses stdin for conversation text to avoid command-line escaping issues
- **Proper UUID Generation**: Uses `uuid.New().String()` for disposable session IDs instead of timestamp-based IDs
- **Debug Logging**: Comprehensive logging for conversation input, raw responses, and formatted output
- **PostMessage API**: Direct Slack API calls for follow-up message delivery

### Fixed - Claude Code CLI Integration
- **Flag Correction**: Changed `--working-dir` to `--add-dir` for proper directory access
- **System Prompt Separation**: Uses `--append-system-prompt` flag instead of inline prompting
- **Input Method**: Switched to stdin piping for message content to avoid shell escaping issues
- **Error Capture**: Proper stdout/stderr separation for better error reporting and debugging

## [2.5.6] - 2025-08-31

### Fixed - Session Path Switching Bug
- **Critical Fix**: Fixed completely broken `/session . <path>` command that only showed success message without creating session
- **Actual Session Creation**: Command now properly creates new session with specified working directory
- **Channel State Update**: Session switching now properly updates channel to use new session with correct working directory
- **Error Handling**: Added proper error handling and logging for session creation failures

### Enhanced - Session Management
- **Working Directory Control**: `/session . <path>` now actually switches to specified working directory
- **Session Consistency**: Ensures new sessions are properly created and linked to channels
- **User Experience**: Command now works as documented and expected

### Technical - Bug Resolution
- **Database Operations**: Added missing `CreateSessionWithPath()` call in session dot command handler
- **State Management**: Proper channel state updating when creating sessions for new paths
- **Logging**: Enhanced error logging for session creation debugging

## [2.5.5] - 2025-08-31

### Simplified - Deployment Notification System
- **Removed DEPLOYMENT_NOTIFY_CHANNELS**: Eliminated redundant environment variable for cleaner configuration
- **Unified Notification System**: Deployment notifications now sent to ALL allowed channels automatically
- **Health Check Enhancement**: Deployment notifications serve as connectivity health check for all allowed channels
- **Better Visibility**: All allowed channels receive deployment notifications for improved monitoring

### Enhanced - Channel Management
- **Consistent Behavior**: All allowed channels treated equally for notifications
- **Simplified Configuration**: One less environment variable to manage
- **Operational Visibility**: Deployment notifications confirm bot can interact with all configured channels

### Technical - Code Cleanup
- **Removed Environment Parsing**: Eliminated `os.Getenv("DEPLOYMENT_NOTIFY_CHANNELS")` logic
- **Simplified Notification Logic**: Direct usage of `AllowedChannels` instead of complex fallback system
- **Unused Import Cleanup**: Removed unused `os` import from bot service

## [2.5.4] - 2025-08-31

### Simplified - Configuration Cleanup  
- **Removed AUTO_RESPONSE_CHANNELS**: Eliminated redundant environment variable for cleaner configuration
- **Unified Bot Behavior**: Bot now responds to ALL messages in ALLOWED_CHANNELS (no mention needed)
- **Simplified Logic**: Removed dual response mode complexity - consistent behavior across all allowed channels
- **Configuration Reduction**: One less environment variable to configure and manage

### Enhanced - User Experience
- **Consistent Behavior**: Bot works the same way in all allowed channels - no more confusion about mention vs auto-response modes
- **Predictable Operation**: If you're in an allowed channel, bot responds to all messages
- **Simplified Setup**: Only need to configure ALLOWED_CHANNELS instead of both ALLOWED_CHANNELS and AUTO_RESPONSE_CHANNELS

### Technical - Code Simplification
- **Removed Config Field**: Eliminated `AutoResponseChannels []string` from config struct
- **Removed Method**: Removed `IsAutoResponseChannel()` configuration method
- **Simplified Message Processing**: Streamlined message handling logic by removing dual response mode checks
- **Updated Fallback Logic**: Startup notifications now fallback to `AllowedChannels` instead of `AutoResponseChannels`

## [2.5.3] - 2025-08-31

### Enhanced - Session Display Improvement
- **Meaningful Session Labels**: Replaced confusing "Claude Session ID" and "Bot Session ID" (which showed identical values) with clear "Parent Session" and "Leaf Session" information
- **Database-Driven Display**: Session information now retrieved directly from `slack_channels` table using existing parent/leaf session architecture
- **Smart Data Resolution**: Added methods to convert database foreign key IDs to user-friendly session UUIDs for display
- **Hierarchical Session Visibility**: Users can now see the actual session hierarchy structure (root conversation vs current active child)

### Technical - Database Access Enhancement
- **New Repository Method**: Added `GetChildSessionByID()` for retrieving child session information by database ID
- **Public Session Access**: Added `GetChannelState()`, `GetChildSessionByID()`, and `LoadSessionByID()` methods to DatabaseManager
- **Type-Safe Conversion**: Proper conversion from database integer foreign keys to session UUID strings for user display

## [2.5.2] - 2025-08-31

### Fixed - Session Switching Bug
- **Critical Fix**: Fixed completely broken `/session <uuid>` command that only returned fake success messages without actually switching sessions
- **Database Updates**: Session switching now properly updates both `slack_channels` table and memory cache
- **Session Validation**: Added proper session existence validation before attempting to switch
- **Error Handling**: Comprehensive error handling with context logging for session switching operations
- **Memory Consistency**: Fixed cache synchronization to maintain consistency between database and memory state

### Enhanced - Session Management
- **Interface Extension**: Added `SwitchToSessionInChannel()` method to SessionManager interface
- **Database Integration**: Full database-backed session switching with proper transaction handling
- **Leaf Session Tracking**: Automatic detection and setting of leaf child sessions during switch operations

## [2.5.1] - 2025-08-31

### Fixed - Database Query Robustness
- **Schema-Safe Queries**: Replaced all `SELECT *` queries with explicit column names to prevent schema evolution issues
- **Column Order Independence**: Fixed critical SQL scan error where database column order didn't match struct field order  
- **Migration-Proof Architecture**: Future database migrations won't break existing queries due to column reordering
- **Child Session Queries**: Fixed `GetConversationTree()` and `FindLeafChild()` functions that were failing due to `summary` column positioning
- **Complete Query Overhaul**: Updated all repository functions (`GetSessionBySessionID`, `ListAllSessions`, `GetChannelState`, etc.) to use explicit column selection

### Technical - Database Layer Improvements
- **Explicit Column Selection**: All queries now specify exact columns needed, improving performance and maintainability
- **Type Safety**: Eliminated runtime SQL scan errors caused by column/struct field misalignment
- **Future-Proof Design**: Database schema can evolve safely without breaking application queries
- **Debugging Enhancement**: Clear column-to-field mapping makes debugging database issues much easier

## [2.5.0] - 2025-08-31

### Added - Centralized Error Logging System
- **Dual-Output Logger**: All errors now sent to both console logs and originating Slack channel
- **Real-time Error Visibility**: Detailed stack traces and error context visible directly in Slack chat  
- **Context-Aware Reporting**: Every error includes channel, user, component, operation, and session information
- **Ephemeral Error Messages**: Error details visible only to the user who triggered them (not channel-wide)
- **Enhanced Debugging**: Stack trace extraction with exact code locations for faster troubleshooting

### Fixed - Critical Database Issues
- **SQL Scan Mismatch**: Fixed column order mismatch in `slack_channels` table scan operation
- **Hardcoded Error Messages**: Replaced hardcoded error returns with centralized error logging
- **Database Schema Alignment**: Corrected `GetChannelState()` scan order to match actual database column sequence
- **Permission Column Position**: Fixed scanning `permission` field from correct database position (column 6)

### Enhanced - Error Handling
- **Stack Trace Integration**: Added automatic stack trace capture and location detection
- **JSON-Safe Formatting**: Error messages properly formatted for Slack API compatibility
- **Component Identification**: All errors tagged with specific component and operation context
- **Background Error Logging**: Errors logged to console even if Slack posting fails

### Technical - Architecture Improvements  
- **DualLogger Package**: New `internal/logging/dual_logger.go` for centralized error management
- **Service Integration**: Updated main Service struct to use dual logger throughout
- **Channel Context Propagation**: All error-prone functions now receive channel/user context
- **Error Context Structure**: Structured error reporting with predefined context fields

## [2.3.0] - 2025-08-30

### Fixed - Database Schema Redesign  
- **Critical Fix**: Corrected fatal database schema error where `previous_session_id` was integer instead of varchar UUID
- **Schema Migration**: Updated migration 001 to use `VARCHAR(255)` for proper UUID chain storage
- **Repository Update**: Fixed `ChildSession.PreviousSessionID` type from `*int` to `*string`
- **Chain Logic**: Previous session ID now stores UUIDs enabling proper conversation reconstruction

### Enhanced - Session Design Implementation
- **Message Counting**: Simplified formula to `child_count` instead of `(child_count * 2) + 1` 
- **Root Session**: Properly implemented as blank state with `UserPrompt = NULL`
- **Conversation Chain**: Each child session represents one exchange linked via UUID chain
- **Future Ready**: Design supports conversation reconstruction and compression features

### Technical - Conversation Flow
- **Root Session**: Created blank, gets `UserPrompt` when user sends message
- **Child Creation**: Claude response creates child session with `AIResponse`, clears `UserPrompt`  
- **Chain Building**: `PreviousSessionID` links to previous session's UUID (root or child)
- **Traceability**: All sessions linked via `RootParentID` for complete conversation trees

### Database Impact
- **Breaking Change**: Requires database cleanup and recreation with new schema
- **UUID Storage**: `previous_session_id` column now stores session UUIDs instead of database IDs
- **Chain Structure**: Enables traversal: Root → Child1 → Child2 → Child3 via UUID references

## [2.2.12] - 2025-08-30

### Fixed - Session Continuity Issues
- **Critical Fix**: Resolved "No conversation found with session ID" errors that prevented successful message processing
- **Session Logic**: Fixed session determination to check child sessions instead of message count for proper new/resume detection
- **New Session Support**: `/session new` followed by messages now works correctly using `--session-id` for first message
- **Resume Logic**: Subsequent messages properly use `--resume` with Claude's returned session IDs from child sessions
- **Debug Enhancement**: Added comprehensive debug information to error messages including session IDs, commands, and execution context

### Technical - Session Management Improvements
- **Child Session Detection**: Modified logic to check for existence of child sessions (actual Claude conversations) rather than total message count
- **Command Selection**: Proper use of `--session-id` for new conversations and `--resume` for continuing conversations
- **Error Diagnostics**: Enhanced error messages with complete debug information including full Claude Code CLI commands
- **Session Flow**: Corrected flow where parent sessions (bot UUIDs) are used for new conversations, child sessions (Claude UUIDs) for resuming
- **Database Integration**: Improved parent-child session relationship tracking and retrieval

### User Impact
- **Before**: New sessions immediately failed with "No conversation found with session ID" error
- **After**: Seamless conversation flow with proper session creation and continuation
- **Debugging**: Error messages now include full execution context for faster troubleshooting

## [2.2.4] - 2025-08-30

### Enhanced - Error Reporting & Diagnostics
- **Smart Error Categorization**: Automatic classification of Claude Code execution errors into specific types (permission, network, syntax, file, timeout)
- **Contextual Troubleshooting**: Actionable suggestions and solutions provided for each error category
- **Detailed Stderr Output**: Full capture and preservation of stderr output from Claude Code CLI execution
- **Enhanced Error Messages**: Rich, formatted error messages with markdown styling and emoji indicators
- **Faster Debugging**: Eliminated guesswork by providing comprehensive error context and specific failure details

### Technical - Error Handling Improvements
- **createEnhancedError()**: New method for generating detailed error messages with category-specific troubleshooting
- **categorizeError()**: Intelligent error analysis to classify failures based on stderr and error patterns
- **Stderr Preservation**: Modified executor to capture and include stderr output in all error responses
- **Pattern Matching**: Advanced text analysis to identify common error patterns and provide targeted help
- **Duration Tracking**: Include execution time in error messages for performance troubleshooting

### Fixed - User Experience
- **Generic Error Messages**: Replaced "claude code execution failed: exit status 1" with detailed diagnostic information
- **Missing Context**: Stderr output no longer lost; now displayed with proper formatting in Slack messages
- **Troubleshooting Gaps**: Added specific guidance for permission issues, missing commands, network problems, etc.
- **Error Classification**: Unknown failures now properly categorized with relevant troubleshooting steps

### User Impact
- **Before**: `❌ Claude Code processing failed: failed to execute Claude Code: claude code execution failed: exit status 1`
- **After**: `🔒 **Permission Denied** - The system denied access to required resources. [Full stderr output + specific troubleshooting steps]`

## [2.2.3] - 2025-08-29

### Enhanced - System Reliability & Compatibility
- **Application Auto-Restart**: Added robust application-level restart capability with exponential backoff (max 5 attempts, 10s delay)
- **SystemD Compatibility**: Fixed `user.Current()` failures in systemd environment with fallback to default "claude-bot" username
- **Database Connection**: Resolved PostgreSQL networking and authentication issues through container recreation
- **HTTP Server Validation**: Enhanced server startup error detection and propagation to main application
- **Error Recovery**: Comprehensive error handling with graceful failure recovery and detailed logging

### Technical - Infrastructure Improvements
- **Connection Resilience**: Added retry logic with timeout handling for database connections
- **Service Monitoring**: Improved startup validation with health checks and timeout detection  
- **Process Management**: Enhanced service lifecycle management with proper error propagation
- **Network Troubleshooting**: Diagnostic improvements for Docker networking and PostgreSQL connectivity
- **Authentication**: Proper password setup for PostgreSQL user authentication with SCRAM-SHA-256

### Fixed - Critical Issues
- **Database Timeouts**: Fixed TCP connection timeouts by recreating PostgreSQL container with clean networking state
- **Service Crashes**: Resolved service crashes from `user.Current()` system calls in restricted environments
- **Connection Failures**: Fixed "connection reset by peer" errors through proper authentication setup
- **Startup Failures**: Added startup validation to prevent silent HTTP server failures
- **Error Propagation**: Fixed issues where server errors weren't properly reported to the main application

### Operational - Deployment
- **Service Stability**: Bot now maintains continuous operation with automatic failure recovery
- **Database Persistence**: Fully operational database-backed session management
- **Slack Integration**: Confirmed operational Slack webhook connectivity and message handling
- **Monitoring**: Enhanced logging and error reporting for better operational visibility

## [2.1.1] - 2025-08-27

### Enhanced - Session Information Display
- **Message Counting**: Display total message count from root/parent session in every bot response
- **Bullet Format**: Changed session information footer to clean bullet point format
- **Session Tracking**: Behind-the-scenes tracking of parent sessions for accurate message counts
- **User Experience**: No longer need to call `/session` command to see message count

### Technical - Session Management  
- **Interface Extension**: Added `GetTotalMessageCount()` method to SessionManager interface
- **Database Integration**: Conversation tree message counting for database-backed sessions
- **Memory Management**: Message count tracking for memory-based session manager
- **Response Formatting**: Updated both "Thinking..." and final response footers

### UI/UX - Response Format
- **Before**: `_Mode: \`default\`, Session: \`abc123\`, Working Dir: \`/path\`_`
- **After**: `_• Mode: \`default\`\n• Session: \`abc123\`\n• Working Dir: \`/path\`\n• Messages: 15_`

## [2.1.0] - 2025-08-27

### Added - Image Processing Support
- **Image Analysis**: Complete image processing pipeline for Slack file uploads
- **File Download System**: Automatic download and temporary storage of Slack images
- **File Event Handling**: Support for `file_shared` events and message file attachments
- **Supported Formats**: JPEG, PNG, GIF, and WebP image analysis
- **Automatic Cleanup**: Intelligent file cleanup with 2-hour retention and 5-minute delay
- **Claude Integration**: Seamless image path passing to Claude Code CLI with `--add-dir` support

### Enhanced - Bot Capabilities  
- **Natural Language + Images**: Process text and images together in conversations
- **Error Handling**: Comprehensive error handling for file downloads and processing
- **Security**: File size limits (50MB max), format validation, and safe filename handling
- **Performance**: Concurrent file processing and background cleanup services
- **User Experience**: Clear feedback for unsupported files and processing errors

### Technical - File Management
- **Storage System**: `/tmp/claude-slack-images/` with user-specific file naming
- **Cleanup Service**: Periodic cleanup service running every 30 minutes
- **File Security**: Sanitized filenames and validated MIME types
- **API Integration**: Slack Files API with proper bot token authentication
- **Memory Management**: Efficient file handling with deferred cleanup

### Architecture - New Components
- **Files Package**: `internal/files/downloader.go` and `internal/files/cleanup.go`
- **Bot Integration**: Enhanced message processing with image attachment detection
- **Claude Executor**: Updated to include image directory access permissions
- **Event Handling**: Extended Events API processing for file-related events

### Requirements - Bot Permissions
- **Slack OAuth Scope**: `files:read` permission required for image downloads
- **Directory Access**: `/tmp/claude-slack-images/` directory creation and management
- **Claude Code CLI**: `--add-dir` flag support for image directory access

## [2.0.0] - 2025-08-26

### Added - PostgreSQL Migration
- **Database Integration**: Complete PostgreSQL session persistence system
- **Docker Compose**: PostgreSQL container configuration for easy deployment
- **Database Schema**: 3-table design (sessions, child_sessions, slack_channels)
- **Migration System**: SQL migration files for database setup
- **Repository Layer**: Database abstraction layer with CRUD operations
- **Session Persistence**: Sessions survive service restarts and crashes
- **Conversation Chains**: Complete conversation history recording
- **Performance Optimization**: O(1) session lookup with memory optimization

### Enhanced
- **Configuration System**: Added database configuration options
- **Version Management**: Version tracking and build-time information
- **Deployment Notifications**: Slack notification system for deployments
- **Deployment Scripts**: Enhanced redeploy script with database support

### Technical
- **Dependencies**: Added PostgreSQL driver (lib/pq v1.10.9)
- **Architecture**: Database-backed session manager with repository pattern
- **Testing**: Database integration tests and connection validation
- **Documentation**: Complete migration plan and implementation guides

### Configuration
- **Database Environment Variables**: Full database configuration support
- **Feature Flags**: `ENABLE_DATABASE_PERSISTENCE` for gradual migration
- **Docker Integration**: PostgreSQL container with health checks
- **Migration Support**: Automated database schema deployment

## [1.0.0] - 2025-08-25

### Added - Initial Release
- **Natural Language Interface**: Chat with Claude without command parsing
- **Session Management**: Persistent conversation context
- **Permission System**: Multiple permission modes (default, acceptEdits, bypassPermissions, plan)  
- **Slack Integration**: Events API and Socket Mode support
- **Message Queuing**: Smart message combining for rapid interactions
- **Working Directory Tracking**: Real-time directory information
- **Slash Commands**: Session and permission management
- **Admin Features**: `/stop` and `/debug` commands
- **Authentication**: User allowlisting and access control
- **SystemD Integration**: Production-ready service deployment

### Features
- **Modes**: `default`, `acceptEdits`, `bypassPermissions`, `plan`
- **Commands**: `/session`, `/permission`, `/stop`, `/debug`
- **Auto-Response**: Configurable channels for automatic responses
- **Rate Limiting**: Per-user rate limiting and timeout protection
- **Logging**: Structured JSON logging with configurable levels
- **Health Checks**: HTTP health endpoint for monitoring

### Technical
- **Go Implementation**: High-performance concurrent Slack bot
- **Claude Code Integration**: Direct CLI integration with session support
- **Security**: Signature verification and user authentication
- **Performance**: Message queuing and session optimization