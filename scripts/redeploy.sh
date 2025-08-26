#!/bin/bash

set -e  # Exit on any error

echo "🚀 Starting claude-on-slack redeploy v2.0.0..."

# Get current directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_DIR"

# 1. Stop current service
echo "Stopping current service..."
pkill -f slack-claude-bot || true
sleep 2

# 2. Build new binary
echo "Building new binary..."
BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
/usr/local/go/bin/go build -ldflags "-X github.com/ghabxph/claude-on-slack/internal/version.BuildTime=$BUILD_TIME" -o slack-claude-bot ./cmd/slack-claude-bot

# 3. Start Docker Compose services
echo "Starting PostgreSQL..."
docker-compose up -d postgres

# 4. Wait for database readiness
echo "Waiting for database to be ready..."
for i in {1..30}; do
    if docker-compose exec -T postgres pg_isready -U claude_bot -d claude_slack; then
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
    docker-compose exec -T postgres psql -U claude_bot -d claude_slack -f /host_migrations/001_initial_schema.sql || true
    docker-compose exec -T postgres psql -U claude_bot -d claude_slack -f /host_migrations/002_indexes.sql || true
    docker-compose exec -T postgres psql -U claude_bot -d claude_slack -f /host_migrations/003_initial_data.sql || true
fi

# 6. Start new service in background
echo "Starting new service..."
nohup ./slack-claude-bot > slack-bot.log 2>&1 &

# 7. Wait for service startup
echo "Waiting for service to start..."
sleep 5

# 8. Verify service health
echo "Verifying service health..."
if curl -f http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Deployment completed successfully!"
    echo "Service is running and healthy"
else
    echo "❌ Health check failed!"
    echo "Check logs: tail -f slack-bot.log"
    exit 1
fi

echo ""
echo "📊 Service status:"
curl -s http://localhost:8080/health | head -c 200
echo ""