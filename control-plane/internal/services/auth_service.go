package services

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/naijcloud/control-plane/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userService  *UserService
	orgService   *OrganizationService
	emailService *EmailService
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{
		userService:  NewUserService(db),
		orgService:   NewOrganizationService(db),
		emailService: NewEmailService(db),
	}
}

// RegisterUser creates a new user and organization
func (s *AuthService) RegisterUser(ctx context.Context, req *models.RegisterUserRequest) (*models.AuthResponse, error) {
	// Validate input
	if err := s.validateRegistration(ctx, req); err != nil {
		return nil, err
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	// Create user
	user, err := s.userService.CreateUser(ctx, req.Email, req.Name, string(hashedPassword))
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	// Create organization
	organization, err := s.orgService.CreateOrganization(
		ctx,
		req.OrganizationName,
		req.OrganizationSlug,
		"",     // description
		"free", // plan
		user.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating organization: %w", err)
	}

	// Send verification email (don't fail registration if email fails)
	if err := s.emailService.SendEmailVerification(user.ID); err != nil {
		// Log the error but don't fail the registration
		fmt.Printf("Warning: Failed to send verification email to %s: %v\n", user.Email, err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	return &models.AuthResponse{
		User:         user,
		Organization: organization,
		Message:      "User and organization created successfully. Please check your email to verify your account.",
	}, nil
}

// LoginUser authenticates a user
func (s *AuthService) LoginUser(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Get user with password
	user, err := s.userService.GetUserByEmailWithPassword(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Get user's first organization for session
	organizations, err := s.orgService.GetUserOrganizations(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user organizations: %w", err)
	}

	// Remove password hash from response
	user.PasswordHash = ""

	response := &models.AuthResponse{
		User:    user,
		Message: "Login successful",
	}

	// Include first organization if user has any
	if len(organizations) > 0 {
		response.Organization = &organizations[0]
	}

	return response, nil
}

// validateRegistration validates user registration data
func (s *AuthService) validateRegistration(ctx context.Context, req *models.RegisterUserRequest) error {
	// Check if passwords match
	if req.Password != req.ConfirmPassword {
		return fmt.Errorf("passwords do not match")
	}

	// Validate password strength
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Check if email already exists
	emailExists, err := s.userService.CheckEmailExists(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("error checking email: %w", err)
	}
	if emailExists {
		return fmt.Errorf("email already registered")
	}

	// Check if organization slug already exists
	slugExists, err := s.orgService.CheckSlugExists(ctx, req.OrganizationSlug)
	if err != nil {
		return fmt.Errorf("error checking organization slug: %w", err)
	}
	if slugExists {
		return fmt.Errorf("organization slug already taken")
	}

	// Validate organization slug format
	if err := s.validateSlug(req.OrganizationSlug); err != nil {
		return err
	}

	return nil
}

// validateSlug validates organization slug format
func (s *AuthService) validateSlug(slug string) error {
	if len(slug) < 3 {
		return fmt.Errorf("organization slug must be at least 3 characters long")
	}

	if len(slug) > 50 {
		return fmt.Errorf("organization slug must be no more than 50 characters long")
	}

	// Check if slug contains only allowed characters (lowercase letters, numbers, hyphens)
	validSlug := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !validSlug.MatchString(slug) {
		return fmt.Errorf("organization slug can only contain lowercase letters, numbers, and hyphens")
	}

	// Check if slug starts or ends with hyphen
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return fmt.Errorf("organization slug cannot start or end with a hyphen")
	}

	// Check for consecutive hyphens
	if strings.Contains(slug, "--") {
		return fmt.Errorf("organization slug cannot contain consecutive hyphens")
	}

	// Check for reserved slugs
	reservedSlugs := []string{
		"admin", "api", "www", "mail", "ftp", "blog", "help", "support",
		"docs", "status", "app", "dashboard", "control", "panel", "cdn",
		"edge", "proxy", "cache", "assets", "static", "files", "images",
	}

	for _, reserved := range reservedSlugs {
		if slug == reserved {
			return fmt.Errorf("organization slug '%s' is reserved", slug)
		}
	}

	return nil
}

// CheckEmailExists checks if an email is already registered
func (s *AuthService) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	return s.userService.CheckEmailExists(ctx, email)
}

// CheckSlugExists checks if an organization slug is already taken
func (s *AuthService) CheckSlugExists(ctx context.Context, slug string) (bool, error) {
	return s.orgService.CheckSlugExists(ctx, slug)
}
