#!/bin/bash

# NaijCloud Load Testing Script
# Uses k6 for performance testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if k6 is installed
if ! command -v k6 &> /dev/null; then
    print_error "k6 is not installed. Please install it first:"
    echo "  macOS: brew install k6"
    echo "  Linux: sudo apt-get install k6"
    echo "  Windows: choco install k6"
    exit 1
fi

# Configuration
CONTROL_PLANE_URL=${CONTROL_PLANE_URL:-"http://localhost:8080"}
EDGE_PROXY_URL=${EDGE_PROXY_URL:-"http://localhost:8081"}

# Create k6 test script
cat > /tmp/naijcloud_load_test.js << 'EOF'
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

const errorRate = new Rate('errors');

export let options = {
  stages: [
    { duration: '2m', target: 10 },  // Ramp up to 10 users
    { duration: '5m', target: 10 },  // Stay at 10 users
    { duration: '2m', target: 20 },  // Ramp up to 20 users
    { duration: '5m', target: 20 },  // Stay at 20 users
    { duration: '2m', target: 0 },   // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.1'],    // Error rate should be less than 10%
  },
};

const BASE_URL = __ENV.CONTROL_PLANE_URL || 'http://localhost:8080';
const EDGE_URL = __ENV.EDGE_PROXY_URL || 'http://localhost:8081';

export default function() {
  // Test 1: Health check endpoints
  let healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 200ms': (r) => r.timings.duration < 200,
  }) || errorRate.add(1);

  // Test 2: API endpoints
  let domainsRes = http.get(`${BASE_URL}/v1/domains`);
  check(domainsRes, {
    'domains API status is 200 or 401': (r) => r.status === 200 || r.status === 401,
    'domains API response time < 500ms': (r) => r.timings.duration < 500,
  }) || errorRate.add(1);

  // Test 3: Edge proxy (if available)
  let edgeRes = http.get(`${EDGE_URL}/health`);
  check(edgeRes, {
    'edge proxy health status is 200': (r) => r.status === 200,
    'edge proxy response time < 200ms': (r) => r.timings.duration < 200,
  }) || errorRate.add(1);

  // Test 4: Metrics endpoints
  let metricsRes = http.get(`${BASE_URL}/metrics`);
  check(metricsRes, {
    'metrics endpoint accessible': (r) => r.status === 200,
    'metrics response contains prometheus format': (r) => r.body.includes('# HELP'),
  }) || errorRate.add(1);

  sleep(1); // 1 second delay between iterations
}

export function teardown(data) {
  console.log('Load test completed!');
  console.log('Check the results above for performance metrics.');
}
EOF

print_status "Starting NaijCloud load test..."
print_status "Control Plane URL: $CONTROL_PLANE_URL"
print_status "Edge Proxy URL: $EDGE_PROXY_URL"

# Wait for services to be ready
print_status "Checking if services are ready..."
for i in {1..30}; do
    if curl -s "$CONTROL_PLANE_URL/health" > /dev/null 2>&1; then
        print_success "Control Plane is ready!"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "Control Plane is not responding. Please start services first."
        exit 1
    fi
    sleep 2
done

# Run the load test
print_status "Running load test with k6..."
k6 run \
  --env CONTROL_PLANE_URL="$CONTROL_PLANE_URL" \
  --env EDGE_PROXY_URL="$EDGE_PROXY_URL" \
  /tmp/naijcloud_load_test.js

print_success "Load test completed!"

# Cleanup
rm -f /tmp/naijcloud_load_test.js

echo ""
echo "📊 Load Test Summary:"
echo "  • Test focused on API endpoints and health checks"
echo "  • Tested with up to 20 concurrent users"
echo "  • Validated response times and error rates"
echo ""
echo "🔍 Next steps:"
echo "  • Check Prometheus metrics: http://localhost:9090"
echo "  • View Grafana dashboards: http://localhost:3000"
echo "  • Analyze application logs: ./dev.sh logs"
