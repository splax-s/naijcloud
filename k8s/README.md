# NaijCloud Kubernetes Deployment Guide

This guide provides instructions for deploying NaijCloud D-CDN to a Kubernetes cluster.

## Prerequisites

### Required Tools
- `kubectl` - Kubernetes command-line tool
- `docker` - For building container images
- `helm` (optional) - For advanced deployments
- Access to a Kubernetes cluster (v1.20+)

### Required Cluster Features
- LoadBalancer support (for cloud providers)
- StorageClass for persistent volumes
- Ingress controller (nginx recommended)
- cert-manager (optional, for automatic TLS)

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/splax-s/naijcloud.git
cd naijcloud
```

### 2. Configure Environment

Edit `k8s/00-namespace-secrets.yaml` and update the following:

```yaml
# Update these values for production
stringData:
  DATABASE_URL: "postgres://naijcloud:YOUR_DB_PASSWORD@postgres:5432/naijcloud?sslmode=disable"
  REDIS_URL: "redis://redis:6379"
  NEXTAUTH_SECRET: "YOUR_LONG_RANDOM_SECRET_STRING"
  POSTGRES_PASSWORD: "YOUR_DB_PASSWORD"
```

Edit `k8s/00-namespace-secrets.yaml` ConfigMap:

```yaml
data:
  NEXTAUTH_URL: "https://your-domain.com"
  NEXT_PUBLIC_API_URL: "https://api.your-domain.com"
```

Edit `k8s/07-ingress.yaml` and update domains:

```yaml
# Replace example.com with your actual domain
- host: api.your-domain.com
- host: your-domain.com
- host: prometheus.your-domain.com
```

### 3. Deploy

```bash
# Build and deploy everything
./deploy.sh deploy

# Or deploy step by step:
./deploy.sh build
kubectl apply -f k8s/
```

### 4. Verify Deployment

```bash
# Check status
./deploy.sh status

# Watch pods start up
kubectl get pods -n naijcloud -w
```

## Architecture Overview

### Components

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Dashboard     │    │  Control Plane  │    │   Edge Proxy    │
│   (Next.js)     │    │    (Go API)     │    │  (Go Proxy)     │
│   Port: 3000    │    │   Port: 8080    │    │  Port: 80/443   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────┐    ┌─────────────────┐
         │   PostgreSQL    │    │     Redis       │
         │   Port: 5432    │    │   Port: 6379    │
         └─────────────────┘    └─────────────────┘
```

### Services

1. **Dashboard** (`dashboard`) - Management interface
2. **Control Plane** (`control-plane`) - API and orchestration
3. **Edge Proxy** (`edge-proxy`) - CDN proxy nodes (DaemonSet)
4. **PostgreSQL** (`postgres`) - Primary database
5. **Redis** (`redis`) - Cache and session storage
6. **Prometheus** (`prometheus`) - Metrics collection

## Networking

### Internal Services
- All services communicate via ClusterIP services
- Edge proxies deployed as DaemonSet on all nodes
- Control plane has multiple replicas for HA

### External Access
- **LoadBalancer Services**: Direct access to services
- **Ingress**: Domain-based routing with TLS termination
- **NodePort**: Alternative for on-premise clusters

### Default Ports
- Dashboard: 3000 (internal), 80 (external)
- Control Plane: 8080 (internal), 80 (external)
- Edge Proxy: 80/443 (host network)
- Prometheus: 9090 (internal)

## Storage

### PostgreSQL
- Uses PersistentVolumeClaim for data persistence
- Default size: 20Gi (adjust in `01-postgres.yaml`)
- Automatic schema initialization

### Redis
- Uses emptyDir for ephemeral cache storage
- Memory limit: 1Gi (adjust in `02-redis.yaml`)
- LRU eviction policy

## Security

### Secrets Management
- Database credentials stored in Kubernetes secrets
- NextAuth secret for session security
- TLS certificates managed by cert-manager

### Network Policies
- Services isolated by namespace
- Edge proxies have host network access
- Internal services use ClusterIP only

### RBAC
- Minimal service account permissions
- Prometheus has cluster-wide read access for metrics

## Monitoring

### Metrics Collection
- Prometheus scrapes all services
- Control plane exposes metrics on port 9091
- Edge proxies expose metrics on port 8081

### Health Checks
- All services have liveness and readiness probes
- Database connectivity checked in probes
- Custom health endpoints for applications

## Configuration

### Environment Variables

#### Control Plane
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `METRICS_PORT` - Prometheus metrics port

#### Edge Proxy
- `CONTROL_PLANE_URL` - Control plane endpoint
- `REDIS_URL` - Redis connection for caching
- `NODE_REGION` - Kubernetes node region label
- `NODE_HOSTNAME` - Kubernetes node name

#### Dashboard
- `NEXTAUTH_URL` - Full URL for authentication
- `NEXTAUTH_SECRET` - Session encryption secret
- `NEXT_PUBLIC_API_URL` - Control plane API URL

### Resource Limits

Default resource allocation:

```yaml
# Control Plane
requests: {cpu: 200m, memory: 256Mi}
limits: {cpu: 1000m, memory: 1Gi}

# Edge Proxy
requests: {cpu: 500m, memory: 512Mi}
limits: {cpu: 2000m, memory: 2Gi}

# Dashboard
requests: {cpu: 100m, memory: 256Mi}
limits: {cpu: 500m, memory: 512Mi}
```

## Scaling

### Horizontal Scaling
- Control plane: Increase replicas in deployment
- Dashboard: Increase replicas in deployment
- Edge proxy: Automatically scales with cluster nodes

### Vertical Scaling
- Adjust resource requests and limits
- Monitor CPU and memory usage via Prometheus

### Database Scaling
- PostgreSQL: Consider read replicas for high load
- Redis: Consider Redis Cluster for larger datasets

## Troubleshooting

### Common Issues

#### Pods Stuck in Pending
```bash
kubectl describe pod <pod-name> -n naijcloud
# Check for resource constraints or scheduling issues
```

#### Database Connection Errors
```bash
kubectl logs deployment/control-plane -n naijcloud
kubectl exec -it deployment/postgres -n naijcloud -- psql -U naijcloud
```

#### TLS Certificate Issues
```bash
kubectl describe certificatechain naijcloud-tls -n naijcloud
kubectl logs -n cert-manager deployment/cert-manager
```

### Log Access
```bash
# Application logs
kubectl logs deployment/control-plane -n naijcloud
kubectl logs deployment/dashboard -n naijcloud
kubectl logs daemonset/edge-proxy -n naijcloud

# Follow logs
kubectl logs -f deployment/control-plane -n naijcloud
```

### Database Access
```bash
# Connect to PostgreSQL
kubectl exec -it deployment/postgres -n naijcloud -- psql -U naijcloud

# Connect to Redis
kubectl exec -it deployment/redis -n naijcloud -- redis-cli
```

## Maintenance

### Updates
```bash
# Update images
export TAG=v1.1.0
./deploy.sh build

# Rolling update
kubectl set image deployment/control-plane control-plane=naijcloud/control-plane:v1.1.0 -n naijcloud
```

### Backup
```bash
# Database backup
kubectl exec deployment/postgres -n naijcloud -- pg_dump -U naijcloud naijcloud > backup.sql

# Restore
kubectl exec -i deployment/postgres -n naijcloud -- psql -U naijcloud naijcloud < backup.sql
```

### Cleanup
```bash
# Remove everything
./deploy.sh cleanup

# Or selectively
kubectl delete namespace naijcloud
```

## Production Considerations

### High Availability
- Deploy across multiple availability zones
- Use external databases (RDS, Cloud SQL)
- Configure anti-affinity rules
- Set up cross-region replication

### Performance
- Tune PostgreSQL settings for workload
- Configure Redis persistence if needed
- Set appropriate resource limits
- Monitor and adjust based on metrics

### Security
- Enable network policies
- Use private container registry
- Rotate secrets regularly
- Enable audit logging

### Cost Optimization
- Use spot instances for development
- Right-size resource requests
- Consider reserved instances for production
- Monitor resource utilization
