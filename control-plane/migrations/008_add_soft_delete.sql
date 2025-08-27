-- Migration 008: Add soft delete support
-- Adds deleted_at columns to tables that need soft delete functionality

-- Add deleted_at to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Add deleted_at to organizations table  
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Add deleted_at to api_keys table
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Create indexes on deleted_at columns for performance
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_organizations_deleted_at ON organizations(deleted_at);  
CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON api_keys(deleted_at);

-- Add compound indexes for queries that check both active status and deleted_at
CREATE INDEX IF NOT EXISTS idx_users_email_not_deleted ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organizations_slug_not_deleted ON organizations(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_not_deleted ON api_keys(organization_id) WHERE deleted_at IS NULL;
