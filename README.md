# Claude on Slack

A Slack bot that enables natural language interaction with Claude Code directly in Slack channels, providing seamless AI assistance for development teams.

## 🎯 Overview

Claude on Slack bridges the gap between Slack conversations and Claude Code's powerful AI capabilities, allowing teams to:

- Have natural conversations with Claude without command parsing
- Maintain session context across multiple interactions  
- Collaborate on code analysis, debugging, and development tasks
- Access Claude's file operations, code generation, and analysis tools
- Control permissions and access levels through slash commands

## 🚀 Quick Start

### Prerequisites

- [Claude Code](https://github.com/anthropics/claude-code) installed and configured
- Slack workspace with bot creation permissions
- Go 1.21+ for building the service

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/ghabxph/claude-on-slack.git
   cd claude-on-slack
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your actual tokens and working directory
   ```

3. **Install as systemd service (recommended)**
   ```bash
   # Install as current user (for security)
   sudo ./scripts/install.sh

   # Or specify a different user
   sudo CLAUDE_SERVICE_USER=youruser ./scripts/install.sh
   ```

4. **Manual build and run**
   ```bash
   go mod tidy
   go build -o slack-claude-bot ./cmd/slack-claude-bot
   ./slack-claude-bot
   ```

### Configuration

Copy `.env.example` to `.env` and configure:

```bash
# Required configuration
SLACK_BOT_TOKEN=xoxb-your-slack-bot-token
SLACK_APP_TOKEN=xapp-your-slack-app-token  
SLACK_SIGNING_SECRET=your-slack-signing-secret

# Claude Code CLI configuration
CLAUDE_CODE_PATH=claude
ALLOWED_TOOLS=                    # Empty = all tools (full access)

# Access control
ALLOWED_USERS=user1@domain.com,user2@domain.com
ADMIN_USERS=admin@domain.com

# Auto-response channels (optional)
AUTO_RESPONSE_CHANNELS=C1234567890  # Channel ID where bot responds to ALL messages

# Server settings (for SSH tunnel setup)
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Working directory (set to your home directory for full system access)
WORKING_DIRECTORY=/home/yourusername
```

## 📖 Usage

### Basic Interaction

Just talk naturally to Claude! No special commands or formatting needed:

```
@claude-bot Can you help me optimize this code?

def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

@claude-bot Can you create a dockerfile for a python web app?
```

### Slash Commands

#### Session Management
- `/session` - Show current session info and help
- `/session <claude-session-id>` - Switch to specific session
- `/session new` - Start fresh conversation

#### Permission Control
- `/permission` - Show current permission mode and help
- `/permission default` - Standard permissions with prompts
- `/permission acceptEdits` - Auto-accept file edits
- `/permission bypassPermissions` - Bypass permission checks
- `/permission plan` - Planning mode, won't execute actions

### Advanced Features

- **Natural Language Processing**: Just chat normally, no command parsing
- **Session Continuity**: Conversations maintain context across messages
- **Message Queuing**: Multiple rapid messages get combined intelligently
- **Working Directory**: Current directory shown in responses
- **Permission Modes**: Control Claude's behavior with slash commands

## 🏗️ Architecture

### Core Components

- **Slack Integration**: 
  - Events API support
  - Slash command handling
  - Message queueing system
  
- **Session Management**:
  - Persistent conversations
  - Manual session control
  - Working directory tracking

- **Permission System**:
  - Multiple permission modes
  - Automatic mode reset
  - Fine-grained control

- **Response Handling**:
  - Slack-friendly formatting
  - Progress indicators
  - Directory tracking

## 🔐 Security

- User authentication via Slack
- Signature verification for all requests
- Permission mode system for access control
- Working directory isolation
- Rate limiting and timeout protection

## 🛠️ Development

### Building
```bash
go build -o slack-claude-bot ./cmd/slack-claude-bot
```

### Redeploying
```bash
./scripts/redeploy.sh
```

### Monitoring
```bash
# Check service status
sudo systemctl status slack-claude-bot

# View real-time logs
sudo journalctl -u slack-claude-bot -f
```

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## 🔮 Upcoming Features

### Concurrent Multi-Session Support (In Development)

We're working on enhancing the session management system to better support multi-tasking and concurrent conversations:

#### Key Features
- **Concurrent Sessions**: Run multiple Claude conversations simultaneously
- **Per-Session Mode Control**: Each session maintains its own permission mode and settings
- **Latest Session Tracking**: Automatically manage session lifecycle
  - Keep track of latest/active sessions
  - Discard old/completed sessions
  - Monitor session status (in-progress/completed)

#### Implementation Plan
1. **Phase 1 - Memory-Based Implementation**
   - Simple in-memory session tracking
   - Latest session prioritization
   - Session status monitoring
   - Per-session mode management

2. **Phase 2 - Database Integration**
   - PostgreSQL integration for persistent storage
   - Enhanced session switching capabilities
   - Historical session lookup
   - Comprehensive session metadata

The initial implementation will be kept simple and memory-based, with database integration planned for future scalability.

### Image Support (Research)
• Support for processing images uploaded to Slack channels (not yet implemented)
• Research in progress for integrating Claude's image analysis capabilities
• Will enable visual debugging, code review from screenshots, and more

### Code Edit Visualization
We're working on showing code changes directly in chat:
```diff
# Example future output:
File: src/utils.ts:45
- function getData(id: string): Promise<Data> {
+ async function getData(id: string): Promise<Data | null> {
    const result = await db.query('SELECT * FROM data WHERE id = ?', [id]);
-   return result[0];
+   return result[0] || null;
  }
```

This will help you:
- See exactly what Claude changed
- Track file and line locations
- Understand the context of changes
- Review changes before accepting them

Check `CLAUDE.md` for more planned features!

## 🆘 Support

- Check `CLAUDE.md` for detailed feature list and roadmap
- Report issues via [GitHub Issues](https://github.com/ghabxph/claude-on-slack/issues)
- Join discussions in [GitHub Discussions](https://github.com/ghabxph/claude-on-slack/discussions)

---

Built with ❤️ for development teams who want natural AI assistance in their Slack workflow.