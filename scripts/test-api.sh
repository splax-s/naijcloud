#!/bin/bash
set -e

echo "🧪 Running NaijCloud D-CDN API tests..."

BASE_URL="http://localhost:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Helper function to run tests
run_test() {
    local test_name="$1"
    local command="$2"
    local expected_code="$3"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  ${test_name}... "
    
    response=$(eval "$command" 2>&1)
    exit_code=$?
    
    if [ $exit_code -eq 0 ] && [ -z "$expected_code" ]; then
        echo -e "${GREEN}✓${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    elif [ $exit_code -eq "$expected_code" ]; then
        echo -e "${GREEN}✓${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC}"
        echo "    Expected exit code: $expected_code, Got: $exit_code"
        echo "    Response: $response"
    fi
}

# Check if control plane is running
echo "🔍 Checking if control plane is available..."
if ! curl -s -f "$BASE_URL/health" > /dev/null; then
    echo -e "${RED}❌ Control plane is not running at $BASE_URL${NC}"
    echo "   Run: ./scripts/dev-setup.sh first"
    exit 1
fi

echo -e "${GREEN}✅ Control plane is available${NC}"
echo ""

# Health check test
echo "🏥 Health Check Tests"
run_test "GET /health" "curl -s -f $BASE_URL/health"

echo ""

# Domain management tests
echo "🌐 Domain Management Tests"

# Create a test domain
DOMAIN_NAME="test-$(date +%s).example.com"
CREATE_DOMAIN_DATA="{\"domain\":\"$DOMAIN_NAME\",\"origin_url\":\"https://httpbin.org\",\"cache_ttl\":7200}"

run_test "POST /v1/domains (create domain)" "curl -s -f -X POST $BASE_URL/v1/domains -H 'Content-Type: application/json' -d '$CREATE_DOMAIN_DATA'"

run_test "GET /v1/domains (list domains)" "curl -s -f $BASE_URL/v1/domains"

run_test "GET /v1/domains/$DOMAIN_NAME (get specific domain)" "curl -s -f $BASE_URL/v1/domains/$DOMAIN_NAME"

# Update domain
UPDATE_DOMAIN_DATA="{\"cache_ttl\":3600,\"rate_limit\":2000}"
run_test "PUT /v1/domains/$DOMAIN_NAME (update domain)" "curl -s -f -X PUT $BASE_URL/v1/domains/$DOMAIN_NAME -H 'Content-Type: application/json' -d '$UPDATE_DOMAIN_DATA'"

# Test cache purge
PURGE_DATA="{\"paths\":[\"/\",\"/api/*\"]}"
run_test "POST /v1/domains/$DOMAIN_NAME/purge (purge cache)" "curl -s -f -X POST $BASE_URL/v1/domains/$DOMAIN_NAME/purge -H 'Content-Type: application/json' -d '$PURGE_DATA'"

echo ""

# Edge management tests
echo "🌍 Edge Management Tests"

# Register an edge node
EDGE_DATA="{\"region\":\"test-region\",\"ip_address\":\"10.0.1.100\",\"hostname\":\"test-edge-01\",\"capacity\":1000}"
EDGE_RESPONSE=$(curl -s -f -X POST $BASE_URL/v1/edges -H 'Content-Type: application/json' -d "$EDGE_DATA")
EDGE_ID=$(echo "$EDGE_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

run_test "POST /v1/edges (register edge)" "echo '$EDGE_RESPONSE' | grep -q id"

run_test "GET /v1/edges (list edges)" "curl -s -f $BASE_URL/v1/edges"

if [ -n "$EDGE_ID" ]; then
    run_test "GET /v1/edges/$EDGE_ID (get specific edge)" "curl -s -f $BASE_URL/v1/edges/$EDGE_ID"
    
    # Send heartbeat
    HEARTBEAT_DATA="{\"status\":\"healthy\",\"metrics\":{\"requests_per_second\":100,\"cache_hit_ratio\":0.85}}"
    run_test "POST /v1/edges/$EDGE_ID/heartbeat (send heartbeat)" "curl -s -f -X POST $BASE_URL/v1/edges/$EDGE_ID/heartbeat -H 'Content-Type: application/json' -d '$HEARTBEAT_DATA'"
    
    run_test "GET /v1/edges/$EDGE_ID/purges (get pending purges)" "curl -s -f $BASE_URL/v1/edges/$EDGE_ID/purges"
fi

echo ""

# Analytics tests
echo "📊 Analytics Tests"

run_test "GET /v1/analytics/domains/$DOMAIN_NAME (get analytics)" "curl -s -f $BASE_URL/v1/analytics/domains/$DOMAIN_NAME"

run_test "GET /v1/analytics/domains/$DOMAIN_NAME/paths (get top paths)" "curl -s -f $BASE_URL/v1/analytics/domains/$DOMAIN_NAME/paths"

run_test "GET /v1/analytics/domains/$DOMAIN_NAME/timeline (get timeline)" "curl -s -f '$BASE_URL/v1/analytics/domains/$DOMAIN_NAME/timeline?interval=1%20hour'"

echo ""

# Metrics tests
echo "📈 Metrics Tests"

run_test "GET /metrics (Prometheus metrics)" "curl -s -f http://localhost:9091/metrics"

echo ""

# Error handling tests
echo "❌ Error Handling Tests"

run_test "GET /v1/domains/nonexistent.com (404 error)" "curl -s -f $BASE_URL/v1/domains/nonexistent.com" 22

run_test "POST /v1/domains (duplicate domain)" "curl -s -f -X POST $BASE_URL/v1/domains -H 'Content-Type: application/json' -d '$CREATE_DOMAIN_DATA'" 22

run_test "GET /v1/edges/invalid-uuid (bad request)" "curl -s -f $BASE_URL/v1/edges/invalid-uuid" 22

echo ""

# Cleanup
echo "🧹 Cleanup"
if [ -n "$DOMAIN_NAME" ]; then
    run_test "DELETE /v1/domains/$DOMAIN_NAME (cleanup domain)" "curl -s -f -X DELETE $BASE_URL/v1/domains/$DOMAIN_NAME"
fi

if [ -n "$EDGE_ID" ]; then
    run_test "DELETE /v1/edges/$EDGE_ID (cleanup edge)" "curl -s -f -X DELETE $BASE_URL/v1/edges/$EDGE_ID"
fi

echo ""

# Summary
echo "📋 Test Summary"
echo "  Tests run: $TESTS_RUN"
echo "  Tests passed: $TESTS_PASSED"
echo "  Tests failed: $((TESTS_RUN - TESTS_PASSED))"

if [ $TESTS_PASSED -eq $TESTS_RUN ]; then
    echo -e "  ${GREEN}🎉 All tests passed!${NC}"
    exit 0
else
    echo -e "  ${RED}❌ Some tests failed${NC}"
    exit 1
fi
