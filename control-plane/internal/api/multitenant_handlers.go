package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/middleware"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
	"github.com/sirupsen/logrus"
)

type DomainHandler struct {
	domainService *services.DomainService
	cacheService  *services.CacheService
}

func NewDomainHandler(domainService *services.DomainService, cacheService *services.CacheService) *DomainHandler {
	return &DomainHandler{
		domainService: domainService,
		cacheService:  cacheService,
	}
}

// ListDomains retrieves all domains for an organization
func (h *DomainHandler) ListDomains(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	domains, err := h.domainService.ListDomains(orgID)
	if err != nil {
		logrus.WithError(err).Error("Failed to list domains")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list domains"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

// CreateDomain creates a new domain for an organization
func (h *DomainHandler) CreateDomain(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	var req models.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain, err := h.domainService.CreateDomain(orgID, &req)
	if err != nil {
		logrus.WithError(err).WithField("domain", req.Domain).Error("Failed to create domain")
		if err.Error() == "domain "+req.Domain+" already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create domain"})
		return
	}

	c.JSON(http.StatusCreated, domain)
}

// GetDomain retrieves a specific domain for an organization
func (h *DomainHandler) GetDomain(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	domainName := c.Param("domain")

	domain, err := h.domainService.GetDomain(orgID, domainName)
	if err != nil {
		if err.Error() == "domain not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
			return
		}
		logrus.WithError(err).WithField("domain", domainName).Error("Failed to get domain")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get domain"})
		return
	}

	c.JSON(http.StatusOK, domain)
}

// UpdateDomain updates an existing domain for an organization
func (h *DomainHandler) UpdateDomain(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	domainName := c.Param("domain")

	var req models.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain, err := h.domainService.UpdateDomain(orgID, domainName, &req)
	if err != nil {
		if err.Error() == "domain not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
			return
		}
		logrus.WithError(err).WithField("domain", domainName).Error("Failed to update domain")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update domain"})
		return
	}

	c.JSON(http.StatusOK, domain)
}

// DeleteDomain removes a domain for an organization
func (h *DomainHandler) DeleteDomain(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	domainName := c.Param("domain")

	err = h.domainService.DeleteDomain(orgID, domainName)
	if err != nil {
		if err.Error() == "domain not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
			return
		}
		logrus.WithError(err).WithField("domain", domainName).Error("Failed to delete domain")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Domain deleted successfully"})
}

// PurgeDomainCache purges cache for a domain
func (h *DomainHandler) PurgeDomainCache(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	domainName := c.Param("domain")

	// Verify domain belongs to organization
	domain, err := h.domainService.GetDomain(orgID, domainName)
	if err != nil {
		if err.Error() == "domain not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
			return
		}
		logrus.WithError(err).WithField("domain", domainName).Error("Failed to get domain")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get domain"})
		return
	}

	var req models.PurgeRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := middleware.ExtractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	purgeRequest, err := h.cacheService.PurgeCache(domain.ID, req.Paths, userID.String())
	if err != nil {
		logrus.WithError(err).WithField("domain", domainName).Error("Failed to create purge request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create purge request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"purge_request": purgeRequest})
}

// EdgeHandler handles organization-scoped edge node management
type EdgeHandler struct {
	edgeService  *services.EdgeService
	cacheService *services.CacheService
}

func NewEdgeHandler(edgeService *services.EdgeService, cacheService *services.CacheService) *EdgeHandler {
	return &EdgeHandler{
		edgeService:  edgeService,
		cacheService: cacheService,
	}
}

// AnalyticsHandler handles organization-scoped analytics and reporting
type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetOrganizationOverview retrieves overview analytics for an organization
func (h *AnalyticsHandler) GetOrganizationOverview(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period format"})
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-period)

	overview, err := h.analyticsService.GetOrganizationOverview(orgID.String(), startTime, endTime)
	if err != nil {
		logrus.WithError(err).Error("Failed to get organization overview")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get overview"})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// GetOrganizationDomainAnalytics retrieves domain-specific analytics for an organization
func (h *AnalyticsHandler) GetOrganizationDomainAnalytics(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	domain := c.Param("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain not specified"})
		return
	}

	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period format"})
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-period)

	analytics, err := h.analyticsService.GetOrganizationDomainAnalytics(orgID.String(), domain, startTime, endTime)
	if err != nil {
		logrus.WithError(err).Error("Failed to get domain analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get domain analytics"})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// GetOrganizationUsageStats retrieves usage statistics for an organization
func (h *AnalyticsHandler) GetOrganizationUsageStats(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	periodStr := c.DefaultQuery("period", "30d")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period format"})
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-period)

	usageStats, err := h.analyticsService.GetOrganizationUsageStats(orgID.String(), startTime, endTime)
	if err != nil {
		logrus.WithError(err).Error("Failed to get usage statistics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage statistics"})
		return
	}

	c.JSON(http.StatusOK, usageStats)
}

// GetOrganizationPerformanceMetrics retrieves performance metrics for an organization
func (h *AnalyticsHandler) GetOrganizationPerformanceMetrics(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period format"})
		return
	}

	endTime := time.Now()
	startTime := endTime.Add(-period)

	metrics, err := h.analyticsService.GetOrganizationPerformanceMetrics(orgID.String(), startTime, endTime)
	if err != nil {
		logrus.WithError(err).Error("Failed to get performance metrics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get performance metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// ListEdges retrieves all edge nodes for an organization
func (h *EdgeHandler) ListEdges(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	edges, err := h.edgeService.ListEdgesByOrganization(orgID.(uuid.UUID))
	if err != nil {
		logrus.WithError(err).Error("Failed to list edge nodes for organization")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list edge nodes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"edges": edges})
}

// GetEdge retrieves a specific edge node
func (h *EdgeHandler) GetEdge(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	edgeIDStr := c.Param("edgeId")
	edgeID, err := uuid.Parse(edgeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
		return
	}

	edge, err := h.edgeService.GetEdgeByOrganization(orgID.(uuid.UUID), edgeID)
	if err != nil {
		if err.Error() == "edge not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Edge not found"})
			return
		}
		logrus.WithError(err).Error("Failed to get edge node")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get edge node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"edge": edge})
}

// CreateEdge creates a new edge node for the organization
func (h *EdgeHandler) CreateEdge(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	var req models.RegisterEdgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	edge, err := h.edgeService.RegisterEdgeForOrganization(orgID.(uuid.UUID), &req)
	if err != nil {
		logrus.WithError(err).Error("Failed to create edge node")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create edge node"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"edge": edge})
}

// UpdateEdge updates an edge node configuration
func (h *EdgeHandler) UpdateEdge(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	edgeIDStr := c.Param("edgeId")
	edgeID, err := uuid.Parse(edgeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	edge, err := h.edgeService.UpdateEdgeForOrganization(orgID.(uuid.UUID), edgeID, req)
	if err != nil {
		if err.Error() == "edge not found" || err.Error() == "edge not found or access denied" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Edge node not found"})
			return
		}
		logrus.WithError(err).Error("Failed to update edge node")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update edge node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"edge": edge})
}

// DeleteEdge removes an edge node
func (h *EdgeHandler) DeleteEdge(c *gin.Context) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
		return
	}

	edgeIDStr := c.Param("edgeId")
	edgeID, err := uuid.Parse(edgeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
		return
	}

	err = h.edgeService.DeleteEdgeForOrganization(orgID.(uuid.UUID), edgeID)
	if err != nil {
		if err.Error() == "edge not found or access denied" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Edge node not found"})
			return
		}
		logrus.WithError(err).WithField("edge_id", edgeID).Error("Failed to delete edge node")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete edge node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Edge node deleted successfully"})
}

// GetEdgeMetrics retrieves metrics for a specific edge node
func (h *EdgeHandler) GetEdgeMetrics(c *gin.Context) {
	edgeIDStr := c.Param("edgeId")
	edgeID, err := uuid.Parse(edgeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
		return
	}

	// Get edge node details
	edge, err := h.edgeService.GetEdge(edgeID)
	if err != nil {
		if err.Error() == "edge not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Edge node not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get edge node"})
		return
	}

	// Generate sample metrics (replace with real metrics later)
	metrics := gin.H{
		"edge_id":           edge.ID,
		"requests_per_sec":  500 + (int(edgeID[0]) % 300),
		"cache_hit_ratio":   0.85 + (float64(int(edgeID[1])%10) / 100),
		"avg_response_time": 45 + (int(edgeID[2]) % 50),
		"bandwidth_mbps":    100 + (int(edgeID[3]) % 400),
		"cpu_usage":         0.3 + (float64(int(edgeID[4])%30) / 100),
		"memory_usage":      0.6 + (float64(int(edgeID[5])%20) / 100),
		"uptime_hours":      24 + (int(edgeID[6]) % 168), // Up to 1 week
	}

	c.JSON(http.StatusOK, metrics)
}

// SetupMultiTenantRoutes sets up the multi-tenant API routes
func SetupMultiTenantRoutes(
	router *gin.Engine,
	orgService *services.OrganizationService,
	userService *services.UserService,
	domainService *services.DomainService,
	edgeService *services.EdgeService,
	analyticsService *services.AnalyticsService,
	cacheService *services.CacheService,
	apiKeyService *services.APIKeyService,
	authService *services.AuthService,
	emailService *services.EmailService,
) {
	// Create auth handler
	authHandler := NewAuthHandler(authService)

	// Create email handler
	emailHandler := NewEmailHandler(emailService, userService)

	// Public authentication routes (no auth required)
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/check-email", authHandler.CheckEmail)
		auth.GET("/check-slug", authHandler.CheckSlug)
		
		// Email verification routes
		auth.POST("/send-verification", emailHandler.SendEmailVerification)
		auth.POST("/verify-email", emailHandler.VerifyEmail)
		auth.POST("/forgot-password", emailHandler.RequestPasswordReset)
		auth.POST("/reset-password", emailHandler.ResetPassword)
	}

	// Setup organization routes
	SetupOrganizationRoutes(router, orgService, userService)

	// Create API key middleware
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(apiKeyService)

	// Organization-scoped API routes
	api := router.Group("/api/v1/orgs/:slug")
	api.Use(middleware.AuthMiddleware(userService))
	api.Use(middleware.MultiTenancyMiddleware(orgService, userService))
	api.Use(middleware.RequireOrganizationAccess(orgService))

	// API Key management routes
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	apiKeys := api.Group("/api-keys")
	apiKeys.Use(middleware.RequireOrganizationAccess(orgService, "owner", "admin"))
	{
		apiKeys.POST("", apiKeyHandler.CreateAPIKey)
		apiKeys.GET("", apiKeyHandler.ListAPIKeys)
		apiKeys.GET("/:keyId", apiKeyHandler.GetAPIKey)
		apiKeys.PUT("/:keyId", apiKeyHandler.UpdateAPIKey)
		apiKeys.DELETE("/:keyId", apiKeyHandler.DeleteAPIKey)
		apiKeys.GET("/:keyId/usage", apiKeyHandler.GetAPIKeyUsage)
	}

	// Domain management
	domainHandler := NewDomainHandler(domainService, cacheService)
	domains := api.Group("/domains")
	domains.Use(middleware.RequireOrganizationAccess(orgService, "owner", "admin", "member"))
	{
		domains.GET("", domainHandler.ListDomains)
		domains.POST("", domainHandler.CreateDomain)
		domains.GET("/:domain", domainHandler.GetDomain)
		domains.PUT("/:domain", domainHandler.UpdateDomain)
		domains.DELETE("/:domain", domainHandler.DeleteDomain)
		domains.POST("/:domain/purge", domainHandler.PurgeDomainCache)
	}

	// Edge node management
	edgeHandler := NewEdgeHandler(edgeService, cacheService)
	edges := api.Group("/edges")
	edges.Use(middleware.RequireOrganizationAccess(orgService, "owner", "admin", "member"))
	{
		edges.GET("", edgeHandler.ListEdges)
		edges.POST("", edgeHandler.CreateEdge)
		edges.GET("/:edgeId", edgeHandler.GetEdge)
		edges.PUT("/:edgeId", edgeHandler.UpdateEdge)
		edges.DELETE("/:edgeId", edgeHandler.DeleteEdge)
		edges.GET("/:edgeId/metrics", edgeHandler.GetEdgeMetrics)
	}

	// Analytics and reporting
	analyticsHandler := NewAnalyticsHandler(analyticsService)
	analytics := api.Group("/analytics")
	analytics.Use(middleware.RequireOrganizationAccess(orgService, "owner", "admin", "member"))
	{
		analytics.GET("/overview", analyticsHandler.GetOrganizationOverview)
		analytics.GET("/domains/:domain", analyticsHandler.GetOrganizationDomainAnalytics)
		analytics.GET("/usage", analyticsHandler.GetOrganizationUsageStats)
		analytics.GET("/performance", analyticsHandler.GetOrganizationPerformanceMetrics)
	}

	// Programmatic API routes (API key authentication)
	programmatic := router.Group("/api/v1/programmatic")
	// Require API key authentication for programmatic access
	programmatic.Use(apiKeyMiddleware.RequireAPIKey())
	programmatic.Use(apiKeyMiddleware.LogAPIKeyUsage())
	{
		// Domain routes for API access
		progDomains := programmatic.Group("/domains")
		progDomains.Use(apiKeyMiddleware.RequireAnyScope("domains:read", "domains:write"))
		{
			progDomains.GET("", func(c *gin.Context) {
				// Get organization ID from context (set by API key auth)
				orgID, exists := c.Get("organization_id")
				if !exists {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
					return
				}

				domains, err := domainService.ListDomains(orgID.(uuid.UUID))
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list domains"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"domains": domains})
			})

			progDomains.POST("", apiKeyMiddleware.RequireScope("domains:write"), func(c *gin.Context) {
				orgID, exists := c.Get("organization_id")
				if !exists {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Organization context required"})
					return
				}

				var req models.CreateDomainRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				domain, err := domainService.CreateDomain(orgID.(uuid.UUID), &req)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create domain"})
					return
				}

				c.JSON(http.StatusCreated, domain)
			})
		}
	}

	// Global API routes (for edge nodes and dashboard)
	global := router.Group("/api/v1")
	{
		// Edge routes - these are not organization-scoped
		edges := global.Group("/edges")
		{
			edges.GET("", func(c *gin.Context) {
				edgeList, err := edgeService.ListEdges()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list edges"})
					return
				}

				// Enhance edge data with calculated metrics
				enhancedEdges := make([]gin.H, len(edgeList))
				for i, edge := range edgeList {
					// Calculate metrics based on edge characteristics
					healthScore := 95
					if edge.Status != "healthy" {
						healthScore = 70
					}

					// Use edge ID bytes to generate consistent pseudo-random metrics
					idBytes := []byte(edge.ID.String())
					avgResponseTime := 30 + (int(idBytes[0]) % 40)              // 30-70ms
					totalRequests := 10000 + (int(idBytes[1]) * 1000)           // 10k-265k requests
					cacheHitRatio := 0.75 + (float64(int(idBytes[2])%25) / 100) // 75-99%

					enhancedEdges[i] = gin.H{
						"id":              edge.ID,
						"organization_id": edge.OrganizationID,
						"hostname":        edge.Hostname,
						"ip_address":      edge.IPAddress,
						"region":          edge.Region,
						"capacity":        edge.Capacity,
						"status":          edge.Status,
						"last_heartbeat":  edge.LastHeartbeat,
						"created_at":      edge.CreatedAt,
						"metadata":        edge.Metadata,
						// Enhanced fields for dashboard
						"health_score":      healthScore,
						"avg_response_time": avgResponseTime,
						"total_requests":    totalRequests,
						"cache_hit_ratio":   cacheHitRatio,
						"version":           "v1.0.0",
					}
				}

				c.JSON(http.StatusOK, gin.H{"edges": enhancedEdges})
			})

			edges.POST("", func(c *gin.Context) {
				var req models.RegisterEdgeRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				edge, err := edgeService.RegisterEdge(&req)
				if err != nil {
					logrus.WithError(err).Error("Failed to register edge")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register edge"})
					return
				}

				c.JSON(http.StatusCreated, edge)
			})

			// Heartbeat endpoint for edge nodes
			edges.POST("/:edgeId/heartbeat", func(c *gin.Context) {
				edgeIDStr := c.Param("edgeId")
				edgeID, err := uuid.Parse(edgeIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
					return
				}

				var req struct {
					Status  string                 `json:"status"`
					Metrics map[string]interface{} `json:"metrics"`
				}

				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// Update edge heartbeat (for now, just return success)
				// TODO: Update edge last_heartbeat timestamp in database
				logrus.WithFields(logrus.Fields{
					"edge_id": edgeID,
					"status":  req.Status,
				}).Debug("Received heartbeat from edge node")

				c.JSON(http.StatusOK, gin.H{"status": "acknowledged"})
			})

			// Purges endpoint for edge nodes
			edges.GET("/:edgeId/purges", func(c *gin.Context) {
				edgeIDStr := c.Param("edgeId")
				_, err := uuid.Parse(edgeIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
					return
				}

				// For now, return empty purges list
				// TODO: Implement actual purge requests from database
				c.JSON(http.StatusOK, gin.H{"purges": []interface{}{}})
			})

			// Complete purge endpoint for edge nodes
			edges.POST("/:edgeId/purges/:purgeId/complete", func(c *gin.Context) {
				edgeIDStr := c.Param("edgeId")
				edgeID, err := uuid.Parse(edgeIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
					return
				}

				purgeIDStr := c.Param("purgeId")
				purgeID, err := uuid.Parse(purgeIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid purge ID"})
					return
				}

				// For now, just acknowledge completion
				// TODO: Update purge status in database
				logrus.WithFields(logrus.Fields{
					"edge_id":  edgeID,
					"purge_id": purgeID,
				}).Info("Purge completed by edge node")

				c.JSON(http.StatusOK, gin.H{"status": "completed"})
			})
		}

		// Dashboard backward-compatibility routes (use demo organization)
		demoOrgID := uuid.MustParse("3fbdbdad-dbf5-4ac1-9335-e644302769ad")

		// Domains endpoint for dashboard
		global.GET("/domains", func(c *gin.Context) {
			domains, err := domainService.ListDomains(demoOrgID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list domains"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"domains": domains})
		})

		// Dashboard metrics endpoints
		SetupDashboardMetrics(global, domainService, edgeService, analyticsService, demoOrgID)
	}
}

// SetupDashboardMetrics sets up dashboard-specific metrics endpoints
func SetupDashboardMetrics(
	api *gin.RouterGroup,
	domainService *services.DomainService,
	edgeService *services.EdgeService,
	analyticsService *services.AnalyticsService,
	orgID uuid.UUID,
) {
	// Dashboard overview metrics
	api.GET("/metrics/dashboard", func(c *gin.Context) {
		// Get total domains count
		domains, err := domainService.ListDomains(orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get domains"})
			return
		}

		// Get edge nodes count
		edges, err := edgeService.ListEdges()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get edges"})
			return
		}

		activeEdges := 0
		totalResponseTime := 0
		totalCacheHitRatio := 0.0
		totalRequests := 0
		for _, edge := range edges {
			if edge.Status == "healthy" {
				activeEdges++
			}
			// Calculate response time based on edge ID (same logic as in edges endpoint)
			idBytes := []byte(edge.ID.String())
			avgResponseTime := 30 + (int(idBytes[0]) % 40)
			totalResponseTime += avgResponseTime

			// Calculate cache hit ratio (same logic as edges endpoint)
			cacheHitRatio := 0.75 + (float64(int(idBytes[2])%25) / 100) // 75-99%
			totalCacheHitRatio += cacheHitRatio

			// Calculate total requests (same logic as edges endpoint)
			requests := 10000 + (int(idBytes[1]) * 1000) // 10k-265k requests
			totalRequests += requests
		}

		avgResponseTime := 45    // Default
		avgCacheHitRatio := 0.85 // Default
		if len(edges) > 0 {
			avgResponseTime = totalResponseTime / len(edges)
			avgCacheHitRatio = totalCacheHitRatio / float64(len(edges))
		}

		// Calculate basic metrics
		metrics := gin.H{
			"total_domains":      len(domains),
			"active_edge_nodes":  activeEdges,
			"cache_hit_ratio":    avgCacheHitRatio,
			"avg_response_time":  avgResponseTime,
			"total_requests_24h": totalRequests, // Add total requests
		}

		c.JSON(http.StatusOK, metrics)
	})

	// Traffic data for charts
	api.GET("/metrics/traffic", func(c *gin.Context) {
		hoursStr := c.DefaultQuery("hours", "24")
		hours, err := strconv.Atoi(hoursStr)
		if err != nil {
			hours = 24
		}

		// Get real edge nodes to base traffic on
		edges, err := edgeService.ListEdges()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get edges"})
			return
		}

		trafficData := generateTrafficDataFromEdges(hours, edges)
		c.JSON(http.StatusOK, trafficData)
	})

	// Top domains by traffic
	api.GET("/metrics/top-domains", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "10")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 10
		}

		domains, err := domainService.ListDomains(orgID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get domains"})
			return
		}

		// Generate top domains data (for now, just return the domains with mock data)
		topDomains := generateTopDomainsData(domains, limit)
		c.JSON(http.StatusOK, topDomains)
	})

	// Recent activity
	api.GET("/activity", func(c *gin.Context) {
		limitStr := c.DefaultQuery("limit", "10")
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 10
		}

		// Generate recent activity data
		activity := generateRecentActivity(limit)
		c.JSON(http.StatusOK, activity)
	})
}

// Helper functions for generating dashboard data
func generateTrafficDataFromEdges(hours int, edges []*models.Edge) []gin.H {
	data := make([]gin.H, hours)
	now := time.Now()

	// Calculate base traffic from edge nodes
	baseRequests := 0
	totalCacheHits := 0
	totalCacheMisses := 0

	for _, edge := range edges {
		idBytes := []byte(edge.ID.String())
		edgeRequests := 10000 + (int(idBytes[1]) * 1000) // Same calculation as in edges endpoint
		baseRequests += edgeRequests

		cacheHitRatio := 0.75 + (float64(int(idBytes[2])%25) / 100)
		cacheHits := int(float64(edgeRequests) * cacheHitRatio)
		cacheMisses := edgeRequests - cacheHits

		totalCacheHits += cacheHits
		totalCacheMisses += cacheMisses
	}

	// If no edges, use minimal data
	if len(edges) == 0 {
		baseRequests = 1000
		totalCacheHits = 800
		totalCacheMisses = 200
	}

	for i := 0; i < hours; i++ {
		timestamp := now.Add(time.Duration(-i) * time.Hour)

		// Create realistic traffic patterns (higher during business hours, lower at night)
		hour := timestamp.Hour()
		timeMultiplier := 1.0
		if hour >= 9 && hour <= 17 {
			timeMultiplier = 1.5 // Business hours peak
		} else if hour >= 0 && hour <= 6 {
			timeMultiplier = 0.4 // Night time low
		}

		// Add some randomness but keep it realistic
		randomFactor := 0.8 + (float64(i%5) * 0.1) // 0.8 to 1.2

		hourlyRequests := int(float64(baseRequests/24) * timeMultiplier * randomFactor)
		hourlyBandwidth := int64(hourlyRequests * 2048) // 2KB average per request

		// Distribute cache hits/misses proportionally
		cacheHitRatio := float64(totalCacheHits) / float64(totalCacheHits+totalCacheMisses)
		hourlyCacheHits := int(float64(hourlyRequests) * cacheHitRatio)
		hourlyCacheMisses := hourlyRequests - hourlyCacheHits

		data[hours-1-i] = gin.H{
			"timestamp":    timestamp.Format(time.RFC3339),
			"requests":     hourlyRequests,
			"bandwidth":    hourlyBandwidth,
			"cache_hits":   hourlyCacheHits,
			"cache_misses": hourlyCacheMisses,
		}
	}

	return data
}

func generateTopDomainsData(domains []*models.Domain, limit int) []gin.H {
	topDomains := make([]gin.H, 0, limit)

	if len(domains) == 0 {
		// Return empty if no domains
		return topDomains
	}

	for i, domain := range domains {
		if i >= limit {
			break
		}

		// Generate realistic traffic data based on domain characteristics
		domainBytes := []byte(domain.Domain)
		baseRequests := 50000 + (int(domainBytes[0]) * 1000) // 50k-305k requests

		// Decrease requests for each subsequent domain (realistic ranking)
		requests := baseRequests - (i * 8000)
		if requests < 1000 {
			requests = 1000 + (i * 100) // Minimum floor
		}

		// Calculate realistic bandwidth (varies by domain type)
		avgBytesPerRequest := 2048 // Default 2KB
		if strings.Contains(domain.Domain, "api.") {
			avgBytesPerRequest = 512 // API responses are smaller
		} else if strings.Contains(domain.Domain, "static.") || strings.Contains(domain.Domain, "cdn.") {
			avgBytesPerRequest = 8192 // Static assets are larger
		}

		bandwidth := int64(requests * avgBytesPerRequest)

		// Calculate cache hit ratio based on domain type
		cacheHitRatio := 0.85 // Default
		if strings.Contains(domain.Domain, "api.") {
			cacheHitRatio = 0.65 // APIs cache less
		} else if strings.Contains(domain.Domain, "static.") {
			cacheHitRatio = 0.98 // Static content caches very well
		}

		topDomains = append(topDomains, gin.H{
			"domain":          domain.Domain,
			"requests":        requests,
			"bandwidth":       bandwidth,
			"cache_hit_ratio": cacheHitRatio,
		})
	}

	return topDomains
}

func generateRecentActivity(limit int) []gin.H {
	now := time.Now()

	activities := []gin.H{
		{
			"id":        "1",
			"type":      "edge",
			"action":    "Edge node registered",
			"target":    "local-dev region",
			"timestamp": now.Add(-2 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":        "2",
			"type":      "system",
			"action":    "API endpoints activated",
			"target":    "dashboard metrics",
			"timestamp": now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":        "3",
			"type":      "security",
			"action":    "Authentication enabled",
			"target":    "API key system",
			"timestamp": now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":        "4",
			"type":      "system",
			"action":    "Services started",
			"target":    "control plane",
			"timestamp": now.Add(-20 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":        "5",
			"type":      "database",
			"action":    "Connection established",
			"target":    "PostgreSQL",
			"timestamp": now.Add(-25 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":        "6",
			"type":      "cache",
			"action":    "Cache initialized",
			"target":    "Redis cluster",
			"timestamp": now.Add(-25 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":        "7",
			"type":      "domain",
			"action":    "Domain added",
			"target":    "example.com",
			"timestamp": now.Add(-2 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":        "8",
			"type":      "cache",
			"action":    "Cache purged",
			"target":    "api.example.com",
			"timestamp": now.Add(-4 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":        "9",
			"type":      "security",
			"action":    "SSL certificate renewed",
			"target":    "secure.example.com",
			"timestamp": now.Add(-8 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":        "10",
			"type":      "monitoring",
			"action":    "High traffic detected",
			"target":    "cdn.example.com",
			"timestamp": now.Add(-12 * time.Hour).Format(time.RFC3339),
		},
	}

	if limit < len(activities) {
		return activities[:limit]
	}

	return activities
}

// EmailHandler handles email verification and password reset operations
type EmailHandler struct {
	emailService *services.EmailService
	userService  *services.UserService
}

// NewEmailHandler creates a new email handler
func NewEmailHandler(emailService *services.EmailService, userService *services.UserService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
		userService:  userService,
	}
}

// SendEmailVerification sends an email verification token
func (h *EmailHandler) SendEmailVerification(c *gin.Context) {
	var req models.SendEmailVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Find user by email
	user, err := h.userService.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		logrus.WithError(err).Error("Failed to find user for email verification")
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	err = h.emailService.SendEmailVerification(user.ID)
	if err != nil {
		logrus.WithError(err).Error("Failed to send verification email")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}

// VerifyEmail verifies an email using a token
func (h *EmailHandler) VerifyEmail(c *gin.Context) {
	var req models.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.emailService.VerifyEmail(req.Token)
	if err != nil {
		logrus.WithError(err).Error("Failed to verify email")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// RequestPasswordReset sends a password reset email
func (h *EmailHandler) RequestPasswordReset(c *gin.Context) {
	var req models.RequestPasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	err := h.emailService.SendPasswordReset(req.Email)
	if err != nil {
		logrus.WithError(err).Error("Failed to send password reset email")
		// Don't reveal the actual error to prevent email enumeration
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send password reset email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset email sent (if email exists)"})
}

// ResetPassword resets a password using a token
func (h *EmailHandler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate passwords match
	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passwords do not match"})
		return
	}

	err := h.emailService.ResetPassword(req.Token, req.Password)
	if err != nil {
		logrus.WithError(err).Error("Failed to reset password")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
