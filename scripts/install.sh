#!/bin/bash

# claude-on-slack Installation Script
# Installs the claude-on-slack bot as a systemd service
#
# Environment Variables:
#   CLAUDE_SERVICE_USER - User to run service as (default: current user)
#   CLAUDE_PROJECT_DIR  - Directory to install project in (default: /home/user/claude-on-slack)
#
# Usage:
#   ./install.sh
#   CLAUDE_PROJECT_DIR=/opt/my-claude ./install.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="slack-claude-bot"
# Default to current user if not specified
SERVICE_USER="${CLAUDE_SERVICE_USER:-$(logname 2>/dev/null || whoami)}"
INSTALL_DIR="/opt/claude-on-slack"
PROJECT_DIR="${CLAUDE_PROJECT_DIR:-/home/$SERVICE_USER/claude-on-slack}"  # Configurable project directory
CONFIG_DIR="/home/$SERVICE_USER/.config/claude-on-slack"  # User config directory
LOG_DIR="/home/$SERVICE_USER/.local/share/claude-on-slack/logs"  # User log directory  
WORK_DIR="/home/$SERVICE_USER"  # User's home directory

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    print_status "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21+ first."
        exit 1
    fi
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_success "Go version: $GO_VERSION"
    
    # Check if Claude Code is available
    if ! command -v claude &> /dev/null; then
        print_error "Claude Code CLI is not installed or not in PATH."
        print_error "Please install Claude Code first: https://github.com/anthropics/claude-code"
        exit 1
    fi
    
    print_success "Claude Code CLI detected: $(which claude)"
    
    # Check if systemd is available
    if ! command -v systemctl &> /dev/null; then
        print_error "systemd is not available on this system"
        exit 1
    fi
}

# Check service user exists
check_service_user() {
    print_status "Checking service user '$SERVICE_USER'..."
    
    if id "$SERVICE_USER" &>/dev/null; then
        print_success "User '$SERVICE_USER' exists"
    else
        print_error "User '$SERVICE_USER' does not exist"
        print_error "Please create the user first or change SERVICE_USER in this script"
        exit 1
    fi
}

# Create directories
create_directories() {
    print_status "Creating directories..."
    
    # Create installation directory
    mkdir -p "$INSTALL_DIR"
    chown root:root "$INSTALL_DIR"
    chmod 755 "$INSTALL_DIR"
    
    # Create configuration directory in user's home
    sudo -u $SERVICE_USER mkdir -p "$CONFIG_DIR"
    chown $SERVICE_USER:$SERVICE_USER "$CONFIG_DIR"
    chmod 755 "$CONFIG_DIR"
    
    # Create log directory in user's home
    sudo -u $SERVICE_USER mkdir -p "$LOG_DIR"
    chown $SERVICE_USER:$SERVICE_USER "$LOG_DIR"
    chmod 755 "$LOG_DIR"
    
    # Ensure working directory (user's home) has correct permissions
    chown $SERVICE_USER:$SERVICE_USER "$WORK_DIR"
    chmod 755 "$WORK_DIR"
    
    print_success "Created directories"
}

# Build the application
build_application() {
    print_status "Building claude-on-slack application..."
    
    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
    
    cd "$PROJECT_DIR"
    
    # Build the application
    print_status "Running: go mod tidy"
    go mod tidy
    
    print_status "Running: go build -o $PROJECT_DIR/$SERVICE_NAME ./cmd/slack-claude-bot"
    go build -o "$PROJECT_DIR/$SERVICE_NAME" ./cmd/slack-claude-bot
    
    # Set permissions
    chown $SERVICE_USER:$SERVICE_USER "$PROJECT_DIR/$SERVICE_NAME"
    chmod 755 "$PROJECT_DIR/$SERVICE_NAME"
    
    print_success "Built and installed application binary"
}

# Copy configuration files
copy_configuration() {
    print_status "Setting up configuration files..."
    
    # Get the directory where this script is located
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
    
    # Copy .env.example to config directory
    if [[ -f "$PROJECT_DIR/.env.example" ]]; then
        sudo -u $SERVICE_USER cp "$PROJECT_DIR/.env.example" "$CONFIG_DIR/claude-on-slack.env.example"
        chown $SERVICE_USER:$SERVICE_USER "$CONFIG_DIR/claude-on-slack.env.example"
        chmod 640 "$CONFIG_DIR/claude-on-slack.env.example"
        print_success "Copied .env.example to $CONFIG_DIR/"
    fi
    
    # Check if .env file exists in project directory and copy it
    if [[ -f "$PROJECT_DIR/.env" ]]; then
        print_status "Found existing .env file in project directory, copying it..."
        sudo -u $SERVICE_USER cp "$PROJECT_DIR/.env" "$PROJECT_DIR/.env"
        chown $SERVICE_USER:$SERVICE_USER "$PROJECT_DIR/.env"
        chmod 640 "$PROJECT_DIR/.env"
        print_success "Copied existing .env to $PROJECT_DIR/.env"
    # Create default environment file if it doesn't exist and no .env was copied
    elif [[ ! -f "$PROJECT_DIR/.env" ]]; then
        print_status "Creating default environment file..."
        sudo -u $SERVICE_USER tee "$PROJECT_DIR/.env" > /dev/null << 'EOF'
# claude-on-slack Configuration
# Copy from claude-on-slack.env.example and customize

# Slack Configuration (Required)
SLACK_BOT_TOKEN=
SLACK_APP_TOKEN=
SLACK_SIGNING_SECRET=

# Claude Code CLI Configuration
CLAUDE_CODE_PATH=claude
CLAUDE_TIMEOUT=5m

# Claude Code Tool Configuration
# Empty = all tools available for full system access
ALLOWED_TOOLS=
DISALLOWED_TOOLS=

# Bot Configuration
BOT_NAME=claude-bot
BOT_DISPLAY_NAME=Claude Bot
COMMAND_PREFIX=!claude

# User Access Control
ALLOWED_USERS=
ADMIN_USERS=

# Security & Rate Limiting
ENABLE_AUTH=true
RATE_LIMIT_PER_MINUTE=20
MAX_MESSAGE_LENGTH=4000

# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
HEALTH_CHECK_PATH=/health

# Working Directory & Commands  
# Full system access - set to your home directory
WORKING_DIRECTORY=/home/$SERVICE_USER
COMMAND_TIMEOUT=10m
MAX_OUTPUT_LENGTH=50000
# Minimal restrictions for personal use
BLOCKED_COMMANDS=

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
ENABLE_DEBUG=false
EOF
        chown $SERVICE_USER:$SERVICE_USER "$PROJECT_DIR/.env"
        chmod 640 "$PROJECT_DIR/.env"
        print_success "Created default environment file"
    else
        print_warning "Environment file already exists at $PROJECT_DIR/.env"
    fi
}

# Install systemd service
install_systemd_service() {
    print_status "Installing systemd service..."
    
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=Claude on Slack Bot
Documentation=https://github.com/ghabxph/claude-on-slack
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER

# Security settings (minimal for personal use with full system access)
NoNewPrivileges=yes
PrivateTmp=no

# Full file system access for personal bot
# ProtectSystem=false (commented out for full access)
# ProtectHome=false (commented out for home access)

# Process settings
Restart=always
RestartSec=10
TimeoutStartSec=30
TimeoutStopSec=30

# Resource limits
LimitNOFILE=65536
MemoryMax=1G

# Environment
Environment=HOME=$WORK_DIR
EnvironmentFile=$PROJECT_DIR/.env
WorkingDirectory=$PROJECT_DIR

# Command
ExecStart=$PROJECT_DIR/slack-claude-bot
ExecReload=/bin/kill -HUP \$MAINPID

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

[Install]
WantedBy=multi-user.target
EOF

    # Set permissions
    chown root:root "/etc/systemd/system/$SERVICE_NAME.service"
    chmod 644 "/etc/systemd/system/$SERVICE_NAME.service"
    
    # Reload systemd
    systemctl daemon-reload
    
    print_success "Installed systemd service"
}

# Create workspace directory
create_workspace() {
    print_status "Creating workspace directory..."
    
    mkdir -p "$WORK_DIR/workspace"
    chown -R $SERVICE_USER:$SERVICE_USER "$WORK_DIR/workspace"
    chmod 755 "$WORK_DIR/workspace"
    
    print_success "Created workspace directory"
}

# Show post-install instructions
show_instructions() {
    print_success "Installation completed successfully!"
    echo
    print_status "Next steps:"
    echo "1. Edit the configuration file:"
    echo "   nano $PROJECT_DIR/.env"
    echo
    echo "2. Add your Slack tokens and configure user access:"
    echo "   SLACK_BOT_TOKEN=xoxb-your-token"
    echo "   SLACK_APP_TOKEN=xapp-your-token"  
    echo "   SLACK_SIGNING_SECRET=your-secret"
    echo "   ALLOWED_USERS=your-email@domain.com"
    echo
    echo "3. Enable and start the service:"
    echo "   sudo systemctl enable $SERVICE_NAME"
    echo "   sudo systemctl start $SERVICE_NAME"
    echo
    echo "4. Check service status:"
    echo "   sudo systemctl status $SERVICE_NAME"
    echo
    echo "5. View logs:"
    echo "   sudo journalctl -u $SERVICE_NAME -f"
    echo
    echo "6. Test health endpoint:"
    echo "   curl http://localhost:8080/health"
    echo
    print_status "Service files located at:"
    echo "- Binary: $PROJECT_DIR/$SERVICE_NAME"
    echo "- Config: $PROJECT_DIR/.env"
    echo "- Logs: $LOG_DIR/ (also in journalctl)"
    echo "- Working Dir: $WORK_DIR ($SERVICE_USER's home)"
    echo "- Service: /etc/systemd/system/$SERVICE_NAME.service"
    echo
    print_status "Security Note:"
    echo "- Service runs as user '$SERVICE_USER' (not root)"
    echo "- For root operations, handle manually via remote access"
    echo "- Claude has same permissions as user '$SERVICE_USER'"
}

# Main installation function
main() {
    print_status "Starting claude-on-slack installation..."
    echo
    
    check_root
    check_prerequisites
    check_service_user
    create_directories
    build_application
    copy_configuration
    install_systemd_service
    create_workspace
    
    echo
    show_instructions
}

# Run main function
main "$@"