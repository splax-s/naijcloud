package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/naijcloud/control-plane/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// EmailService handles email operations
type EmailService struct {
	db       *sql.DB
	smtpHost string
	smtpPort string
	smtpUser string
	smtpPass string
	fromAddr string
	baseURL  string
}

// NewEmailService creates a new email service
func NewEmailService(db *sql.DB) *EmailService {
	return &EmailService{
		db:       db,
		smtpHost: getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		smtpPort: getEnvOrDefault("SMTP_PORT", "587"),
		smtpUser: getEnvOrDefault("SMTP_USER", ""),
		smtpPass: getEnvOrDefault("SMTP_PASS", ""),
		fromAddr: getEnvOrDefault("FROM_EMAIL", "noreply@naijcloud.com"),
		baseURL:  getEnvOrDefault("BASE_URL", "http://localhost:3000"),
	}
}

// generateSecureToken generates a cryptographically secure random token
func (s *EmailService) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// SendEmailVerification sends an email verification token to the user
func (s *EmailService) SendEmailVerification(userID uuid.UUID) error {
	// Find the user and check if email is already verified
	var user models.User
	query := `
		SELECT id, email, name, email_verified
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL
	`
	
	err := s.db.QueryRow(query, userID).Scan(
		&user.ID, &user.Email, &user.Name, &user.EmailVerified,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user not found")
		}
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	// Check if email is already verified
	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Generate verification token
	token, err := s.generateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Set token expiry (24 hours from now)
	expiry := time.Now().Add(24 * time.Hour)

	// Update user with verification token
	updateQuery := `
		UPDATE users 
		SET email_verification_token = $1, email_verification_expiry = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err = s.db.Exec(updateQuery, token, expiry, userID)
	if err != nil {
		return fmt.Errorf("failed to save verification token: %w", err)
	}

	// Send verification email
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)
	subject := "Verify Your Email - NaijCloud"
	body := s.buildEmailVerificationTemplate(user.Name, verificationURL)

	if err := s.sendEmail(user.Email, subject, body); err != nil {
		log.Printf("Failed to send verification email to %s: %v", user.Email, err)
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	log.Printf("Verification email sent to %s", user.Email)
	return nil
}

// VerifyEmail verifies an email using the provided token
func (s *EmailService) VerifyEmail(token string) error {
	// Find user by verification token
	var user models.User
	var expiry *time.Time
	query := `
		SELECT id, email, email_verification_expiry
		FROM users 
		WHERE email_verification_token = $1 AND deleted_at IS NULL
	`
	
	err := s.db.QueryRow(query, token).Scan(&user.ID, &user.Email, &expiry)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid verification token")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Check if token has expired
	if expiry == nil || time.Now().After(*expiry) {
		return fmt.Errorf("verification token has expired")
	}

	// Mark email as verified and clear token
	updateQuery := `
		UPDATE users 
		SET email_verified = true, 
		    email_verification_token = NULL, 
		    email_verification_expiry = NULL,
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err = s.db.Exec(updateQuery, user.ID)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	log.Printf("Email verified for user %s", user.Email)
	return nil
}

// SendPasswordReset sends a password reset token to the user
func (s *EmailService) SendPasswordReset(email string) error {
	// Find the user by email
	var user models.User
	query := `
		SELECT id, email, name
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL
	`
	
	err := s.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			// Don't reveal if email exists or not for security
			log.Printf("Password reset requested for non-existent email: %s", email)
			return nil
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Generate reset token
	token, err := s.generateSecureToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Set token expiry (1 hour from now)
	expiry := time.Now().Add(1 * time.Hour)

	// Update user with reset token
	updateQuery := `
		UPDATE users 
		SET password_reset_token = $1, password_reset_expiry = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err = s.db.Exec(updateQuery, token, expiry, user.ID)
	if err != nil {
		return fmt.Errorf("failed to save reset token: %w", err)
	}

	// Send reset email
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	subject := "Reset Your Password - NaijCloud"
	body := s.buildPasswordResetTemplate(user.Name, resetURL)

	if err := s.sendEmail(user.Email, subject, body); err != nil {
		log.Printf("Failed to send password reset email to %s: %v", user.Email, err)
		return fmt.Errorf("failed to send password reset email: %w", err)
	}

	log.Printf("Password reset email sent to %s", user.Email)
	return nil
}

// ResetPassword resets a user's password using the provided token
func (s *EmailService) ResetPassword(token, newPassword string) error {
	// Find user by reset token
	var user models.User
	var expiry *time.Time
	query := `
		SELECT id, email, password_reset_expiry
		FROM users 
		WHERE password_reset_token = $1 AND deleted_at IS NULL
	`
	
	err := s.db.QueryRow(query, token).Scan(&user.ID, &user.Email, &expiry)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("invalid reset token")
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Check if token has expired
	if expiry == nil || time.Now().After(*expiry) {
		return fmt.Errorf("reset token has expired")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token
	updateQuery := `
		UPDATE users 
		SET password_hash = $1, 
		    password_reset_token = NULL, 
		    password_reset_expiry = NULL,
		    updated_at = NOW()
		WHERE id = $2
	`
	_, err = s.db.Exec(updateQuery, string(hashedPassword), user.ID)
	if err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}

	log.Printf("Password reset successfully for user %s", user.Email)
	return nil
}

// sendEmail sends an email using SMTP
func (s *EmailService) sendEmail(to, subject, body string) error {
	// Skip sending if SMTP not configured
	if s.smtpUser == "" || s.smtpPass == "" {
		log.Printf("SMTP not configured, email would be sent to %s with subject: %s", to, subject)
		return nil
	}

	// Set up authentication information
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPass, s.smtpHost)

	// Compose message
	msg := []string{
		fmt.Sprintf("From: %s", s.fromAddr),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"",
		body,
	}

	// Send email
	err := smtp.SendMail(
		s.smtpHost+":"+s.smtpPort,
		auth,
		s.fromAddr,
		[]string{to},
		[]byte(strings.Join(msg, "\r\n")),
	)

	return err
}

// buildEmailVerificationTemplate creates the email verification email body
func (s *EmailService) buildEmailVerificationTemplate(name, verificationURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #007bff; color: white; padding: 20px; text-align: center; }
        .content { padding: 30px 20px; }
        .button { display: inline-block; padding: 12px 30px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>NaijCloud</h1>
        </div>
        <div class="content">
            <h2>Hi %s,</h2>
            <p>Thank you for signing up for NaijCloud! To complete your registration and start using our CDN platform, please verify your email address by clicking the button below:</p>
            
            <a href="%s" class="button">Verify Email Address</a>
            
            <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
            <p><a href="%s">%s</a></p>
            
            <p>This verification link will expire in 24 hours for security reasons.</p>
            
            <p>If you didn't create an account with NaijCloud, you can safely ignore this email.</p>
            
            <p>Best regards,<br>The NaijCloud Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 NaijCloud. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, name, verificationURL, verificationURL, verificationURL)
}

// buildPasswordResetTemplate creates the password reset email body
func (s *EmailService) buildPasswordResetTemplate(name, resetURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Reset Your Password</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #dc3545; color: white; padding: 20px; text-align: center; }
        .content { padding: 30px 20px; }
        .button { display: inline-block; padding: 12px 30px; background: #dc3545; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
        .footer { background: #f8f9fa; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>NaijCloud</h1>
        </div>
        <div class="content">
            <h2>Hi %s,</h2>
            <p>You requested to reset your password for your NaijCloud account. Click the button below to create a new password:</p>
            
            <a href="%s" class="button">Reset Password</a>
            
            <p>If the button doesn't work, you can copy and paste this link into your browser:</p>
            <p><a href="%s">%s</a></p>
            
            <p>This reset link will expire in 1 hour for security reasons.</p>
            
            <p>If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.</p>
            
            <p>Best regards,<br>The NaijCloud Team</p>
        </div>
        <div class="footer">
            <p>&copy; 2024 NaijCloud. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`, name, resetURL, resetURL, resetURL)
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
