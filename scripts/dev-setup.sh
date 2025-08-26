#!/bin/bash
set -e

echo "🚀 Starting NaijCloud D-CDN local development environment..."

# Check dependencies
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required but not installed. Aborting." >&2; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "❌ Docker Compose is required but not installed. Aborting." >&2; exit 1; }

# Create necessary directories
mkdir -p observability/grafana/dashboards
mkdir -p observability/grafana/provisioning/dashboards
mkdir -p observability/grafana/provisioning/datasources

echo "📋 Setting up observability configuration..."

# Create Prometheus configuration
cat > observability/prometheus.yml << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'control-plane'
    static_configs:
      - targets: ['control-plane:9091']
    metrics_path: '/metrics'
    scrape_interval: 10s
    
  - job_name: 'edge-proxy'
    static_configs:
      - targets: ['edge-proxy:9092']
    metrics_path: '/metrics'
    scrape_interval: 10s

  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
EOF

# Create Loki configuration
cat > observability/loki-config.yml << 'EOF'
auth_enabled: false

server:
  http_listen_port: 3100

common:
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

ruler:
  alertmanager_url: http://localhost:9093
EOF

# Create Grafana datasource configuration
cat > observability/grafana/provisioning/datasources/datasources.yml << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    editable: true
EOF

# Create Grafana dashboard configuration
cat > observability/grafana/provisioning/dashboards/dashboards.yml << 'EOF'
apiVersion: 1

providers:
  - name: 'NaijCloud Dashboards'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /var/lib/grafana/dashboards
EOF

echo "🐳 Starting infrastructure services..."
docker-compose up -d postgres redis prometheus grafana loki

echo "⏳ Waiting for services to be ready..."
sleep 10

# Wait for PostgreSQL
echo "🔍 Waiting for PostgreSQL..."
until docker-compose exec -T postgres pg_isready -U naijcloud; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done
echo "✅ PostgreSQL is ready!"

# Wait for Redis
echo "🔍 Waiting for Redis..."
until docker-compose exec -T redis redis-cli ping | grep PONG; do
  echo "Redis is unavailable - sleeping"
  sleep 2
done
echo "✅ Redis is ready!"

echo "🏗️  Building application services..."
docker-compose build control-plane

echo "🚀 Starting control plane..."
docker-compose up -d control-plane

# Wait for control plane
echo "🔍 Waiting for control plane..."
until curl -f http://localhost:8080/health >/dev/null 2>&1; do
  echo "Control plane is unavailable - sleeping"
  sleep 2
done
echo "✅ Control plane is ready!"

echo ""
echo "🎉 NaijCloud D-CDN development environment is ready!"
echo ""
echo "📊 Available services:"
echo "  • Control Plane API: http://localhost:8080"
echo "  • Control Plane Metrics: http://localhost:9091/metrics"
echo "  • Prometheus: http://localhost:9090"
echo "  • Grafana: http://localhost:3000 (admin/admin)"
echo "  • PostgreSQL: localhost:5432 (naijcloud/naijcloud_pass)"
echo "  • Redis: localhost:6379"
echo ""
echo "🔧 Quick commands:"
echo "  • View logs: docker-compose logs -f [service]"
echo "  • Stop all: docker-compose down"
echo "  • Restart: docker-compose restart [service]"
echo ""
echo "📚 Next steps:"
echo "  1. Test the API: curl http://localhost:8080/health"
echo "  2. Create a domain: curl -X POST http://localhost:8080/v1/domains -H 'Content-Type: application/json' -d '{\"domain\":\"example.com\",\"origin_url\":\"https://httpbin.org\"}'"
echo "  3. View metrics: open http://localhost:9090"
echo "  4. Check Grafana: open http://localhost:3000"
echo ""
