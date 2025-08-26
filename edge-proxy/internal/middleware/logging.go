package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingMiddleware creates a structured logging middleware
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)

		logrus.WithFields(logrus.Fields{
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"host":       c.Request.Host,
			"proto":      c.Request.Proto,
			"size":       c.Writer.Size(),
		}).Info("HTTP request")
	}
}

// MetricsMiddleware for collecting request metrics
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Record metrics
		duration := time.Since(start)

		// Here you would increment Prometheus counters
		// For now, we'll just log metrics
		logrus.WithFields(logrus.Fields{
			"type":         "metrics",
			"method":       c.Request.Method,
			"path":         c.Request.URL.Path,
			"status":       c.Writer.Status(),
			"duration_ms":  duration.Milliseconds(),
			"cache_status": c.GetHeader("X-Cache-Status"),
		}).Debug("Request metrics")
	}
}
