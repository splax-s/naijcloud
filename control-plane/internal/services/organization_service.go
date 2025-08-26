package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/naijcloud/control-plane/internal/models"

	"github.com/google/uuid"
)

type OrganizationService struct {
	db *sql.DB
}

func NewOrganizationService(db *sql.DB) *OrganizationService {
	return &OrganizationService{db: db}
}

// GetOrganization retrieves an organization by ID
func (s *OrganizationService) GetOrganization(ctx context.Context, orgID uuid.UUID) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, description, plan, settings, created_at, updated_at
		FROM organizations 
		WHERE id = $1 AND deleted_at IS NULL
	`

	org := &models.Organization{}
	err := s.db.QueryRowContext(ctx, query, orgID).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description,
		&org.Plan, &org.Settings, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, fmt.Errorf("error retrieving organization: %w", err)
	}

	return org, nil
}

// GetOrganizationBySlug retrieves an organization by slug
func (s *OrganizationService) GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	query := `
		SELECT id, name, slug, description, plan, settings, created_at, updated_at
		FROM organizations 
		WHERE slug = $1 AND deleted_at IS NULL
	`

	org := &models.Organization{}
	err := s.db.QueryRowContext(ctx, query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.Description,
		&org.Plan, &org.Settings, &org.CreatedAt, &org.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("organization not found")
		}
		return nil, fmt.Errorf("error retrieving organization: %w", err)
	}

	return org, nil
}

// GetUserOrganizations retrieves all organizations for a user
func (s *OrganizationService) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]models.Organization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.description, o.plan, o.settings, o.created_at, o.updated_at
		FROM organizations o
		JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1 AND o.deleted_at IS NULL
		ORDER BY o.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving user organizations: %w", err)
	}
	defer rows.Close()

	var organizations []models.Organization
	for rows.Next() {
		var org models.Organization
		err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.Description,
			&org.Plan, &org.Settings, &org.CreatedAt, &org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning organization: %w", err)
		}
		organizations = append(organizations, org)
	}

	return organizations, nil
}

// CheckUserAccess verifies if a user has access to an organization
func (s *OrganizationService) CheckUserAccess(ctx context.Context, userID, orgID uuid.UUID) (*models.OrganizationMember, error) {
	query := `
		SELECT id, organization_id, user_id, role, permissions, invited_by, invited_at, joined_at, created_at, updated_at
		FROM organization_members
		WHERE user_id = $1 AND organization_id = $2
	`

	member := &models.OrganizationMember{}
	err := s.db.QueryRowContext(ctx, query, userID, orgID).Scan(
		&member.ID, &member.OrganizationID, &member.UserID, &member.Role,
		&member.Permissions, &member.InvitedBy, &member.InvitedAt,
		&member.JoinedAt, &member.CreatedAt, &member.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user does not have access to organization")
		}
		return nil, fmt.Errorf("error checking user access: %w", err)
	}

	return member, nil
}

// CreateOrganization creates a new organization
func (s *OrganizationService) CreateOrganization(ctx context.Context, name, slug, description, plan string, ownerID uuid.UUID) (*models.Organization, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	// Create organization
	orgID := uuid.New()
	query := `
		INSERT INTO organizations (id, name, slug, description, plan)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	org := &models.Organization{
		ID:          orgID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Plan:        plan,
	}

	err = tx.QueryRowContext(ctx, query, orgID, name, slug, description, plan).Scan(
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating organization: %w", err)
	}

	// Add owner membership
	memberQuery := `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`
	_, err = tx.ExecContext(ctx, memberQuery, orgID, ownerID)
	if err != nil {
		return nil, fmt.Errorf("error creating organization membership: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return org, nil
}

// InviteUser invites a user to an organization
func (s *OrganizationService) InviteUser(ctx context.Context, orgID, inviterID uuid.UUID, email, role string) error {
	// Check if inviter has permission
	inviter, err := s.CheckUserAccess(ctx, inviterID, orgID)
	if err != nil {
		return fmt.Errorf("inviter does not have access: %w", err)
	}

	if inviter.Role != "owner" && inviter.Role != "admin" {
		return fmt.Errorf("insufficient permissions to invite users")
	}

	// Get user by email
	userQuery := `SELECT id FROM users WHERE email = $1`
	var userID uuid.UUID
	err = s.db.QueryRowContext(ctx, userQuery, email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user with email %s not found", email)
		}
		return fmt.Errorf("error finding user: %w", err)
	}

	// Check if already a member
	existingQuery := `SELECT id FROM organization_members WHERE organization_id = $1 AND user_id = $2`
	var existingID uuid.UUID
	err = s.db.QueryRowContext(ctx, existingQuery, orgID, userID).Scan(&existingID)
	if err == nil {
		return fmt.Errorf("user is already a member of this organization")
	}

	// Create membership
	insertQuery := `
		INSERT INTO organization_members (organization_id, user_id, role, invited_by, invited_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err = s.db.ExecContext(ctx, insertQuery, orgID, userID, role, inviterID)
	if err != nil {
		return fmt.Errorf("error creating organization membership: %w", err)
	}

	log.Printf("User %s invited to organization %s with role %s", email, orgID, role)
	return nil
}

// GetOrganizationMembers retrieves all members of an organization
func (s *OrganizationService) GetOrganizationMembers(ctx context.Context, orgID uuid.UUID) ([]models.OrganizationMember, error) {
	query := `
		SELECT om.id, om.organization_id, om.user_id, om.role, om.permissions, 
		       om.invited_by, om.invited_at, om.joined_at, om.created_at, om.updated_at
		FROM organization_members om
		WHERE om.organization_id = $1
		ORDER BY om.created_at
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving organization members: %w", err)
	}
	defer rows.Close()

	var members []models.OrganizationMember
	for rows.Next() {
		var member models.OrganizationMember
		err := rows.Scan(
			&member.ID, &member.OrganizationID, &member.UserID, &member.Role,
			&member.Permissions, &member.InvitedBy, &member.InvitedAt,
			&member.JoinedAt, &member.CreatedAt, &member.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning organization member: %w", err)
		}
		members = append(members, member)
	}

	return members, nil
}

type UserService struct {
	db *sql.DB
}

func NewUserService(db *sql.DB) *UserService {
	return &UserService{db: db}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, name, email_verified, COALESCE(avatar_url, ''), 
		       COALESCE(settings::text, '{}'), created_at, updated_at
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	var settingsStr string
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.EmailVerified,
		&user.AvatarURL, &settingsStr, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	// Convert settings string to []byte
	user.Settings = []byte(settingsStr)

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, name, email_verified, COALESCE(avatar_url, ''), 
		       COALESCE(settings::text, '{}'), created_at, updated_at
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &models.User{}
	var settingsStr string
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.Name, &user.EmailVerified,
		&user.AvatarURL, &settingsStr, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("error retrieving user: %w", err)
	}

	// Convert settings string to []byte
	user.Settings = []byte(settingsStr)

	return user, nil
} // CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	userID := uuid.New()
	query := `
		INSERT INTO users (id, email, name, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at, updated_at
	`

	user := &models.User{
		ID:           userID,
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
	}

	err := s.db.QueryRowContext(ctx, query, userID, email, name, passwordHash).Scan(
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return user, nil
}
