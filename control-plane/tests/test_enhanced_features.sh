#!/bin/bash
# Enhanced Feature Test Script
# Tests JWT authentication, activity logging, and notifications

set -e

# Configuration
BASE_URL="http://localhost:8080"
API_BASE="$BASE_URL/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
    ((PASSED_TESTS++))
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((FAILED_TESTS++))
}

test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local headers=$4
    local expected_status=$5
    local test_name=$6
    
    ((TOTAL_TESTS++))
    
    log_info "Testing: $test_name"
    
    local cmd="curl -s -w '%{http_code}' -X $method"
    
    if [ ! -z "$headers" ]; then
        cmd="$cmd $headers"
    fi
    
    if [ ! -z "$data" ]; then
        cmd="$cmd -d '$data'"
    fi
    
    cmd="$cmd '$endpoint'"
    
    local response=$(eval $cmd)
    local status_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$status_code" = "$expected_status" ]; then
        log_success "$test_name - Status: $status_code"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        return 0
    else
        log_error "$test_name - Expected: $expected_status, Got: $status_code"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        return 1
    fi
}

# Wait for server to be ready
wait_for_server() {
    log_info "Waiting for server to be ready..."
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
            log_success "Server is ready!"
            return 0
        fi
        
        echo -n "."
        sleep 1
        ((attempt++))
    done
    
    log_error "Server did not become ready within $max_attempts seconds"
    return 1
}

# Test user registration
test_registration() {
    log_info "=== Testing User Registration ==="
    
    local test_email="test-enhanced-$(date +%s)@example.com"
    local test_org_slug="test-enhanced-$(date +%s)"
    
    local registration_data='{
        "email": "'$test_email'",
        "name": "Enhanced Test User",
        "password": "SecurePassword123!",
        "confirm_password": "SecurePassword123!",
        "organization_name": "Enhanced Test Org",
        "organization_slug": "'$test_org_slug'"
    }'
    
    test_endpoint "POST" "$API_BASE/auth/register" "$registration_data" "-H 'Content-Type: application/json'" "201" "User Registration"
    
    # Store email for later tests
    echo "$test_email" > /tmp/test_email.txt
}

# Test user login and get tokens
test_login() {
    log_info "=== Testing Enhanced Authentication ==="
    
    local test_email=$(cat /tmp/test_email.txt 2>/dev/null || echo "test@example.com")
    
    local login_data='{
        "email": "'$test_email'",
        "password": "SecurePassword123!"
    }'
    
    local response=$(curl -s -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "$login_data")
    
    if echo "$response" | jq -e '.tokens.access_token' > /dev/null; then
        log_success "Login successful with JWT tokens"
        
        # Extract tokens
        local access_token=$(echo "$response" | jq -r '.tokens.access_token')
        local refresh_token=$(echo "$response" | jq -r '.tokens.refresh_token')
        
        # Store tokens for later tests
        echo "$access_token" > /tmp/access_token.txt
        echo "$refresh_token" > /tmp/refresh_token.txt
        
        log_info "Access token obtained: ${access_token:0:20}..."
        log_info "Refresh token obtained: ${refresh_token:0:20}..."
        
        return 0
    else
        log_error "Login failed or no tokens returned"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# Test token refresh
test_token_refresh() {
    log_info "=== Testing Token Refresh ==="
    
    local refresh_token=$(cat /tmp/refresh_token.txt 2>/dev/null)
    
    if [ -z "$refresh_token" ]; then
        log_error "No refresh token available"
        return 1
    fi
    
    local refresh_data='{
        "refresh_token": "'$refresh_token'"
    }'
    
    local response=$(curl -s -X POST "$API_BASE/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "$refresh_data")
    
    if echo "$response" | jq -e '.tokens.access_token' > /dev/null; then
        log_success "Token refresh successful"
        
        # Update stored access token
        local new_access_token=$(echo "$response" | jq -r '.tokens.access_token')
        echo "$new_access_token" > /tmp/access_token.txt
        
        return 0
    else
        log_error "Token refresh failed"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
}

# Test authenticated endpoints
test_authenticated_endpoints() {
    log_info "=== Testing Authenticated Endpoints ==="
    
    local access_token=$(cat /tmp/access_token.txt 2>/dev/null)
    
    if [ -z "$access_token" ]; then
        log_error "No access token available"
        return 1
    fi
    
    local auth_header="-H 'Authorization: Bearer $access_token'"
    
    # Test profile endpoint
    test_endpoint "GET" "$API_BASE/auth/profile" "" "$auth_header" "200" "Get User Profile"
    
    # Test activities endpoint
    test_endpoint "GET" "$API_BASE/activities" "" "$auth_header" "200" "Get Activities"
    
    # Test activity summary
    test_endpoint "GET" "$API_BASE/activities/summary" "" "$auth_header" "200" "Get Activity Summary"
    
    # Test notifications endpoint
    test_endpoint "GET" "$API_BASE/notifications" "" "$auth_header" "200" "Get Notifications"
    
    # Test unread notifications count
    test_endpoint "GET" "$API_BASE/notifications/unread-count" "" "$auth_header" "200" "Get Unread Count"
}

# Test notification functionality
test_notifications() {
    log_info "=== Testing Notification System ==="
    
    local access_token=$(cat /tmp/access_token.txt 2>/dev/null)
    
    if [ -z "$access_token" ]; then
        log_error "No access token available"
        return 1
    fi
    
    local auth_header="-H 'Authorization: Bearer $access_token'"
    
    # Get notifications with filters
    test_endpoint "GET" "$API_BASE/notifications?unread=true&limit=10" "" "$auth_header" "200" "Get Unread Notifications"
    
    # Test mark all as read
    test_endpoint "POST" "$API_BASE/notifications/mark-all-read" "" "$auth_header" "200" "Mark All Notifications as Read"
}

# Test password change
test_password_change() {
    log_info "=== Testing Password Change ==="
    
    local access_token=$(cat /tmp/access_token.txt 2>/dev/null)
    
    if [ -z "$access_token" ]; then
        log_error "No access token available"
        return 1
    fi
    
    local auth_header="-H 'Authorization: Bearer $access_token' -H 'Content-Type: application/json'"
    
    local password_data='{
        "current_password": "SecurePassword123!",
        "new_password": "NewSecurePassword123!",
        "confirm_password": "NewSecurePassword123!"
    }'
    
    test_endpoint "POST" "$API_BASE/auth/change-password" "$password_data" "$auth_header" "200" "Change Password"
}

# Test logout
test_logout() {
    log_info "=== Testing Logout ==="
    
    local access_token=$(cat /tmp/access_token.txt 2>/dev/null)
    
    if [ -z "$access_token" ]; then
        log_error "No access token available"
        return 1
    fi
    
    local auth_header="-H 'Authorization: Bearer $access_token'"
    
    test_endpoint "POST" "$API_BASE/auth/logout" "" "$auth_header" "200" "User Logout"
}

# Test security features
test_security() {
    log_info "=== Testing Security Features ==="
    
    # Test without authentication
    test_endpoint "GET" "$API_BASE/auth/profile" "" "" "401" "Unauthorized Access"
    
    # Test with invalid token
    local invalid_auth="-H 'Authorization: Bearer invalid.token.here'"
    test_endpoint "GET" "$API_BASE/auth/profile" "" "$invalid_auth" "401" "Invalid Token"
    
    # Test CORS headers
    local response=$(curl -s -I -X OPTIONS "$API_BASE/auth/profile" -H "Origin: http://localhost:3000")
    if echo "$response" | grep -i "access-control-allow" > /dev/null; then
        log_success "CORS headers present"
    else
        log_error "CORS headers missing"
    fi
}

# Test rate limiting (if enabled)
test_rate_limiting() {
    log_info "=== Testing Rate Limiting ==="
    
    # Make multiple rapid requests to test rate limiting
    for i in {1..10}; do
        local status_code=$(curl -s -w '%{http_code}' -o /dev/null "$API_BASE/auth/profile")
        if [ "$status_code" = "429" ]; then
            log_success "Rate limiting is working (got 429)"
            return 0
        fi
    done
    
    log_warning "Rate limiting not detected (this may be expected if not configured)"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test files..."
    rm -f /tmp/test_email.txt /tmp/access_token.txt /tmp/refresh_token.txt
}

# Main test execution
main() {
    log_info "🚀 Starting Enhanced Feature Tests"
    log_info "Testing against: $BASE_URL"
    
    # Wait for server
    if ! wait_for_server; then
        log_error "Server is not available. Please start the control-plane server."
        exit 1
    fi
    
    # Run tests
    test_registration
    test_login
    test_token_refresh
    test_authenticated_endpoints
    test_notifications
    test_password_change
    test_logout
    test_security
    test_rate_limiting
    
    # Cleanup
    cleanup
    
    # Summary
    echo ""
    log_info "=== Test Summary ==="
    log_info "Total Tests: $TOTAL_TESTS"
    log_success "Passed: $PASSED_TESTS"
    
    if [ $FAILED_TESTS -gt 0 ]; then
        log_error "Failed: $FAILED_TESTS"
        exit 1
    else
        log_success "All tests passed! 🎉"
        exit 0
    fi
}

# Run main function
main "$@"
