package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
)

// MultiTenancyMiddleware handles organization-scoped requests
func MultiTenancyMiddleware(orgService *services.OrganizationService, userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract organization from header or path
		orgID := c.GetHeader("X-Organization-ID")
		orgSlug := c.GetHeader("X-Organization-Slug")

		// If no org in headers, try to extract from path
		if orgID == "" && orgSlug == "" {
			// Check if path has /orgs/{slug} pattern
			path := c.Request.URL.Path
			parts := strings.Split(path, "/")
			for i, part := range parts {
				if part == "orgs" && i+1 < len(parts) {
					orgSlug = parts[i+1]
					break
				}
			}
		}

		var org *models.Organization
		var err error

		// Get organization by ID or slug
		if orgID != "" {
			orgUUID, parseErr := uuid.Parse(orgID)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid organization ID format"})
				c.Abort()
				return
			}
			org, err = orgService.GetOrganization(context.Background(), orgUUID)
		} else if orgSlug != "" {
			org, err = orgService.GetOrganizationBySlug(context.Background(), orgSlug)
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
			c.Abort()
			return
		}

		if org != nil {
			// Store organization in context
			c.Set("organization", org)
			c.Set("organization_id", org.ID)
		}

		c.Next()
	}
}

// AuthMiddleware handles user authentication
func AuthMiddleware(userService *services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user from header (in production, this would be JWT or session)
		userIDHeader := c.GetHeader("X-User-ID")
		userEmail := c.GetHeader("X-User-Email")

		var user *models.User
		var err error

		if userIDHeader != "" {
			userID, parseErr := uuid.Parse(userIDHeader)
			if parseErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
				c.Abort()
				return
			}
			user, err = userService.GetUser(context.Background(), userID)
		} else if userEmail != "" {
			user, err = userService.GetUserByEmail(context.Background(), userEmail)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No user authentication provided"})
			c.Abort()
			return
		}

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Store user in context
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// RequireOrganizationAccess ensures user has access to the organization
func RequireOrganizationAccess(orgService *services.OrganizationService, requiredRole ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			c.Abort()
			return
		}

		orgID, exists := c.Get("organization_id")
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
			c.Abort()
			return
		}

		// Check user access to organization
		member, err := orgService.CheckUserAccess(context.Background(), userID.(uuid.UUID), orgID.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to organization"})
			c.Abort()
			return
		}

		// Check role requirements if specified
		if len(requiredRole) > 0 {
			hasRequiredRole := false
			for _, role := range requiredRole {
				if member.Role == role {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				c.Abort()
				return
			}
		}

		// Store member info in context
		c.Set("organization_member", member)
		c.Set("user_role", member.Role)

		c.Next()
	}
}

// ExtractOrganizationID helper function to get organization ID from context
func ExtractOrganizationID(c *gin.Context) (uuid.UUID, error) {
	orgID, exists := c.Get("organization_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("organization ID not found in context")
	}
	return orgID.(uuid.UUID), nil
}

// ExtractUserID helper function to get user ID from context
func ExtractUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, fmt.Errorf("user ID not found in context")
	}
	return userID.(uuid.UUID), nil
}
