package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/naijcloud/control-plane/internal/middleware"
	"github.com/naijcloud/control-plane/internal/services"
)

type OrganizationHandler struct {
	orgService  *services.OrganizationService
	userService *services.UserService
}

func NewOrganizationHandler(orgService *services.OrganizationService, userService *services.UserService) *OrganizationHandler {
	return &OrganizationHandler{
		orgService:  orgService,
		userService: userService,
	}
}

// GetUserOrganizations retrieves all organizations for the authenticated user
func (h *OrganizationHandler) GetUserOrganizations(c *gin.Context) {
	userID, err := middleware.ExtractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	organizations, err := h.orgService.GetUserOrganizations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve organizations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organizations": organizations})
}

// GetOrganization retrieves organization details
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	org, exists := c.Get("organization")
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"organization": org})
}

// CreateOrganization creates a new organization
func (h *OrganizationHandler) CreateOrganization(c *gin.Context) {
	userID, err := middleware.ExtractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Slug        string `json:"slug" binding:"required"`
		Description string `json:"description"`
		Plan        string `json:"plan"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to free plan if not specified
	if req.Plan == "" {
		req.Plan = "free"
	}

	org, err := h.orgService.CreateOrganization(c.Request.Context(), req.Name, req.Slug, req.Description, req.Plan, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create organization"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"organization": org})
}

// GetOrganizationMembers retrieves all members of an organization
func (h *OrganizationHandler) GetOrganizationMembers(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	members, err := h.orgService.GetOrganizationMembers(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"members": members})
}

// InviteUser invites a user to the organization
func (h *OrganizationHandler) InviteUser(c *gin.Context) {
	orgID, err := middleware.ExtractOrganizationID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Organization not specified"})
		return
	}

	userID, err := middleware.ExtractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role
	validRoles := []string{"owner", "admin", "member", "viewer"}
	roleValid := false
	for _, validRole := range validRoles {
		if req.Role == validRole {
			roleValid = true
			break
		}
	}
	if !roleValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role specified"})
		return
	}

	err = h.orgService.InviteUser(c.Request.Context(), orgID, userID, req.Email, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User invited successfully"})
}

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// GetCurrentUser retrieves the current authenticated user
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// CreateUser creates a new user (for development purposes)
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Name     string `json:"name" binding:"required"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// In production, you would hash the password properly
	passwordHash := "$2a$10$dummy.hash.for.development" // Placeholder

	user, err := h.userService.CreateUser(c.Request.Context(), req.Email, req.Name, passwordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// SetupOrganizationRoutes sets up the organization-related routes
func SetupOrganizationRoutes(router *gin.Engine, orgService *services.OrganizationService, userService *services.UserService) {
	orgHandler := NewOrganizationHandler(orgService, userService)
	userHandler := NewUserHandler(userService)

	// Public routes
	router.POST("/users", userHandler.CreateUser)

	// Authenticated routes
	auth := router.Group("/")
	auth.Use(middleware.AuthMiddleware(userService))
	{
		// User routes
		auth.GET("/user", userHandler.GetCurrentUser)
		auth.GET("/user/organizations", orgHandler.GetUserOrganizations)

		// Organization management
		auth.POST("/organizations", orgHandler.CreateOrganization)
	}

	// Organization-scoped routes
	orgRoutes := router.Group("/orgs/:slug")
	orgRoutes.Use(middleware.AuthMiddleware(userService))
	orgRoutes.Use(middleware.MultiTenancyMiddleware(orgService, userService))
	orgRoutes.Use(middleware.RequireOrganizationAccess(orgService))
	{
		orgRoutes.GET("", orgHandler.GetOrganization)

		// Member management (admin+ only)
		memberRoutes := orgRoutes.Group("/members")
		memberRoutes.Use(middleware.RequireOrganizationAccess(orgService, "owner", "admin"))
		{
			memberRoutes.GET("", orgHandler.GetOrganizationMembers)
			memberRoutes.POST("/invite", orgHandler.InviteUser)
		}
	}
}
