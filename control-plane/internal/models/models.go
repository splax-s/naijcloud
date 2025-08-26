package models

import (
	"time"

	"github.com/google/uuid"
)

// Domain represents a registered domain in the system
type Domain struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Domain    string    `json:"domain" db:"domain"`
	OriginURL string    `json:"origin_url" db:"origin_url"`
	CacheTTL  int       `json:"cache_ttl" db:"cache_ttl"`
	RateLimit int       `json:"rate_limit" db:"rate_limit"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Edge represents an edge proxy node
type Edge struct {
	ID            uuid.UUID   `json:"id" db:"id"`
	Region        string      `json:"region" db:"region"`
	IPAddress     string      `json:"ip_address" db:"ip_address"`
	Hostname      string      `json:"hostname" db:"hostname"`
	Capacity      int         `json:"capacity" db:"capacity"`
	Status        string      `json:"status" db:"status"`
	LastHeartbeat time.Time   `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt     time.Time   `json:"created_at" db:"created_at"`
	Metadata      interface{} `json:"metadata" db:"metadata"`
}

// CachePolicy represents caching rules for a domain
type CachePolicy struct {
	ID                uuid.UUID `json:"id" db:"id"`
	DomainID          uuid.UUID `json:"domain_id" db:"domain_id"`
	PathPattern       string    `json:"path_pattern" db:"path_pattern"`
	CacheTTL          int       `json:"cache_ttl" db:"cache_ttl"`
	CacheKeyTemplate  string    `json:"cache_key_template" db:"cache_key_template"`
	HeadersToVary     []string  `json:"headers_to_vary" db:"headers_to_vary"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// RequestLog represents a logged HTTP request
type RequestLog struct {
	ID             uuid.UUID `json:"id" db:"id"`
	DomainID       uuid.UUID `json:"domain_id" db:"domain_id"`
	EdgeID         uuid.UUID `json:"edge_id" db:"edge_id"`
	RequestTime    time.Time `json:"request_time" db:"request_time"`
	Method         string    `json:"method" db:"method"`
	Path           string    `json:"path" db:"path"`
	StatusCode     int       `json:"status_code" db:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms" db:"response_time_ms"`
	BytesSent      int64     `json:"bytes_sent" db:"bytes_sent"`
	CacheStatus    string    `json:"cache_status" db:"cache_status"`
	ClientIP       string    `json:"client_ip" db:"client_ip"`
	UserAgent      string    `json:"user_agent" db:"user_agent"`
	Referer        string    `json:"referer" db:"referer"`
}

// PurgeRequest represents a cache purge operation
type PurgeRequest struct {
	ID          uuid.UUID `json:"id" db:"id"`
	DomainID    uuid.UUID `json:"domain_id" db:"domain_id"`
	Paths       []string  `json:"paths" db:"paths"`
	Status      string    `json:"status" db:"status"`
	RequestedBy string    `json:"requested_by" db:"requested_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
}

// Analytics represents aggregated analytics data
type Analytics struct {
	Domain           string  `json:"domain"`
	TotalRequests    int64   `json:"total_requests"`
	CacheHitRatio    float64 `json:"cache_hit_ratio"`
	BandwidthSaved   int64   `json:"bandwidth_saved"`
	AvgResponseTime  float64 `json:"avg_response_time"`
	P50ResponseTime  float64 `json:"p50_response_time"`
	P95ResponseTime  float64 `json:"p95_response_time"`
	P99ResponseTime  float64 `json:"p99_response_time"`
}

// CreateDomainRequest represents the request to create a new domain
type CreateDomainRequest struct {
	Domain    string `json:"domain" binding:"required"`
	OriginURL string `json:"origin_url" binding:"required"`
	CacheTTL  int    `json:"cache_ttl"`
}

// UpdateDomainRequest represents the request to update a domain
type UpdateDomainRequest struct {
	OriginURL string `json:"origin_url"`
	CacheTTL  int    `json:"cache_ttl"`
	RateLimit int    `json:"rate_limit"`
}

// RegisterEdgeRequest represents the request to register an edge node
type RegisterEdgeRequest struct {
	Region    string `json:"region" binding:"required"`
	IPAddress string `json:"ip_address" binding:"required"`
	Hostname  string `json:"hostname"`
	Capacity  int    `json:"capacity"`
}

// HeartbeatRequest represents an edge node heartbeat
type HeartbeatRequest struct {
	Status  string                 `json:"status" binding:"required"`
	Metrics map[string]interface{} `json:"metrics"`
}

// PurgeRequestBody represents a cache purge request
type PurgeRequestBody struct {
	Paths []string `json:"paths"`
}
