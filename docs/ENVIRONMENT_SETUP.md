# Environment Setup Guide

This guide covers setting up development and production environments for NaijCloud D-CDN.

## Development Environment

### Prerequisites

- Go 1.21 or later
- Node.js 18+ with npm
- Docker and Docker Compose
- PostgreSQL 15+
- Redis 6+

### Local Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/splax-s/naijcloud.git
   cd naijcloud
   ```

2. **Set up Go environment:**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Install Node.js dependencies:**
   ```bash
   cd dashboard
   npm install
   cd ..
   ```

4. **Start local services with Docker:**
   ```bash
   docker-compose up -d postgres redis
   ```

5. **Run database migrations:**
   ```bash
   cd control-plane
   go run cmd/migrate/main.go
   cd ..
   ```

6. **Start development servers:**
   ```bash
   # Terminal 1: Control Plane
   cd control-plane
   go run cmd/server/main.go

   # Terminal 2: Edge Proxy
   cd edge-proxy
   go run cmd/server/main.go

   # Terminal 3: Dashboard
   cd dashboard
   npm run dev
   ```

### Environment Variables

Create `.env` files in each component directory:

**control-plane/.env:**
```env
DATABASE_URL=postgres://naijcloud:password@localhost:5432/naijcloud?sslmode=disable
REDIS_URL=redis://localhost:6379
LOG_LEVEL=debug
METRICS_PORT=9091
PORT=8080
```

**edge-proxy/.env:**
```env
CONTROL_PLANE_URL=http://localhost:8080
REDIS_URL=redis://localhost:6379
LOG_LEVEL=debug
METRICS_PORT=8081
PORT=8000
```

**dashboard/.env.local:**
```env
NEXTAUTH_URL=http://localhost:3000
NEXTAUTH_SECRET=your-secret-key-here
NEXT_PUBLIC_API_URL=http://localhost:8080
DATABASE_URL=postgres://naijcloud:password@localhost:5432/naijcloud?sslmode=disable
```

## Production Environment

### Cloud Provider Requirements

#### AWS
- EKS cluster (v1.20+)
- RDS PostgreSQL instance
- ElastiCache Redis cluster
- Application Load Balancer
- Route 53 for DNS
- Certificate Manager for TLS

#### Google Cloud
- GKE cluster (v1.20+)
- Cloud SQL PostgreSQL
- Memorystore Redis
- Cloud Load Balancing
- Cloud DNS
- Certificate Manager

#### Azure
- AKS cluster (v1.20+)
- Azure Database for PostgreSQL
- Azure Cache for Redis
- Application Gateway
- Azure DNS
- Key Vault for secrets

### Kubernetes Cluster Setup

#### Resource Requirements

**Minimum:**
- 3 nodes, 2 vCPUs, 4GB RAM each
- 100GB persistent storage
- Load balancer support

**Recommended:**
- 5+ nodes, 4 vCPUs, 8GB RAM each
- 500GB+ persistent storage
- Multi-AZ deployment

#### Required Addons

1. **Ingress Controller:**
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
   ```

2. **cert-manager (optional):**
   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml
   ```

3. **Metrics Server:**
   ```bash
   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
   ```

### External Services

#### PostgreSQL Database

**AWS RDS:**
```bash
aws rds create-db-instance \
  --db-instance-identifier naijcloud-prod \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --master-username naijcloud \
  --master-user-password YOUR_PASSWORD \
  --allocated-storage 100 \
  --vpc-security-group-ids sg-xxx \
  --db-subnet-group-name your-subnet-group
```

**Google Cloud SQL:**
```bash
gcloud sql instances create naijcloud-prod \
  --database-version=POSTGRES_15 \
  --tier=db-n1-standard-2 \
  --region=us-central1 \
  --root-password=YOUR_PASSWORD
```

#### Redis Cache

**AWS ElastiCache:**
```bash
aws elasticache create-cache-cluster \
  --cache-cluster-id naijcloud-redis \
  --engine redis \
  --cache-node-type cache.t3.micro \
  --num-cache-nodes 1
```

**Google Memorystore:**
```bash
gcloud redis instances create naijcloud-redis \
  --size=1 \
  --region=us-central1 \
  --redis-version=redis_6_x
```

### DNS and TLS

#### Domain Configuration

1. **Create DNS records:**
   ```
   api.yourdomain.com    -> Load Balancer IP
   yourdomain.com        -> Load Balancer IP
   prometheus.yourdomain.com -> Load Balancer IP
   ```

2. **TLS Certificate (Let's Encrypt):**
   ```yaml
   apiVersion: cert-manager.io/v1
   kind: ClusterIssuer
   metadata:
     name: letsencrypt-prod
   spec:
     acme:
       server: https://acme-v02.api.letsencrypt.org/directory
       email: admin@yourdomain.com
       privateKeySecretRef:
         name: letsencrypt-prod
       solvers:
       - http01:
           ingress:
             class: nginx
   ```

### Security Configuration

#### Network Security

1. **Firewall Rules:**
   - Allow HTTP/HTTPS (80, 443) from internet
   - Allow SSH (22) from admin IPs only
   - Deny all other inbound traffic

2. **Kubernetes Network Policies:**
   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: naijcloud-netpol
     namespace: naijcloud
   spec:
     podSelector: {}
     policyTypes:
     - Ingress
     - Egress
     ingress:
     - from:
       - namespaceSelector:
           matchLabels:
             name: naijcloud
     egress:
     - {}
   ```

#### Secret Management

1. **Create production secrets:**
   ```bash
   kubectl create secret generic naijcloud-secrets \
     --from-literal=DATABASE_URL="postgres://..." \
     --from-literal=REDIS_URL="redis://..." \
     --from-literal=NEXTAUTH_SECRET="..." \
     -n naijcloud
   ```

2. **Use external secret management:**
   - AWS Secrets Manager
   - Google Secret Manager
   - Azure Key Vault
   - HashiCorp Vault

### Monitoring Setup

#### Prometheus Configuration

1. **External Prometheus (recommended):**
   - AWS Managed Prometheus
   - Google Cloud Monitoring
   - Grafana Cloud

2. **Self-hosted Prometheus:**
   ```bash
   helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
   helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring
   ```

#### Grafana Dashboards

Import these dashboard IDs:
- Node Exporter: 1860
- Kubernetes Cluster: 7249
- Go Applications: 10826
- NGINX Ingress: 9614

### Backup Strategy

#### Database Backups

1. **Automated backups:**
   ```bash
   # Daily backup script
   kubectl create cronjob postgres-backup \
     --image=postgres:15 \
     --schedule="0 2 * * *" \
     -- pg_dump $DATABASE_URL | gzip > /backup/$(date +%Y%m%d).sql.gz
   ```

2. **Cross-region replication:**
   - Configure read replicas
   - Set up disaster recovery

#### Application Backups

1. **Configuration backup:**
   ```bash
   kubectl get all,configmap,secret -n naijcloud -o yaml > naijcloud-backup.yaml
   ```

2. **Persistent volume backups:**
   - Use cloud provider snapshot features
   - Regular automated snapshots

### Performance Tuning

#### Resource Optimization

1. **Horizontal Pod Autoscaler:**
   ```yaml
   apiVersion: autoscaling/v2
   kind: HorizontalPodAutoscaler
   metadata:
     name: control-plane-hpa
   spec:
     scaleTargetRef:
       apiVersion: apps/v1
       kind: Deployment
       name: control-plane
     minReplicas: 2
     maxReplicas: 10
     metrics:
     - type: Resource
       resource:
         name: cpu
         target:
           type: Utilization
           averageUtilization: 70
   ```

2. **Vertical Pod Autoscaler:**
   ```bash
   kubectl apply -f https://github.com/kubernetes/autoscaler/releases/download/vertical-pod-autoscaler-0.13.0/vpa-release.yaml
   ```

#### Database Optimization

1. **PostgreSQL tuning:**
   ```sql
   -- Increase connection limits
   ALTER SYSTEM SET max_connections = 200;
   
   -- Optimize memory settings
   ALTER SYSTEM SET shared_buffers = '256MB';
   ALTER SYSTEM SET effective_cache_size = '1GB';
   
   -- Enable query optimization
   ALTER SYSTEM SET random_page_cost = 1.1;
   ```

2. **Redis optimization:**
   ```redis
   # Memory optimization
   maxmemory 1gb
   maxmemory-policy allkeys-lru
   
   # Persistence
   save 900 1
   save 300 10
   save 60 10000
   ```

### Deployment Pipeline

#### CI/CD Setup

1. **GitHub Actions workflow:**
   ```yaml
   name: Deploy to Production
   on:
     push:
       branches: [main]
   jobs:
     deploy:
       runs-on: ubuntu-latest
       steps:
       - uses: actions/checkout@v3
       - name: Build and Deploy
         run: |
           ./deploy.sh build
           ./deploy.sh deploy
   ```

2. **Blue-Green Deployment:**
   ```bash
   # Deploy to staging
   kubectl apply -f k8s/ --namespace=naijcloud-staging
   
   # Test and validate
   ./scripts/smoke-test.sh staging
   
   # Switch traffic
   kubectl patch service control-plane -p '{"spec":{"selector":{"version":"v2"}}}' -n naijcloud
   ```

### Troubleshooting

#### Common Production Issues

1. **High Memory Usage:**
   ```bash
   kubectl top pods -n naijcloud
   kubectl describe pod <pod-name> -n naijcloud
   ```

2. **Database Connection Issues:**
   ```bash
   kubectl exec -it deployment/control-plane -n naijcloud -- nc -zv postgres 5432
   ```

3. **Certificate Problems:**
   ```bash
   kubectl describe certificate naijcloud-tls -n naijcloud
   kubectl logs -n cert-manager deployment/cert-manager
   ```

#### Emergency Procedures

1. **Quick rollback:**
   ```bash
   kubectl rollout undo deployment/control-plane -n naijcloud
   ```

2. **Emergency scaling:**
   ```bash
   kubectl scale deployment control-plane --replicas=0 -n naijcloud
   kubectl scale deployment control-plane --replicas=3 -n naijcloud
   ```

3. **Database emergency access:**
   ```bash
   kubectl port-forward svc/postgres 5432:5432 -n naijcloud
   psql -h localhost -U naijcloud
   ```
