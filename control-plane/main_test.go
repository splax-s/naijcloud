package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestCreateDomainRequest(t *testing.T) {
	createReq := models.CreateDomainRequest{
		Domain:    "example.com",
		OriginURL: "https://origin.example.com",
		CacheTTL:  3600,
	}

	jsonData, err := json.Marshal(createReq)
	assert.NoError(t, err)

	// Test JSON marshaling/unmarshaling
	var parsed models.CreateDomainRequest
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, createReq.Domain, parsed.Domain)
	assert.Equal(t, createReq.OriginURL, parsed.OriginURL)
	assert.Equal(t, createReq.CacheTTL, parsed.CacheTTL)
}

func TestRegisterEdgeRequest(t *testing.T) {
	edgeReq := models.RegisterEdgeRequest{
		Region:    "us-east-1",
		IPAddress: "10.0.1.100",
		Hostname:  "edge-01.us-east-1",
		Capacity:  1000,
	}

	jsonData, err := json.Marshal(edgeReq)
	assert.NoError(t, err)

	// Test JSON marshaling/unmarshaling
	var parsed models.RegisterEdgeRequest
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, edgeReq.Region, parsed.Region)
	assert.Equal(t, edgeReq.IPAddress, parsed.IPAddress)
	assert.Equal(t, edgeReq.Hostname, parsed.Hostname)
	assert.Equal(t, edgeReq.Capacity, parsed.Capacity)
}

func TestPurgeRequestBody(t *testing.T) {
	purgeReq := models.PurgeRequestBody{
		Paths: []string{"/", "/api/*", "/images/*"},
	}

	jsonData, err := json.Marshal(purgeReq)
	assert.NoError(t, err)

	// Test JSON marshaling/unmarshaling
	var parsed models.PurgeRequestBody
	err = json.Unmarshal(jsonData, &parsed)
	assert.NoError(t, err)
	assert.Equal(t, len(purgeReq.Paths), len(parsed.Paths))
	assert.Equal(t, purgeReq.Paths[0], parsed.Paths[0])
}
