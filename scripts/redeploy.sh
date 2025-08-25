#!/bin/bash

# claude-on-slack Redeploy Script
# Stops services, deploys new binary, and restarts everything

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="claude-on-slack"
TUNNEL_SERVICE="claude-on-slack-tunnel"  # Adjust this to your actual tunnel service name
BINARY_NAME="slack-claude-bot"
INSTALL_DIR="/opt/claude-on-slack"

# Slack notification settings (read from bot config)
CONFIG_FILE="/home/$(logname)/.config/claude-on-slack/claude-on-slack.env"

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Build the application
build_application() {
    print_status "Building $BINARY_NAME..."
    
    # Use full Go path
    if ! /usr/local/go/bin/go build -o "$BINARY_NAME" ./cmd/slack-claude-bot; then
        print_error "Failed to build application"
        exit 1
    fi
    
    print_success "Built $BINARY_NAME successfully"
}

# Send deployment notification to Slack
send_slack_notification() {
    print_status "Sending deployment notification..."
    
    # Read Slack bot token and channel from config
    if [[ -f "$CONFIG_FILE" ]]; then
        SLACK_BOT_TOKEN=$(grep "^SLACK_BOT_TOKEN=" "$CONFIG_FILE" | cut -d'=' -f2)
        AUTO_RESPONSE_CHANNELS=$(grep "^AUTO_RESPONSE_CHANNELS=" "$CONFIG_FILE" | cut -d'=' -f2)
        
        if [[ -n "$SLACK_BOT_TOKEN" && -n "$AUTO_RESPONSE_CHANNELS" ]]; then
            # Use the first auto-response channel
            CHANNEL_ID=$(echo "$AUTO_RESPONSE_CHANNELS" | cut -d',' -f1)
            
            # Send notification message
            curl -s -X POST "https://slack.com/api/chat.postMessage" \
                -H "Authorization: Bearer $SLACK_BOT_TOKEN" \
                -H "Content-Type: application/json" \
                -d "{
                    \"channel\": \"$CHANNEL_ID\",
                    \"text\": \"🚀 **Bot Redeployed Successfully**\\n\\n✅ New version is now running\\n⏰ $(date '+%Y-%m-%d %H:%M:%S')\\n🔄 Ready for conversations!\"
                }" > /dev/null
            
            print_success "Deployment notification sent to Slack"
        else
            print_warning "Slack token or channel not configured, skipping notification"
        fi
    else
        print_warning "Config file not found, skipping notification"
    fi
}

# Stop services
stop_services() {
    print_status "Stopping services..."
    
    # Stop Claude on Slack service
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        print_status "Stopping $SERVICE_NAME..."
        sudo systemctl stop "$SERVICE_NAME"
        print_success "Stopped $SERVICE_NAME"
    else
        print_warning "$SERVICE_NAME is not running"
    fi
    
    # Stop tunnel service (adjust service name as needed)
    if systemctl is-active --quiet "$TUNNEL_SERVICE" 2>/dev/null; then
        print_status "Stopping $TUNNEL_SERVICE..."
        sudo systemctl stop "$TUNNEL_SERVICE"
        print_success "Stopped $TUNNEL_SERVICE"
    else
        print_warning "$TUNNEL_SERVICE is not running or doesn't exist"
    fi
}

# Deploy binary
deploy_binary() {
    print_status "Deploying new binary..."
    
    # Copy binary to installation directory
    sudo cp "$BINARY_NAME" "$INSTALL_DIR/$SERVICE_NAME"
    sudo chown root:root "$INSTALL_DIR/$SERVICE_NAME"
    sudo chmod 755 "$INSTALL_DIR/$SERVICE_NAME"
    
    print_success "Deployed binary to $INSTALL_DIR/$SERVICE_NAME"
}

# Start services
start_services() {
    print_status "Starting services..."
    
    # Start Claude on Slack service first
    print_status "Starting $SERVICE_NAME..."
    sudo systemctl start "$SERVICE_NAME"
    sleep 2  # Give service time to start
    
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        print_success "Started $SERVICE_NAME"
    else
        print_error "Failed to start $SERVICE_NAME"
        print_error "Check logs: journalctl -u $SERVICE_NAME -n 20"
        exit 1
    fi
    
    # Start tunnel service last
    if systemctl list-unit-files | grep -q "$TUNNEL_SERVICE"; then
        print_status "Starting $TUNNEL_SERVICE..."
        sudo systemctl start "$TUNNEL_SERVICE"
        sleep 2  # Give tunnel time to establish
        if systemctl is-active --quiet "$TUNNEL_SERVICE"; then
            print_success "Started $TUNNEL_SERVICE"
        else
            print_error "Failed to start $TUNNEL_SERVICE"
        fi
    else
        print_warning "$TUNNEL_SERVICE service not found, skipping"
    fi
}

# Show status
show_status() {
    print_status "Service status:"
    echo
    
    # Check Claude on Slack status
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo -e "${GREEN}● $SERVICE_NAME${NC} - Active"
    else
        echo -e "${RED}● $SERVICE_NAME${NC} - Inactive"
    fi
    
    # Check tunnel status
    if systemctl is-active --quiet "$TUNNEL_SERVICE" 2>/dev/null; then
        echo -e "${GREEN}● $TUNNEL_SERVICE${NC} - Active"
    else
        echo -e "${YELLOW}● $TUNNEL_SERVICE${NC} - Inactive or not found"
    fi
    
    echo
    print_status "Endpoints:"
    echo "- Health: http://localhost:8080/health"
    echo "- Events: http://localhost:8080/slack/events"
    echo "- Metrics: http://localhost:8080/metrics"
    echo
    print_status "Logs:"
    echo "- Bot: journalctl -u $SERVICE_NAME -f"
    echo "- Tunnel: journalctl -u $TUNNEL_SERVICE -f"
}

# Main function
main() {
    print_status "Starting claude-on-slack redeploy..."
    echo
    
    build_application
    stop_services
    deploy_binary
    start_services
    
    echo
    show_status
    
    # Send Slack notification
    send_slack_notification
    
    echo
    print_success "Redeploy completed successfully!"
}

# Handle script arguments
case "${1:-}" in
    "stop")
        print_status "Stopping services only..."
        stop_services
        ;;
    "start") 
        print_status "Starting services only..."
        start_services
        show_status
        ;;
    "status")
        show_status
        ;;
    *)
        main
        ;;
esac
