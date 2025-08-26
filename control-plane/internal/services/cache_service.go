package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type CacheService struct {
	redis       *redis.Client
	edgeService *EdgeService
}

func NewCacheService(redis *redis.Client, edgeService *EdgeService) *CacheService {
	return &CacheService{
		redis:       redis,
		edgeService: edgeService,
	}
}

// PurgeCache initiates a cache purge for specified paths
func (s *CacheService) PurgeCache(domainID uuid.UUID, paths []string, requestedBy string) (*models.PurgeRequest, error) {
	purgeReq := &models.PurgeRequest{
		ID:          uuid.New(),
		DomainID:    domainID,
		Paths:       paths,
		Status:      "pending",
		RequestedBy: requestedBy,
		CreatedAt:   time.Now(),
	}

	// Store purge request in Redis for edge nodes to pick up
	purgeKey := fmt.Sprintf("purge:%s", purgeReq.ID)

	// Convert paths slice to JSON string for Redis storage
	pathsJSON, err := json.Marshal(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal paths: %w", err)
	}

	// Use individual SET operations instead of HMSet to avoid marshalling issues
	pipe := s.redis.Pipeline()
	pipe.HSet(context.Background(), purgeKey, "id", purgeReq.ID.String())
	pipe.HSet(context.Background(), purgeKey, "domain_id", domainID.String())
	pipe.HSet(context.Background(), purgeKey, "paths", string(pathsJSON))
	pipe.HSet(context.Background(), purgeKey, "created_at", fmt.Sprintf("%d", purgeReq.CreatedAt.Unix()))
	pipe.Expire(context.Background(), purgeKey, time.Hour)

	if _, err := pipe.Exec(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to store purge request: %w", err)
	}
	if err := s.redis.Expire(context.Background(), purgeKey, time.Hour).Err(); err != nil {
		logrus.WithError(err).Warn("Failed to set purge request expiration")
	}

	// Notify all healthy edge nodes
	if err := s.notifyEdgeNodes(purgeReq); err != nil {
		logrus.WithError(err).Warn("Failed to notify some edge nodes")
	}

	logrus.WithFields(logrus.Fields{
		"purge_id":  purgeReq.ID,
		"domain_id": domainID,
		"paths":     paths,
	}).Info("Cache purge initiated")

	return purgeReq, nil
}

// GetPendingPurges returns pending purge requests for an edge node
func (s *CacheService) GetPendingPurges(edgeID uuid.UUID) ([]*models.PurgeRequest, error) {
	// Get all purge requests from the edge's queue
	queueKey := fmt.Sprintf("edge:%s:purge_queue", edgeID)
	purgeIDs, err := s.redis.LRange(context.Background(), queueKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get purge queue: %w", err)
	}

	var purgeRequests []*models.PurgeRequest
	for _, purgeIDStr := range purgeIDs {
		purgeKey := fmt.Sprintf("purge:%s", purgeIDStr)
		purgeData, err := s.redis.HGetAll(context.Background(), purgeKey).Result()
		if err != nil {
			logrus.WithError(err).WithField("purge_id", purgeIDStr).Warn("Failed to get purge request")
			continue
		}

		if len(purgeData) == 0 {
			// Purge request expired or doesn't exist, remove from queue
			s.redis.LRem(context.Background(), queueKey, 1, purgeIDStr)
			continue
		}

		purgeID, _ := uuid.Parse(purgeData["id"])
		domainID, _ := uuid.Parse(purgeData["domain_id"])

		// Parse paths JSON string back to slice
		var paths []string
		if pathsStr, exists := purgeData["paths"]; exists {
			if err := json.Unmarshal([]byte(pathsStr), &paths); err != nil {
				logrus.WithError(err).WithField("purge_id", purgeIDStr).Warn("Failed to parse purge paths")
				paths = []string{pathsStr} // Fallback to single path
			}
		}

		purgeReq := &models.PurgeRequest{
			ID:       purgeID,
			DomainID: domainID,
			Paths:    paths,
			Status:   "pending",
		}
		purgeRequests = append(purgeRequests, purgeReq)
	}

	return purgeRequests, nil
}

// CompletePurge marks a purge request as completed for an edge node
func (s *CacheService) CompletePurge(edgeID, purgeID uuid.UUID) error {
	// Remove from edge's purge queue
	queueKey := fmt.Sprintf("edge:%s:purge_queue", edgeID)
	if err := s.redis.LRem(context.Background(), queueKey, 1, purgeID.String()).Err(); err != nil {
		return fmt.Errorf("failed to remove from purge queue: %w", err)
	}

	// Mark as completed in the edge's completion set
	completionKey := fmt.Sprintf("purge:%s:completed", purgeID)
	if err := s.redis.SAdd(context.Background(), completionKey, edgeID.String()).Err(); err != nil {
		return fmt.Errorf("failed to mark purge as completed: %w", err)
	}

	// Set expiration for completion tracking
	s.redis.Expire(context.Background(), completionKey, 24*time.Hour)

	logrus.WithFields(logrus.Fields{
		"edge_id":  edgeID,
		"purge_id": purgeID,
	}).Info("Purge completed by edge node")

	return nil
}

// InvalidateDomainCache invalidates all cached content for a domain
func (s *CacheService) InvalidateDomainCache(domain string) error {
	// Use Redis pattern matching to find and delete domain-related cache keys
	pattern := fmt.Sprintf("cache:%s:*", domain)
	keys, err := s.redis.Keys(context.Background(), pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find cache keys: %w", err)
	}

	if len(keys) > 0 {
		if err := s.redis.Del(context.Background(), keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
	}

	logrus.WithFields(logrus.Fields{
		"domain":     domain,
		"keys_count": len(keys),
	}).Info("Domain cache invalidated")

	return nil
}

// SetCache stores content in the cache
func (s *CacheService) SetCache(key string, content []byte, ttl time.Duration) error {
	if err := s.redis.Set(context.Background(), key, content, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

// GetCache retrieves content from the cache
func (s *CacheService) GetCache(key string) ([]byte, error) {
	content, err := s.redis.Get(context.Background(), key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}
	return content, nil
}

// Helper method to notify edge nodes about purge requests
func (s *CacheService) notifyEdgeNodes(purgeReq *models.PurgeRequest) error {
	// Get all healthy edge nodes
	edges, err := s.edgeService.GetHealthyEdges("")
	if err != nil {
		return fmt.Errorf("failed to get healthy edges: %w", err)
	}

	// Add purge request to each edge's queue
	for _, edge := range edges {
		queueKey := fmt.Sprintf("edge:%s:purge_queue", edge.ID)
		if err := s.redis.LPush(context.Background(), queueKey, purgeReq.ID.String()).Err(); err != nil {
			logrus.WithError(err).WithField("edge_id", edge.ID).Warn("Failed to add purge to edge queue")
			continue
		}

		// Set queue expiration
		s.redis.Expire(context.Background(), queueKey, 24*time.Hour)
	}

	return nil
}
