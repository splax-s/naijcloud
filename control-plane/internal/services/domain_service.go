package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type DomainService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewDomainService(db *sql.DB, redis *redis.Client) *DomainService {
	return &DomainService{
		db:    db,
		redis: redis,
	}
}

// CreateDomain creates a new domain registration for an organization
func (s *DomainService) CreateDomain(orgID uuid.UUID, req *models.CreateDomainRequest) (*models.Domain, error) {
	// Check if domain already exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM domains WHERE domain = $1)", req.Domain).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check domain existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("domain %s already exists", req.Domain)
	}

	// Set default cache TTL if not provided
	cacheTTL := req.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = 3600 // 1 hour default
	}

	domain := &models.Domain{
		ID:             uuid.New(),
		OrganizationID: &orgID,
		Domain:         req.Domain,
		OriginURL:      req.OriginURL,
		CacheTTL:       cacheTTL,
		RateLimit:      1000, // Default rate limit
		Status:         "active",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `
		INSERT INTO domains (id, organization_id, domain, origin_url, cache_ttl, rate_limit, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = s.db.Exec(query, domain.ID, domain.OrganizationID, domain.Domain, domain.OriginURL, domain.CacheTTL,
		domain.RateLimit, domain.Status, domain.CreatedAt, domain.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create domain: %w", err)
	}

	// Cache domain configuration in Redis
	s.cacheDomainConfig(domain)

	logrus.WithFields(logrus.Fields{
		"domain":          domain.Domain,
		"organization_id": orgID,
	}).Info("Domain created successfully")
	return domain, nil
}

// GetDomain retrieves a domain by name for an organization
func (s *DomainService) GetDomain(orgID uuid.UUID, domainName string) (*models.Domain, error) {
	var domain models.Domain
	row := s.db.QueryRow("SELECT id, organization_id, domain, origin_url, cache_ttl, rate_limit, status, created_at, updated_at FROM domains WHERE domain = $1 AND organization_id = $2", domainName, orgID)
	err := row.Scan(&domain.ID, &domain.OrganizationID, &domain.Domain, &domain.OriginURL, &domain.CacheTTL, &domain.RateLimit, &domain.Status, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain not found")
		}
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	return &domain, nil
}

// GetDomainByID retrieves a domain by ID for an organization
func (s *DomainService) GetDomainByID(orgID uuid.UUID, domainID uuid.UUID) (*models.Domain, error) {
	var domain models.Domain
	row := s.db.QueryRow("SELECT id, organization_id, domain, origin_url, cache_ttl, rate_limit, status, created_at, updated_at FROM domains WHERE id = $1 AND organization_id = $2", domainID, orgID)
	err := row.Scan(&domain.ID, &domain.OrganizationID, &domain.Domain, &domain.OriginURL, &domain.CacheTTL, &domain.RateLimit, &domain.Status, &domain.CreatedAt, &domain.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("domain not found")
		}
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	return &domain, nil
}

// ListDomains retrieves all domains for an organization
func (s *DomainService) ListDomains(orgID uuid.UUID) ([]*models.Domain, error) {
	query := `
		SELECT id, organization_id, domain, origin_url, cache_ttl, rate_limit, status, created_at, updated_at
		FROM domains WHERE organization_id = $1 ORDER BY created_at DESC
	`
	rows, err := s.db.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}
	defer rows.Close()

	var domains []*models.Domain
	for rows.Next() {
		domain := &models.Domain{}
		err := rows.Scan(
			&domain.ID, &domain.OrganizationID, &domain.Domain, &domain.OriginURL, &domain.CacheTTL,
			&domain.RateLimit, &domain.Status, &domain.CreatedAt, &domain.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan domain: %w", err)
		}
		domains = append(domains, domain)
	}

	return domains, nil
}

// UpdateDomain updates an existing domain for an organization
func (s *DomainService) UpdateDomain(orgID uuid.UUID, domainName string, req *models.UpdateDomainRequest) (*models.Domain, error) {
	// Get existing domain
	domain, err := s.GetDomain(orgID, domainName)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.OriginURL != "" {
		domain.OriginURL = req.OriginURL
	}
	if req.CacheTTL > 0 {
		domain.CacheTTL = req.CacheTTL
	}
	if req.RateLimit > 0 {
		domain.RateLimit = req.RateLimit
	}
	domain.UpdatedAt = time.Now()

	query := `
		UPDATE domains 
		SET origin_url = $1, cache_ttl = $2, rate_limit = $3, updated_at = $4
		WHERE domain = $5 AND organization_id = $6
	`
	_, err = s.db.Exec(query, domain.OriginURL, domain.CacheTTL, domain.RateLimit, domain.UpdatedAt, domainName, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	// Update cache
	s.cacheDomainConfig(domain)

	logrus.WithFields(logrus.Fields{
		"domain":          domainName,
		"organization_id": orgID,
	}).Info("Domain updated successfully")
	return domain, nil
}

// DeleteDomain removes a domain for an organization
func (s *DomainService) DeleteDomain(orgID uuid.UUID, domainName string) error {
	result, err := s.db.Exec("DELETE FROM domains WHERE domain = $1 AND organization_id = $2", domainName, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("domain not found")
	}

	// Remove from cache
	s.redis.Del(context.Background(), fmt.Sprintf("domain:%s", domainName))

	logrus.WithFields(logrus.Fields{
		"domain":          domainName,
		"organization_id": orgID,
	}).Info("Domain deleted successfully")
	return nil
}

// Helper methods for caching
func (s *DomainService) cacheDomainConfig(domain *models.Domain) {
	data, err := json.Marshal(domain)
	if err != nil {
		logrus.WithError(err).Warn("Failed to marshal domain for cache")
		return
	}

	key := fmt.Sprintf("domain:%s", domain.Domain)
	if err := s.redis.Set(context.Background(), key, data, 5*time.Minute).Err(); err != nil {
		logrus.WithError(err).Warn("Failed to cache domain config")
	}
}

func (s *DomainService) getCachedDomainConfig(domainName string) (*models.Domain, error) {
	key := fmt.Sprintf("domain:%s", domainName)
	data, err := s.redis.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var domain models.Domain
	if err := json.Unmarshal([]byte(data), &domain); err != nil {
		return nil, err
	}

	return &domain, nil
}
