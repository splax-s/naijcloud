package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naijcloud/control-plane/internal/api"
	"github.com/naijcloud/control-plane/internal/config"
	"github.com/naijcloud/control-plane/internal/database"
	"github.com/naijcloud/control-plane/internal/middleware"
	"github.com/naijcloud/control-plane/internal/services"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	setupLogger(cfg.LogLevel)

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := database.InitializeRedis(cfg.RedisURL)
	if err != nil {
		logrus.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize services
	domainService := services.NewDomainService(db, redisClient)
	edgeService := services.NewEdgeService(db, redisClient)
	analyticsService := services.NewAnalyticsService(db)
	cacheService := services.NewCacheService(redisClient, edgeService)

	// Initialize multi-tenancy services
	orgService := services.NewOrganizationService(db)
	userService := services.NewUserService(db)
	apiKeyService := services.NewAPIKeyService(db)
	authService := services.NewAuthService(db)
	emailService := services.NewEmailService(db)
	activityService := services.NewActivityService(db)
	notificationService := services.NewNotificationService(db)

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())
	router.Use(middleware.Metrics())
	router.Use(middleware.SecurityHeaders())

	// JWT middleware
	jwtMiddleware := middleware.NewJWTMiddleware(db)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// API routes - use multi-tenant setup with enhanced features
	api.SetupMultiTenantRoutes(router, orgService, userService, domainService, edgeService, analyticsService, cacheService, apiKeyService, authService, emailService, activityService, notificationService, jwtMiddleware)

	// Metrics server
	go func() {
		metricsRouter := gin.New()
		metricsRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))
		metricsRouter.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})

		logrus.Infof("Starting metrics server on port %s", cfg.MetricsPort)
		if err := http.ListenAndServe(":"+cfg.MetricsPort, metricsRouter); err != nil {
			logrus.Errorf("Metrics server error: %v", err)
		}
	}()

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		logrus.Infof("Starting control plane server on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logrus.Info("Shutting down server...")

	// Graceful shutdown with 30s timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("Server forced to shutdown: %v", err)
	}

	logrus.Info("Server exited")
}

func setupLogger(level string) {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	switch level {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}
