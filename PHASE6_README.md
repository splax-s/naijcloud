# Phase 6: Complete Advanced API Platform

## Overview

Phase 6 implements a comprehensive advanced API platform with enterprise-grade features including JWT authentication, activity logging, notifications, enhanced middleware, and complete Docker containerization.

## 🚀 Features Implemented

### 1. JWT Authentication System
- **Access & Refresh Tokens**: Secure token-based authentication with automatic renewal
- **Token Management**: Secure storage, validation, and revocation
- **Password Security**: bcrypt hashing with activity logging
- **Session Management**: Redis-backed session storage

### 2. Activity Logging & Audit Trail
- **Comprehensive Logging**: All user actions tracked with metadata
- **Filtering & Search**: Advanced querying capabilities
- **Export Functionality**: CSV/JSON export for compliance
- **Automatic Cleanup**: Configurable retention policies

### 3. Notification System
- **In-App Notifications**: Real-time user notifications
- **User Preferences**: Customizable notification settings
- **Email Integration**: Mailhog for development email testing
- **Notification Types**: Security alerts, system updates, user actions

### 4. Enhanced Middleware
- **Rate Limiting**: Redis-backed rate limiting per user/IP
- **Security Headers**: CORS, XSS protection, content type validation
- **Role-Based Access Control**: Fine-grained permission system
- **Request Logging**: Structured logging with correlation IDs

### 5. Database Enhancements
- **Migration System**: Structured database versioning
- **PostgreSQL 15**: Latest stable version with performance optimizations
- **Indexing**: Optimized indexes for performance
- **JSONB Storage**: Flexible metadata storage

### 6. Docker Containerization
- **Multi-Service Setup**: PostgreSQL, Redis, Mailhog, Monitoring
- **Health Checks**: Service dependency management
- **Development Workflow**: Hot reloading and debugging support
- **Monitoring Stack**: Prometheus, Grafana, Loki integration

## 📁 Project Structure

```
naijcloud/
├── control-plane/
│   ├── internal/
│   │   ├── services/
│   │   │   ├── auth_service.go          # Enhanced JWT authentication
│   │   │   ├── activity_service.go      # Activity logging system
│   │   │   └── notification_service.go  # Notification management
│   │   ├── middleware/
│   │   │   └── jwt_auth.go             # JWT middleware & security
│   │   ├── api/
│   │   │   └── enhanced_auth_handlers.go # Enhanced API endpoints
│   │   └── models/
│   │       ├── refresh_token.go        # Token models
│   │       ├── activity_log.go         # Activity models
│   │       └── notification.go         # Notification models
│   ├── migrations/
│   │   ├── 005_add_refresh_tokens.sql  # JWT token storage
│   │   ├── 006_add_activity_logs.sql   # Activity logging
│   │   └── 007_add_notifications.sql   # Notification system
│   └── Dockerfile                      # Application container
├── docker-compose.yml                  # Multi-service orchestration
└── scripts/
    └── test-phase6.sh                  # Comprehensive test suite
```

## 🛠️ Getting Started

### Prerequisites
- Docker & Docker Compose
- curl & jq (for testing)
- Go 1.21+ (for development)

### Quick Start

1. **Start all services**:
   ```bash
   docker-compose up -d
   ```

2. **Wait for services to be ready**:
   ```bash
   docker-compose logs -f control-plane
   ```

3. **Run integration tests**:
   ```bash
   ./scripts/test-phase6.sh
   ```

### Service URLs
- **API**: http://localhost:8080
- **Mailhog Web UI**: http://localhost:8025
- **Metrics**: http://localhost:9091/metrics
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090

## 🔐 Authentication Flow

### Registration & Login
```bash
# Register new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "full_name": "John Doe",
    "phone": "+1234567890"
  }'

# Response includes access_token and refresh_token
```

### Token Refresh
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your_refresh_token"}'
```

### Password Change
```bash
curl -X POST http://localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer your_access_token" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "current_pass",
    "new_password": "new_pass"
  }'
```

## 📊 Activity Logging

### View Activities
```bash
curl -X GET "http://localhost:8080/api/v1/auth/activities?limit=10&activity_type=login" \
  -H "Authorization: Bearer your_access_token"
```

### Activity Types Tracked
- `user_registered` - User account creation
- `user_login` - Successful login
- `user_logout` - User logout
- `password_changed` - Password updates
- `token_refreshed` - Token renewal
- `profile_updated` - Profile changes

## 🔔 Notifications

### Get Notifications
```bash
curl -X GET "http://localhost:8080/api/v1/auth/notifications?unread_only=true" \
  -H "Authorization: Bearer your_access_token"
```

### Update Preferences
```bash
curl -X PUT http://localhost:8080/api/v1/auth/notification-preferences \
  -H "Authorization: Bearer your_access_token" \
  -H "Content-Type: application/json" \
  -d '{
    "email_notifications": true,
    "push_notifications": false,
    "security_alerts": true
  }'
```

## 🔧 Configuration

### Environment Variables
```bash
# Database
DATABASE_URL=postgres://user:pass@host:port/db?sslmode=disable

# Redis
REDIS_URL=redis://host:port/db

# JWT
JWT_SECRET=your-super-secret-jwt-key

# Email (Mailhog for development)
SMTP_HOST=mailhog
SMTP_PORT=1025
SMTP_FROM=noreply@naijcloud.dev

# API
PORT=8080
METRICS_PORT=9091
LOG_LEVEL=info
FRONTEND_URL=http://localhost:3001
```

## 📝 Database Migrations

Migrations are automatically applied on startup. Manual migration:

```bash
# Apply migrations
docker-compose exec control-plane migrate -path=/app/migrations -database="$DATABASE_URL" up

# Check migration status
docker-compose exec control-plane migrate -path=/app/migrations -database="$DATABASE_URL" version
```

## 🧪 Testing

### Automated Test Suite
```bash
# Run comprehensive tests
./scripts/test-phase6.sh

# Individual service tests
docker-compose exec control-plane go test ./...
```

### Manual Testing
1. **Health Check**: `curl http://localhost:8080/health`
2. **Metrics**: `curl http://localhost:9091/metrics`
3. **Email Testing**: Check Mailhog UI at http://localhost:8025

## 📈 Monitoring

### Metrics Available
- HTTP request metrics
- Database connection pool metrics
- Redis operation metrics
- JWT token metrics
- Rate limiting metrics

### Grafana Dashboards
- API Performance Dashboard
- Authentication Metrics
- Database Performance
- System Resource Usage

## 🔒 Security Features

### Rate Limiting
- **Per User**: 100 requests per minute
- **Per IP**: 200 requests per minute
- **Redis Backend**: Distributed rate limiting

### Security Headers
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security` (in production)

### CORS Configuration
- Configurable origins
- Credential support
- Preflight handling

## 🐛 Troubleshooting

### Common Issues

1. **Database Connection**:
   ```bash
   docker-compose logs postgres
   docker-compose exec postgres psql -U naijcloud -d naijcloud
   ```

2. **Redis Connection**:
   ```bash
   docker-compose logs redis
   docker-compose exec redis redis-cli ping
   ```

3. **Email Testing**:
   ```bash
   # Check Mailhog logs
   docker-compose logs mailhog
   
   # Test SMTP connection
   telnet localhost 1025
   ```

### Debug Mode
Set `LOG_LEVEL=debug` in docker-compose.yml for verbose logging.

## 📋 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Token refresh
- `POST /api/v1/auth/logout` - User logout
- `POST /api/v1/auth/change-password` - Password change
- `GET /api/v1/auth/profile` - User profile
- `PUT /api/v1/auth/profile` - Update profile

### Activity & Notifications
- `GET /api/v1/auth/activities` - Get user activities
- `GET /api/v1/auth/notifications` - Get notifications
- `PUT /api/v1/auth/notifications/:id/read` - Mark as read
- `GET /api/v1/auth/notification-preferences` - Get preferences
- `PUT /api/v1/auth/notification-preferences` - Update preferences

### System
- `GET /health` - Health check
- `GET /metrics` - Prometheus metrics

## 🚀 Next Steps

Phase 6 provides a solid foundation for:
- Multi-tenant architecture
- Advanced security features
- Real-time capabilities
- Microservices decomposition
- Cloud-native deployment

## 📄 License

This project is part of the NaijCloud platform development.
