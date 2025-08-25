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
# Required configuration
SLACK_BOT_TOKEN=xoxb-your-slack-bot-token
SLACK_APP_TOKEN=xapp-your-slack-app-token  
SLACK_SIGNING_SECRET=your-slack-signing-secret

# Claude Code CLI configuration
CLAUDE_CODE_PATH=claude
ALLOWED_TOOLS=Read,Write,Bash,Grep,Glob,WebSearch

# Access control
ALLOWED_USERS=user1@domain.com,user2@domain.com
ADMIN_USERS=admin@domain.com

# Server settings (for SSH tunnel setup)
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Working directory
WORKING_DIRECTORY=/tmp/claude-workspace
```

## 📖 Usage

### Basic Commands

Mention the bot in any channel or send direct messages:

```
@claude-bot analyze this code snippet:

def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

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

    claude-on-slack/
    ├── cmd/
    │   └── slack-claude-bot/
    │       └── main.go
    ├── internal/
    │   ├── auth/
    │   │   └── service.go
    │   ├── bot/
    │   │   └── service.go
    │   ├── claude/
    │   │   └── executor.go
    │   ├── config/
    │   │   └── config.go
    │   └── session/
    │       └── manager.go
    ├── configs/
    ├── scripts/
    ├── docs/
    │   └── examples/
    ├── tests/
    │   ├── unit/
    │   └── integration/
    ├── .env.example
    ├── go.mod
    ├── go.sum
    ├── LICENSE
    └── README.md

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
