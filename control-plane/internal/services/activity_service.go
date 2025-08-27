package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
)

type ActivityService struct {
	db *sql.DB
}

type ActivityLog struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID *uuid.UUID             `json:"organization_id" db:"organization_id"`
	UserID         *uuid.UUID             `json:"user_id" db:"user_id"`
	Action         string                 `json:"action" db:"action"`
	ResourceType   string                 `json:"resource_type" db:"resource_type"`
	ResourceID     *uuid.UUID             `json:"resource_id" db:"resource_id"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	IPAddress      *string                `json:"ip_address" db:"ip_address"`
	UserAgent      *string                `json:"user_agent" db:"user_agent"`
	Severity       string                 `json:"severity" db:"severity"` // info, warning, error, critical
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
}

type ActivityFilter struct {
	OrganizationID *uuid.UUID
	UserID         *uuid.UUID
	Actions        []string
	ResourceTypes  []string
	Severity       []string
	StartDate      *time.Time
	EndDate        *time.Time
	Limit          int
	Offset         int
}

func NewActivityService(db *sql.DB) *ActivityService {
	return &ActivityService{db: db}
}

// LogActivity logs a user action
func (s *ActivityService) LogActivity(ctx context.Context, orgID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]interface{}, ipAddress, userAgent *string) error {
	return s.LogActivityWithSeverity(ctx, orgID, userID, action, resourceType, resourceID, metadata, ipAddress, userAgent, "info")
}

// LogActivityWithSeverity logs a user action with specific severity
func (s *ActivityService) LogActivityWithSeverity(ctx context.Context, orgID, userID *uuid.UUID, action, resourceType string, resourceID *uuid.UUID, metadata map[string]interface{}, ipAddress, userAgent *string, severity string) error {
	id := uuid.New()
	
	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	query := `
		INSERT INTO activity_logs (
			id, organization_id, user_id, action, resource_type, resource_id,
			metadata, ip_address, user_agent, severity
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = s.db.ExecContext(ctx, query,
		id, orgID, userID, action, resourceType, resourceID,
		metadataJSON, ipAddress, userAgent, severity,
	)

	if err != nil {
		return fmt.Errorf("error logging activity: %w", err)
	}

	return nil
}

// GetActivities retrieves activity logs with filtering
func (s *ActivityService) GetActivities(ctx context.Context, filter *ActivityFilter) ([]ActivityLog, error) {
	query := `
		SELECT id, organization_id, user_id, action, resource_type, resource_id,
		       metadata, ip_address, user_agent, severity, created_at
		FROM activity_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	// Build dynamic WHERE clause
	if filter.OrganizationID != nil {
		argCount++
		query += fmt.Sprintf(" AND organization_id = $%d", argCount)
		args = append(args, *filter.OrganizationID)
	}

	if filter.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
	}

	if len(filter.Actions) > 0 {
		argCount++
		query += fmt.Sprintf(" AND action = ANY($%d)", argCount)
		args = append(args, filter.Actions)
	}

	if len(filter.ResourceTypes) > 0 {
		argCount++
		query += fmt.Sprintf(" AND resource_type = ANY($%d)", argCount)
		args = append(args, filter.ResourceTypes)
	}

	if len(filter.Severity) > 0 {
		argCount++
		query += fmt.Sprintf(" AND severity = ANY($%d)", argCount)
		args = append(args, filter.Severity)
	}

	if filter.StartDate != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartDate)
	}

	if filter.EndDate != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.EndDate)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying activities: %w", err)
	}
	defer rows.Close()

	var activities []ActivityLog
	for rows.Next() {
		var activity ActivityLog
		var metadataJSON []byte

		err := rows.Scan(
			&activity.ID, &activity.OrganizationID, &activity.UserID,
			&activity.Action, &activity.ResourceType, &activity.ResourceID,
			&metadataJSON, &activity.IPAddress, &activity.UserAgent,
			&activity.Severity, &activity.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning activity: %w", err)
		}

		// Unmarshal metadata
		if metadataJSON != nil {
			err = json.Unmarshal(metadataJSON, &activity.Metadata)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling metadata: %w", err)
			}
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// GetActivitySummary returns activity statistics
func (s *ActivityService) GetActivitySummary(ctx context.Context, orgID *uuid.UUID, startDate, endDate time.Time) (*models.ActivitySummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_activities,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(CASE WHEN severity = 'error' THEN 1 END) as error_count,
			COUNT(CASE WHEN severity = 'warning' THEN 1 END) as warning_count,
			COUNT(CASE WHEN created_at >= $2 THEN 1 END) as recent_activities
		FROM activity_logs 
		WHERE organization_id = $1 AND created_at BETWEEN $2 AND $3
	`

	summary := &models.ActivitySummary{}
	err := s.db.QueryRowContext(ctx, query, orgID, startDate, endDate).Scan(
		&summary.TotalActivities,
		&summary.UniqueUsers,
		&summary.ErrorCount,
		&summary.WarningCount,
		&summary.RecentActivities,
	)

	if err != nil {
		return nil, fmt.Errorf("error getting activity summary: %w", err)
	}

	return summary, nil
}

// GetTopActions returns most frequent actions
func (s *ActivityService) GetTopActions(ctx context.Context, orgID *uuid.UUID, limit int) ([]models.ActionCount, error) {
	query := `
		SELECT action, COUNT(*) as count
		FROM activity_logs 
		WHERE organization_id = $1 
		GROUP BY action 
		ORDER BY count DESC 
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("error querying top actions: %w", err)
	}
	defer rows.Close()

	var actions []models.ActionCount
	for rows.Next() {
		var action models.ActionCount
		err := rows.Scan(&action.Action, &action.Count)
		if err != nil {
			return nil, fmt.Errorf("error scanning action count: %w", err)
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// CleanupOldActivities removes activity logs older than specified duration
func (s *ActivityService) CleanupOldActivities(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoffDate := time.Now().Add(-olderThan)
	
	query := `DELETE FROM activity_logs WHERE created_at < $1`
	result, err := s.db.ExecContext(ctx, query, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("error cleaning up old activities: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error getting rows affected: %w", err)
	}

	return rowsAffected, nil
}

// Export activities to JSON for compliance/backup
func (s *ActivityService) ExportActivities(ctx context.Context, filter *ActivityFilter) ([]byte, error) {
	activities, err := s.GetActivities(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("error getting activities for export: %w", err)
	}

	export := map[string]interface{}{
		"exported_at": time.Now(),
		"filter":      filter,
		"count":       len(activities),
		"activities":  activities,
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling export data: %w", err)
	}

	return data, nil
}
