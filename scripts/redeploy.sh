#!/bin/bash

set -e  # Exit on any error

echo "🚀 Starting claude-on-slack redeploy v2.6.1..."

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

# Update systemd service configuration
echo "Updating systemd service configuration..."
sudo sed -i 's|EnvironmentFile=.*|EnvironmentFile=-/opt/claude-on-slack/.env|g' /etc/systemd/system/claude-on-slack.service
sudo sed -i 's|Restart=always|Restart=no|g' /etc/systemd/system/claude-on-slack.service
sudo sed -i '/^RestartSec=/d' /etc/systemd/system/claude-on-slack.service
sudo systemctl daemon-reload
echo "✅ Systemd service updated:"
echo "  • Environment: /opt/claude-on-slack/.env"
echo "  • Restart policy: disabled (app handles restarts)"

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
if [ -d "migrations" ]; then
    # Run all migration files in sequential order (001, 002, 003, etc.)
    for migration_file in migrations/*.sql; do
        if [ -f "$migration_file" ]; then
            migration_name=$(basename "$migration_file")
            echo "Running migration: $migration_name"
            $DOCKER_COMPOSE exec -T postgres psql -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack} -f "/host_migrations/$migration_name" || true
        fi
    done
    echo "All migrations completed"
    
    # Wait for database to be ready again after migrations
    echo "Verifying database readiness after migrations..."
    for i in {1..15}; do
        if $DOCKER_COMPOSE exec -T postgres pg_isready -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack} >/dev/null 2>&1; then
            echo "Database confirmed ready after migrations!"
            break
        fi
        if [ $i -eq 15 ]; then
            echo "Database not ready after migrations - waiting 5 more seconds"
            sleep 5
        fi
        sleep 2
    done
else
    echo "No migrations directory found"
fi

# 6. Final database connection test before starting services
echo "Final database connection test..."
for i in {1..10}; do
    if $DOCKER_COMPOSE exec -T postgres psql -U ${DB_USER:-claude_bot} -d ${DB_NAME:-claude_slack} -c "SELECT 1;" >/dev/null 2>&1; then
        echo "Database connection test successful!"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Database connection test failed after 10 attempts"
        echo "Check PostgreSQL logs: docker compose logs postgres"
        exit 1
    fi
    echo "Database connection attempt $i/10 failed, retrying..."
    sleep 3
done

# 7. Start services back up
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

# Note: Legacy systemd service creation code has been removed.
# The service now uses /opt/claude-on-slack/ for deployment.