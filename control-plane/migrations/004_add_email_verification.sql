-- Migration: Add email verification and password reset fields to users table
-- Version: 004
-- Description: Add email verification token, expiry, password reset token, and expiry fields

-- Add email verification fields
ALTER TABLE users 
ADD COLUMN email_verification_token VARCHAR(255),
ADD COLUMN email_verification_expiry TIMESTAMP WITH TIME ZONE,
ADD COLUMN password_reset_token VARCHAR(255),
ADD COLUMN password_reset_expiry TIMESTAMP WITH TIME ZONE;

-- Add indexes for token lookups (important for performance)
CREATE INDEX idx_users_email_verification_token ON users(email_verification_token) WHERE email_verification_token IS NOT NULL;
CREATE INDEX idx_users_password_reset_token ON users(password_reset_token) WHERE password_reset_token IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN users.email_verification_token IS 'Token for email verification, expires after 24 hours';
COMMENT ON COLUMN users.email_verification_expiry IS 'Expiry time for email verification token';
COMMENT ON COLUMN users.password_reset_token IS 'Token for password reset, expires after 1 hour';
COMMENT ON COLUMN users.password_reset_expiry IS 'Expiry time for password reset token';
