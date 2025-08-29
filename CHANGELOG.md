# Changelog

All notable changes to claude-on-slack will be documented in this file.

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