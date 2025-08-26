#!/bin/bash

# Set test environment variables for edge proxy
export TEST_REDIS_URL="redis://localhost:6379/2"
export CONTROL_PLANE_URL="http://localhost:8080"
export REGION="test-region"
export LOG_LEVEL="debug"

# Navigate to edge-proxy directory
cd edge-proxy

# Run the integration tests
echo "Running edge proxy integration tests..."
go test -v ./tests/...
