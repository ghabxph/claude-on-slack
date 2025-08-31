#!/bin/bash

set -e  # Exit on any error

echo "🚀 Starting claude-on-slack redeploy v2.5.0..."

# Configuration
SERVICE_NAME="slack-claude-bot"
# Use current user if not specified
CURRENT_USER="${CLAUDE_SERVICE_USER:-$(logname 2>/dev/null || whoami)}"

# Get current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

echo "📋 Deployment Info:"
echo "• Service User: $CURRENT_USER"
echo "• Project Dir: $PROJECT_DIR"
echo "• Working Dir: /home/$CURRENT_USER"
echo ""

# 1. Build new binary (ALWAYS compile)
echo "Building new binary..."
echo "Working directory: $(pwd)"
echo "Go version: $(/usr/local/go/bin/go version)"

# Build to temporary location first (so running binary isn't locked)
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo "Building with timestamp: $BUILD_TIME"
TEMP_BINARY="slack-claude-bot.new"

if ! /usr/local/go/bin/go build -v -ldflags "-X github.com/ghabxph/claude-on-slack/internal/version.BuildTime=$BUILD_TIME" -o "$TEMP_BINARY" ./cmd/slack-claude-bot; then
    echo "❌ Build failed! Stopping deployment."
    exit 1
fi

# Verify binary was created and show info
if [ ! -f "$TEMP_BINARY" ]; then
    echo "❌ Binary $TEMP_BINARY was not created! Stopping deployment."
    exit 1
fi

echo "✅ Binary built successfully:"
ls -la "$TEMP_BINARY"
echo "SHA256: $(sha256sum "$TEMP_BINARY")"

# 2. Stop current services (only after successful build)
echo "Stopping claude-on-slack services..."
sudo systemctl stop claude-on-slack.service || true

# Only stop tunnel service if it exists
if systemctl list-units --full -all | grep -Fq "claude-on-slack-tunnel.service"; then
    echo "Stopping claude-on-slack-tunnel.service..."
    sudo systemctl stop claude-on-slack-tunnel.service || true
fi

sleep 2

# 3. Deploy binary and environment file to /opt/claude-on-slack/
echo "Deploying binary to /opt/claude-on-slack/claude-on-slack..."
sudo cp "$TEMP_BINARY" /opt/claude-on-slack/claude-on-slack
sudo chmod +x /opt/claude-on-slack/claude-on-slack
rm -f "$TEMP_BINARY"

echo "Deploying .env file to /opt/claude-on-slack/.env..."
if [ -f ".env" ]; then
    sudo cp .env /opt/claude-on-slack/.env
    sudo chmod 600 /opt/claude-on-slack/.env
    echo "✅ Environment file deployed"
else
    echo "⚠️  Warning: .env file not found in project directory"
fi

echo "✅ Binary deployed successfully:"
ls -la /opt/claude-on-slack/claude-on-slack
echo "SHA256: $(sudo sha256sum /opt/claude-on-slack/claude-on-slack)"

# Update systemd service to use the secure environment file path
echo "Updating systemd service environment file path..."
sudo sed -i 's|EnvironmentFile=.*|EnvironmentFile=-/opt/claude-on-slack/.env|g' /etc/systemd/system/claude-on-slack.service
sudo systemctl daemon-reload
echo "✅ Systemd service updated to use /opt/claude-on-slack/.env"

# Determine docker compose command
if command -v docker-compose >/dev/null 2>&1; then
    DOCKER_COMPOSE="docker-compose"
else
    DOCKER_COMPOSE="docker compose"
fi

# 3. Start Docker Compose services
echo "Starting PostgreSQL..."
$DOCKER_COMPOSE up -d postgres

# 4. Wait for database readiness
echo "Waiting for database to be ready..."
# Extract database credentials from .env (safer than sourcing)
if [ -f ".env" ]; then
    DB_USER=$(grep "^DB_USER=" .env | cut -d'=' -f2)
    DB_NAME=$(grep "^DB_NAME=" .env | cut -d'=' -f2)
    DB_PASSWORD=$(grep "^DB_PASSWORD=" .env | cut -d'=' -f2)
fi
for i in {1..30}; do
    if $DOCKER_COMPOSE exec -T postgres pg_isready -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack}; then
        echo "Database is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Database failed to become ready after 30 attempts"
        exit 1
    fi
    sleep 2
done

# 5. Run database migrations
echo "Running database migrations..."
if [ -f "migrations/001_initial_schema.sql" ]; then
    $DOCKER_COMPOSE exec -T postgres psql -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack} -f /host_migrations/001_initial_schema.sql || true
    $DOCKER_COMPOSE exec -T postgres psql -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack} -f /host_migrations/002_indexes.sql || true
    $DOCKER_COMPOSE exec -T postgres psql -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack} -f /host_migrations/003_initial_data.sql || true
fi

# 4. Start services back up
echo "Starting claude-on-slack services..."
sudo systemctl start claude-on-slack.service

# Only start tunnel service if it exists
if systemctl list-units --full -all | grep -Fq "claude-on-slack-tunnel.service"; then
    echo "Starting claude-on-slack-tunnel.service..."
    sudo systemctl start claude-on-slack-tunnel.service
fi

# 5. Wait for service startup and verify
echo "Waiting for services to start..."
sleep 5

# Check service status
if sudo systemctl is-active --quiet claude-on-slack.service; then
	echo "✅ claude-on-slack.service is running"
else
	echo "❌ claude-on-slack.service failed to start!"
	echo "Check status: sudo systemctl status claude-on-slack.service"
	echo "Check logs: sudo journalctl -u claude-on-slack.service -f"
	exit 1
fi

# Only check tunnel service if it exists
if systemctl list-units --full -all | grep -Fq "claude-on-slack-tunnel.service"; then
	if sudo systemctl is-active --quiet claude-on-slack-tunnel.service; then
		echo "✅ claude-on-slack-tunnel.service is running"
	else
		echo "❌ claude-on-slack-tunnel.service failed to start!"
		echo "Check status: sudo systemctl status claude-on-slack-tunnel.service"
		echo "Check logs: sudo journalctl -u claude-on-slack-tunnel.service -f"
		exit 1
	fi
fi

echo ""
echo "✅ Deployment completed successfully!"
echo ""
echo "📋 Service Management:"
echo "• Status: sudo systemctl status claude-on-slack.service"
echo "• Logs: sudo journalctl -u claude-on-slack.service -f"
echo "• Stop: sudo systemctl stop claude-on-slack.service"
echo "• Restart: sudo systemctl restart claude-on-slack.service"

# Skip old systemd service creation
if false; then
echo "Updating systemd service file..."
sudo tee "/etc/systemd/system/$SERVICE_NAME.service" > /dev/null << EOF
[Unit]
Description=Claude on Slack Bot (Auto-deployed)
Documentation=https://github.com/ghabxph/claude-on-slack
After=network.target
Wants=network.target

[Service]
Type=simple
User=$CURRENT_USER
Group=$CURRENT_USER
WorkingDirectory=/home/$CURRENT_USER/files/projects/ghabxph/claude-on-slack
ExecStart=/home/$CURRENT_USER/files/projects/ghabxph/claude-on-slack/slack-claude-bot
EnvironmentFile=-/home/$CURRENT_USER/files/projects/ghabxph/claude-on-slack/.env
Restart=always
RestartSec=10

# Minimal security settings for personal use
NoNewPrivileges=yes
PrivateTmp=no

# Full file system access for personal bot
# ProtectSystem=false (commented out for full access)
# ProtectHome=false (commented out for home access)

# Resource limits
LimitNOFILE=65536
MemoryMax=1G

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and start service
echo "Reloading systemd configuration..."
sudo systemctl daemon-reload

echo "Starting systemd service..."
sudo systemctl enable $SERVICE_NAME || true
sudo systemctl start $SERVICE_NAME

# 7. Wait for service startup and verify
echo "Waiting for service to start..."
sleep 5

# Check systemd service status
if sudo systemctl is-active --quiet $SERVICE_NAME; then
    echo "✅ SystemD service is running"
else
    echo "❌ SystemD service failed to start!"
    echo "Check status: sudo systemctl status $SERVICE_NAME"
    echo "Check logs: sudo journalctl -u $SERVICE_NAME -f"
    exit 1
fi

# 8. Verify service health endpoint
echo "Verifying service health..."
HEALTH_PORT="${SERVER_PORT:-8080}"
for i in {1..10}; do
    if curl -f http://localhost:$HEALTH_PORT/health > /dev/null 2>&1; then
        echo "✅ Health check passed!"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Health check failed after 10 attempts!"
        echo "Check logs: sudo journalctl -u $SERVICE_NAME -f"
        exit 1
    fi
    echo "⏳ Waiting for health endpoint... (attempt $i/10)"
    sleep 2
done

echo ""
echo "📊 Final Status:"
echo "• SystemD Service: $(sudo systemctl is-active $SERVICE_NAME)"
echo "• Health Check: $(curl -s http://localhost:$HEALTH_PORT/health | head -c 100)"
echo ""
echo "✅ Deployment completed successfully!"
echo ""
echo "📋 Service Management:"
echo "• Status: sudo systemctl status $SERVICE_NAME"
echo "• Logs: sudo journalctl -u $SERVICE_NAME -f"
echo "• Stop: sudo systemctl stop $SERVICE_NAME"
echo "• Restart: sudo systemctl restart claude-on-slack.service"
fi