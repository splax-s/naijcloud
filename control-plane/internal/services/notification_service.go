package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type NotificationService struct {
	db *sql.DB
}

type Notification struct {
	ID             uuid.UUID              `json:"id" db:"id"`
	OrganizationID *uuid.UUID             `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID              `json:"user_id" db:"user_id"`
	Type           string                 `json:"type" db:"type"` // email, in_app, push, sms
	Channel        string                 `json:"channel" db:"channel"` // email_verification, password_reset, organization_invite, etc.
	Title          string                 `json:"title" db:"title"`
	Message        string                 `json:"message" db:"message"`
	Data           map[string]interface{} `json:"data" db:"data"`
	Priority       string                 `json:"priority" db:"priority"` // low, normal, high, urgent
	Status         string                 `json:"status" db:"status"` // pending, sent, delivered, failed, read
	ScheduledFor   *time.Time             `json:"scheduled_for" db:"scheduled_for"`
	SentAt         *time.Time             `json:"sent_at" db:"sent_at"`
	ReadAt         *time.Time             `json:"read_at" db:"read_at"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
}

type NotificationPreferences struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Channel        string    `json:"channel" db:"channel"`
	EmailEnabled   bool      `json:"email_enabled" db:"email_enabled"`
	InAppEnabled   bool      `json:"in_app_enabled" db:"in_app_enabled"`
	PushEnabled    bool      `json:"push_enabled" db:"push_enabled"`
	SMSEnabled     bool      `json:"sms_enabled" db:"sms_enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type NotificationFilter struct {
	UserID         *uuid.UUID
	OrganizationID *uuid.UUID
	Types          []string
	Channels       []string
	Status         []string
	Priority       []string
	Unread         bool
	StartDate      *time.Time
	EndDate        *time.Time
	Limit          int
	Offset         int
}

func NewNotificationService(db *sql.DB) *NotificationService {
	return &NotificationService{db: db}
}

// CreateNotification creates a new notification
func (s *NotificationService) CreateNotification(ctx context.Context, notification *Notification) error {
	notification.ID = uuid.New()
	notification.Status = "pending"
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	// Convert data to JSON
	dataJSON, err := json.Marshal(notification.Data)
	if err != nil {
		return fmt.Errorf("error marshaling notification data: %w", err)
	}

	query := `
		INSERT INTO notifications (
			id, organization_id, user_id, type, channel, title, message,
			data, priority, status, scheduled_for
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = s.db.ExecContext(ctx, query,
		notification.ID, notification.OrganizationID, notification.UserID,
		notification.Type, notification.Channel, notification.Title, notification.Message,
		dataJSON, notification.Priority, notification.Status, notification.ScheduledFor,
	)

	if err != nil {
		return fmt.Errorf("error creating notification: %w", err)
	}

	return nil
}

// GetNotifications retrieves notifications with filtering
func (s *NotificationService) GetNotifications(ctx context.Context, filter *NotificationFilter) ([]Notification, error) {
	query := `
		SELECT id, organization_id, user_id, type, channel, title, message,
		       data, priority, status, scheduled_for, sent_at, read_at,
		       created_at, updated_at
		FROM notifications
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	// Build dynamic WHERE clause
	if filter.UserID != nil {
		argCount++
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
	}

	if filter.OrganizationID != nil {
		argCount++
		query += fmt.Sprintf(" AND organization_id = $%d", argCount)
		args = append(args, *filter.OrganizationID)
	}

	if len(filter.Types) > 0 {
		argCount++
		query += fmt.Sprintf(" AND type = ANY($%d)", argCount)
		args = append(args, filter.Types)
	}

	if len(filter.Channels) > 0 {
		argCount++
		query += fmt.Sprintf(" AND channel = ANY($%d)", argCount)
		args = append(args, filter.Channels)
	}

	if len(filter.Status) > 0 {
		argCount++
		query += fmt.Sprintf(" AND status = ANY($%d)", argCount)
		args = append(args, filter.Status)
	}

	if len(filter.Priority) > 0 {
		argCount++
		query += fmt.Sprintf(" AND priority = ANY($%d)", argCount)
		args = append(args, filter.Priority)
	}

	if filter.Unread {
		query += " AND read_at IS NULL"
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
		return nil, fmt.Errorf("error querying notifications: %w", err)
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		var dataJSON []byte

		err := rows.Scan(
			&notification.ID, &notification.OrganizationID, &notification.UserID,
			&notification.Type, &notification.Channel, &notification.Title, &notification.Message,
			&dataJSON, &notification.Priority, &notification.Status,
			&notification.ScheduledFor, &notification.SentAt, &notification.ReadAt,
			&notification.CreatedAt, &notification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning notification: %w", err)
		}

		// Unmarshal data
		if dataJSON != nil {
			err = json.Unmarshal(dataJSON, &notification.Data)
			if err != nil {
				return nil, fmt.Errorf("error unmarshaling notification data: %w", err)
			}
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID, userID uuid.UUID) error {
	query := `
		UPDATE notifications 
		SET read_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2 AND read_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("error marking notification as read: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found or already read")
	}

	return nil
}

// MarkAllAsRead marks all unread notifications for a user as read
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE notifications 
		SET read_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND read_at IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("error marking all notifications as read: %w", err)
	}

	return nil
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`

	var count int64
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error getting unread count: %w", err)
	}

	return count, nil
}

// DeleteNotification deletes a notification
func (s *NotificationService) DeleteNotification(ctx context.Context, notificationID, userID uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1 AND user_id = $2`

	result, err := s.db.ExecContext(ctx, query, notificationID, userID)
	if err != nil {
		return fmt.Errorf("error deleting notification: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification not found")
	}

	return nil
}

// UpdateNotificationStatus updates the status of a notification
func (s *NotificationService) UpdateNotificationStatus(ctx context.Context, notificationID uuid.UUID, status string) error {
	query := `
		UPDATE notifications 
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`

	if status == "sent" {
		query = `
			UPDATE notifications 
			SET status = $1, sent_at = NOW(), updated_at = NOW()
			WHERE id = $2
		`
	}

	_, err := s.db.ExecContext(ctx, query, status, notificationID)
	if err != nil {
		return fmt.Errorf("error updating notification status: %w", err)
	}

	return nil
}

// GetNotificationPreferences retrieves user notification preferences
func (s *NotificationService) GetNotificationPreferences(ctx context.Context, userID uuid.UUID) ([]NotificationPreferences, error) {
	query := `
		SELECT id, user_id, channel, email_enabled, in_app_enabled, 
		       push_enabled, sms_enabled, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying notification preferences: %w", err)
	}
	defer rows.Close()

	var preferences []NotificationPreferences
	for rows.Next() {
		var pref NotificationPreferences
		err := rows.Scan(
			&pref.ID, &pref.UserID, &pref.Channel,
			&pref.EmailEnabled, &pref.InAppEnabled,
			&pref.PushEnabled, &pref.SMSEnabled,
			&pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning notification preference: %w", err)
		}
		preferences = append(preferences, pref)
	}

	return preferences, nil
}

// UpdateNotificationPreferences updates user notification preferences
func (s *NotificationService) UpdateNotificationPreferences(ctx context.Context, pref *NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			id, user_id, channel, email_enabled, in_app_enabled, 
			push_enabled, sms_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, channel) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			in_app_enabled = EXCLUDED.in_app_enabled,
			push_enabled = EXCLUDED.push_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			updated_at = NOW()
	`

	pref.ID = uuid.New()
	_, err := s.db.ExecContext(ctx, query,
		pref.ID, pref.UserID, pref.Channel,
		pref.EmailEnabled, pref.InAppEnabled,
		pref.PushEnabled, pref.SMSEnabled,
	)

	if err != nil {
		return fmt.Errorf("error updating notification preferences: %w", err)
	}

	return nil
}

// SendInAppNotification creates an in-app notification
func (s *NotificationService) SendInAppNotification(ctx context.Context, userID uuid.UUID, orgID *uuid.UUID, channel, title, message string, data map[string]interface{}) error {
	notification := &Notification{
		OrganizationID: orgID,
		UserID:         userID,
		Type:           "in_app",
		Channel:        channel,
		Title:          title,
		Message:        message,
		Data:           data,
		Priority:       "normal",
	}

	return s.CreateNotification(ctx, notification)
}
