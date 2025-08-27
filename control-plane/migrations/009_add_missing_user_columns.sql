-- Migration 009: Add missing user columns for compatibility
-- Adds missing columns to users table to match service expectations

-- Add name column (for compatibility, we'll keep full_name as the main field)
ALTER TABLE users ADD COLUMN IF NOT EXISTS name VARCHAR(255);

-- Add email_verified column (for compatibility, we'll keep is_verified as the main field)  
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN DEFAULT FALSE;

-- Add email verification token fields for proper email verification flow
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_expiry TIMESTAMP WITH TIME ZONE;

-- Add password reset token fields for password reset flow
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_expiry TIMESTAMP WITH TIME ZONE;

-- Add avatar_url column for user profile pictures
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;

-- Add settings column for user preferences
ALTER TABLE users ADD COLUMN IF NOT EXISTS settings JSONB DEFAULT '{}';

-- Create indexes for the new columns
CREATE INDEX IF NOT EXISTS idx_users_email_verification_token ON users(email_verification_token);
CREATE INDEX IF NOT EXISTS idx_users_password_reset_token ON users(password_reset_token);

-- Update name column to match full_name where name is null
UPDATE users SET name = full_name WHERE name IS NULL;
