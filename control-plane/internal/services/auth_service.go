package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userService     *UserService
	orgService      *OrganizationService
	emailService    *EmailService
	activityService *ActivityService
	db              *sql.DB
	jwtSecret       []byte
	tokenExpiry     time.Duration
	refreshExpiry   time.Duration
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

type Claims struct {
	UserID    uuid.UUID  `json:"user_id"`
	Email     string     `json:"email"`
	Role      string     `json:"role,omitempty"`
	OrgID     *uuid.UUID `json:"org_id,omitempty"`
	TokenType string     `json:"token_type"`
	jwt.RegisteredClaims
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{
		userService:     NewUserService(db),
		orgService:      NewOrganizationService(db),
		emailService:    NewEmailService(db),
		activityService: NewActivityService(db),
		db:              db,
		jwtSecret:       []byte("your-secret-key"), // TODO: Get from environment
		tokenExpiry:     15 * time.Minute,         // Short-lived access tokens
		refreshExpiry:   7 * 24 * time.Hour,       // 7 days refresh tokens
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

// LoginUser authenticates a user and returns JWT tokens
func (s *AuthService) LoginUser(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Get user with password
	user, err := s.userService.GetUserByEmailWithPassword(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if email is verified
	if !user.EmailVerified {
		return nil, fmt.Errorf("email not verified")
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

	// Generate token pair
	tokenPair, err := s.GenerateTokenPair(ctx, user.ID, user.Email, "")
	if err != nil {
		return nil, fmt.Errorf("error generating tokens: %w", err)
	}

	// Convert to models.TokenPair
	modelTokenPair := &models.TokenPair{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    tokenPair.TokenType,
	}

	// Remove password hash from response
	user.PasswordHash = ""

	response := &models.AuthResponse{
		User:    user,
		Tokens:  modelTokenPair,
		Message: "Login successful",
	}

	// Include first organization if user has any
	if len(organizations) > 0 {
		response.Organization = &organizations[0]
	}

	// Log login activity
	go s.logActivity(ctx, user.ID, "user_login", map[string]interface{}{
		"login_time": time.Now(),
		"ip_address": "unknown", // TODO: Extract from request context
	})

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

// GenerateTokenPair creates access and refresh tokens
func (s *AuthService) GenerateTokenPair(ctx context.Context, userID uuid.UUID, email, role string) (*TokenPair, error) {
	now := time.Now()
	
	// Generate access token
	accessClaims := &Claims{
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "naijcloud",
			Subject:   userID.String(),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("error signing access token: %w", err)
	}

	// Generate refresh token
	refreshTokenBytes := make([]byte, 32)
	_, err = rand.Read(refreshTokenBytes)
	if err != nil {
		return nil, fmt.Errorf("error generating refresh token: %w", err)
	}
	refreshTokenString := base64.URLEncoding.EncodeToString(refreshTokenBytes)

	// Store refresh token in database
	err = s.storeRefreshToken(ctx, userID, refreshTokenString, now.Add(s.refreshExpiry))
	if err != nil {
		return nil, fmt.Errorf("error storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(s.tokenExpiry),
		TokenType:    "Bearer",
	}, nil
}

// RefreshToken generates new token pair using refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	userID, err := s.validateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user details
	user, err := s.userService.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Revoke old refresh token
	err = s.revokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("error revoking refresh token: %w", err)
	}

	// Generate new token pair
	return s.GenerateTokenPair(ctx, user.ID, user.Email, "")
}

// ValidateToken validates and parses JWT token
func (s *AuthService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("error parsing token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		if claims.TokenType != "access" {
			return nil, fmt.Errorf("invalid token type")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Logout revokes all user tokens
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("error revoking tokens: %w", err)
	}

	// Log logout activity
	go s.logActivity(ctx, userID, "user_logout", map[string]interface{}{
		"logout_time": time.Now(),
	})

	return nil
}

// ChangePassword changes user password with proper verification
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// Get user with password
	userQuery := `
		SELECT id, email, password_hash 
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`
	var user models.User
	err := s.db.QueryRowContext(ctx, userQuery, userID).Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword))
	if err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	// Update password
	updateQuery := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err = s.db.ExecContext(ctx, updateQuery, string(hashedPassword), userID)
	if err != nil {
		return fmt.Errorf("error updating password: %w", err)
	}

	// Revoke all existing tokens
	err = s.Logout(ctx, userID)
	if err != nil {
		return fmt.Errorf("error revoking existing tokens: %w", err)
	}

	// Log password change
	go s.logActivity(ctx, userID, "password_changed", map[string]interface{}{
		"changed_at": time.Now(),
	})

	return nil
}

// Helper functions

func (s *AuthService) storeRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := s.db.ExecContext(ctx, query, userID, token, expiresAt)
	return err
}

func (s *AuthService) validateRefreshToken(ctx context.Context, token string) (uuid.UUID, error) {
	query := `
		SELECT user_id 
		FROM refresh_tokens 
		WHERE token = $1 AND expires_at > NOW() AND revoked_at IS NULL
	`
	var userID uuid.UUID
	err := s.db.QueryRowContext(ctx, query, token).Scan(&userID)
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (s *AuthService) revokeRefreshToken(ctx context.Context, token string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE token = $1`
	_, err := s.db.ExecContext(ctx, query, token)
	return err
}

func (s *AuthService) logActivity(ctx context.Context, userID uuid.UUID, action string, metadata map[string]interface{}) {
	// Use the activity service to log the activity
	err := s.activityService.LogActivity(
		ctx,
		nil,     // organization ID - can be determined from user context
		&userID, // user ID
		action,
		"user", // resource type
		&userID, // resource ID (the user themselves)
		metadata,
		nil, // IP address - TODO: extract from context
		nil, // user agent - TODO: extract from context
	)
	if err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Error logging activity: %v\n", err)
	}
}
