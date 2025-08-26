package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
	"github.com/sirupsen/logrus"
)

// SetupRoutes configures all API routes
func SetupRoutes(
	r *gin.RouterGroup,
	domainService *services.DomainService,
	edgeService *services.EdgeService,
	analyticsService *services.AnalyticsService,
	cacheService *services.CacheService,
) {
	// Domain routes
	domains := r.Group("/domains")
	{
		domains.GET("", listDomains(domainService))
		domains.POST("", createDomain(domainService))
		domains.GET("/:domain", getDomain(domainService))
		domains.GET("/id/:domain_id", getDomainByID(domainService))
		domains.PUT("/:domain", updateDomain(domainService))
		domains.DELETE("/:domain", deleteDomain(domainService))
		domains.POST("/:domain/purge", purgeDomainCache(domainService, cacheService))
	}

	// Edge routes
	edges := r.Group("/edges")
	{
		edges.GET("", listEdges(edgeService))
		edges.POST("", registerEdge(edgeService))
		edges.GET("/:edge_id", getEdge(edgeService))
		edges.DELETE("/:edge_id", deleteEdge(edgeService))
		edges.POST("/:edge_id/heartbeat", updateHeartbeat(edgeService))
		edges.GET("/:edge_id/purges", getPendingPurges(cacheService))
		edges.POST("/:edge_id/purges/:purge_id/complete", completePurge(cacheService))
	}

	// Analytics routes
	analytics := r.Group("/analytics")
	{
		analytics.GET("/domains/:domain", getDomainAnalytics(analyticsService))
		analytics.GET("/domains/:domain/paths", getTopPaths(analyticsService))
		analytics.GET("/domains/:domain/timeline", getRequestsOverTime(analyticsService))
	}
}

// Domain handlers
func listDomains(service *services.DomainService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domains, err := service.ListDomains()
		if err != nil {
			logrus.WithError(err).Error("Failed to list domains")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list domains"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"domains": domains})
	}
}

func createDomain(service *services.DomainService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateDomainRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		domain, err := service.CreateDomain(&req)
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
}

func getDomain(service *services.DomainService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domainName := c.Param("domain")

		domain, err := service.GetDomain(domainName)
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
}

func getDomainByID(service *services.DomainService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domainIDStr := c.Param("domain_id")
		domainID, err := uuid.Parse(domainIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid domain ID"})
			return
		}

		domain, err := service.GetDomainByID(domainID)
		if err != nil {
			if err.Error() == "domain not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
				return
			}
			logrus.WithError(err).WithField("domain_id", domainID).Error("Failed to get domain")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get domain"})
			return
		}

		c.JSON(http.StatusOK, domain)
	}
}

func updateDomain(service *services.DomainService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domainName := c.Param("domain")

		var req models.UpdateDomainRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		domain, err := service.UpdateDomain(domainName, &req)
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
}

func deleteDomain(service *services.DomainService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domainName := c.Param("domain")

		err := service.DeleteDomain(domainName)
		if err != nil {
			if err.Error() == "domain not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
				return
			}
			logrus.WithError(err).WithField("domain", domainName).Error("Failed to delete domain")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete domain"})
			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}

func purgeDomainCache(domainService *services.DomainService, cacheService *services.CacheService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domainName := c.Param("domain")

		// Verify domain exists
		domain, err := domainService.GetDomain(domainName)
		if err != nil {
			if err.Error() == "domain not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Domain not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify domain"})
			return
		}

		var req models.PurgeRequestBody
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Default to purging all paths if none specified
		if len(req.Paths) == 0 {
			req.Paths = []string{"/*"}
		}

		purgeReq, err := cacheService.PurgeCache(domain.ID, req.Paths, "api")
		if err != nil {
			logrus.WithError(err).WithField("domain", domainName).Error("Failed to initiate cache purge")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate cache purge"})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"purge_id": purgeReq.ID,
			"status":   "accepted",
			"message":  "Cache purge initiated",
		})
	}
}

// Edge handlers
func listEdges(service *services.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		edges, err := service.ListEdges()
		if err != nil {
			logrus.WithError(err).Error("Failed to list edges")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list edges"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"edges": edges})
	}
}

func registerEdge(service *services.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.RegisterEdgeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		edge, err := service.RegisterEdge(&req)
		if err != nil {
			logrus.WithError(err).Error("Failed to register edge")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register edge"})
			return
		}

		c.JSON(http.StatusCreated, edge)
	}
}

func getEdge(service *services.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		edgeIDStr := c.Param("edge_id")
		edgeID, err := uuid.Parse(edgeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
			return
		}

		edge, err := service.GetEdge(edgeID)
		if err != nil {
			if err.Error() == "edge not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Edge not found"})
				return
			}
			logrus.WithError(err).WithField("edge_id", edgeID).Error("Failed to get edge")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get edge"})
			return
		}

		c.JSON(http.StatusOK, edge)
	}
}

func updateHeartbeat(service *services.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		edgeIDStr := c.Param("edge_id")
		edgeID, err := uuid.Parse(edgeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
			return
		}

		var req models.HeartbeatRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = service.UpdateHeartbeat(edgeID, &req)
		if err != nil {
			if err.Error() == "edge not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Edge not found"})
				return
			}
			logrus.WithError(err).WithField("edge_id", edgeID).Error("Failed to update heartbeat")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heartbeat"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "heartbeat updated"})
	}
}

func deleteEdge(service *services.EdgeService) gin.HandlerFunc {
	return func(c *gin.Context) {
		edgeIDStr := c.Param("edge_id")
		edgeID, err := uuid.Parse(edgeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
			return
		}

		err = service.DeleteEdge(edgeID)
		if err != nil {
			if err.Error() == "edge not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Edge not found"})
				return
			}
			logrus.WithError(err).WithField("edge_id", edgeID).Error("Failed to delete edge")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete edge"})
			return
		}

		c.JSON(http.StatusNoContent, nil)
	}
}

// Cache handlers
func getPendingPurges(service *services.CacheService) gin.HandlerFunc {
	return func(c *gin.Context) {
		edgeIDStr := c.Param("edge_id")
		edgeID, err := uuid.Parse(edgeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
			return
		}

		purges, err := service.GetPendingPurges(edgeID)
		if err != nil {
			logrus.WithError(err).WithField("edge_id", edgeID).Error("Failed to get pending purges")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending purges"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"purges": purges})
	}
}

func completePurge(service *services.CacheService) gin.HandlerFunc {
	return func(c *gin.Context) {
		edgeIDStr := c.Param("edge_id")
		edgeID, err := uuid.Parse(edgeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid edge ID"})
			return
		}

		purgeIDStr := c.Param("purge_id")
		purgeID, err := uuid.Parse(purgeIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid purge ID"})
			return
		}

		err = service.CompletePurge(edgeID, purgeID)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"edge_id":  edgeID,
				"purge_id": purgeID,
			}).Error("Failed to complete purge")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete purge"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "purge completed"})
	}
}

// Analytics handlers
func getDomainAnalytics(service *services.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.Param("domain")

		// Parse time range from query parameters
		startTime, endTime := parseTimeRange(c)

		analytics, err := service.GetDomainAnalytics(domain, startTime, endTime)
		if err != nil {
			logrus.WithError(err).WithField("domain", domain).Error("Failed to get domain analytics")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics"})
			return
		}

		c.JSON(http.StatusOK, analytics)
	}
}

func getTopPaths(service *services.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.Param("domain")
		startTime, endTime := parseTimeRange(c)

		limitStr := c.DefaultQuery("limit", "10")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}

		paths, err := service.GetTopPaths(domain, startTime, endTime, limit)
		if err != nil {
			logrus.WithError(err).WithField("domain", domain).Error("Failed to get top paths")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get top paths"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"paths": paths})
	}
}

func getRequestsOverTime(service *services.AnalyticsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		domain := c.Param("domain")
		startTime, endTime := parseTimeRange(c)
		interval := c.DefaultQuery("interval", "1 hour")

		timeline, err := service.GetRequestsOverTime(domain, startTime, endTime, interval)
		if err != nil {
			logrus.WithError(err).WithField("domain", domain).Error("Failed to get requests over time")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get timeline data"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"timeline": timeline})
	}
}

// Helper function to parse time range from query parameters
func parseTimeRange(c *gin.Context) (time.Time, time.Time) {
	now := time.Now()

	// Default to last 24 hours
	endTime := now
	startTime := now.Add(-24 * time.Hour)

	if startStr := c.Query("start_time"); startStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = parsed
		}
	}

	if endStr := c.Query("end_time"); endStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = parsed
		}
	}

	return startTime, endTime
}
