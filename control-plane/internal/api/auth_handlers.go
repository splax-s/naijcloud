package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naijcloud/control-plane/internal/models"
	"github.com/naijcloud/control-plane/internal/services"
	"github.com/sirupsen/logrus"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration with organization creation
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := h.authService.RegisterUser(c.Request.Context(), &req)
	if err != nil {
		logrus.WithError(err).Error("Failed to register user")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := h.authService.LoginUser(c.Request.Context(), &req)
	if err != nil {
		logrus.WithError(err).Info("Failed login attempt")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CheckEmail validates if an email is available for registration
func (h *AuthHandler) CheckEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email parameter is required",
		})
		return
	}

	exists, err := h.authService.CheckEmailExists(c.Request.Context(), email)
	if err != nil {
		logrus.WithError(err).Error("Failed to check email availability")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check email availability",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available": !exists,
		"message": func() string {
			if exists {
				return "Email is already registered"
			}
			return "Email is available"
		}(),
	})
}

// CheckSlug validates if an organization slug is available
func (h *AuthHandler) CheckSlug(c *gin.Context) {
	slug := c.Query("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Slug parameter is required",
		})
		return
	}

	exists, err := h.authService.CheckSlugExists(c.Request.Context(), slug)
	if err != nil {
		logrus.WithError(err).Error("Failed to check slug availability")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to check slug availability",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available": !exists,
		"message": func() string {
			if exists {
				return "Organization slug is already taken"
			}
			return "Organization slug is available"
		}(),
	})
}
