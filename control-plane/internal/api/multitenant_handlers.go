package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

// SetupMultiTenantRoutes sets up the multi-tenant API routes
func SetupMultiTenantRoutes(
	router *gin.Engine,
	orgService *services.OrganizationService,
	userService *services.UserService,
	domainService *services.DomainService,
	edgeService *services.EdgeService,
	analyticsService *services.AnalyticsService,
	cacheService *services.CacheService,
) {
	// Setup organization routes
	SetupOrganizationRoutes(router, orgService, userService)

	// Organization-scoped API routes
	api := router.Group("/api/v1/orgs/:slug")
	api.Use(middleware.AuthMiddleware(userService))
	api.Use(middleware.MultiTenancyMiddleware(orgService, userService))
	api.Use(middleware.RequireOrganizationAccess(orgService))

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

	// Global API routes (for edge nodes)
	global := router.Group("/api/v1")
	{
		// Edge routes - these are not organization-scoped
		edges := global.Group("/edges")
		{
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
		}
	}
}
