#!/bin/bash

# NaijCloud Kubernetes Deployment Script
set -e

# Configuration
NAMESPACE="naijcloud"
DOCKER_REGISTRY=${DOCKER_REGISTRY:-""}
TAG=${TAG:-"latest"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is required but not installed"
        exit 1
    fi
    
    if ! command -v docker &> /dev/null; then
        error "docker is required but not installed"
        exit 1
    fi
    
    # Check if cluster is accessible
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log "Prerequisites check passed"
}

# Build and push Docker images
build_images() {
    log "Building Docker images..."
    
    # Build control plane
    log "Building control-plane image..."
    docker build -t ${DOCKER_REGISTRY}naijcloud/control-plane:${TAG} ./control-plane/
    
    # Build edge proxy
    log "Building edge-proxy image..."
    docker build -t ${DOCKER_REGISTRY}naijcloud/edge-proxy:${TAG} ./edge-proxy/
    
    # Build dashboard
    log "Building dashboard image..."
    docker build -t ${DOCKER_REGISTRY}naijcloud/dashboard:${TAG} ./dashboard/
    
    if [ -n "$DOCKER_REGISTRY" ]; then
        log "Pushing images to registry..."
        docker push ${DOCKER_REGISTRY}naijcloud/control-plane:${TAG}
        docker push ${DOCKER_REGISTRY}naijcloud/edge-proxy:${TAG}
        docker push ${DOCKER_REGISTRY}naijcloud/dashboard:${TAG}
    fi
    
    log "Images built successfully"
}

# Deploy to Kubernetes
deploy() {
    log "Deploying to Kubernetes..."
    
    # Apply manifests in order
    kubectl apply -f k8s/00-namespace-secrets.yaml
    kubectl apply -f k8s/01-postgres.yaml
    kubectl apply -f k8s/02-redis.yaml
    
    # Wait for database to be ready
    log "Waiting for PostgreSQL to be ready..."
    kubectl wait --for=condition=ready pod -l app=postgres -n $NAMESPACE --timeout=300s
    
    log "Waiting for Redis to be ready..."
    kubectl wait --for=condition=ready pod -l app=redis -n $NAMESPACE --timeout=300s
    
    # Deploy applications
    kubectl apply -f k8s/03-control-plane.yaml
    kubectl apply -f k8s/04-edge-proxy.yaml
    kubectl apply -f k8s/05-dashboard.yaml
    kubectl apply -f k8s/06-monitoring.yaml
    
    # Wait for applications to be ready
    log "Waiting for control-plane to be ready..."
    kubectl wait --for=condition=ready pod -l app=control-plane -n $NAMESPACE --timeout=300s
    
    log "Waiting for dashboard to be ready..."
    kubectl wait --for=condition=ready pod -l app=dashboard -n $NAMESPACE --timeout=300s
    
    # Apply ingress last
    if kubectl get crd clusterissuers.cert-manager.io &> /dev/null; then
        kubectl apply -f k8s/07-ingress.yaml
        log "Ingress with TLS configured"
    else
        warn "cert-manager not found, skipping TLS configuration"
        warn "Install cert-manager first: kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml"
    fi
    
    log "Deployment completed successfully!"
}

# Show status
show_status() {
    log "Deployment status:"
    kubectl get pods -n $NAMESPACE
    echo
    kubectl get services -n $NAMESPACE
    echo
    kubectl get ingress -n $NAMESPACE
}

# Get service URLs
get_urls() {
    log "Service URLs:"
    
    # Get LoadBalancer IPs
    CONTROL_PLANE_IP=$(kubectl get svc control-plane-external -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    DASHBOARD_IP=$(kubectl get svc dashboard-external -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "pending")
    
    echo "Control Plane API: http://$CONTROL_PLANE_IP"
    echo "Dashboard: http://$DASHBOARD_IP"
    
    # If ingress is configured
    if kubectl get ingress naijcloud-ingress -n $NAMESPACE &> /dev/null; then
        echo "API (via ingress): https://api.cdn.example.com"
        echo "Dashboard (via ingress): https://cdn.example.com"
        echo "Prometheus (via ingress): https://prometheus.cdn.example.com"
    fi
}

# Cleanup
cleanup() {
    warn "Removing NaijCloud deployment..."
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    log "Cleanup completed"
}

# Main script
case "${1:-deploy}" in
    "build")
        check_prerequisites
        build_images
        ;;
    "deploy")
        check_prerequisites
        build_images
        deploy
        show_status
        get_urls
        ;;
    "status")
        show_status
        get_urls
        ;;
    "cleanup")
        cleanup
        ;;
    *)
        echo "Usage: $0 {build|deploy|status|cleanup}"
        echo "  build   - Build and push Docker images"
        echo "  deploy  - Build images and deploy to Kubernetes (default)"
        echo "  status  - Show deployment status"
        echo "  cleanup - Remove all resources"
        exit 1
        ;;
esac
