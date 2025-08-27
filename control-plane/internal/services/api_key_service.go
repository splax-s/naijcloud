package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type APIKeyService struct {
	db *sql.DB
}

func NewAPIKeyService(db *sql.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

// GenerateAPIKey generates a new API key with the format: prefix_randomstring
func (s *APIKeyService) GenerateAPIKey(prefix string) (string, string, error) {
	// Generate 32 random bytes
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to hex string
	randomString := hex.EncodeToString(randomBytes)

	// Create the full API key
	fullKey := fmt.Sprintf("%s%s", prefix, randomString)

	// Create hash for storage
	hash, err := bcrypt.GenerateFromPassword([]byte(fullKey), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash API key: %w", err)
	}

	return fullKey, string(hash), nil
}

// CreateAPIKey creates a new API key for an organization
func (s *APIKeyService) CreateAPIKey(ctx context.Context, orgID, userID uuid.UUID, req *models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	// Generate API key with organization prefix
	orgPrefix := "nj_" + orgID.String()[:8] + "_"
	plainKey, keyHash, err := s.GenerateAPIKey(orgPrefix)
	if err != nil {
		return nil, err
	}

	// Set default values
	if req.RateLimit == 0 {
		req.RateLimit = 1000
	}

	// Parse expiration date if provided
	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_at format, use ISO 8601: %w", err)
		}
		expiresAt = &parsed
	}

	// Convert permissions to JSON
	permissionsJSON := "{}"
	if req.Permissions != nil {
		// Simple JSON encoding for permissions
		var parts []string
		for resource, actions := range req.Permissions {
			actionList := `["` + strings.Join(actions, `", "`) + `"]`
			parts = append(parts, fmt.Sprintf(`"%s": %s`, resource, actionList))
		}
		permissionsJSON = "{" + strings.Join(parts, ", ") + "}"
	}

	// Create API key in database
	apiKeyID := uuid.New()
	query := `
		INSERT INTO api_keys (
			id, organization_id, user_id, name, key_hash, key_prefix,
			permissions, scopes, rate_limit, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, organization_id, user_id, name, key_prefix, permissions, scopes, rate_limit, last_used_at, expires_at, created_at, updated_at
	`

	now := time.Now()
	apiKey := &models.APIKey{}
	var permissionsBytes []byte

	err = s.db.QueryRowContext(ctx, query,
		apiKeyID, orgID, userID, req.Name, keyHash, orgPrefix,
		permissionsJSON, req.Scopes, req.RateLimit, expiresAt, now, now,
	).Scan(
		&apiKey.ID, &apiKey.OrganizationID, &apiKey.UserID, &apiKey.Name,
		&apiKey.KeyPrefix, &permissionsBytes, &apiKey.Scopes, &apiKey.RateLimit,
		&apiKey.LastUsedAt, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	apiKey.Permissions = permissionsBytes

	return &models.CreateAPIKeyResponse{
		APIKey:   apiKey,
		PlainKey: plainKey,
		Warning:  "Store this API key securely. You will not be able to see it again.",
	}, nil
}

// GetAPIKey retrieves an API key by ID
func (s *APIKeyService) GetAPIKey(ctx context.Context, orgID, keyID uuid.UUID) (*models.APIKey, error) {
	query := `
		SELECT id, organization_id, user_id, name, key_prefix, 
		       COALESCE(permissions::text, '{}'), scopes, rate_limit,
		       last_used_at, last_used_ip, expires_at, created_at, updated_at
		FROM api_keys 
		WHERE id = $1 AND organization_id = $2 AND deleted_at IS NULL
	`

	apiKey := &models.APIKey{}
	var permissionsStr string
	var lastUsedIP sql.NullString

	err := s.db.QueryRowContext(ctx, query, keyID, orgID).Scan(
		&apiKey.ID, &apiKey.OrganizationID, &apiKey.UserID, &apiKey.Name,
		&apiKey.KeyPrefix, &permissionsStr, &apiKey.Scopes, &apiKey.RateLimit,
		&apiKey.LastUsedAt, &lastUsedIP, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("error retrieving API key: %w", err)
	}

	apiKey.Permissions = []byte(permissionsStr)
	if lastUsedIP.Valid {
		apiKey.LastUsedIP = &lastUsedIP.String
	}

	return apiKey, nil
}

// ListAPIKeys retrieves all API keys for an organization
func (s *APIKeyService) ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]*models.APIKey, error) {
	query := `
		SELECT id, organization_id, user_id, name, key_prefix,
		       COALESCE(permissions::text, '{}'), scopes, rate_limit,
		       last_used_at, last_used_ip, expires_at, created_at, updated_at
		FROM api_keys 
		WHERE organization_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("error listing API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*models.APIKey
	for rows.Next() {
		apiKey := &models.APIKey{}
		var permissionsStr string
		var lastUsedIP sql.NullString

		err := rows.Scan(
			&apiKey.ID, &apiKey.OrganizationID, &apiKey.UserID, &apiKey.Name,
			&apiKey.KeyPrefix, &permissionsStr, &apiKey.Scopes, &apiKey.RateLimit,
			&apiKey.LastUsedAt, &lastUsedIP, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning API key: %w", err)
		}

		apiKey.Permissions = []byte(permissionsStr)
		if lastUsedIP.Valid {
			apiKey.LastUsedIP = &lastUsedIP.String
		}

		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// UpdateAPIKey updates an existing API key
func (s *APIKeyService) UpdateAPIKey(ctx context.Context, orgID, keyID uuid.UUID, req *models.UpdateAPIKeyRequest) (*models.APIKey, error) {
	// Build dynamic update query
	var setParts []string
	var args []interface{}
	argIndex := 1

	if req.Name != "" {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, req.Name)
		argIndex++
	}

	if req.Scopes != nil {
		setParts = append(setParts, fmt.Sprintf("scopes = $%d", argIndex))
		args = append(args, req.Scopes)
		argIndex++
	}

	if req.RateLimit > 0 {
		setParts = append(setParts, fmt.Sprintf("rate_limit = $%d", argIndex))
		args = append(args, req.RateLimit)
		argIndex++
	}

	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			setParts = append(setParts, fmt.Sprintf("expires_at = NULL"))
		} else {
			parsed, err := time.Parse(time.RFC3339, *req.ExpiresAt)
			if err != nil {
				return nil, fmt.Errorf("invalid expires_at format, use ISO 8601: %w", err)
			}
			setParts = append(setParts, fmt.Sprintf("expires_at = $%d", argIndex))
			args = append(args, parsed)
			argIndex++
		}
	}

	if req.Permissions != nil {
		// Convert permissions to JSON
		var parts []string
		for resource, actions := range req.Permissions {
			actionList := `["` + strings.Join(actions, `", "`) + `"]`
			parts = append(parts, fmt.Sprintf(`"%s": %s`, resource, actionList))
		}
		permissionsJSON := "{" + strings.Join(parts, ", ") + "}"

		setParts = append(setParts, fmt.Sprintf("permissions = $%d", argIndex))
		args = append(args, permissionsJSON)
		argIndex++
	}

	if len(setParts) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	// Add updated_at
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE conditions
	args = append(args, keyID, orgID)

	query := fmt.Sprintf(`
		UPDATE api_keys 
		SET %s
		WHERE id = $%d AND organization_id = $%d AND deleted_at IS NULL
	`, strings.Join(setParts, ", "), argIndex-2, argIndex-1)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update API key: %w", err)
	}

	// Return updated API key
	return s.GetAPIKey(ctx, orgID, keyID)
}

// DeleteAPIKey soft deletes an API key
func (s *APIKeyService) DeleteAPIKey(ctx context.Context, orgID, keyID uuid.UUID) error {
	query := `
		UPDATE api_keys 
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND organization_id = $4 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := s.db.ExecContext(ctx, query, now, now, keyID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// AuthenticateAPIKey verifies an API key and returns the associated organization and user
func (s *APIKeyService) AuthenticateAPIKey(ctx context.Context, plainKey string) (*models.APIKey, error) {
	// Extract key prefix to optimize lookup
	parts := strings.Split(plainKey, "_")
	if len(parts) < 3 {
		logrus.WithField("parts", parts).Info("DEBUG: Invalid API key format")
		return nil, fmt.Errorf("invalid API key format")
	}
	keyPrefix := strings.Join(parts[:3], "_") + "_"
	logrus.WithFields(logrus.Fields{
		"plain_key":  plainKey,
		"key_prefix": keyPrefix,
	}).Info("DEBUG: Looking for API key")

	query := `
		SELECT id, organization_id, user_id, name, key_hash, key_prefix,
		       COALESCE(permissions::text, '{}'), scopes, rate_limit,
		       last_used_at, last_used_ip, expires_at, created_at, updated_at
		FROM api_keys 
		WHERE key_prefix = $1 AND deleted_at IS NULL
	`

	rows, err := s.db.QueryContext(ctx, query, keyPrefix)
	if err != nil {
		logrus.WithError(err).Info("DEBUG: Database query failed")
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	// Check each API key with this prefix
	foundRows := 0
	for rows.Next() {
		foundRows++
		logrus.WithField("row_number", foundRows).Info("DEBUG: Found API key row")
		apiKey := &models.APIKey{}
		var permissionsStr string
		var lastUsedIP sql.NullString

		err := rows.Scan(
			&apiKey.ID, &apiKey.OrganizationID, &apiKey.UserID, &apiKey.Name,
			&apiKey.KeyHash, &apiKey.KeyPrefix, &permissionsStr, &apiKey.Scopes, &apiKey.RateLimit,
			&apiKey.LastUsedAt, &lastUsedIP, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.UpdatedAt,
		)
		if err != nil {
			logrus.WithError(err).Info("DEBUG: Error scanning row")
			continue // Skip invalid rows
		}

		logrus.WithFields(logrus.Fields{
			"plain_key":  plainKey,
			"hash_start": apiKey.KeyHash[:20] + "...",
		}).Info("DEBUG: Testing bcrypt comparison")
		// Check if this key matches
		if err := bcrypt.CompareHashAndPassword([]byte(apiKey.KeyHash), []byte(plainKey)); err == nil {
			logrus.Info("DEBUG: ✓ bcrypt match successful!")
			// Check if key is expired
			if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
				return nil, fmt.Errorf("API key has expired")
			}

			apiKey.Permissions = []byte(permissionsStr)
			if lastUsedIP.Valid {
				apiKey.LastUsedIP = &lastUsedIP.String
			}

			// Update last used timestamp
			go s.updateLastUsed(context.Background(), apiKey.ID, "")

			return apiKey, nil
		} else {
			logrus.WithError(err).Info("DEBUG: ✗ bcrypt comparison failed")
		}
	}

	logrus.WithField("rows_checked", foundRows).Info("DEBUG: No matching API key found")
	return nil, fmt.Errorf("invalid API key")
}

// updateLastUsed updates the last used timestamp for an API key
func (s *APIKeyService) updateLastUsed(ctx context.Context, keyID uuid.UUID, clientIP string) {
	query := `
		UPDATE api_keys 
		SET last_used_at = $1, last_used_ip = $2, updated_at = $3
		WHERE id = $4
	`

	now := time.Now()
	var ip *string
	if clientIP != "" {
		ip = &clientIP
	}

	s.db.ExecContext(ctx, query, now, ip, now, keyID)
}

// RecordUsage records API key usage for analytics
func (s *APIKeyService) RecordUsage(ctx context.Context, keyID, orgID uuid.UUID, endpoint, method string, statusCode int, responseTimeMs int, clientIP, userAgent string) error {
	query := `
		INSERT INTO api_key_usage (
			id, api_key_id, organization_id, endpoint, method, status_code,
			response_time_ms, client_ip, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	usageID := uuid.New()
	var ip, ua *string
	if clientIP != "" {
		ip = &clientIP
	}
	if userAgent != "" {
		ua = &userAgent
	}

	_, err := s.db.ExecContext(ctx, query,
		usageID, keyID, orgID, endpoint, method, statusCode,
		responseTimeMs, ip, ua, time.Now(),
	)

	return err
}
