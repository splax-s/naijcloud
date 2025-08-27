package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant in the multi-tenant system
type Organization struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Description string    `json:"description" db:"description"`
	Plan        string    `json:"plan" db:"plan"` // free, basic, pro, enterprise
	Settings    []byte    `json:"settings" db:"settings"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// User represents an individual user in the system
type User struct {
	ID                      uuid.UUID  `json:"id" db:"id"`
	Email                   string     `json:"email" db:"email"`
	Name                    string     `json:"name" db:"name"`
	PasswordHash            string     `json:"-" db:"password_hash"` // Never expose password hash in JSON
	EmailVerified           bool       `json:"email_verified" db:"email_verified"`
	EmailVerificationToken  *string    `json:"-" db:"email_verification_token"` // Never expose token in JSON
	EmailVerificationExpiry *time.Time `json:"-" db:"email_verification_expiry"`
	PasswordResetToken      *string    `json:"-" db:"password_reset_token"` // Never expose token in JSON
	PasswordResetExpiry     *time.Time `json:"-" db:"password_reset_expiry"`
	AvatarURL               string     `json:"avatar_url" db:"avatar_url"`
	Settings                []byte     `json:"settings" db:"settings"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
}

// OrganizationMember represents membership in an organization
type OrganizationMember struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	Role           string     `json:"role" db:"role"` // owner, admin, member, viewer
	Permissions    []byte     `json:"permissions" db:"permissions"`
	InvitedBy      *uuid.UUID `json:"invited_by" db:"invited_by"`
	InvitedAt      *time.Time `json:"invited_at" db:"invited_at"`
	JoinedAt       time.Time  `json:"joined_at" db:"joined_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	Name           string     `json:"name" db:"name"`
	KeyHash        string     `json:"-" db:"key_hash"` // Never expose hash
	KeyPrefix      string     `json:"key_prefix" db:"key_prefix"`
	Permissions    []byte     `json:"permissions" db:"permissions"`
	Scopes         []string   `json:"scopes" db:"scopes"`
	RateLimit      int        `json:"rate_limit" db:"rate_limit"`
	LastUsedAt     *time.Time `json:"last_used_at" db:"last_used_at"`
	LastUsedIP     *string    `json:"last_used_ip" db:"last_used_ip"`
	ExpiresAt      *time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// APIKeyUsage represents usage tracking for API keys
type APIKeyUsage struct {
	ID                uuid.UUID `json:"id" db:"id"`
	APIKeyID          uuid.UUID `json:"api_key_id" db:"api_key_id"`
	OrganizationID    uuid.UUID `json:"organization_id" db:"organization_id"`
	Endpoint          string    `json:"endpoint" db:"endpoint"`
	Method            string    `json:"method" db:"method"`
	StatusCode        int       `json:"status_code" db:"status_code"`
	ResponseTimeMs    *int      `json:"response_time_ms" db:"response_time_ms"`
	RequestSizeBytes  *int64    `json:"request_size_bytes" db:"request_size_bytes"`
	ResponseSizeBytes *int64    `json:"response_size_bytes" db:"response_size_bytes"`
	ClientIP          *string   `json:"client_ip" db:"client_ip"`
	UserAgent         *string   `json:"user_agent" db:"user_agent"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// Domain represents a registered domain in the system
type Domain struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID *uuid.UUID `json:"organization_id" db:"organization_id"`
	Domain         string     `json:"domain" db:"domain"`
	OriginURL      string     `json:"origin_url" db:"origin_url"`
	CacheTTL       int        `json:"cache_ttl" db:"cache_ttl"`
	RateLimit      int        `json:"rate_limit" db:"rate_limit"`
	Status         string     `json:"status" db:"status"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// Edge represents an edge proxy node
type Edge struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	OrganizationID *uuid.UUID  `json:"organization_id" db:"organization_id"`
	Region         string      `json:"region" db:"region"`
	IPAddress      string      `json:"ip_address" db:"ip_address"`
	Hostname       string      `json:"hostname" db:"hostname"`
	Capacity       int         `json:"capacity" db:"capacity"`
	Status         string      `json:"status" db:"status"`
	LastHeartbeat  time.Time   `json:"last_heartbeat" db:"last_heartbeat"`
	CreatedAt      time.Time   `json:"created_at" db:"created_at"`
	Metadata       interface{} `json:"metadata" db:"metadata"`
}

// CachePolicy represents caching rules for a domain
type CachePolicy struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	OrganizationID   *uuid.UUID `json:"organization_id" db:"organization_id"`
	DomainID         uuid.UUID  `json:"domain_id" db:"domain_id"`
	PathPattern      string     `json:"path_pattern" db:"path_pattern"`
	CacheTTL         int        `json:"cache_ttl" db:"cache_ttl"`
	CacheKeyTemplate string     `json:"cache_key_template" db:"cache_key_template"`
	HeadersToVary    []string   `json:"headers_to_vary" db:"headers_to_vary"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// RequestLog represents a logged HTTP request
type RequestLog struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID *uuid.UUID `json:"organization_id" db:"organization_id"`
	DomainID       uuid.UUID  `json:"domain_id" db:"domain_id"`
	EdgeID         uuid.UUID  `json:"edge_id" db:"edge_id"`
	RequestTime    time.Time  `json:"request_time" db:"request_time"`
	Method         string     `json:"method" db:"method"`
	Path           string     `json:"path" db:"path"`
	StatusCode     int        `json:"status_code" db:"status_code"`
	ResponseTimeMs int        `json:"response_time_ms" db:"response_time_ms"`
	BytesSent      int64      `json:"bytes_sent" db:"bytes_sent"`
	CacheStatus    string     `json:"cache_status" db:"cache_status"`
	ClientIP       string     `json:"client_ip" db:"client_ip"`
	UserAgent      string     `json:"user_agent" db:"user_agent"`
	Referer        string     `json:"referer" db:"referer"`
}

// PurgeRequest represents a cache purge operation
type PurgeRequest struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID *uuid.UUID `json:"organization_id" db:"organization_id"`
	DomainID       uuid.UUID  `json:"domain_id" db:"domain_id"`
	Paths          []string   `json:"paths" db:"paths"`
	Status         string     `json:"status" db:"status"`
	RequestedBy    string     `json:"requested_by" db:"requested_by"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time `json:"completed_at" db:"completed_at"`
}

// Analytics represents aggregated analytics data
type Analytics struct {
	Domain          string  `json:"domain"`
	TotalRequests   int64   `json:"total_requests"`
	CacheHitRatio   float64 `json:"cache_hit_ratio"`
	BandwidthSaved  int64   `json:"bandwidth_saved"`
	AvgResponseTime float64 `json:"avg_response_time"`
	P50ResponseTime float64 `json:"p50_response_time"`
	P95ResponseTime float64 `json:"p95_response_time"`
	P99ResponseTime float64 `json:"p99_response_time"`
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

// CreateAPIKeyRequest represents the request to create a new API key
type CreateAPIKeyRequest struct {
	Name        string              `json:"name" binding:"required"`
	Scopes      []string            `json:"scopes" binding:"required"`
	RateLimit   int                 `json:"rate_limit"`
	ExpiresAt   *string             `json:"expires_at"` // ISO 8601 date string
	Permissions map[string][]string `json:"permissions"`
}

// UpdateAPIKeyRequest represents the request to update an API key
type UpdateAPIKeyRequest struct {
	Name        string              `json:"name"`
	Scopes      []string            `json:"scopes"`
	RateLimit   int                 `json:"rate_limit"`
	ExpiresAt   *string             `json:"expires_at"` // ISO 8601 date string
	Permissions map[string][]string `json:"permissions"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	APIKey   *APIKey `json:"api_key"`
	PlainKey string  `json:"plain_key"` // Only returned once during creation
	Warning  string  `json:"warning,omitempty"`
}

// RegisterUserRequest represents the request to register a new user
type RegisterUserRequest struct {
	Email            string `json:"email" binding:"required,email"`
	Name             string `json:"name" binding:"required"`
	Password         string `json:"password" binding:"required,min=8"`
	ConfirmPassword  string `json:"confirm_password" binding:"required"`
	OrganizationName string `json:"organization_name" binding:"required"`
	OrganizationSlug string `json:"organization_slug" binding:"required"`
}

// LoginRequest represents the request to authenticate a user
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// SendEmailVerificationRequest represents the request to send email verification
type SendEmailVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// VerifyEmailRequest represents the request to verify email with token
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// RequestPasswordResetRequest represents the request to reset password
type RequestPasswordResetRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents the request to reset password with token
type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	Password        string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	User         *User         `json:"user"`
	Organization *Organization `json:"organization,omitempty"`
	AccessToken  string        `json:"access_token,omitempty"`
	Message      string        `json:"message"`
}
