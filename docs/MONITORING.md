# Monitoring and Observability Guide

This guide covers monitoring, logging, and observability for NaijCloud D-CDN in production.

## Overview

The NaijCloud monitoring stack includes:
- **Metrics**: Prometheus for collection, Grafana for visualization
- **Logging**: Centralized logging with structured output
- **Tracing**: Distributed tracing for request flows
- **Alerting**: Alert Manager for notifications
- **Health Checks**: Built-in health endpoints and probes

## Metrics Collection

### Prometheus Configuration

The included Prometheus configuration scrapes metrics from all services:

```yaml
# Prometheus scrape configuration
scrape_configs:
  - job_name: 'control-plane'
    static_configs:
      - targets: ['control-plane:9091']
    metrics_path: /metrics
    scrape_interval: 15s

  - job_name: 'edge-proxy'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: edge-proxy
      - source_labels: [__meta_kubernetes_pod_ip]
        target_label: __address__
        replacement: ${1}:8081
```

### Application Metrics

#### Control Plane Metrics

**HTTP Request Metrics:**
- `http_requests_total` - Total HTTP requests by method and status
- `http_request_duration_seconds` - Request duration histogram
- `http_requests_in_flight` - Current number of requests being served

**Business Metrics:**
- `naijcloud_domains_total` - Total number of managed domains
- `naijcloud_cache_hits_total` - Cache hit counter
- `naijcloud_cache_misses_total` - Cache miss counter
- `naijcloud_edge_nodes_active` - Number of active edge nodes

**Database Metrics:**
- `database_connections_active` - Active database connections
- `database_query_duration_seconds` - Query duration histogram
- `database_queries_total` - Total queries by operation

**Example queries:**
```promql
# Request rate per second
rate(http_requests_total[5m])

# 95th percentile response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate percentage
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) * 100
```

#### Edge Proxy Metrics

**Proxy Metrics:**
- `proxy_requests_total` - Total proxy requests
- `proxy_bytes_sent` - Bytes sent to clients
- `proxy_bytes_received` - Bytes received from origins
- `proxy_cache_status` - Cache status (hit/miss/bypass)

**Performance Metrics:**
- `proxy_upstream_duration_seconds` - Time to fetch from origin
- `proxy_response_time_seconds` - Total response time
- `proxy_connections_active` - Active proxy connections

**CDN Metrics:**
- `cdn_bandwidth_bytes_per_second` - Current bandwidth usage
- `cdn_storage_bytes_used` - Cache storage utilization
- `cdn_cache_evictions_total` - Cache evictions

### Custom Metrics

#### Adding Custom Metrics

**Go applications (Control Plane/Edge Proxy):**
```go
import "github.com/prometheus/client_golang/prometheus"

// Counter for business events
var domainsCreated = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "naijcloud_domains_created_total",
        Help: "Total number of domains created",
    },
    []string{"region", "plan"},
)

// Histogram for operation duration
var operationDuration = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Name: "naijcloud_operation_duration_seconds",
        Help: "Duration of operations",
        Buckets: prometheus.DefBuckets,
    },
    []string{"operation", "status"},
)
```

**Next.js Dashboard:**
```javascript
// Custom metrics endpoint
export default function handler(req, res) {
  const metrics = [
    '# HELP dashboard_page_views_total Total page views',
    '# TYPE dashboard_page_views_total counter',
    `dashboard_page_views_total{page="${req.query.page}"} ${getPageViews()}`,
    
    '# HELP dashboard_users_active Current active users',
    '# TYPE dashboard_users_active gauge',
    `dashboard_users_active ${getActiveUsers()}`,
  ];
  
  res.setHeader('Content-Type', 'text/plain');
  res.send(metrics.join('\n'));
}
```

## Grafana Dashboards

### Pre-built Dashboards

1. **Infrastructure Overview**
   - Cluster resource utilization
   - Node health and performance
   - Network and storage metrics

2. **Application Performance**
   - Request rates and response times
   - Error rates and status codes
   - Cache hit ratios

3. **Business Metrics**
   - Domain management metrics
   - User activity and growth
   - Revenue and usage tracking

### Custom Dashboard JSON

```json
{
  "dashboard": {
    "title": "NaijCloud Control Plane",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total{job=\"control-plane\"}[5m])",
            "legendFormat": "{{method}} {{status}}"
          }
        ]
      },
      {
        "title": "Response Time P95",
        "type": "singlestat",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job=\"control-plane\"}[5m]))",
            "legendFormat": "P95"
          }
        ]
      }
    ]
  }
}
```

## Logging

### Structured Logging

All applications use structured JSON logging:

**Go applications:**
```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("Processing request",
    zap.String("method", "GET"),
    zap.String("path", "/api/domains"),
    zap.String("user_id", userID),
    zap.Duration("duration", time.Since(start)),
)
```

**Next.js Dashboard:**
```javascript
const logger = {
  info: (message, meta = {}) => {
    console.log(JSON.stringify({
      level: 'info',
      message,
      timestamp: new Date().toISOString(),
      ...meta
    }));
  }
};

logger.info('User login', {
  user_id: session.user.id,
  method: 'credentials'
});
```

### Log Aggregation

#### Using Fluentd/Fluent Bit

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
data:
  fluent-bit.conf: |
    [SERVICE]
        Flush         5
        Log_Level     info
        Daemon        off
        Parsers_File  parsers.conf

    [INPUT]
        Name              tail
        Tag               kube.*
        Path              /var/log/containers/*naijcloud*.log
        Parser            json
        DB                /var/log/flb_kube.db
        Mem_Buf_Limit     50MB

    [OUTPUT]
        Name  es
        Match *
        Host  elasticsearch.logging.svc.cluster.local
        Port  9200
        Index naijcloud-logs
```

#### Using Loki

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: promtail-config
data:
  promtail.yaml: |
    server:
      http_listen_port: 9080
      grpc_listen_port: 0

    clients:
      - url: http://loki:3100/loki/api/v1/push

    scrape_configs:
      - job_name: kubernetes-pods
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            action: keep
            regex: '(control-plane|edge-proxy|dashboard)'
```

### Log Queries

**Common log queries:**

```bash
# Error logs in the last hour
kubectl logs -n naijcloud --since=1h | grep '"level":"error"'

# Specific user activity
kubectl logs -n naijcloud | grep '"user_id":"user123"'

# API endpoint performance
kubectl logs -n naijcloud | grep '/api/domains' | jq '.duration'
```

## Alerting

### Alert Rules

**Prometheus alert rules:**

```yaml
groups:
  - name: naijcloud.rules
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }} for {{ $labels.job }}"

      - alert: HighResponseTime
        expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High response time"
          description: "95th percentile response time is {{ $value }}s"

      - alert: DatabaseConnectionsHigh
        expr: database_connections_active / database_connections_max > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connections running high"

      - alert: EdgeNodeDown
        expr: up{job="edge-proxy"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Edge node is down"
          description: "Edge node {{ $labels.instance }} is not responding"
```

### Alert Manager Configuration

```yaml
global:
  smtp_smarthost: 'localhost:587'
  smtp_from: 'alerts@naijcloud.com'

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'web.hook'

receivers:
  - name: 'web.hook'
    email_configs:
      - to: 'admin@naijcloud.com'
        subject: 'NaijCloud Alert: {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          {{ end }}
    
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#alerts'
        title: 'NaijCloud Alert'
        text: '{{ .CommonAnnotations.summary }}'
```

### PagerDuty Integration

```yaml
receivers:
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_SERVICE_KEY'
        description: '{{ .CommonAnnotations.summary }}'
        severity: '{{ .CommonLabels.severity }}'
```

## Health Checks

### Application Health Endpoints

**Control Plane:**
```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    health := struct {
        Status   string            `json:"status"`
        Checks   map[string]string `json:"checks"`
        Version  string            `json:"version"`
        Uptime   string            `json:"uptime"`
    }{
        Status:  "ok",
        Version: version.Version,
        Uptime:  time.Since(startTime).String(),
        Checks: map[string]string{
            "database": checkDatabase(),
            "redis":    checkRedis(),
        },
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(health)
}
```

**Dashboard:**
```javascript
// pages/api/health.js
export default async function handler(req, res) {
  try {
    // Check database connection
    const dbCheck = await checkDatabase();
    
    // Check external APIs
    const apiCheck = await fetch('http://control-plane:8080/health');
    
    res.status(200).json({
      status: 'ok',
      checks: {
        database: dbCheck ? 'ok' : 'error',
        api: apiCheck.ok ? 'ok' : 'error'
      },
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    res.status(500).json({
      status: 'error',
      error: error.message
    });
  }
}
```

### Kubernetes Health Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

## Distributed Tracing

### OpenTelemetry Integration

**Go applications:**
```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/jaeger"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracing() {
    exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://jaeger:14268/api/traces")))
    if err != nil {
        log.Fatal(err)
    }
    
    tp := trace.NewTracerProvider(
        trace.WithBatcher(exp),
        trace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String("control-plane"),
        )),
    )
    
    otel.SetTracerProvider(tp)
}
```

### Jaeger Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:latest
        ports:
        - containerPort: 16686
        - containerPort: 14268
        env:
        - name: COLLECTOR_ZIPKIN_HTTP_PORT
          value: "9411"
```

## Performance Monitoring

### SLA/SLO Definitions

**Service Level Objectives:**
- API Availability: 99.9% uptime
- Response Time: 95% of requests < 500ms
- Error Rate: < 1% of total requests
- Cache Hit Ratio: > 80%

**SLI Queries:**
```promql
# Availability
(sum(rate(http_requests_total[5m])) - sum(rate(http_requests_total{status=~"5.."}[5m]))) / sum(rate(http_requests_total[5m]))

# Response Time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error Rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
```

### Load Testing

**k6 load test script:**
```javascript
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 100 },
    { duration: '2m', target: 200 },
    { duration: '5m', target: 200 },
    { duration: '2m', target: 0 },
  ],
};

export default function() {
  let response = http.get('https://api.naijcloud.com/v1/domains');
  check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
}
```

## Troubleshooting

### Common Monitoring Issues

1. **Missing Metrics:**
   ```bash
   # Check if metrics endpoints are accessible
   kubectl port-forward svc/control-plane 9091:9091 -n naijcloud
   curl http://localhost:9091/metrics
   ```

2. **High Memory Usage:**
   ```bash
   # Check Prometheus memory usage
   kubectl exec -it prometheus-0 -n monitoring -- du -sh /prometheus
   
   # Reduce retention or increase resources
   kubectl patch statefulset prometheus -n monitoring -p '{"spec":{"template":{"spec":{"containers":[{"name":"prometheus","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
   ```

3. **Alert Fatigue:**
   ```yaml
   # Adjust alert thresholds
   - alert: HighErrorRate
     expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.10  # Increased from 0.05
     for: 10m  # Increased from 5m
   ```

### Debugging Checklist

- [ ] Are all pods running and ready?
- [ ] Are metrics endpoints accessible?
- [ ] Is Prometheus scraping all targets?
- [ ] Are alert rules loading correctly?
- [ ] Is Alert Manager routing notifications?
- [ ] Are dashboards showing data?
- [ ] Are logs being collected?
- [ ] Are traces being generated?
