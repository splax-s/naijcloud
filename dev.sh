#!/bin/bash

# NaijCloud Development Environment Startup Script

set -e

echo "🚀 Starting NaijCloud Development Environment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Function to wait for service to be healthy
wait_for_service() {
    local service_name=$1
    local max_attempts=30
    local attempt=1
    
    print_status "Waiting for $service_name to be healthy..."
    
    while [ $attempt -le $max_attempts ]; do
        if docker-compose ps $service_name | grep -q "healthy\|Up"; then
            print_success "$service_name is healthy!"
            return 0
        fi
        
        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done
    
    print_error "$service_name failed to start within expected time"
    return 1
}

# Parse command line arguments
ACTION=${1:-start}

case $ACTION in
    "start")
        print_status "Starting infrastructure services..."
        docker-compose up -d postgres redis prometheus grafana loki
        
        # Wait for infrastructure to be ready
        wait_for_service postgres
        wait_for_service redis
        
        print_status "Building and starting application services..."
        docker-compose up -d control-plane edge-proxy
        
        # Wait for application services
        wait_for_service control-plane
        wait_for_service edge-proxy
        
        print_success "All services are running!"
        
        echo ""
        echo "📊 Service URLs:"
        echo "  Control Plane API:    http://localhost:8080"
        echo "  Edge Proxy:           http://localhost:8081"
        echo "  Dashboard:            http://localhost:3001 (if enabled)"
        echo "  Prometheus:           http://localhost:9090"
        echo "  Grafana:              http://localhost:3000 (admin/admin)"
        echo "  Loki:                 http://localhost:3100"
        echo "  Database Admin:       http://localhost:8081 (if adminer enabled)"
        echo ""
        echo "🔧 Next steps:"
        echo "  1. Check service status: ./dev.sh status"
        echo "  2. View logs: ./dev.sh logs [service-name]"
        echo "  3. Run tests: ./dev.sh test"
        echo "  4. Stop services: ./dev.sh stop"
        ;;
        
    "stop")
        print_status "Stopping all services..."
        docker-compose down
        print_success "All services stopped!"
        ;;
        
    "restart")
        print_status "Restarting all services..."
        docker-compose down
        docker-compose up -d
        print_success "All services restarted!"
        ;;
        
    "status")
        print_status "Service status:"
        docker-compose ps
        
        echo ""
        print_status "Health checks:"
        
        # Check Control Plane
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            print_success "Control Plane: Healthy"
        else
            print_error "Control Plane: Unhealthy"
        fi
        
        # Check Edge Proxy
        if curl -s http://localhost:8081/health > /dev/null 2>&1; then
            print_success "Edge Proxy: Healthy"
        else
            print_error "Edge Proxy: Unhealthy"
        fi
        
        # Check Prometheus
        if curl -s http://localhost:9090/-/healthy > /dev/null 2>&1; then
            print_success "Prometheus: Healthy"
        else
            print_error "Prometheus: Unhealthy"
        fi
        ;;
        
    "logs")
        SERVICE_NAME=${2:-""}
        if [ -z "$SERVICE_NAME" ]; then
            print_status "Showing logs for all services..."
            docker-compose logs -f
        else
            print_status "Showing logs for $SERVICE_NAME..."
            docker-compose logs -f $SERVICE_NAME
        fi
        ;;
        
    "test")
        print_status "Running integration tests..."
        
        # Run control plane tests
        print_status "Testing Control Plane..."
        cd control-plane && go test -v ./tests/
        cd ..
        
        # Run edge proxy tests
        print_status "Testing Edge Proxy..."
        cd edge-proxy && go test -v ./tests/
        cd ..
        
        print_success "All tests completed!"
        ;;
        
    "build")
        print_status "Building all services..."
        docker-compose build
        print_success "Build completed!"
        ;;
        
    "clean")
        print_status "Cleaning up containers, volumes, and images..."
        docker-compose down -v --rmi all
        docker system prune -f
        print_success "Cleanup completed!"
        ;;
        
    "shell")
        SERVICE_NAME=${2:-"control-plane"}
        print_status "Opening shell in $SERVICE_NAME..."
        docker-compose exec $SERVICE_NAME /bin/sh
        ;;
        
    "dashboard")
        print_status "Starting dashboard in development mode..."
        cd dashboard
        if [ ! -d "node_modules" ]; then
            print_status "Installing dependencies..."
            npm install
        fi
        print_status "Starting Next.js development server..."
        npm run dev
        ;;
        
    "help")
        echo "NaijCloud Development Environment"
        echo ""
        echo "Usage: $0 [command] [options]"
        echo ""
        echo "Commands:"
        echo "  start       Start all services (default)"
        echo "  stop        Stop all services"
        echo "  restart     Restart all services"
        echo "  status      Show service status and health"
        echo "  logs [svc]  Show logs (optionally for specific service)"
        echo "  test        Run integration tests"
        echo "  build       Build all Docker images"
        echo "  clean       Remove all containers, volumes, and images"
        echo "  shell [svc] Open shell in service container"
        echo "  dashboard   Start dashboard in development mode"
        echo "  help        Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0 start"
        echo "  $0 logs control-plane"
        echo "  $0 shell redis"
        echo "  $0 test"
        ;;
        
    *)
        print_error "Unknown command: $ACTION"
        echo "Use '$0 help' for available commands."
        exit 1
        ;;
esac
