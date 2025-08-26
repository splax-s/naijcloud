# NaijCloud - Decentralized CDN / Proxy Network (D-CDN) MVP

A minimal, production-minded decentralized CDN with edge proxy nodes, control plane, and admin dashboard.

## Architecture Overview

```text
[DNS Layer] → [Edge Regions] → [Control Plane] → [Admin Dashboard]
                     ↓
              [Observability Stack]
```

## Services

- **Edge Proxy** (Go): HTTP/S proxying, caching, TLS termination, rate limiting
- **Control Plane** (Go): Domain/edge management, cache policies, metrics collection
- **Admin Dashboard** (Next.js): Domain management UI, analytics dashboard
- **Observability**: Prometheus, Loki, Grafana

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Kubernetes cluster (optional, for production deployment)
- Go 1.21+
- Node.js 18+

### Local Development

```bash
# Start infrastructure services
./scripts/dev-setup.sh

# Test the API
./scripts/test-api.sh

# View logs
docker-compose logs -f control-plane

# Stop all services
docker-compose down
```

### Production Deployment

```bash
# Deploy with Helm
helm install naijcloud ./helm/naijcloud \
  --set controlPlane.database.host=postgres.example.com \
  --set redis.host=redis.example.com
```

## Repository Structure

```text
├── edge-proxy/          # Go-based edge proxy service
├── control-plane/       # Go-based control plane API
├── dashboard/           # Next.js admin dashboard
├── helm/               # Kubernetes Helm charts
├── scripts/            # Deployment and testing scripts
├── docs/               # Documentation
├── .github/workflows/  # CI/CD pipelines
├── docker-compose.yml  # Local development setup
└── README.md
```

## API Documentation

See [api-spec.yaml](./api-spec.yaml) for the complete OpenAPI specification.

## Database Schema

See [schema.sql](./schema.sql) for the PostgreSQL database schema.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for your changes
4. Ensure CI passes
5. Submit a pull request

## License

MIT License - see [LICENSE](./LICENSE) for details.
