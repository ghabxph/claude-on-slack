# Claude on Slack

**🚧 DEVELOPMENT STATUS - ALPHA/BETA VERSION 🚧**

**⚠️ This project is currently in active development and not yet production-ready. Expect bugs, breaking changes, and incomplete features.**

**🤖 Your Personal AI Assistant - Jarvis in Chat Mode (In Development)**

**Code from anywhere - even your mobile phone!** Claude on Slack brings the power of programming to your fingertips through natural conversation with AI, accessible wherever you have Slack.

## 🏠 Vision: The Gateway to AI-Powered Home Automation

Imagine having **Jarvis, but in chat mode** - that's the dream I'm building toward! Right now, this project lets you:

### 📱 **Current Reality: Code Anywhere**
- **Mobile Development**: Write, debug, and deploy code from your phone through Slack chat
- **Natural Conversation**: No command syntax - just talk to the AI about what you want to build
- **Persistent Sessions**: Your coding conversations continue across devices and time
- **Image Analysis**: Upload screenshots of errors, diagrams, or hardware setups for instant feedback

### 🚀 **Future Vision: True Home Control** 
The real reason I built this is to eventually control **all my devices** by just chatting with the AI. The plan:

- **IoT Integration**: Connect smart lights, sensors, cameras, and home systems (coming soon!)
- **Remote Machine Control**: Manage home servers, deploy apps, monitor systems - all through chat
- **Automated Workflows**: Create intelligent scripts that respond to natural language commands
- **Device Orchestration**: Coordinate multiple systems and devices through conversational AI

### 🔧 **Current State: Active Development**
**Honest Reality**: This project is in active development with ongoing bugs and stability issues. It's not production-ready yet.

**Development Roadmap to Stability**: 

### 🎯 **Phase 1: Alpha Stabilization (Current)**
- **Current Focus**: Core feature implementation and critical bug fixes
- **Status**: Actively adding features while maintaining basic functionality
- **Risk**: Things can easily break as we prioritize rapid development over stability

### 🧪 **Phase 2: Test-Driven Development Migration (Next)**
- **Goal**: Transition from "move fast and break things" to "move fast and don't break things"
- **Approach**: 
  - **High Code Coverage**: Implement comprehensive tests aiming for 80%+ coverage
  - **Code Refactoring**: Restructure existing code into testable, mockable components
  - **Test-First Development**: All new features must be written with tests first
- **Key Principle**: **Tests should rarely change** - only when requirements are genuinely misunderstood
- **Expected Duration**: 2-4 weeks of focused testing implementation

### 🏆 **Phase 3: Production Stability (Future)**
- **Milestone**: Project considered stable when TDD practices are fully integrated
- **Definition of Stable**:
  - Every new feature requires passing tests before merge
  - Existing functionality protected by comprehensive test suite
  - Breaking changes become extremely rare and intentional only
  - Confident deployments with automated testing pipeline

### 📋 **Testing Strategy**
- **Unit Tests**: Core business logic, session management, database operations
- **Integration Tests**: Slack API interactions, database connections, file operations  
- **End-to-End Tests**: Complete user workflows from Slack message to Claude response
- **Mocking Strategy**: External dependencies (Slack API, Claude CLI, database) properly mocked
- **Continuous Integration**: All tests must pass before deployment

**Timeline**: Currently in Phase 1, targeting Phase 2 completion by Q4 2025

**But here's the magic**: Even in development, being able to code through conversation is the **entry point** to controlling anything programmatically! Want to control your lights? Write a script. Need to monitor your servers? Code it up. Want to automate your morning routine? Program it through chat.

**Your imagination is your limit** - and now you have the tools to build anything you can dream up! 🎨

## 📸 **Preview: See It In Action**

Here's what natural conversation with your personal AI assistant looks like in Slack:

<img width="1348" height="1258" alt="image" src="https://github.com/user-attachments/assets/348341fc-7931-4908-8e56-8cb9a74f1f5c" />

*Real conversation: Planning a PostgreSQL migration through natural chat - no command syntax needed, just talk to your AI like you would a colleague!*

**What you're seeing:**
- 💬 **Natural Language**: Just chat normally about what you want to accomplish
- 🤖 **AI Planning**: Your assistant understands complex technical requirements  
- 📋 **Detailed Responses**: Comprehensive plans, code suggestions, and implementation steps
- 🔄 **Session Continuity**: Conversations persist across devices and time
- ⚡ **Real-time**: Immediate responses for coding, debugging, and automation tasks

This is your "Jarvis in chat mode" - working **right now** on real projects! 🚀

### 🖼️ **Image Analysis in Action**

Upload any image and get instant AI analysis - perfect for debugging, technical analysis, or understanding complex diagrams:

<img width="1251" height="1167" alt="image" src="https://github.com/user-attachments/assets/1b2a3c84-4a1a-4a11-8667-d748b21b8001" />

*Real example: Upload a TradingView screenshot and Claude provides comprehensive technical analysis including price movements, trend analysis, and trading insights - all through natural conversation!*

**Image Processing Features:**
- 🔍 **Smart Analysis**: Detailed breakdown of charts, diagrams, code screenshots, and technical content
- 📊 **Technical Insights**: Understanding of financial charts, system architecture, and data visualizations  
- 💬 **Conversational**: Just upload and ask questions - no special commands needed
- 🔄 **Seamless Integration**: Combine image analysis with your ongoing coding and automation conversations

## ⚠️ **Current Limitations & Cost Considerations**

### 🔗 **Dependency: Claude Code CLI**
This project is **heavily integrated** with [Claude Code CLI](https://claude.ai/code) - that's both its superpower and current limitation:

- **✅ Pros**: Extremely reliable, powerful AI capabilities, seamless integration
- **⚠️ Cons**: Requires Anthropic's Claude subscription, not fully self-hosted

### 💰 **Real Cost Reality**  
**In my home setup, I pay $100/month for Claude Max subscription.**

- **Expensive?** Yes, kinda pricey for a personal setup
- **Worth it?** For me, totally worth it - but I understand it's a barrier for many
- **Future Goal**: Make this setup cheaper with alternative LLM integrations

### 🔮 **Future Alternatives (Maybe)**
Looking toward more self-hosted friendly options:

- **[AIDE](https://github.com/paul-gauthier/aider)**: Potential integration for local LLM support
- **Custom Agent Development**: May develop custom agents for other LLMs  
- **Cost Reduction**: Working toward cheaper, self-hosted alternatives
- **Current Reality**: Claude Code CLI remains the main core due to reliability

### 🌐 **Network Requirements: Exposing Your Home Setup**
**Reality Check**: Your home is behind NAT, so Slack can't reach your endpoints directly.

**Solution Options**:
1. **Static IP**: If you have a machine with public static IP (lucky you!)
2. **Bastion Server**: Buy a cheap VPS for SSH tunneling (**Recommended**)

**💰 Bastion Server Cost**: ~$5/month from:
- **Digital Ocean**: Basic droplet
- **Linode**: Nanode plan  
- **Contabo**: VPS S plan

**Setup**: Use SSH port forwarding to expose Slack endpoints from your home network through the bastion server.

### 🔒 **IMPORTANT: Create Your Own Private Slack Workspace**

**⚠️ Security & Privacy Warning:**
**Create your own private Slack workspace** - don't use your work or shared Slack!

**Why this matters:**
- **Slack is FREE** for personal use - take advantage of it! 
- **You'll be the only user** in your personal workspace
- **Privacy**: Your home automation commands stay private
- **Security**: You don't want coworkers accidentally turning off your lights! 💡
- **Control**: Full admin control over your personal AI assistant

**Setup Steps:**
1. Go to slack.com and create a **new workspace** 
2. Make it **private** and **personal** to you
3. Install your Claude-on-Slack bot there
4. Enjoy complete control over your digital home

**Remember**: This is YOUR personal Jarvis - keep it personal! 🏠🔒

### 🎯 **Total Real Cost**
- **Claude Max**: $100/month
- **Bastion Server**: $5/month  
- **Private Slack Workspace**: **FREE** 🆓
- **Total**: ~$105/month for full "Jarvis" experience

Worth it? For me, absolutely. Your mileage may vary! 💸

## 🚀 Quick Start

**⚠️ DEVELOPMENT WARNING**: This setup is for **developers and early adopters** only. Expect to encounter bugs and may need to troubleshoot issues during setup and usage.

### 🤖 **Golden Rule: Let Claude Do The Heavy Lifting!**

**Most importantly, to save yourself from headache, just use Claude to help you set this up - like I do!**

**💡 Pro Tip**: I literally ask Claude to set everything up for me. It's the perfect self-referential loop:
- Use Claude to set up Claude-on-Slack 
- Claude knows its own integration better than anyone
- Avoid configuration nightmares and debugging sessions

**Simply open Claude Code CLI and ask:**
```
"Help me set up claude-on-slack from https://github.com/ghabxph/claude-on-slack. 
I need help with:
• Slack app creation and OAuth configuration
• Environment setup (.env file configuration)
• Database setup (PostgreSQL with Docker)
• SSH tunneling setup for bastion server
• Deployment and systemd service configuration"
```

**Claude will walk you through**:
- Slack workspace configuration
- Bastion server SSH tunneling setup  
- PostgreSQL database initialization
- Environment variables configuration
- Service deployment and monitoring

**Why this works so well**: Claude understands the entire stack, knows common pitfalls, and can troubleshoot issues in real-time. Don't suffer through manual configuration - let the AI do what it does best! 🚀

### Manual Installation

**⚠️ PROCEED WITH CAUTION**: Manual installation is **not recommended** for production use. This is an active development project with frequent breaking changes and known bugs.

If you prefer to set up manually for **development/testing purposes**, here are the prerequisites and steps:

#### Prerequisites

- [Claude Code](https://github.com/anthropics/claude-code) installed and configured
- Slack workspace with bot creation permissions  
- Go 1.21+ for building the service

#### Installation Steps

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

# Bot responds to ALL messages in allowed channels (no mention needed)
# Deployment notifications are automatically sent to all allowed channels
ALLOWED_CHANNELS=C1234567890,C0987654321  # Channel IDs where bot is allowed to operate

# Server settings (for SSH tunnel setup)
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Working directory (set to your home directory for full system access)
WORKING_DIRECTORY=/home/yourusername
```

### Slack App Configuration

For image processing support, ensure your Slack app has the following OAuth permissions:

#### Bot Token Scopes (Required):
- `app_mentions:read` - Read mentions of the bot
- `channels:read` - Read channel information  
- `chat:write` - Send messages as the bot
- `files:read` - **Download and analyze uploaded images**
- `users:read` - Read user information

#### Event Subscriptions (Required):
- `app_mention` - When someone mentions the bot
- `message.channels` - Messages in channels where bot is member
- `message.groups` - Messages in private channels  
- `message.im` - Direct messages to the bot
- `file_shared` - **When files/images are shared**

#### Features and Functionality:
- ✅ **Slash Commands** - For `/session`, `/permission` commands
- ✅ **Bots** - Enable bot user

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
- `/session` - Show current session info, available sessions, and suggested paths
- `/session list` - Show detailed list of all sessions grouped by path
- `/session <claude-session-id>` - Switch to specific session
- `/session new` - Start fresh conversation in current directory  
- `/session new <path>` - Start fresh conversation in specific path
- `/session . <path>` - Switch to or create session for specific path

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

## 🌟 Open Source Philosophy

**Fork It. It's Yours. Adapt the Open Source Culture.**

This project is my personal contribution to the world 🌍 - a working example of how AI can transform your home into a smart, automated environment. I use this daily in my own home setup and am sharing it with the DIY community.

### 🚨 **Important Notes:**
- **No Pull Requests Accepted**: This is my personal home automation system. I won't be accepting PRs as this could affect my home setup
- **Fork & Modify**: Feel free to fork this project and make it your own! Adapt it to your needs, add your own features
- **You're On Your Own**: For any issues with your fork or modifications, you'll need to debug and fix them yourself
- **Community Spirit**: I encourage you to share your forks and improvements with others in the DIY community

### 💡 **Recommended Approach:**
1. **Fork** this repository
2. **Study** the code to understand how it works  
3. **Adapt** it to your specific home automation needs
4. **Experiment** with IoT integrations, custom commands, and automation workflows
5. **Share** your innovations with the community (but maintain your own fork)

This is how open source should work - take what's useful, make it better for your needs, and contribute back to the ecosystem by sharing your innovations!

## 📄 License

This project is open source and available under the [MIT License](LICENSE).

## 🆕 Latest Features

### Real-Time Thinking Process Visibility (v2.7.0)

Watch Claude work in real-time with transparent tool execution display:

#### Key Features
- **Live Tool Updates**: See Claude's progress as it reads files, runs commands, and writes code
- **Opt-In Activation**: Enable with `FEATURES=THINKING_PROCESS` environment variable
- **Backwards Compatible**: Existing behavior unchanged - traditional "🤔 Thinking..." unless enabled
- **Rich Progress Display**: Custom emojis and descriptions for each tool type (🔍 Read, ⚙️ Bash, ✏️ Write, etc.)
- **Non-Blocking Performance**: Real-time updates don't slow down Claude's execution

#### User Experience Transformation
**Before (Default):**
```
🤔 Thinking...
[Final response appears]
```

**After (With THINKING_PROCESS enabled):**
```
🤔 Claude is thinking...
🔍 Reading package.json...
⚙️ Running npm install...
✏️ Writing src/components/Button.tsx...
🔍 Reading existing test files...
✏️ Writing tests/Button.test.tsx...
[Complete response with full context]
```

#### Setup
Add to your `.env` file:
```bash
# Enable real-time thinking process visibility
FEATURES=THINKING_PROCESS
```

No other configuration needed! The feature integrates seamlessly with Claude Code CLI's streaming JSON output.

### Image Processing Support (v2.1.0)

Upload and analyze images directly in Slack conversations:

#### Key Features
- **Multi-format Support**: JPEG, PNG, GIF, and WebP image analysis
- **Automatic Processing**: Upload images and Claude analyzes them automatically
- **Natural Integration**: Combine image analysis with text conversations
- **Smart Storage**: Temporary storage with automatic cleanup (2-hour retention)
- **File Security**: Size limits (50MB), format validation, and safe handling

#### Usage
Simply upload an image to any channel where the bot is active:
- Drop an image file into the chat
- Add optional text: "What's in this image? Can you explain the architecture?"
- Claude will download, analyze, and respond with detailed insights

#### Requirements
- Slack bot requires `files:read` OAuth scope for image downloads
- Automatic storage directory creation at `/tmp/claude-slack-images/`
- Background cleanup service runs every 30 minutes

### PostgreSQL Session Persistence (v2.0.0)

Enhanced session management with database-backed persistence:

#### Key Features
- **Session Persistence**: Sessions survive service restarts
- **Conversation Chains**: Complete conversation history recording
- **Database-Backed Storage**: PostgreSQL integration with Docker Compose
- **O(1) Performance**: Optimized memory loading for fast session access
- **Migration Ready**: Easy upgrade from memory-based to database storage

#### Database Integration
- **3-Table Design**: sessions, child_sessions, slack_channels
- **Conversation Trees**: Complete conversation chain recording  
- **Session Branching**: Support for conversation branching and switching
- **Performance Optimized**: Single query loads entire conversation tree

#### Setup
```bash
# Enable database persistence
ENABLE_DATABASE_PERSISTENCE=true

# Start PostgreSQL
docker compose up -d postgres

# Run migrations and start service  
./scripts/redeploy.sh
```

### Enhanced Session Management (v2.1.0)

Interactive and stateful session management with database integration:

#### New Features
- **Interactive Session Listing**: `/session` shows available sessions with timestamps and paths
- **Path Suggestions**: Smart path suggestions based on previous sessions
- **Default Path Suggestion**: Uses the configured `$WORKING_DIRECTORY` when no sessions exist
- **Path-based Session Management**: Switch to or create sessions for specific paths with `/session . <path>`
- **Session Creation with Paths**: Create new sessions with specific working directories
- **Session History**: Browse and resume from previous conversations

#### Enhanced Commands
- `/session` - Interactive session browser with available sessions and suggested paths
- `/session list` - Organized session listing grouped by working directory
- `/session new <path>` - Start new conversation in specific directory
- `/session . <path>` - Switch to or create session for specific path
- Displays recent sessions with creation dates and working directories
- Shows commonly used paths for quick access
- Intelligently handles path-session relationships (sessions are tied to their workspace directories)
- Offers session selection when multiple sessions exist for the same path

### Multi-Session Support (Enhanced)
- **Concurrent Sessions**: Run multiple Claude conversations simultaneously
- **Per-Session Modes**: Each session maintains independent permission settings
- **Database Persistence**: Sessions persist across service restarts
- **Session Management**: Advanced switching and branching capabilities

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

## 🚧 Development Status & Roadmap

### **Current State: Alpha/Beta**
- **Functionality**: Core features working but with known bugs
- **Stability**: Expect breaking changes and system instability  
- **Testing**: Limited test coverage - manual testing only
- **Production Readiness**: **NOT READY** for production use

### **Path to Stability**
- **Phase 1**: Bug fixes and stability improvements (current focus)
- **Phase 2**: Implement Test-Driven Development (TDD) practices
- **Phase 3**: Comprehensive test suite covering all features
- **Phase 4**: Production-ready release with stability guarantees

### **TDD Implementation Priority**
The project will be considered **stable and production-ready** when:
- ✅ Full TDD practices are implemented
- ✅ Comprehensive test coverage (unit, integration, end-to-end)
- ✅ CI/CD pipeline with automated testing
- ✅ Feature additions don't break existing functionality
- ✅ Regression testing for all critical paths

**Until TDD is fully integrated, expect bugs and breaking changes with each update.**

## 🆘 Support & Contributing

**For Developers & Early Adopters:**
- Check `CLAUDE.md` for detailed feature list and current development status
- Report bugs via [GitHub Issues](https://github.com/ghabxph/claude-on-slack/issues) - **include full logs and reproduction steps**
- Join discussions in [GitHub Discussions](https://github.com/ghabxph/claude-on-slack/discussions)
- **Contributions welcome** - especially for test coverage and stability improvements!

---

Built with ❤️ for DIY enthusiasts who dream of having their own personal AI assistant - Jarvis in chat mode! 🤖🏠
