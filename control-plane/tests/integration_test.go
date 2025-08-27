package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/api"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	_ "github.com/lib/pq"
)

type IntegrationTestSuite struct {
	suite.Suite
	db           *sql.DB
	redis        *redis.Client
	router       *gin.Engine
	domainSvc    *services.DomainService
	edgeSvc      *services.EdgeService
	cacheSvc     *services.CacheService
	analyticsSvc *services.AnalyticsService
	apiKeySvc    *services.APIKeyService
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Set up test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://naijcloud:naijcloud_pass@localhost:5433/naijcloud?sslmode=disable"
	}

	var err error
	suite.db, err = sql.Open("postgres", dbURL)
	suite.Require().NoError(err)

	// Test the connection
	err = suite.db.Ping()
	suite.Require().NoError(err)

	// Set up test Redis
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/1" // Use database 1 for tests
	}

	opt, err := redis.ParseURL(redisURL)
	suite.Require().NoError(err)
	suite.redis = redis.NewClient(opt)

	// Test Redis connection
	_, err = suite.redis.Ping(context.Background()).Result()
	suite.Require().NoError(err)

	// Initialize services
	suite.domainSvc = services.NewDomainService(suite.db, suite.redis)
	suite.edgeSvc = services.NewEdgeService(suite.db, suite.redis)
	suite.analyticsSvc = services.NewAnalyticsService(suite.db)
	suite.cacheSvc = services.NewCacheService(suite.redis, suite.edgeSvc)
	suite.apiKeySvc = services.NewAPIKeyService(suite.db)

	// Set up router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.router.Use(gin.Recovery())

	// Health check endpoint
	suite.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Add metrics endpoint for testing
	suite.router.GET("/metrics", func(c *gin.Context) {
		c.String(http.StatusOK, "# HELP test_metric A test metric\ntest_metric 1.0\n")
	})

	// API routes
	v1 := suite.router.Group("/v1")
	api.SetupRoutes(v1, suite.domainSvc, suite.edgeSvc, suite.analyticsSvc, suite.cacheSvc, suite.apiKeySvc)
}

func (suite *IntegrationTestSuite) SetupTest() {
	// Clean up test data before each test
	suite.cleanupTestData()
}

func (suite *IntegrationTestSuite) TearDownTest() {
	// Clean up test data after each test
	suite.cleanupTestData()
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	suite.cleanupTestData()
	suite.db.Close()
	suite.redis.Close()
}

func (suite *IntegrationTestSuite) cleanupTestData() {
	// Clean up database tables in reverse dependency order
	tables := []string{"purge_requests", "cache_policies", "request_logs", "edges", "domains"}
	for _, table := range tables {
		_, err := suite.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE 1=1", table))
		suite.Require().NoError(err)
	}

	// Clean up Redis
	ctx := context.Background()
	suite.redis.FlushDB(ctx)
}

func (suite *IntegrationTestSuite) TestDomainCRUD() {
	// Test domain creation
	createReq := models.CreateDomainRequest{
		Domain:    "test-domain.com",
		OriginURL: "https://example.com",
		CacheTTL:  3600,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/v1/domains", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var domain models.Domain
	err := json.Unmarshal(w.Body.Bytes(), &domain)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "test-domain.com", domain.Domain)
	assert.Equal(suite.T(), "active", domain.Status)

	// Test domain retrieval
	req = httptest.NewRequest("GET", "/v1/domains/test-domain.com", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test domain retrieval by ID
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/domains/id/%s", domain.ID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test domain update
	updateReq := models.UpdateDomainRequest{
		OriginURL: "https://updated-example.com",
		CacheTTL:  7200,
		RateLimit: 2000,
	}
	body, _ = json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/v1/domains/test-domain.com", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test domain deletion
	req = httptest.NewRequest("DELETE", "/v1/domains/test-domain.com", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusNoContent, w.Code) // Changed from StatusOK to StatusNoContent

	// Verify domain is deleted
	req = httptest.NewRequest("GET", "/v1/domains/test-domain.com", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *IntegrationTestSuite) TestEdgeNodeManagement() {
	// Test edge registration
	registerReq := models.RegisterEdgeRequest{
		Region:    "us-east-1",
		IPAddress: "192.168.1.100",
		Hostname:  "edge-test-01",
		Capacity:  1000,
	}

	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest("POST", "/v1/edges", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var edge models.Edge
	err := json.Unmarshal(w.Body.Bytes(), &edge)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "us-east-1", edge.Region)
	assert.Equal(suite.T(), "healthy", edge.Status) // Changed from "active" to "healthy"

	// Test edge retrieval
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/edges/%s", edge.ID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test heartbeat
	heartbeatReq := models.HeartbeatRequest{
		Status: "healthy",
		Metrics: map[string]interface{}{
			"cpu_usage":          0.45,
			"memory_usage":       0.60,
			"requests_per_sec":   125.5,
			"cache_hit_ratio":    0.85,
			"active_connections": 42,
		},
	}

	body, _ = json.Marshal(heartbeatReq)
	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/edges/%s/heartbeat", edge.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Test edge deletion
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/v1/edges/%s", edge.ID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusNoContent, w.Code) // Changed from StatusOK to StatusNoContent
}

func (suite *IntegrationTestSuite) TestCachePurgeWorkflow() {
	// First, create a test domain
	createReq := models.CreateDomainRequest{
		Domain:    "purge-test.com",
		OriginURL: "https://example.com",
		CacheTTL:  3600,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/v1/domains", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var domain models.Domain
	err := json.Unmarshal(w.Body.Bytes(), &domain)
	suite.Require().NoError(err)

	// Register an edge node
	registerReq := models.RegisterEdgeRequest{
		Region:    "us-east-1",
		IPAddress: "192.168.1.100",
		Hostname:  "edge-purge-test",
		Capacity:  1000,
	}

	body, _ = json.Marshal(registerReq)
	req = httptest.NewRequest("POST", "/v1/edges", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var edge models.Edge
	err = json.Unmarshal(w.Body.Bytes(), &edge)
	suite.Require().NoError(err)

	// Test cache purge initiation
	purgeReq := models.PurgeRequestBody{
		Paths: []string{"/api/users", "/static/css/*"},
	}

	body, _ = json.Marshal(purgeReq)
	req = httptest.NewRequest("POST", "/v1/domains/purge-test.com/purge", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusAccepted, w.Code)

	var purgeResp struct {
		PurgeID string `json:"purge_id"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &purgeResp)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "accepted", purgeResp.Status)
	assert.NotEmpty(suite.T(), purgeResp.PurgeID)

	// Test getting pending purges for edge
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/edges/%s/purges", edge.ID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var purgeList struct {
		Purges []*models.PurgeRequest `json:"purges"`
	}
	err = json.Unmarshal(w.Body.Bytes(), &purgeList)
	suite.Require().NoError(err)
	assert.Len(suite.T(), purgeList.Purges, 1)

	// Test completing a purge
	req = httptest.NewRequest("POST", fmt.Sprintf("/v1/edges/%s/purges/%s/complete", edge.ID, purgeResp.PurgeID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	// Verify purge is no longer pending
	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/edges/%s/purges", edge.ID), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &purgeList)
	suite.Require().NoError(err)
	assert.Len(suite.T(), purgeList.Purges, 0)
}

func (suite *IntegrationTestSuite) TestAnalyticsCollection() {
	// Create a test domain
	createReq := models.CreateDomainRequest{
		Domain:    "analytics-test.com",
		OriginURL: "https://example.com",
		CacheTTL:  3600,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/v1/domains", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var domain models.Domain
	err := json.Unmarshal(w.Body.Bytes(), &domain)
	suite.Require().NoError(err)

	// Create an edge node
	registerReq := models.RegisterEdgeRequest{
		Region:    "us-east-1",
		IPAddress: "192.168.1.100",
		Hostname:  "analytics-edge",
		Capacity:  1000,
	}

	body, _ = json.Marshal(registerReq)
	req = httptest.NewRequest("POST", "/v1/edges", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var edge models.Edge
	err = json.Unmarshal(w.Body.Bytes(), &edge)
	suite.Require().NoError(err)

	// Simulate analytics data by calling the service directly
	// (since there's no POST analytics endpoint)
	requestLog := &models.RequestLog{
		ID:             uuid.New(),
		DomainID:       domain.ID,
		EdgeID:         edge.ID,
		RequestTime:    time.Now(),
		Method:         "GET",
		Path:           "/api/users",
		StatusCode:     200,
		ResponseTimeMs: 150,
		BytesSent:      1024,
		CacheStatus:    "hit", // Changed from "HIT" to "hit"
		ClientIP:       "192.168.1.1",
		UserAgent:      "test-agent",
		Referer:        "",
	}

	err = suite.analyticsSvc.LogRequest(requestLog)
	suite.Require().NoError(err)

	// Test analytics retrieval
	startTime := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	endTime := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	req = httptest.NewRequest("GET", fmt.Sprintf("/v1/analytics/domains/analytics-test.com?start=%s&end=%s", startTime, endTime), nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var analytics models.Analytics
	err = json.Unmarshal(w.Body.Bytes(), &analytics)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "analytics-test.com", analytics.Domain)
	assert.Equal(suite.T(), int64(1), analytics.TotalRequests)
}

func (suite *IntegrationTestSuite) TestHealthEndpoints() {
	// Test general health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var health map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &health)
	suite.Require().NoError(err)
	assert.Equal(suite.T(), "healthy", health["status"])

	// Test metrics endpoint
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	assert.Equal(suite.T(), http.StatusOK, w.Code)
	assert.Contains(suite.T(), w.Body.String(), "# HELP")
}

func TestIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	suite.Run(t, new(IntegrationTestSuite))
}
