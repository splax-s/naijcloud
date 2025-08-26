#!/bin/bash

# Test runner script for NaijCloud Control Plane Integration Tests

set -e

echo "🧪 NaijCloud Control Plane Integration Tests"
echo "============================================"

# Check if required services are running
echo "📋 Checking prerequisites..."

# Check PostgreSQL
if ! docker-compose exec -T postgres pg_isready -U naijcloud -d naijcloud >/dev/null 2>&1; then
    echo "❌ PostgreSQL is not ready. Please start services with: docker-compose up -d"
    exit 1
fi

# Check Redis
if ! docker-compose exec -T redis redis-cli ping >/dev/null 2>&1; then
    echo "❌ Redis is not ready. Please start services with: docker-compose up -d"
    exit 1
fi

echo "✅ All prerequisites are ready"

# Set test environment variables
export TEST_DATABASE_URL="postgres://naijcloud:naijcloud@localhost:5433/naijcloud?sslmode=disable"
export TEST_REDIS_URL="redis://localhost:6379/1"

# Create test database if it doesn't exist
echo "🗄️  Setting up test database..."
docker-compose exec -T postgres psql -U naijcloud -d postgres -c "CREATE DATABASE naijcloud_test;" 2>/dev/null || true

# Run migrations on test database
echo "🔄 Running database migrations..."
export DATABASE_URL="postgres://naijcloud:naijcloud@localhost:5433/naijcloud_test?sslmode=disable"
cd control-plane

# Apply schema to test database
docker-compose exec -T postgres psql -U naijcloud -d naijcloud_test -f /docker-entrypoint-initdb.d/init.sql >/dev/null 2>&1 || true

echo "🚀 Running integration tests..."

# Run the tests
go test -v ./tests/... -timeout 30s

echo ""
echo "✅ Integration tests completed successfully!"
echo ""
echo "📊 Test Coverage:"
go test -v ./tests/... -cover

echo ""
echo "🧹 Cleaning up test database..."
docker-compose exec -T postgres psql -U naijcloud -d postgres -c "DROP DATABASE IF EXISTS naijcloud_test;" >/dev/null 2>&1

echo "✨ Test run complete!"
