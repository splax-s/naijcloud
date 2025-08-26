# NaijCloud Control Plane

The control plane is the central management service for the NaijCloud D-CDN. It provides REST APIs for domain registration, edge node management, cache policies, and analytics.

## Features

- **Domain Management**: Register, configure, and manage domains
- **Edge Node Registry**: Track and manage edge proxy instances
- **Cache Control**: Purge cache and manage cache policies
- **Analytics**: Request metrics and performance analytics
- **Health Monitoring**: Edge node heartbeats and health tracking

## API Endpoints

### Domains

- `GET /v1/domains` - List all domains
- `POST /v1/domains` - Register a new domain
- `GET /v1/domains/{domain}` - Get domain configuration
- `PUT /v1/domains/{domain}` - Update domain configuration
- `DELETE /v1/domains/{domain}` - Delete domain
- `POST /v1/domains/{domain}/purge` - Purge domain cache

### Edge Nodes

- `GET /v1/edges` - List all edge nodes
- `POST /v1/edges` - Register a new edge node
- `GET /v1/edges/{edge_id}` - Get edge node details
- `POST /v1/edges/{edge_id}/heartbeat` - Update edge heartbeat
- `DELETE /v1/edges/{edge_id}` - Deregister edge node

### Analytics

- `GET /v1/analytics/domains/{domain}` - Get domain analytics
- `GET /v1/analytics/domains/{domain}/paths` - Get top requested paths
- `GET /v1/analytics/domains/{domain}/timeline` - Get request timeline

## Configuration

Environment variables:

- `DATABASE_URL` - PostgreSQL connection string (required)
- `REDIS_URL` - Redis connection string (required)
- `PORT` - HTTP server port (default: 8080)
- `METRICS_PORT` - Metrics server port (default: 9091)
- `LOG_LEVEL` - Log level: debug, info, warn, error (default: info)
- `JWT_SECRET` - JWT signing secret (default: dev-secret-change-in-production)

## Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run with race detection
go test -race ./...

# Build
go build -o control-plane

# Run locally
export DATABASE_URL="postgres://user:pass@localhost:5432/naijcloud?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
./control-plane
```

## Docker

```bash
# Build image
docker build -t naijcloud/control-plane .

# Run container
docker run -p 8080:8080 -p 9091:9091 \
  -e DATABASE_URL="postgres://user:pass@db:5432/naijcloud" \
  -e REDIS_URL="redis://redis:6379" \
  naijcloud/control-plane
```

## Metrics

Prometheus metrics are exposed on `/metrics` endpoint (port 9091 by default):

- `http_requests_total` - Total HTTP requests by method, path, and status
- `http_request_duration_seconds` - HTTP request duration histogram

## Health Check

- `GET /health` - Returns 200 if service is healthy and database is accessible
