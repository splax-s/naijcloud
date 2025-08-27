package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
)

type APIKeyHandler struct {
	apiKeyService *services.APIKeyService
}

func NewAPIKeyHandler(apiKeyService *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// CreateAPIKey creates a new API key for the organization
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate scopes
	validScopes := map[string]bool{
		"domains:read":   true,
		"domains:write":  true,
		"analytics:read": true,
		"edges:read":     true,
		"edges:write":    true,
		"api_keys:read":  true,
		"api_keys:write": true,
	}

	for _, scope := range req.Scopes {
		if !validScopes[scope] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scope", "scope": scope})
			return
		}
	}

	response, err := h.apiKeyService.CreateAPIKey(c.Request.Context(), orgID.(uuid.UUID), userID.(uuid.UUID), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create API key", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetAPIKey retrieves a specific API key
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	keyIDStr := c.Param("keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKey(c.Request.Context(), orgID.(uuid.UUID), keyID)
	if err != nil {
		if err.Error() == "API key not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API key", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
}

// ListAPIKeys retrieves all API keys for the organization
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	apiKeys, err := h.apiKeyService.ListAPIKeys(c.Request.Context(), orgID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list API keys", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_keys": apiKeys})
}

// UpdateAPIKey updates an existing API key
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	keyIDStr := c.Param("keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	var req models.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate scopes if provided
	if req.Scopes != nil {
		validScopes := map[string]bool{
			"domains:read":   true,
			"domains:write":  true,
			"analytics:read": true,
			"edges:read":     true,
			"edges:write":    true,
			"api_keys:read":  true,
			"api_keys:write": true,
		}

		for _, scope := range req.Scopes {
			if !validScopes[scope] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scope", "scope": scope})
				return
			}
		}
	}

	apiKey, err := h.apiKeyService.UpdateAPIKey(c.Request.Context(), orgID.(uuid.UUID), keyID, &req)
	if err != nil {
		if err.Error() == "API key not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update API key", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": apiKey})
}

// DeleteAPIKey deletes an API key
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	keyIDStr := c.Param("keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	err = h.apiKeyService.DeleteAPIKey(c.Request.Context(), orgID.(uuid.UUID), keyID)
	if err != nil {
		if err.Error() == "API key not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API key", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}

// GetAPIKeyUsage retrieves usage statistics for an API key
func (h *APIKeyHandler) GetAPIKeyUsage(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	keyIDStr := c.Param("keyId")
	keyID, err := uuid.Parse(keyIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	// Parse query parameters for date range
	var startDate, endDate time.Time
	if startStr := c.Query("start_date"); startStr != "" {
		if parsed, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = parsed
		}
	}
	if endStr := c.Query("end_date"); endStr != "" {
		if parsed, err := time.Parse("2006-01-02", endStr); err == nil {
			endDate = parsed.Add(24 * time.Hour) // Include the full end date
		}
	}

	// Default to last 30 days if no date range provided
	if startDate.IsZero() {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}

	// Parse limit parameter
	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	// For now, return a placeholder response with the limit applied
	// In a real implementation, you would query the api_key_usage table
	usage := gin.H{
		"api_key_id":        keyID,
		"organization_id":   orgID,
		"start_date":        startDate.Format("2006-01-02"),
		"end_date":          endDate.Format("2006-01-02"),
		"limit":             limit,
		"total_requests":    0,
		"success_rate":      100.0,
		"avg_response_time": 0,
		"endpoints":         []gin.H{},
		"daily_stats":       []gin.H{},
	}

	c.JSON(http.StatusOK, gin.H{"usage": usage})
}
