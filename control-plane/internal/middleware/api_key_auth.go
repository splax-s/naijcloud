package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/services"
	"github.com/sirupsen/logrus"
)

type APIKeyMiddleware struct {
	apiKeyService *services.APIKeyService
}

func NewAPIKeyMiddleware(apiKeyService *services.APIKeyService) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		apiKeyService: apiKeyService,
	}
}

// RequireAPIKey is middleware that validates API key authentication
// It looks for the API key in the Authorization header with "Bearer" prefix
// or in the X-API-Key header
func (m *APIKeyMiddleware) RequireAPIKey() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		logrus.Info("DEBUG: API Key middleware called")
		var apiKey string

		// Check Authorization header first (Bearer token format)
		authHeader := c.GetHeader("Authorization")
		logrus.WithField("auth_header", authHeader).Info("DEBUG: Authorization header")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Fallback to X-API-Key header
			apiKey = c.GetHeader("X-API-Key")
		}

		logrus.WithField("api_key", apiKey).Info("DEBUG: Extracted API key")
		if apiKey == "" {
			logrus.Info("DEBUG: No API key provided")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "API key required",
				"details": "Provide API key in Authorization header (Bearer token) or X-API-Key header",
			})
			c.Abort()
			return
		}

		logrus.Info("DEBUG: Calling AuthenticateAPIKey")
		// Authenticate the API key
		authenticatedKey, err := m.apiKeyService.AuthenticateAPIKey(c.Request.Context(), apiKey)
		if err != nil {
			logrus.WithError(err).Info("DEBUG: Authentication failed")
			if err.Error() == "invalid API key" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key [DEBUG: middleware path]"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication failed", "details": err.Error()})
			}
			c.Abort()
			return
		}

		// Set authenticated context
		c.Set("api_key_id", authenticatedKey.ID)
		c.Set("organization_id", authenticatedKey.OrganizationID)
		c.Set("user_id", authenticatedKey.UserID)
		c.Set("api_key_name", authenticatedKey.Name)
		c.Set("api_key_scopes", authenticatedKey.Scopes)
		c.Set("auth_type", "api_key")

		c.Next()
	})
}

// RequireScope is middleware that validates API key has required scope
func (m *APIKeyMiddleware) RequireScope(requiredScope string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Check if this request was authenticated with an API key
		authType, exists := c.Get("auth_type")
		if !exists || authType != "api_key" {
			// If not API key auth, skip scope checking (let other auth handle it)
			c.Next()
			return
		}

		scopes, exists := c.Get("api_key_scopes")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No scopes available"})
			c.Abort()
			return
		}

		apiKeyScopes, ok := scopes.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid scope format"})
			c.Abort()
			return
		}

		// Check if required scope is present
		hasScope := false
		for _, scope := range apiKeyScopes {
			if scope == requiredScope {
				hasScope = true
				break
			}
		}

		if !hasScope {
			c.JSON(http.StatusForbidden, gin.H{
				"error":            "Insufficient permissions",
				"required_scope":   requiredScope,
				"available_scopes": apiKeyScopes,
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// RequireAnyScope is middleware that validates API key has at least one of the required scopes
func (m *APIKeyMiddleware) RequireAnyScope(requiredScopes ...string) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Check if this request was authenticated with an API key
		authType, exists := c.Get("auth_type")
		if !exists || authType != "api_key" {
			// If not API key auth, skip scope checking (let other auth handle it)
			c.Next()
			return
		}

		scopes, exists := c.Get("api_key_scopes")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No scopes available"})
			c.Abort()
			return
		}

		apiKeyScopes, ok := scopes.([]string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid scope format"})
			c.Abort()
			return
		}

		// Check if any required scope is present
		hasAnyScope := false
		for _, requiredScope := range requiredScopes {
			for _, scope := range apiKeyScopes {
				if scope == requiredScope {
					hasAnyScope = true
					break
				}
			}
			if hasAnyScope {
				break
			}
		}

		if !hasAnyScope {
			c.JSON(http.StatusForbidden, gin.H{
				"error":            "Insufficient permissions",
				"required_scopes":  requiredScopes,
				"available_scopes": apiKeyScopes,
			})
			c.Abort()
			return
		}

		c.Next()
	})
}

// Optional API key middleware that tries to authenticate but doesn't require it
// Useful for endpoints that can work with or without authentication
func (m *APIKeyMiddleware) OptionalAPIKey() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		var apiKey string

		// Check Authorization header first (Bearer token format)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			apiKey = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Fallback to X-API-Key header
			apiKey = c.GetHeader("X-API-Key")
		}

		if apiKey != "" {
			// Try to authenticate the API key
			authenticatedKey, err := m.apiKeyService.AuthenticateAPIKey(c.Request.Context(), apiKey)
			if err == nil {
				// Set authenticated context
				c.Set("api_key_id", authenticatedKey.ID)
				c.Set("organization_id", authenticatedKey.OrganizationID)
				c.Set("user_id", authenticatedKey.UserID)
				c.Set("api_key_name", authenticatedKey.Name)
				c.Set("api_key_scopes", authenticatedKey.Scopes)
				c.Set("auth_type", "api_key")
			}
		}

		c.Next()
	})
}

// LogAPIKeyUsage is middleware that logs API key usage for analytics
func (m *APIKeyMiddleware) LogAPIKeyUsage() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Process the request first
		c.Next()

		// Check if this request was authenticated with an API key
		authType, exists := c.Get("auth_type")
		if !exists || authType != "api_key" {
			return
		}

		apiKeyID, exists := c.Get("api_key_id")
		if !exists {
			return
		}

		_, ok := apiKeyID.(uuid.UUID)
		if !ok {
			return
		}

		// TODO: Implement usage logging in the service
		// For now, this is a placeholder that doesn't fail
	})
}
