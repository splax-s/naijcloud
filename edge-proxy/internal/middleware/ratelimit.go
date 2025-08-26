package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rate.Limit(rps),
		burst:    burst,
	}
}

// GetLimiter returns a rate limiter for the given key
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rps, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

// CleanupExpired removes expired limiters (call periodically)
func (rl *RateLimiter) CleanupExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Remove limiters that haven't been used recently
	for key, limiter := range rl.limiters {
		if limiter.Tokens() == float64(rl.burst) {
			delete(rl.limiters, key)
		}
	}
}

// RateLimit middleware function
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.CleanupExpired()
		}
	}()

	return func(c *gin.Context) {
		// Use client IP as the key
		key := c.ClientIP()

		limiter := rl.GetLimiter(key)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": "60s",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PerDomainRateLimit implements rate limiting per domain
func (rl *RateLimiter) PerDomainRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use domain + client IP as the key for more granular rate limiting
		key := c.Request.Host + ":" + c.ClientIP()

		limiter := rl.GetLimiter(key)

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded for this domain",
				"retry_after": "60s",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
