package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type EdgeService struct {
	db    *sql.DB
	redis *redis.Client
}

func NewEdgeService(db *sql.DB, redis *redis.Client) *EdgeService {
	return &EdgeService{
		db:    db,
		redis: redis,
	}
}

// RegisterEdge registers a new edge node
func (s *EdgeService) RegisterEdge(req *models.RegisterEdgeRequest) (*models.Edge, error) {
	edge := &models.Edge{
		ID:            uuid.New(),
		Region:        req.Region,
		IPAddress:     req.IPAddress,
		Hostname:      req.Hostname,
		Capacity:      req.Capacity,
		Status:        "healthy",
		LastHeartbeat: time.Now(),
		CreatedAt:     time.Now(),
		Metadata:      map[string]interface{}{},
	}

	if edge.Capacity == 0 {
		edge.Capacity = 1000 // Default capacity
	}

	query := `
		INSERT INTO edges (id, region, ip_address, hostname, capacity, status, last_heartbeat, created_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	metadataJSON, _ := json.Marshal(edge.Metadata)
	_, err := s.db.Exec(query, edge.ID, edge.Region, edge.IPAddress, edge.Hostname,
		edge.Capacity, edge.Status, edge.LastHeartbeat, edge.CreatedAt, metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to register edge: %w", err)
	}

	// Cache edge information
	s.cacheEdge(edge)

	logrus.WithFields(logrus.Fields{
		"edge_id": edge.ID,
		"region":  edge.Region,
		"ip":      edge.IPAddress,
	}).Info("Edge node registered successfully")

	return edge, nil
}

// GetEdge retrieves an edge by ID
func (s *EdgeService) GetEdge(edgeID uuid.UUID) (*models.Edge, error) {
	// Try cache first
	if cached, err := s.getCachedEdge(edgeID); err == nil && cached != nil {
		return cached, nil
	}

	edge := &models.Edge{}
	var metadataJSON []byte

	query := `
		SELECT id, region, ip_address, hostname, capacity, status, last_heartbeat, created_at, metadata
		FROM edges WHERE id = $1
	`
	err := s.db.QueryRow(query, edgeID).Scan(
		&edge.ID, &edge.Region, &edge.IPAddress, &edge.Hostname, &edge.Capacity,
		&edge.Status, &edge.LastHeartbeat, &edge.CreatedAt, &metadataJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("edge not found")
		}
		return nil, fmt.Errorf("failed to get edge: %w", err)
	}

	// Parse metadata
	if len(metadataJSON) > 0 {
		json.Unmarshal(metadataJSON, &edge.Metadata)
	}

	// Cache the result
	s.cacheEdge(edge)

	return edge, nil
}

// ListEdges retrieves all edge nodes
func (s *EdgeService) ListEdges() ([]*models.Edge, error) {
	query := `
		SELECT id, region, ip_address, hostname, capacity, status, last_heartbeat, created_at, metadata
		FROM edges ORDER BY region, created_at DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list edges: %w", err)
	}
	defer rows.Close()

	var edges []*models.Edge
	for rows.Next() {
		edge := &models.Edge{}
		var metadataJSON []byte

		err := rows.Scan(
			&edge.ID, &edge.Region, &edge.IPAddress, &edge.Hostname, &edge.Capacity,
			&edge.Status, &edge.LastHeartbeat, &edge.CreatedAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &edge.Metadata)
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// UpdateHeartbeat updates edge heartbeat and metrics
func (s *EdgeService) UpdateHeartbeat(edgeID uuid.UUID, req *models.HeartbeatRequest) error {
	edge, err := s.GetEdge(edgeID)
	if err != nil {
		return err
	}

	// Update edge status and heartbeat
	edge.Status = req.Status
	edge.LastHeartbeat = time.Now()

	// Merge metrics into metadata
	if edge.Metadata == nil {
		edge.Metadata = make(map[string]interface{})
	}

	metadata := edge.Metadata.(map[string]interface{})
	metadata["last_metrics"] = req.Metrics
	metadata["metrics_timestamp"] = time.Now().Unix()

	metadataJSON, _ := json.Marshal(metadata)

	query := `
		UPDATE edges 
		SET status = $1, last_heartbeat = $2, metadata = $3
		WHERE id = $4
	`
	_, err = s.db.Exec(query, edge.Status, edge.LastHeartbeat, metadataJSON, edgeID)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	// Update cache
	edge.Metadata = metadata
	s.cacheEdge(edge)

	return nil
}

// GetHealthyEdges returns edges that are healthy and recently heartbeated
func (s *EdgeService) GetHealthyEdges(region string) ([]*models.Edge, error) {
	query := `
		SELECT id, region, ip_address, hostname, capacity, status, last_heartbeat, created_at, metadata
		FROM edges 
		WHERE status = 'healthy' 
		AND last_heartbeat > NOW() - INTERVAL '5 minutes'
		AND ($1 = '' OR region = $1)
		ORDER BY region, last_heartbeat DESC
	`
	rows, err := s.db.Query(query, region)
	if err != nil {
		return nil, fmt.Errorf("failed to get healthy edges: %w", err)
	}
	defer rows.Close()

	var edges []*models.Edge
	for rows.Next() {
		edge := &models.Edge{}
		var metadataJSON []byte

		err := rows.Scan(
			&edge.ID, &edge.Region, &edge.IPAddress, &edge.Hostname, &edge.Capacity,
			&edge.Status, &edge.LastHeartbeat, &edge.CreatedAt, &metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &edge.Metadata)
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// DeleteEdge removes an edge node
func (s *EdgeService) DeleteEdge(edgeID uuid.UUID) error {
	result, err := s.db.Exec("DELETE FROM edges WHERE id = $1", edgeID)
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("edge not found")
	}

	// Remove from cache
	s.redis.Del(context.Background(), fmt.Sprintf("edge:%s", edgeID))

	logrus.WithField("edge_id", edgeID).Info("Edge node deleted successfully")
	return nil
}

// Organization-scoped edge management methods

// ListEdgesByOrganization retrieves all edge nodes for a specific organization
func (s *EdgeService) ListEdgesByOrganization(orgID uuid.UUID) ([]models.Edge, error) {
	query := `
		SELECT id, organization_id, region, ip_address, hostname, capacity, status, 
		       last_heartbeat, created_at, metadata
		FROM edges 
		WHERE organization_id = $1 OR organization_id IS NULL
		ORDER BY created_at DESC
	`
	
	rows, err := s.db.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer rows.Close()

	var edges []models.Edge
	for rows.Next() {
		var edge models.Edge
		var metadataBytes []byte
		
		err := rows.Scan(
			&edge.ID, &edge.OrganizationID, &edge.Region, &edge.IPAddress,
			&edge.Hostname, &edge.Capacity, &edge.Status, &edge.LastHeartbeat,
			&edge.CreatedAt, &metadataBytes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}

		// Parse metadata JSON
		if len(metadataBytes) > 0 {
			var metadata map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &metadata); err == nil {
				edge.Metadata = metadata
			}
		}

		edges = append(edges, edge)
	}

	return edges, nil
}

// GetEdgeByOrganization retrieves a specific edge node for an organization
func (s *EdgeService) GetEdgeByOrganization(orgID, edgeID uuid.UUID) (*models.Edge, error) {
	query := `
		SELECT id, organization_id, region, ip_address, hostname, capacity, status, 
		       last_heartbeat, created_at, metadata
		FROM edges 
		WHERE id = $1 AND (organization_id = $2 OR organization_id IS NULL)
	`
	
	var edge models.Edge
	var metadataBytes []byte
	
	err := s.db.QueryRow(query, edgeID, orgID).Scan(
		&edge.ID, &edge.OrganizationID, &edge.Region, &edge.IPAddress,
		&edge.Hostname, &edge.Capacity, &edge.Status, &edge.LastHeartbeat,
		&edge.CreatedAt, &metadataBytes,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("edge not found")
		}
		return nil, fmt.Errorf("failed to get edge: %w", err)
	}

	// Parse metadata JSON
	if len(metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err == nil {
			edge.Metadata = metadata
		}
	}

	return &edge, nil
}

// RegisterEdgeForOrganization registers a new edge node for a specific organization
func (s *EdgeService) RegisterEdgeForOrganization(orgID uuid.UUID, req *models.RegisterEdgeRequest) (*models.Edge, error) {
	edge := &models.Edge{
		ID:             uuid.New(),
		OrganizationID: &orgID,
		Region:         req.Region,
		IPAddress:      req.IPAddress,
		Hostname:       req.Hostname,
		Capacity:       req.Capacity,
		Status:         "healthy",
		LastHeartbeat:  time.Now(),
		CreatedAt:      time.Now(),
		Metadata:       make(map[string]interface{}),
	}

	// Set default hostname if not provided
	if edge.Hostname == "" {
		edge.Hostname = fmt.Sprintf("edge-%s", edge.ID.String()[:8])
	}

	// Set default capacity if not provided
	if edge.Capacity == 0 {
		edge.Capacity = 1000
	}

	// Serialize metadata
	metadataBytes, err := json.Marshal(edge.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO edges (id, organization_id, region, ip_address, hostname, capacity, status, last_heartbeat, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = s.db.Exec(query, edge.ID, edge.OrganizationID, edge.Region, edge.IPAddress, 
		edge.Hostname, edge.Capacity, edge.Status, edge.LastHeartbeat, metadataBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"edge_id":         edge.ID,
		"organization_id": orgID,
		"region":          edge.Region,
		"hostname":        edge.Hostname,
	}).Info("Edge node registered for organization")

	return edge, nil
}

// UpdateEdgeForOrganization updates an edge node for a specific organization
func (s *EdgeService) UpdateEdgeForOrganization(orgID, edgeID uuid.UUID, updates map[string]interface{}) (*models.Edge, error) {
	// First verify the edge belongs to this organization
	edge, err := s.GetEdgeByOrganization(orgID, edgeID)
	if err != nil {
		return nil, err
	}

	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		switch field {
		case "region", "hostname", "capacity", "status":
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		case "metadata":
			metadataBytes, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}
			setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
			args = append(args, metadataBytes)
			argIndex++
		}
	}

	if len(setParts) == 0 {
		return edge, nil // No updates to make
	}

	// Add updated_at timestamp
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add WHERE clause parameters
	args = append(args, edgeID, orgID)

	query := fmt.Sprintf(`
		UPDATE edges 
		SET %s
		WHERE id = $%d AND (organization_id = $%d OR organization_id IS NULL)
		RETURNING id, organization_id, region, ip_address, hostname, capacity, status, 
		          last_heartbeat, created_at, metadata
	`, strings.Join(setParts, ", "), argIndex-1, argIndex)

	var updatedEdge models.Edge
	var metadataBytes []byte

	err = s.db.QueryRow(query, args...).Scan(
		&updatedEdge.ID, &updatedEdge.OrganizationID, &updatedEdge.Region, 
		&updatedEdge.IPAddress, &updatedEdge.Hostname, &updatedEdge.Capacity, 
		&updatedEdge.Status, &updatedEdge.LastHeartbeat, &updatedEdge.CreatedAt, 
		&metadataBytes,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update edge: %w", err)
	}

	// Parse metadata JSON
	if len(metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &metadata); err == nil {
			updatedEdge.Metadata = metadata
		}
	}

	// Remove from cache
	s.redis.Del(context.Background(), fmt.Sprintf("edge:%s", edgeID))

	return &updatedEdge, nil
}

// DeleteEdgeForOrganization removes an edge node for a specific organization
func (s *EdgeService) DeleteEdgeForOrganization(orgID, edgeID uuid.UUID) error {
	result, err := s.db.Exec(`
		DELETE FROM edges 
		WHERE id = $1 AND (organization_id = $2 OR organization_id IS NULL)
	`, edgeID, orgID)
	
	if err != nil {
		return fmt.Errorf("failed to delete edge: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("edge not found or access denied")
	}

	// Remove from cache
	s.redis.Del(context.Background(), fmt.Sprintf("edge:%s", edgeID))

	logrus.WithFields(logrus.Fields{
		"edge_id":         edgeID,
		"organization_id": orgID,
	}).Info("Edge node deleted for organization")

	return nil
}

// Helper methods for caching
func (s *EdgeService) cacheEdge(edge *models.Edge) {
	data, err := json.Marshal(edge)
	if err != nil {
		logrus.WithError(err).Warn("Failed to marshal edge for cache")
		return
	}

	key := fmt.Sprintf("edge:%s", edge.ID)
	if err := s.redis.Set(context.Background(), key, data, 2*time.Minute).Err(); err != nil {
		logrus.WithError(err).Warn("Failed to cache edge")
	}
}

func (s *EdgeService) getCachedEdge(edgeID uuid.UUID) (*models.Edge, error) {
	key := fmt.Sprintf("edge:%s", edgeID)
	data, err := s.redis.Get(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var edge models.Edge
	if err := json.Unmarshal([]byte(data), &edge); err != nil {
		return nil, err
	}

	return &edge, nil
}
