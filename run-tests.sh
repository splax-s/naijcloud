#!/bin/bash

# Set test environment variables
export TEST_DATABASE_URL="postgres://naijcloud:naijcloud_pass@localhost:5433/naijcloud?sslmode=disable"
export TEST_REDIS_URL="redis://localhost:6379/1"

# Navigate to control-plane directory
cd control-plane

# Run the integration tests
echo "Running integration tests..."
go test -v ./tests/...
