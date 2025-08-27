package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/middleware"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
)

type AuthAPIHandlers struct {
	authService         *services.AuthService
	notificationService *services.NotificationService
}

func NewAuthAPIHandlers(authService *services.AuthService, notificationService *services.NotificationService) *AuthAPIHandlers {
	return &AuthAPIHandlers{
		authService:         authService,
		notificationService: notificationService,
	}
}

// RefreshToken handles token refresh requests
func (h *AuthAPIHandlers) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid_token",
			"message": "Invalid or expired refresh token",
		})
		return
	}

	// Convert to models.TokenPair
	modelTokenPair := &models.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": modelTokenPair,
		"message": "Tokens refreshed successfully",
	})
}

// Logout handles logout requests
func (h *AuthAPIHandlers) Logout(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	err := h.authService.Logout(c.Request.Context(), authCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "logout_failed",
			"message": "Failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

// ChangePassword handles password change requests
func (h *AuthAPIHandlers) ChangePassword(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate password confirmation
	if req.NewPassword != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation_failed",
			"message": "New password and confirmation do not match",
		})
		return
	}

	err := h.authService.ChangePassword(c.Request.Context(), authCtx.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "password_change_failed",
			"message": err.Error(),
		})
		return
	}

	// Send notification about password change
	go h.notificationService.SendInAppNotification(
		c.Request.Context(),
		authCtx.UserID,
		authCtx.OrgID,
		"security_alerts",
		"Password Changed",
		"Your password has been changed successfully.",
		map[string]interface{}{
			"timestamp": time.Now(),
			"action":    "password_change",
		},
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// GetProfile returns the current user's profile
func (h *AuthAPIHandlers) GetProfile(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    authCtx.UserID,
			"email": authCtx.Email,
			"role":  authCtx.Role,
		},
		"organization_id": authCtx.OrgID,
	})
}

type ActivityAPIHandlers struct {
	activityService *services.ActivityService
}

func NewActivityAPIHandlers(activityService *services.ActivityService) *ActivityAPIHandlers {
	return &ActivityAPIHandlers{
		activityService: activityService,
	}
}

// GetActivities returns activity logs with filtering
func (h *ActivityAPIHandlers) GetActivities(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Parse query parameters
	filter := &services.ActivityFilter{
		OrganizationID: authCtx.OrgID,
		Limit:          50, // Default limit
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if actions := c.QueryArray("action"); len(actions) > 0 {
		filter.Actions = actions
	}

	if resourceTypes := c.QueryArray("resource_type"); len(resourceTypes) > 0 {
		filter.ResourceTypes = resourceTypes
	}

	if severities := c.QueryArray("severity"); len(severities) > 0 {
		filter.Severity = severities
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			filter.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			filter.EndDate = &endDate
		}
	}

	activities, err := h.activityService.GetActivities(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "query_failed",
			"message": "Failed to retrieve activities",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities": activities,
		"count":      len(activities),
		"filter":     filter,
	})
}

// GetActivitySummary returns activity statistics
func (h *ActivityAPIHandlers) GetActivitySummary(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	// Default to last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			startDate = parsed
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			endDate = parsed
		}
	}

	summary, err := h.activityService.GetActivitySummary(c.Request.Context(), authCtx.OrgID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "query_failed",
			"message": "Failed to retrieve activity summary",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary":    summary,
		"start_date": startDate,
		"end_date":   endDate,
	})
}

type NotificationAPIHandlers struct {
	notificationService *services.NotificationService
}

func NewNotificationAPIHandlers(notificationService *services.NotificationService) *NotificationAPIHandlers {
	return &NotificationAPIHandlers{
		notificationService: notificationService,
	}
}

// GetNotifications returns user notifications
func (h *NotificationAPIHandlers) GetNotifications(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	filter := &services.NotificationFilter{
		UserID: &authCtx.UserID,
		Limit:  50, // Default limit
	}

	if types := c.QueryArray("type"); len(types) > 0 {
		filter.Types = types
	}

	if channels := c.QueryArray("channel"); len(channels) > 0 {
		filter.Channels = channels
	}

	if c.Query("unread") == "true" {
		filter.Unread = true
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	notifications, err := h.notificationService.GetNotifications(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "query_failed",
			"message": "Failed to retrieve notifications",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

// MarkNotificationAsRead marks a notification as read
func (h *NotificationAPIHandlers) MarkNotificationAsRead(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	notificationIDStr := c.Param("id")
	notificationID, err := uuid.Parse(notificationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_id",
			"message": "Invalid notification ID",
		})
		return
	}

	err = h.notificationService.MarkAsRead(c.Request.Context(), notificationID, authCtx.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "mark_failed",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Notification marked as read",
	})
}

// MarkAllNotificationsAsRead marks all notifications as read
func (h *NotificationAPIHandlers) MarkAllNotificationsAsRead(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	err := h.notificationService.MarkAllAsRead(c.Request.Context(), authCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "mark_failed",
			"message": "Failed to mark notifications as read",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All notifications marked as read",
	})
}

// GetUnreadCount returns the count of unread notifications
func (h *NotificationAPIHandlers) GetUnreadCount(c *gin.Context) {
	authCtx := middleware.GetAuthContext(c)
	if authCtx == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Authentication required",
		})
		return
	}

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), authCtx.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "query_failed",
			"message": "Failed to get unread count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"unread_count": count,
	})
}
