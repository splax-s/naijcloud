#!/bin/bash

# Local Kubernetes Setup with k3s
# This script sets up a local k3s cluster for testing NaijCloud

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

ACTION=${1:-"install"}

case $ACTION in
    "install")
        print_status "Installing k3s for local Kubernetes testing..."
        
        # Check if k3s is already installed
        if command -v k3s &> /dev/null; then
            print_warning "k3s is already installed"
            k3s --version
        else
            print_status "Downloading and installing k3s..."
            curl -sfL https://get.k3s.io | sh -
        fi
        
        print_status "Waiting for k3s to be ready..."
        sleep 10
        
        # Check if kubectl is available
        if ! command -v kubectl &> /dev/null; then
            print_status "Installing kubectl..."
            # Use k3s kubectl
            sudo ln -sf /usr/local/bin/k3s /usr/local/bin/kubectl
        fi
        
        # Copy kubeconfig for regular user
        print_status "Setting up kubeconfig..."
        mkdir -p ~/.kube
        sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
        sudo chown $(id -u):$(id -g) ~/.kube/config
        export KUBECONFIG=~/.kube/config
        
        # Wait for node to be ready
        print_status "Waiting for cluster to be ready..."
        for i in {1..30}; do
            if kubectl get nodes | grep -q Ready; then
                print_success "k3s cluster is ready!"
                break
            fi
            sleep 5
        done
        
        print_success "k3s installation completed!"
        echo ""
        echo "🎯 Next steps:"
        echo "  1. Deploy NaijCloud: ./k3s-setup.sh deploy"
        echo "  2. Check status: ./k3s-setup.sh status"
        echo "  3. Access services: ./k3s-setup.sh port-forward"
        ;;
        
    "deploy")
        print_status "Deploying NaijCloud to k3s..."
        
        # Check if k3s is running
        if ! kubectl get nodes &> /dev/null; then
            print_error "k3s cluster is not accessible. Run './k3s-setup.sh install' first."
            exit 1
        fi
        
        # Build Docker images
        print_status "Building Docker images..."
        docker build -t naijcloud/control-plane:local ./control-plane
        docker build -t naijcloud/edge-proxy:local ./edge-proxy
        docker build -t naijcloud/dashboard:local ./dashboard
        
        # Import images to k3s
        print_status "Importing images to k3s..."
        k3s ctr images import <(docker save naijcloud/control-plane:local)
        k3s ctr images import <(docker save naijcloud/edge-proxy:local)
        k3s ctr images import <(docker save naijcloud/dashboard:local)
        
        # Update image tags in k8s manifests for local deployment
        print_status "Preparing local Kubernetes manifests..."
        cp -r k8s k8s-local
        
        # Update image references for local deployment
        sed -i.bak 's|image: naijcloud/control-plane:.*|image: naijcloud/control-plane:local|g' k8s-local/*.yaml
        sed -i.bak 's|image: naijcloud/edge-proxy:.*|image: naijcloud/edge-proxy:local|g' k8s-local/*.yaml
        sed -i.bak 's|image: naijcloud/dashboard:.*|image: naijcloud/dashboard:local|g' k8s-local/*.yaml
        
        # Add imagePullPolicy: Never for local images
        sed -i.bak 's|image: naijcloud/|imagePullPolicy: Never\
          image: naijcloud/|g' k8s-local/*.yaml
        
        # Deploy to k3s
        print_status "Applying Kubernetes manifests..."
        kubectl apply -f k8s-local/
        
        # Wait for deployment
        print_status "Waiting for pods to be ready..."
        kubectl wait --for=condition=ready pod -l app=postgres -n naijcloud --timeout=300s
        kubectl wait --for=condition=ready pod -l app=redis -n naijcloud --timeout=300s
        kubectl wait --for=condition=ready pod -l app=control-plane -n naijcloud --timeout=300s
        
        print_success "NaijCloud deployed to k3s!"
        
        echo ""
        echo "📊 Deployment status:"
        kubectl get pods -n naijcloud
        ;;
        
    "status")
        print_status "Checking NaijCloud deployment status..."
        
        if ! kubectl get ns naijcloud &> /dev/null; then
            print_warning "NaijCloud namespace not found. Run './k3s-setup.sh deploy' first."
            exit 0
        fi
        
        echo ""
        echo "🚀 Pods:"
        kubectl get pods -n naijcloud
        
        echo ""
        echo "🌐 Services:"
        kubectl get services -n naijcloud
        
        echo ""
        echo "📡 Ingress:"
        kubectl get ingress -n naijcloud 2>/dev/null || echo "No ingress found"
        ;;
        
    "port-forward")
        print_status "Setting up port forwarding for local access..."
        
        if ! kubectl get pods -n naijcloud &> /dev/null; then
            print_error "NaijCloud is not deployed. Run './k3s-setup.sh deploy' first."
            exit 1
        fi
        
        print_status "Starting port forwarding (press Ctrl+C to stop)..."
        
        # Start port forwarding in background
        kubectl port-forward -n naijcloud svc/control-plane 8080:80 &
        CP_PID=$!
        
        kubectl port-forward -n naijcloud svc/dashboard 3000:80 &
        DASH_PID=$!
        
        kubectl port-forward -n naijcloud svc/prometheus 9090:9090 &
        PROM_PID=$!
        
        echo ""
        print_success "Port forwarding active:"
        echo "  Control Plane API: http://localhost:8080"
        echo "  Dashboard:         http://localhost:3000"
        echo "  Prometheus:        http://localhost:9090"
        echo ""
        echo "Press Ctrl+C to stop port forwarding..."
        
        # Wait for interrupt
        trap "kill $CP_PID $DASH_PID $PROM_PID 2>/dev/null; print_status 'Port forwarding stopped'" INT
        wait
        ;;
        
    "logs")
        SERVICE=${2:-"control-plane"}
        print_status "Showing logs for $SERVICE..."
        kubectl logs -f -n naijcloud deployment/$SERVICE
        ;;
        
    "shell")
        SERVICE=${2:-"control-plane"}
        print_status "Opening shell in $SERVICE pod..."
        kubectl exec -it -n naijcloud deployment/$SERVICE -- /bin/sh
        ;;
        
    "test")
        print_status "Running tests against k3s deployment..."
        
        # Port forward for testing
        kubectl port-forward -n naijcloud svc/control-plane 8080:80 &
        PF_PID=$!
        
        sleep 5  # Wait for port forward to establish
        
        # Run basic connectivity tests
        if curl -s http://localhost:8080/health | grep -q "ok"; then
            print_success "Control Plane health check passed"
        else
            print_error "Control Plane health check failed"
        fi
        
        # Cleanup
        kill $PF_PID 2>/dev/null || true
        ;;
        
    "cleanup")
        print_status "Removing NaijCloud from k3s..."
        kubectl delete namespace naijcloud --ignore-not-found=true
        rm -rf k8s-local
        print_success "Cleanup completed!"
        ;;
        
    "uninstall")
        print_status "Uninstalling k3s..."
        print_warning "This will remove the entire k3s cluster!"
        read -p "Are you sure? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            /usr/local/bin/k3s-uninstall.sh
            print_success "k3s uninstalled!"
        else
            print_status "Uninstall cancelled"
        fi
        ;;
        
    "help")
        echo "Local Kubernetes Setup for NaijCloud"
        echo ""
        echo "Usage: $0 [command] [options]"
        echo ""
        echo "Commands:"
        echo "  install         Install k3s cluster"
        echo "  deploy          Deploy NaijCloud to k3s"
        echo "  status          Show deployment status"
        echo "  port-forward    Set up port forwarding for local access"
        echo "  logs [service]  Show logs for service"
        echo "  shell [service] Open shell in service pod"
        echo "  test            Run basic connectivity tests"
        echo "  cleanup         Remove NaijCloud from k3s"
        echo "  uninstall       Uninstall k3s completely"
        echo "  help            Show this help"
        echo ""
        echo "Examples:"
        echo "  $0 install"
        echo "  $0 deploy"
        echo "  $0 port-forward"
        echo "  $0 logs control-plane"
        ;;
        
    *)
        print_error "Unknown command: $ACTION"
        echo "Use '$0 help' for available commands."
        exit 1
        ;;
esac
