package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naijcloud/edge-proxy/internal/cache"
	"github.com/naijcloud/edge-proxy/internal/config"
	"github.com/naijcloud/edge-proxy/internal/middleware"
	"github.com/naijcloud/edge-proxy/internal/proxy"
	"github.com/naijcloud/edge-proxy/internal/services"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Configure logging
	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.WithError(err).Warn("Invalid log level, using info")
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	logrus.WithField("config", cfg).Info("Starting edge proxy")

	// Initialize cache
	var cacheImpl cache.Cache
	if cfg.RedisURL != "" {
		redisCache, err := cache.NewRedisCache(cfg.RedisURL, "edge-cache:", time.Duration(cfg.DefaultTTL)*time.Second)
		if err != nil {
			logrus.WithError(err).Warn("Failed to initialize Redis cache, falling back to memory cache")
			cacheImpl = cache.NewMemoryCache(parseSize(cfg.CacheSize))
		} else {
			cacheImpl = redisCache
			logrus.Info("Initialized Redis cache")
		}
	} else {
		cacheImpl = cache.NewMemoryCache(parseSize(cfg.CacheSize))
		logrus.Info("Initialized memory cache")
	}

	// Initialize proxy service
	proxyConfig := proxy.ProxyConfig{
		DefaultTTL:       time.Duration(cfg.DefaultTTL) * time.Second,
		MaxBodySize:      10 * 1024 * 1024, // 10MB
		ConnectTimeout:   10 * time.Second,
		ResponseTimeout:  30 * time.Second,
		IdleConnTimeout:  90 * time.Second,
		MaxIdleConns:     100,
		MaxIdleConnsHost: 10,
	}
	proxyService := proxy.NewProxyService(cacheImpl, proxyConfig)

	// Initialize control plane client
	controlPlane := services.NewControlPlaneClient(cfg.ControlPlaneURL, cfg.Region)

	// Register with control plane
	hostname, _ := os.Hostname()
	ipAddress := getLocalIP()
	_, err = controlPlane.RegisterEdge(context.Background(), ipAddress, hostname, 1000)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to register with control plane")
	}

	// Start heartbeat goroutine
	go startHeartbeat(controlPlane, cacheImpl)

	// Start purge handler goroutine
	go startPurgeHandler(controlPlane, proxyService)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)

	// Setup main HTTP server
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add middleware
	router.Use(middleware.LoggingMiddleware())
	router.Use(middleware.MetricsMiddleware())
	router.Use(rateLimiter.PerDomainRateLimit())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "healthy",
			"timestamp":  time.Now().UTC(),
			"cache_size": cacheImpl.Size(),
		})
	})

	// Proxy handler - catch all other requests
	router.NoRoute(func(c *gin.Context) {
		handleProxyRequest(c, controlPlane, proxyService)
	})

	// Start metrics server
	go func() {
		metricsRouter := gin.New()
		metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
		metricsRouter.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.MetricsPort),
			Handler: metricsRouter,
		}

		logrus.WithField("port", cfg.MetricsPort).Info("Starting metrics server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("Metrics server failed")
		}
	}()

	// Start main server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		logrus.WithField("port", cfg.Port).Info("Starting edge proxy server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Server failed to start")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down edge proxy...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Server forced to shutdown")
	}

	logrus.Info("Edge proxy stopped")
}

func handleProxyRequest(c *gin.Context, controlPlane *services.ControlPlaneClient, proxyService *proxy.ProxyService) {
	domain := c.Request.Host

	// Remove port from domain if present
	if colonPos := strings.Index(domain, ":"); colonPos != -1 {
		domain = domain[:colonPos]
	}

	// Get domain configuration from control plane
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	domainInfo, err := controlPlane.GetDomain(ctx, domain)
	if err != nil {
		logrus.WithError(err).WithField("domain", domain).Warn("Domain not found or control plane error")
		c.JSON(http.StatusNotFound, gin.H{"error": "Domain not configured"})
		return
	}

	if domainInfo.Status != "active" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Domain not active"})
		return
	}

	// Proxy the request
	proxyService.ServeHTTP(c.Writer, c.Request, domainInfo.OriginURL)
}

func startHeartbeat(controlPlane *services.ControlPlaneClient, cache cache.Cache) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		metrics := map[string]interface{}{
			"cache_size":       cache.Size(),
			"timestamp":        time.Now().Unix(),
			"requests_handled": 0, // TODO: implement request counter
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := controlPlane.SendHeartbeat(ctx, "healthy", metrics); err != nil {
			logrus.WithError(err).Warn("Failed to send heartbeat")
		}
		cancel()
	}
}

func startPurgeHandler(controlPlane *services.ControlPlaneClient, proxyService *proxy.ProxyService) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		purges, err := controlPlane.GetPendingPurges(ctx)
		if err != nil {
			logrus.WithError(err).Warn("Failed to get pending purges")
			cancel()
			continue
		}

		for _, purge := range purges {
			// Get domain info using the domain ID from the purge request
			domainInfo, err := controlPlane.GetDomainByID(ctx, purge.DomainID)
			if err != nil {
				logrus.WithError(err).WithField("purge_id", purge.ID).Warn("Failed to get domain info for purge")
				continue
			}

			// Purge cache entries
			if err := proxyService.PurgeCache(ctx, domainInfo.Domain, purge.Paths); err != nil {
				logrus.WithError(err).WithField("purge_id", purge.ID).Warn("Failed to purge cache")
				continue
			}

			// Mark purge as complete
			if err := controlPlane.CompletePurge(ctx, purge.ID); err != nil {
				logrus.WithError(err).WithField("purge_id", purge.ID).Warn("Failed to mark purge as complete")
			}

			logrus.WithFields(logrus.Fields{
				"purge_id": purge.ID,
				"domain":   domainInfo.Domain,
				"paths":    purge.Paths,
			}).Info("Cache purge completed")
		}

		cancel()
	}
}

func parseSize(sizeStr string) int64 {
	sizeStr = strings.ToUpper(sizeStr)
	if strings.HasSuffix(sizeStr, "MB") {
		return 100 * 1024 * 1024 // Default 100MB
	}
	return 100 * 1024 * 1024 // Default 100MB
}

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
