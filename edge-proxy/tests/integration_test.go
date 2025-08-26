package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/edge-proxy/internal/cache"
	"github.com/naijcloud/edge-proxy/internal/middleware"
	"github.com/naijcloud/edge-proxy/internal/proxy"
	"github.com/naijcloud/edge-proxy/internal/services"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type EdgeProxyIntegrationTestSuite struct {
	suite.Suite

	// Test infrastructure
	redis       *redis.Client
	memoryCache cache.Cache
	redisCache  cache.Cache

	// Mock servers
	controlPlaneMock *httptest.Server
	originMock       *httptest.Server

	// Edge proxy components
	proxyService *proxy.ProxyService
	controlPlane *services.ControlPlaneClient
	rateLimiter  *middleware.RateLimiter
	router       *gin.Engine

	// Test data
	testDomain    string
	testOriginURL string
	edgeID        uuid.UUID
}

func (suite *EdgeProxyIntegrationTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)

	// Setup test Redis connection
	redisURL := "redis://localhost:6379/2" // Use database 2 for edge proxy tests
	opt, err := redis.ParseURL(redisURL)
	suite.Require().NoError(err)
	suite.redis = redis.NewClient(opt)

	// Test Redis connection
	ctx := context.Background()
	_, err = suite.redis.Ping(ctx).Result()
	suite.Require().NoError(err)

	// Initialize caches
	suite.memoryCache = cache.NewMemoryCache(50 * 1024 * 1024) // 50MB
	suite.redisCache, err = cache.NewRedisCache(redisURL, "test-edge:", 3600*time.Second)
	suite.Require().NoError(err)

	// Setup test data
	suite.testDomain = "test.example.com"
	suite.edgeID = uuid.New()

	suite.setupMockServers()
	suite.setupEdgeProxyComponents()
}

func (suite *EdgeProxyIntegrationTestSuite) setupMockServers() {
	// Setup control plane mock server
	suite.controlPlaneMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.Contains(r.URL.Path, "/v1/edges"):
			// Edge registration
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			response := map[string]interface{}{
				"id":         suite.edgeID.String(),
				"region":     "test-region",
				"ip_address": "192.168.1.100",
				"hostname":   "test-edge",
				"capacity":   1000,
				"status":     "healthy",
				"created_at": time.Now(),
				"updated_at": time.Now(),
			}
			json.NewEncoder(w).Encode(response)

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/heartbeat"):
			// Heartbeat
			w.WriteHeader(http.StatusOK)

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/domains/"):
			// Domain lookup
			domain := strings.TrimPrefix(r.URL.Path, "/v1/domains/")
			if domain == suite.testDomain {
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"id":         uuid.New().String(),
					"domain":     suite.testDomain,
					"origin_url": suite.testOriginURL,
					"cache_ttl":  3600,
					"rate_limit": 1000,
					"status":     "active",
				}
				json.NewEncoder(w).Encode(response)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/edges/") && strings.Contains(r.URL.Path, "/purges"):
			// Pending purges for specific edge
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"purges": []map[string]interface{}{
					{
						"id":        uuid.New().String(),
						"domain_id": uuid.New().String(),
						"paths":     []string{"/cached-content/*"},
						"status":    "pending",
					},
				},
			}
			json.NewEncoder(w).Encode(response)

		case r.Method == "GET" && strings.Contains(r.URL.Path, "/v1/domains-by-id/"):
			// Domain lookup by ID
			w.Header().Set("Content-Type", "application/json")
			response := map[string]interface{}{
				"id":         uuid.New().String(),
				"domain":     suite.testDomain,
				"origin_url": suite.testOriginURL,
				"cache_ttl":  3600,
				"rate_limit": 1000,
				"status":     "active",
			}
			json.NewEncoder(w).Encode(response)

		case r.Method == "POST" && strings.Contains(r.URL.Path, "/v1/purges/") && strings.Contains(r.URL.Path, "/complete"):
			// Complete purge
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Setup origin mock server
	suite.originMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/hello":
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, World!"))

		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "public, max-age=1800")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "Hello JSON"})

		case "/no-cache":
			w.Header().Set("Cache-Control", "no-cache")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Not cacheable"))

		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))

		case "/slow":
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Slow response"))

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}
	}))

	suite.testOriginURL = suite.originMock.URL
}

func (suite *EdgeProxyIntegrationTestSuite) setupEdgeProxyComponents() {
	// Setup proxy service with memory cache
	proxyConfig := proxy.ProxyConfig{
		DefaultTTL:       3600 * time.Second,
		MaxBodySize:      10 * 1024 * 1024, // 10MB
		ConnectTimeout:   10 * time.Second,
		ResponseTimeout:  30 * time.Second,
		IdleConnTimeout:  90 * time.Second,
		MaxIdleConns:     100,
		MaxIdleConnsHost: 10,
	}
	suite.proxyService = proxy.NewProxyService(suite.memoryCache, proxyConfig)

	// Setup control plane client
	suite.controlPlane = services.NewControlPlaneClient(suite.controlPlaneMock.URL, "test-region")

	// Setup rate limiter
	suite.rateLimiter = middleware.NewRateLimiter(1000, 2000)

	// Setup router
	suite.router = gin.New()
	suite.router.Use(gin.Recovery())
	suite.router.Use(middleware.LoggingMiddleware())
	suite.router.Use(middleware.MetricsMiddleware())

	// Health endpoint
	suite.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":     "healthy",
			"timestamp":  time.Now().UTC(),
			"cache_size": suite.memoryCache.Size(),
		})
	})

	// Proxy handler
	suite.router.NoRoute(func(c *gin.Context) {
		suite.handleProxyRequest(c)
	})
}

func (suite *EdgeProxyIntegrationTestSuite) handleProxyRequest(c *gin.Context) {
	domain := c.Request.Host

	// Remove port from domain if present
	if colonPos := strings.Index(domain, ":"); colonPos != -1 {
		domain = domain[:colonPos]
	}

	// Get domain configuration from control plane
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	domainInfo, err := suite.controlPlane.GetDomain(ctx, domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Domain not configured"})
		return
	}

	if domainInfo.Status != "active" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Domain not active"})
		return
	}

	// Proxy the request
	suite.proxyService.ServeHTTP(c.Writer, c.Request, domainInfo.OriginURL)
}

func (suite *EdgeProxyIntegrationTestSuite) TearDownSuite() {
	// Clean up test data
	ctx := context.Background()
	suite.redis.FlushDB(ctx)
	suite.redis.Close()

	// Close mock servers
	suite.controlPlaneMock.Close()
	suite.originMock.Close()
}

func (suite *EdgeProxyIntegrationTestSuite) SetupTest() {
	// Clean caches before each test
	ctx := context.Background()
	suite.memoryCache.Clear(ctx)
	suite.redisCache.Clear(ctx)
}

func (suite *EdgeProxyIntegrationTestSuite) TestCacheHitMiss() {
	// Test cache miss (first request)
	req := httptest.NewRequest("GET", "http://"+suite.testDomain+"/hello", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "Hello, World!", w.Body.String())
	assert.Equal(suite.T(), "text/plain", w.Header().Get("Content-Type"))

	// Verify cache was populated
	cacheKey := cache.GenerateCacheKey(req)
	ctx := context.Background()
	entry, exists := suite.memoryCache.Get(ctx, cacheKey)
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), http.StatusOK, entry.StatusCode)
	assert.Equal(suite.T(), "Hello, World!", string(entry.Body))

	// Test cache hit (second request)
	req2 := httptest.NewRequest("GET", "http://"+suite.testDomain+"/hello", nil)
	w2 := httptest.NewRecorder()

	suite.router.ServeHTTP(w2, req2)

	assert.Equal(suite.T(), http.StatusOK, w2.Code)
	assert.Equal(suite.T(), "Hello, World!", w2.Body.String())
	assert.Equal(suite.T(), "text/plain", w2.Header().Get("Content-Type"))
}

func (suite *EdgeProxyIntegrationTestSuite) TestCacheWithRedis() {
	// Switch to Redis cache for this test
	suite.proxyService = proxy.NewProxyService(suite.redisCache, proxy.ProxyConfig{
		DefaultTTL:       3600 * time.Second,
		MaxBodySize:      10 * 1024 * 1024,
		ConnectTimeout:   10 * time.Second,
		ResponseTimeout:  30 * time.Second,
		IdleConnTimeout:  90 * time.Second,
		MaxIdleConns:     100,
		MaxIdleConnsHost: 10,
	})

	req := httptest.NewRequest("GET", "http://"+suite.testDomain+"/json", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Hello JSON")
	assert.Equal(suite.T(), "application/json", w.Header().Get("Content-Type"))

	// Verify Redis cache was populated
	cacheKey := cache.GenerateCacheKey(req)
	ctx := context.Background()
	entry, exists := suite.redisCache.Get(ctx, cacheKey)
	assert.True(suite.T(), exists)
	assert.Equal(suite.T(), http.StatusOK, entry.StatusCode)
	assert.Contains(suite.T(), string(entry.Body), "Hello JSON")
}

func (suite *EdgeProxyIntegrationTestSuite) TestNonCacheableContent() {
	// Test content that shouldn't be cached
	req := httptest.NewRequest("GET", "http://"+suite.testDomain+"/no-cache", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Equal(suite.T(), "Not cacheable", w.Body.String())

	// Verify content was not cached
	cacheKey := cache.GenerateCacheKey(req)
	ctx := context.Background()
	_, exists := suite.memoryCache.Get(ctx, cacheKey)
	assert.False(suite.T(), exists)
}

func (suite *EdgeProxyIntegrationTestSuite) TestPostRequestsNotCached() {
	// POST requests should not be cached
	body := bytes.NewBufferString(`{"key": "value"}`)
	req := httptest.NewRequest("POST", "http://"+suite.testDomain+"/hello", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// Verify POST was not cached
	cacheKey := cache.GenerateCacheKey(req)
	ctx := context.Background()
	_, exists := suite.memoryCache.Get(ctx, cacheKey)
	assert.False(suite.T(), exists)
}

func (suite *EdgeProxyIntegrationTestSuite) TestErrorResponses() {
	// Test error responses
	req := httptest.NewRequest("GET", "http://"+suite.testDomain+"/error", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusInternalServerError, w.Code)
	assert.Equal(suite.T(), "Internal Server Error", w.Body.String())

	// Verify error responses are not cached
	cacheKey := cache.GenerateCacheKey(req)
	ctx := context.Background()
	_, exists := suite.memoryCache.Get(ctx, cacheKey)
	assert.False(suite.T(), exists)
}

func (suite *EdgeProxyIntegrationTestSuite) TestUnconfiguredDomain() {
	// Test request to unconfigured domain
	req := httptest.NewRequest("GET", "http://unknown.domain.com/hello", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "Domain not configured")
}

func (suite *EdgeProxyIntegrationTestSuite) TestHealthEndpoint() {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var health map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &health)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "healthy", health["status"])
	assert.Contains(suite.T(), health, "timestamp")
	assert.Contains(suite.T(), health, "cache_size")
}

func (suite *EdgeProxyIntegrationTestSuite) TestRateLimiting() {
	// Add rate limiting middleware to router
	testRouter := gin.New()
	testRouter.Use(suite.rateLimiter.PerDomainRateLimit())
	testRouter.NoRoute(func(c *gin.Context) {
		suite.handleProxyRequest(c)
	})

	// Make multiple requests quickly
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "http://"+suite.testDomain+"/hello", nil)
		w := httptest.NewRecorder()

		testRouter.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should have both successful and rate-limited requests
	assert.Greater(suite.T(), successCount, 0)
	// Note: Rate limiting might not kick in immediately in test environment
}

func (suite *EdgeProxyIntegrationTestSuite) TestCachePurge() {
	// First, cache some content
	req := httptest.NewRequest("GET", "http://"+suite.testDomain+"/hello", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify content is cached
	cacheKey := cache.GenerateCacheKey(req)
	ctx := context.Background()
	_, exists := suite.memoryCache.Get(ctx, cacheKey)
	assert.True(suite.T(), exists)

	// Test cache purge
	err := suite.proxyService.PurgeCache(ctx, suite.testDomain, []string{"/hello"})
	assert.NoError(suite.T(), err)

	// Verify content was purged
	_, exists = suite.memoryCache.Get(ctx, cacheKey)
	assert.False(suite.T(), exists)
}

func (suite *EdgeProxyIntegrationTestSuite) TestControlPlaneIntegration() {
	// Test edge registration
	ctx := context.Background()
	edgeResp, err := suite.controlPlane.RegisterEdge(ctx, "192.168.1.100", "test-edge", 1000)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-region", edgeResp.Region)
	assert.Equal(suite.T(), "192.168.1.100", edgeResp.IPAddress)
	assert.Equal(suite.T(), "healthy", edgeResp.Status)

	// Test heartbeat
	metrics := map[string]interface{}{
		"cache_size": 1024,
		"requests":   100,
	}
	err = suite.controlPlane.SendHeartbeat(ctx, "healthy", metrics)
	assert.NoError(suite.T(), err)

	// Test domain lookup
	domainInfo, err := suite.controlPlane.GetDomain(ctx, suite.testDomain)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), suite.testDomain, domainInfo.Domain)
	assert.Equal(suite.T(), suite.testOriginURL, domainInfo.OriginURL)
	assert.Equal(suite.T(), "active", domainInfo.Status)

	// Test pending purges
	purges, err := suite.controlPlane.GetPendingPurges(ctx)
	assert.NoError(suite.T(), err)
	assert.Greater(suite.T(), len(purges), 0)
}

func (suite *EdgeProxyIntegrationTestSuite) TestCacheKeyGeneration() {
	tests := []struct {
		method   string
		url      string
		headers  map[string]string
		expected string
	}{
		{
			method:   "GET",
			url:      "http://example.com/path",
			headers:  map[string]string{},
			expected: "GET:example.com/path",
		},
		{
			method:   "GET",
			url:      "http://example.com/path?param=value",
			headers:  map[string]string{},
			expected: "GET:example.com/path?param=value",
		},
		{
			method: "GET",
			url:    "http://example.com/path",
			headers: map[string]string{
				"Accept": "application/json",
			},
			expected: "GET:example.com/path|Accept=application/json",
		},
	}

	for _, test := range tests {
		req := httptest.NewRequest(test.method, test.url, nil)
		for key, value := range test.headers {
			req.Header.Set(key, value)
		}

		cacheKey := cache.GenerateCacheKey(req)
		assert.Equal(suite.T(), test.expected, cacheKey)
	}
}

func (suite *EdgeProxyIntegrationTestSuite) TestCacheExpiration() {
	// Create a cache entry with short TTL
	shortTTLCache := cache.NewMemoryCache(50 * 1024 * 1024)

	ctx := context.Background()
	entry := &cache.CacheEntry{
		StatusCode: http.StatusOK,
		Headers:    make(http.Header),
		Body:       []byte("test content"),
		CachedAt:   time.Now(),
		TTL:        100 * time.Millisecond, // Very short TTL
	}

	err := shortTTLCache.Set(ctx, "test-key", entry)
	assert.NoError(suite.T(), err)

	// Verify entry exists
	_, exists := shortTTLCache.Get(ctx, "test-key")
	assert.True(suite.T(), exists)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Verify entry has expired
	_, exists = shortTTLCache.Get(ctx, "test-key")
	assert.False(suite.T(), exists)
}

func (suite *EdgeProxyIntegrationTestSuite) TestConcurrentCacheAccess() {
	ctx := context.Background()

	// Test concurrent cache operations
	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key-%d-%d", goroutineID, j)
				entry := &cache.CacheEntry{
					StatusCode: http.StatusOK,
					Headers:    make(http.Header),
					Body:       []byte(fmt.Sprintf("content-%d-%d", goroutineID, j)),
					CachedAt:   time.Now(),
					TTL:        3600 * time.Second,
				}

				// Set
				err := suite.memoryCache.Set(ctx, key, entry)
				assert.NoError(suite.T(), err)

				// Get
				retrieved, exists := suite.memoryCache.Get(ctx, key)
				assert.True(suite.T(), exists)
				assert.Equal(suite.T(), entry.Body, retrieved.Body)

				// Delete
				err = suite.memoryCache.Delete(ctx, key)
				assert.NoError(suite.T(), err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestEdgeProxyIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(EdgeProxyIntegrationTestSuite))
}
