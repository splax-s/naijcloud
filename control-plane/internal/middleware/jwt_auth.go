package middleware

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/services"
)

type JWTMiddleware struct {
	authService *services.AuthService
}

type AuthContext struct {
	UserID    uuid.UUID
	Email     string
	Role      string
	OrgID     *uuid.UUID
	TokenType string
}

const AuthContextKey = "auth_context"

func NewJWTMiddleware(db *sql.DB) *JWTMiddleware {
	return &JWTMiddleware{
		authService: services.NewAuthService(db),
	}
}

// RequireAuth validates JWT token and sets user context
func (m *JWTMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Missing or invalid authorization header",
			})
			c.Abort()
			return
		}

		claims, err := m.authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set auth context
		authCtx := &AuthContext{
			UserID:    claims.UserID,
			Email:     claims.Email,
			Role:      claims.Role,
			OrgID:     claims.OrgID,
			TokenType: claims.TokenType,
		}

		c.Set(AuthContextKey, authCtx)
		c.Next()
	}
}

// RequireEmailVerified ensures user has verified their email
func (m *JWTMiddleware) RequireEmailVerified() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		// TODO: Add email verification check
		// For now, assume all authenticated users are verified since login requires it

		c.Next()
	}
}

// RequireRole ensures user has specific role
func (m *JWTMiddleware) RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		// Check if user has any of the allowed roles
		hasRole := false
		for _, role := range allowedRoles {
			if authCtx.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireOrganization ensures user belongs to an organization
func (m *JWTMiddleware) RequireOrganization() gin.HandlerFunc {
	return func(c *gin.Context) {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authentication required",
			})
			c.Abort()
			return
		}

		if authCtx.OrgID == nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "Organization context required",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware provides rate limiting by user
func (m *JWTMiddleware) RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	type rateLimitEntry struct {
		requests  int
		resetTime time.Time
	}

	rateLimitMap := make(map[uuid.UUID]*rateLimitEntry)

	return func(c *gin.Context) {
		authCtx := GetAuthContext(c)
		if authCtx == nil {
			// Allow unauthenticated requests to pass through
			c.Next()
			return
		}

		now := time.Now()
		userID := authCtx.UserID

		// Check rate limit
		entry, exists := rateLimitMap[userID]
		if !exists || now.After(entry.resetTime) {
			// Reset or create new entry
			rateLimitMap[userID] = &rateLimitEntry{
				requests:  1,
				resetTime: now.Add(time.Minute),
			}
		} else {
			entry.requests++
			if entry.requests > requestsPerMinute {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please try again later.",
					"retry_after": int(entry.resetTime.Sub(now).Seconds()),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		
		// Only add HSTS in production
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// CORSWithAuth middleware for cross-origin requests with authentication support
func CORSWithAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// In production, you should whitelist specific origins
		allowedOrigins := []string{
			"http://localhost:3000", // Development frontend
			"http://localhost:3001", // Alternative dev port
		}

		isAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				isAllowed = true
				break
			}
		}

		if isAllowed || gin.Mode() != gin.ReleaseMode {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Helper functions

// extractToken extracts JWT token from Authorization header
func extractToken(c *gin.Context) string {
	bearerToken := c.Request.Header.Get("Authorization")
	if len(strings.Split(bearerToken, " ")) == 2 {
		return strings.Split(bearerToken, " ")[1]
	}
	return ""
}

// GetAuthContext retrieves auth context from gin context
func GetAuthContext(c *gin.Context) *AuthContext {
	if authCtx, exists := c.Get(AuthContextKey); exists {
		if auth, ok := authCtx.(*AuthContext); ok {
			return auth
		}
	}
	return nil
}

// GetUserID is a convenience function to get user ID from context
func GetUserID(c *gin.Context) *uuid.UUID {
	authCtx := GetAuthContext(c)
	if authCtx != nil {
		return &authCtx.UserID
	}
	return nil
}

// GetOrganizationID is a convenience function to get organization ID from context
func GetOrganizationID(c *gin.Context) *uuid.UUID {
	authCtx := GetAuthContext(c)
	if authCtx != nil {
		return authCtx.OrgID
	}
	return nil
}

// SetAuthContext sets auth context (useful for testing)
func SetAuthContext(c *gin.Context, userID uuid.UUID, email, role string, orgID *uuid.UUID) {
	authCtx := &AuthContext{
		UserID:    userID,
		Email:     email,
		Role:      role,
		OrgID:     orgID,
		TokenType: "access",
	}
	c.Set(AuthContextKey, authCtx)
}
