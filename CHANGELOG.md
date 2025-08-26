# Changelog

All notable changes to claude-on-slack will be documented in this file.

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