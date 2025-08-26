# NaijCloud API Documentation

This document describes the NaijCloud D-CDN REST API for managing domains, configurations, and monitoring your CDN infrastructure.

## Base URL

```
Production: https://api.naijcloud.com/v1
Development: http://localhost:8080/v1
```

## Authentication

All API requests require authentication using API keys or bearer tokens.

### API Key Authentication

Include your API key in the request header:

```http
Authorization: Bearer your-api-key-here
```

### Session Authentication

For dashboard access, session cookies are used:

```http
Cookie: next-auth.session-token=...
```

## Rate Limiting

API requests are rate limited to:
- 1000 requests per hour for authenticated users
- 100 requests per hour for unauthenticated requests

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## Response Format

All API responses follow a consistent JSON format:

### Success Response

```json
{
  "success": true,
  "data": {
    // Response data here
  },
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z",
    "request_id": "req_12345"
  }
}
```

### Error Response

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid domain name format",
    "details": {
      "field": "domain",
      "value": "invalid-domain"
    }
  },
  "meta": {
    "timestamp": "2024-01-01T00:00:00Z",
    "request_id": "req_12345"
  }
}
```

## Domains API

### List Domains

List all domains in your account.

```http
GET /v1/domains
```

**Query Parameters:**
- `page` (integer, optional): Page number (default: 1)
- `limit` (integer, optional): Items per page (default: 20, max: 100)
- `status` (string, optional): Filter by status (`active`, `pending`, `suspended`)
- `search` (string, optional): Search by domain name

**Response:**

```json
{
  "success": true,
  "data": {
    "domains": [
      {
        "id": "dom_12345",
        "domain": "example.com",
        "status": "active",
        "origin": "origin.example.com",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z",
        "ssl_enabled": true,
        "cache_rules": [
          {
            "path": "*.jpg",
            "ttl": 86400,
            "cache_key": "url"
          }
        ],
        "stats": {
          "requests_24h": 50000,
          "bandwidth_24h": 1073741824,
          "cache_hit_ratio": 0.85
        }
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 1,
      "pages": 1
    }
  }
}
```

### Get Domain

Get details for a specific domain.

```http
GET /v1/domains/{domain_id}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "dom_12345",
    "domain": "example.com",
    "status": "active",
    "origin": "origin.example.com",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z",
    "ssl_enabled": true,
    "ssl_cert": {
      "status": "valid",
      "expires_at": "2024-12-31T23:59:59Z",
      "issuer": "Let's Encrypt"
    },
    "cache_rules": [
      {
        "id": "rule_123",
        "path": "*.jpg",
        "ttl": 86400,
        "cache_key": "url",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "edge_locations": [
      {
        "location": "us-east-1",
        "status": "active",
        "requests_24h": 25000
      },
      {
        "location": "eu-west-1", 
        "status": "active",
        "requests_24h": 15000
      }
    ]
  }
}
```

### Create Domain

Add a new domain to your CDN.

```http
POST /v1/domains
```

**Request Body:**

```json
{
  "domain": "example.com",
  "origin": "origin.example.com",
  "ssl_enabled": true,
  "cache_rules": [
    {
      "path": "*.jpg",
      "ttl": 86400,
      "cache_key": "url"
    }
  ]
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "dom_12345",
    "domain": "example.com",
    "status": "pending",
    "cname_target": "dom-12345.naijcloud.com",
    "verification": {
      "method": "dns",
      "record": {
        "type": "TXT",
        "name": "_naijcloud-verify.example.com",
        "value": "verify-12345"
      }
    }
  }
}
```

### Update Domain

Update an existing domain configuration.

```http
PUT /v1/domains/{domain_id}
```

**Request Body:**

```json
{
  "origin": "new-origin.example.com",
  "ssl_enabled": true,
  "cache_rules": [
    {
      "path": "*.css",
      "ttl": 604800,
      "cache_key": "url"
    }
  ]
}
```

### Delete Domain

Remove a domain from your CDN.

```http
DELETE /v1/domains/{domain_id}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "message": "Domain scheduled for deletion",
    "deletion_date": "2024-01-08T00:00:00Z"
  }
}
```

## Cache API

### Purge Cache

Purge cached content for a domain.

```http
POST /v1/domains/{domain_id}/purge
```

**Request Body:**

```json
{
  "type": "selective",
  "paths": [
    "/images/*",
    "/css/style.css",
    "/api/data.json"
  ]
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "purge_id": "purge_12345",
    "status": "in_progress",
    "estimated_completion": "2024-01-01T00:05:00Z"
  }
}
```

### Get Purge Status

Check the status of a cache purge operation.

```http
GET /v1/purges/{purge_id}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "purge_12345",
    "status": "completed",
    "domain_id": "dom_12345",
    "type": "selective",
    "paths": ["/images/*"],
    "created_at": "2024-01-01T00:00:00Z",
    "completed_at": "2024-01-01T00:04:32Z",
    "edge_locations_purged": [
      "us-east-1",
      "eu-west-1",
      "ap-southeast-1"
    ]
  }
}
```

## Analytics API

### Domain Statistics

Get analytics data for a domain.

```http
GET /v1/domains/{domain_id}/stats
```

**Query Parameters:**
- `start_date` (string, required): Start date (ISO 8601 format)
- `end_date` (string, required): End date (ISO 8601 format)
- `granularity` (string, optional): Data granularity (`hour`, `day`, `week`, `month`)
- `metrics` (string, optional): Comma-separated list of metrics

**Response:**

```json
{
  "success": true,
  "data": {
    "domain_id": "dom_12345",
    "period": {
      "start": "2024-01-01T00:00:00Z",
      "end": "2024-01-02T00:00:00Z"
    },
    "summary": {
      "total_requests": 100000,
      "total_bandwidth": 10737418240,
      "cache_hit_ratio": 0.85,
      "avg_response_time": 150,
      "error_rate": 0.001
    },
    "time_series": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "requests": 4167,
        "bandwidth": 447392768,
        "cache_hits": 3542,
        "cache_misses": 625,
        "response_time_p95": 180,
        "status_codes": {
          "2xx": 4100,
          "3xx": 50,
          "4xx": 15,
          "5xx": 2
        }
      }
    ],
    "top_content": [
      {
        "path": "/images/hero.jpg",
        "requests": 5000,
        "bandwidth": 524288000
      }
    ],
    "geographic_distribution": [
      {
        "country": "US",
        "requests": 60000,
        "bandwidth": 6442450944
      },
      {
        "country": "GB", 
        "requests": 25000,
        "bandwidth": 2684354560
      }
    ]
  }
}
```

### Real-time Statistics

Get real-time statistics for a domain.

```http
GET /v1/domains/{domain_id}/stats/realtime
```

**Response:**

```json
{
  "success": true,
  "data": {
    "domain_id": "dom_12345",
    "timestamp": "2024-01-01T12:30:00Z",
    "requests_per_second": 120,
    "bandwidth_per_second": 12582912,
    "active_connections": 450,
    "cache_hit_ratio": 0.87,
    "edge_locations": [
      {
        "location": "us-east-1",
        "requests_per_second": 60,
        "bandwidth_per_second": 6291456,
        "response_time_avg": 45
      },
      {
        "location": "eu-west-1",
        "requests_per_second": 40,
        "bandwidth_per_second": 4194304,
        "response_time_avg": 55
      }
    ]
  }
}
```

## Edge Locations API

### List Edge Locations

Get all available edge locations.

```http
GET /v1/edge-locations
```

**Response:**

```json
{
  "success": true,
  "data": {
    "edge_locations": [
      {
        "id": "us-east-1",
        "name": "US East (N. Virginia)",
        "country": "US",
        "city": "Ashburn",
        "status": "active",
        "capacity_utilization": 0.65,
        "latency_ms": 15
      },
      {
        "id": "eu-west-1",
        "name": "EU West (Ireland)",
        "country": "IE", 
        "city": "Dublin",
        "status": "active",
        "capacity_utilization": 0.72,
        "latency_ms": 25
      }
    ]
  }
}
```

## SSL Certificates API

### Get SSL Certificate

Get SSL certificate information for a domain.

```http
GET /v1/domains/{domain_id}/ssl
```

**Response:**

```json
{
  "success": true,
  "data": {
    "domain_id": "dom_12345",
    "certificate": {
      "status": "valid",
      "type": "lets_encrypt",
      "domains": ["example.com", "www.example.com"],
      "issued_at": "2024-01-01T00:00:00Z",
      "expires_at": "2024-04-01T00:00:00Z",
      "auto_renew": true,
      "fingerprint": "SHA256:ABCD1234..."
    }
  }
}
```

### Renew SSL Certificate

Manually renew an SSL certificate.

```http
POST /v1/domains/{domain_id}/ssl/renew
```

**Response:**

```json
{
  "success": true,
  "data": {
    "renewal_id": "renewal_12345",
    "status": "in_progress",
    "estimated_completion": "2024-01-01T00:10:00Z"
  }
}
```

## Webhooks API

### List Webhooks

Get all configured webhooks.

```http
GET /v1/webhooks
```

**Response:**

```json
{
  "success": true,
  "data": {
    "webhooks": [
      {
        "id": "webhook_12345",
        "url": "https://example.com/webhook",
        "events": ["domain.created", "domain.ssl_renewed"],
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z",
        "last_delivery": {
          "timestamp": "2024-01-01T12:00:00Z",
          "status": "success",
          "response_time": 150
        }
      }
    ]
  }
}
```

### Create Webhook

Register a new webhook endpoint.

```http
POST /v1/webhooks
```

**Request Body:**

```json
{
  "url": "https://example.com/webhook",
  "events": ["domain.created", "domain.updated", "cache.purged"],
  "secret": "webhook_secret_key"
}
```

## Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Request validation failed |
| `AUTHENTICATION_ERROR` | Invalid or missing authentication |
| `AUTHORIZATION_ERROR` | Insufficient permissions |
| `NOT_FOUND` | Resource not found |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded |
| `DOMAIN_EXISTS` | Domain already exists |
| `DOMAIN_VERIFICATION_FAILED` | Domain verification failed |
| `SSL_PROVISIONING_ERROR` | SSL certificate provisioning failed |
| `INTERNAL_ERROR` | Internal server error |

## SDKs and Libraries

### JavaScript/Node.js

```bash
npm install naijcloud-sdk
```

```javascript
const NaijCloud = require('naijcloud-sdk');

const client = new NaijCloud({
  apiKey: 'your-api-key',
  baseURL: 'https://api.naijcloud.com/v1'
});

// List domains
const domains = await client.domains.list();

// Create domain
const domain = await client.domains.create({
  domain: 'example.com',
  origin: 'origin.example.com'
});

// Purge cache
await client.cache.purge('dom_12345', {
  type: 'selective',
  paths: ['/images/*']
});
```

### Python

```bash
pip install naijcloud-python
```

```python
from naijcloud import NaijCloudClient

client = NaijCloudClient(
    api_key='your-api-key',
    base_url='https://api.naijcloud.com/v1'
)

# List domains
domains = client.domains.list()

# Create domain
domain = client.domains.create(
    domain='example.com',
    origin='origin.example.com'
)

# Get analytics
stats = client.analytics.get_stats(
    domain_id='dom_12345',
    start_date='2024-01-01',
    end_date='2024-01-02'
)
```

### Go

```bash
go get github.com/naijcloud/naijcloud-go
```

```go
package main

import (
    "github.com/naijcloud/naijcloud-go"
)

func main() {
    client := naijcloud.NewClient("your-api-key")
    
    // List domains
    domains, err := client.Domains.List()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create domain
    domain, err := client.Domains.Create(&naijcloud.CreateDomainRequest{
        Domain: "example.com",
        Origin: "origin.example.com",
    })
}
```

## Webhook Events

When webhooks are configured, NaijCloud will send HTTP POST requests to your endpoint for relevant events.

### Event Types

- `domain.created` - New domain added
- `domain.updated` - Domain configuration changed
- `domain.deleted` - Domain removed
- `domain.verified` - Domain verification completed
- `ssl.provisioned` - SSL certificate provisioned
- `ssl.renewed` - SSL certificate renewed
- `ssl.expired` - SSL certificate expired
- `cache.purged` - Cache purge completed
- `edge.node_down` - Edge node went offline
- `edge.node_up` - Edge node came online

### Webhook Payload

```json
{
  "event": "domain.created",
  "timestamp": "2024-01-01T00:00:00Z",
  "data": {
    "domain": {
      "id": "dom_12345",
      "domain": "example.com",
      "status": "pending"
    }
  },
  "meta": {
    "webhook_id": "webhook_12345",
    "delivery_id": "delivery_12345"
  }
}
```

### Webhook Security

Webhooks include a signature header for verification:

```http
X-NaijCloud-Signature: sha256=abcd1234...
```

Verify the signature using your webhook secret:

```javascript
const crypto = require('crypto');

function verifyWebhook(payload, signature, secret) {
  const expectedSignature = crypto
    .createHmac('sha256', secret)
    .update(payload)
    .digest('hex');
  
  return `sha256=${expectedSignature}` === signature;
}
```
