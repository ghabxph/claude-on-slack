# Claude on Slack

A Slack bot that enables non-interactive execution of Claude Code commands directly in Slack channels, providing seamless AI assistance for development teams.

## 🎯 Overview

Claude on Slack bridges the gap between Slack conversations and Claude Code's powerful AI capabilities, allowing teams to:

- Execute Claude Code commands directly from Slack messages
- Maintain conversation context across multiple interactions  
- Collaborate on code analysis, debugging, and development tasks
- Access Claude's file operations, code generation, and analysis tools
- Manage permissions and security controls for team usage

## 🚀 Quick Start

### Prerequisites

- [Claude Code](https://github.com/anthropics/claude-code) installed and configured
- Slack workspace with bot creation permissions
- Anthropic API key with Claude access
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
   # Edit .env with your actual tokens and API keys
   ```

3. **Build and run**
   ```bash
   go mod tidy
   go build -o slack-claude-bot ./cmd/slack-claude-bot
   ./slack-claude-bot
   ```

### Configuration

Copy `.env.example` to `.env` and configure:

```bash
SLACK_BOT_TOKEN=xoxb-your-slack-bot-token
SLACK_APP_TOKEN=xapp-your-slack-app-token  
CLAUDE_API_KEY=sk-your-anthropic-api-key
ALLOWED_USERS=user1@domain.com,user2@domain.com
MAX_DAILY_COST_USD=10.00
WORKING_DIR_BASE=/tmp/slack-claude-sessions
```

## 📖 Usage

### Basic Commands

Mention the bot in any channel or send direct messages:

```
@claude-bot analyze this code snippet:
```python
def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)
```

@claude-bot create a dockerfile for a python web app
```

### Advanced Features

- **Session Continuity**: Conversations maintain context across messages
- **File Operations**: Upload files for analysis or request generated files
- **Tool Control**: Granular permissions for different tools and operations
- **Multi-step Workflows**: Complex tasks broken down across multiple interactions

## 🏗️ Architecture

```
[Slack User] → [Slack API] → [Bot Service] → [Claude Code CLI] → [Response]
```

### Core Components

- **Slack Bot Service**: Go-based service handling Slack events
- **Claude Code Wrapper**: Secure execution of Claude Code commands
- **Session Manager**: Maintains conversation context per user
- **Security Layer**: Authentication, authorization, and command filtering

## 🔐 Security

### Access Control
- User allowlisting and role-based permissions
- Channel restrictions and tool access controls
- Session isolation and working directory management

### Safety Features
- Command sanitization and filtering
- Cost limits and rate limiting
- Comprehensive audit logging
- Timeout protection for long-running operations

## 🛠️ Development

### Project Structure

```
├── cmd/slack-claude-bot/     # Main application entry point
│   └── main.go               # Application entry point with HTTP server
├── internal/                 # Core application modules
│   ├── auth/                 # Authentication & authorization
│   │   └── service.go        # User permissions and access control
│   ├── bot/                  # Slack bot logic  
│   │   └── service.go        # Slack event handling and message processing
│   ├── claude/              # Claude Code executor
│   │   └── executor.go       # Claude Code CLI wrapper and execution
│   ├── config/              # Configuration management
│   │   └── config.go         # Environment variable loading and validation
│   └── session/             # Session management
│       └── manager.go        # User session persistence and context
├── configs/                 # Configuration files and templates
├── scripts/                 # Installation and deployment scripts  
├── docs/                    # Documentation and guides
│   └── examples/            # Usage examples and integration guides
└── tests/                   # Test suites
    ├── unit/                # Unit tests
    └── integration/         # Integration tests
```

### Building

```bash
# Build for current platform
go build -o slack-claude-bot ./cmd/slack-claude-bot

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o slack-claude-bot-linux ./cmd/slack-claude-bot
```

### Testing

```bash
go test ./...
```

## 📦 Deployment

### SystemD Service

For production deployment on Linux:

1. **Copy binary to `/opt/slack-claude-bot/`**
   ```bash
   sudo mkdir -p /opt/slack-claude-bot
   sudo cp slack-claude-bot /opt/slack-claude-bot/
   sudo chmod +x /opt/slack-claude-bot/slack-claude-bot
   ```

2. **Configure environment in `/opt/slack-claude-bot/.env`**
   ```bash
   sudo cp .env.example /opt/slack-claude-bot/.env
   sudo nano /opt/slack-claude-bot/.env  # Edit with your actual values
   ```

3. **Create service user**
   ```bash
   sudo useradd --system --no-create-home --shell /bin/false slack-claude-bot
   sudo chown -R slack-claude-bot:slack-claude-bot /opt/slack-claude-bot
   ```

4. **Install systemd service**
   ```bash
   sudo cp configs/slack-claude-bot.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable slack-claude-bot.service
   sudo systemctl start slack-claude-bot
   ```

### Docker

```bash
docker build -t claude-on-slack .
docker run -d --env-file .env --name claude-bot claude-on-slack
```

## 📊 Monitoring

### Health Check Endpoints
- **Health check**: `http://localhost:8081/health` - Service health status
- **Metrics**: `http://localhost:8081/metrics` - Basic service information

### Logging
- **Structured JSON logging** for production environments
- **Console logging** for development  
- **Comprehensive audit trail** of all user commands and responses
- **Cost tracking** and usage monitoring per user

### Service Monitoring
```bash
# Check service status
sudo systemctl status slack-claude-bot

# View real-time logs
sudo journalctl -u slack-claude-bot -f

# Check health endpoint
curl http://localhost:8081/health
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## 🆘 Support

- **Documentation**: Check the [docs/](docs/) directory for detailed guides
- **Issues**: Report bugs and feature requests via [GitHub Issues](https://github.com/ghabxph/claude-on-slack/issues)
- **Discussions**: Join community discussions in [GitHub Discussions](https://github.com/ghabxph/claude-on-slack/discussions)

## 🔗 Related Projects

- [Claude Code](https://github.com/anthropics/claude-code) - The AI-powered coding assistant
- [Anthropic API](https://docs.anthropic.com/) - Official Anthropic API documentation
- [Slack API](https://api.slack.com/) - Slack platform documentation

---

Built with ❤️ for development teams who want AI assistance directly in their Slack workflow.