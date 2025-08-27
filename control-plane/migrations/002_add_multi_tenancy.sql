-- Multi-tenancy schema migration
-- This adds organizations and users tables with proper relationships

-- Organizations table - top-level tenant entity
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL, -- URL-friendly identifier
    description TEXT,
    plan VARCHAR(50) NOT NULL DEFAULT 'free', -- free, basic, pro, enterprise
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Users table - individual users within organizations
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),  -- Add this for compatibility
    full_name VARCHAR(255),  -- Our original field
    password_hash VARCHAR(255), -- For email/password authentication
    email_verified BOOLEAN DEFAULT FALSE,  -- Add this for compatibility
    is_verified BOOLEAN DEFAULT FALSE,  -- Our original field
    phone VARCHAR(20),  -- Add this field
    is_active BOOLEAN DEFAULT TRUE,  -- Add this field
    avatar_url TEXT,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Organization members - many-to-many relationship between users and organizations
CREATE TABLE IF NOT EXISTS organization_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member', -- owner, admin, member, viewer
    permissions JSONB DEFAULT '{}', -- Additional fine-grained permissions
    invited_by UUID REFERENCES users(id),
    invited_at TIMESTAMP WITH TIME ZONE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, user_id)
);

-- API keys for programmatic access
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE, -- Hashed API key
    key_prefix VARCHAR(20) NOT NULL, -- First few characters for identification
    permissions JSONB DEFAULT '{}', -- Scoped permissions for this key
    scopes TEXT[] DEFAULT '{}', -- Array of scope strings
    rate_limit INTEGER DEFAULT 1000, -- Rate limit per hour
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip INET,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE NULL
);

-- Update existing domains table to include organization_id
ALTER TABLE domains 
ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Update existing edges table to include organization_id
ALTER TABLE edges 
ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Update existing cache_policies table to include organization_id
ALTER TABLE cache_policies 
ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Update existing purge_requests table to include organization_id
ALTER TABLE purge_requests 
ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Update existing request_logs table to include organization_id
ALTER TABLE request_logs 
ADD COLUMN IF NOT EXISTS organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_organizations_slug ON organizations(slug);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_organization_members_org_id ON organization_members(organization_id);
CREATE INDEX IF NOT EXISTS idx_organization_members_user_id ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);

-- Update existing indexes to include organization_id
CREATE INDEX IF NOT EXISTS idx_domains_organization_id ON domains(organization_id);
CREATE INDEX IF NOT EXISTS idx_edges_organization_id ON edges(organization_id);
CREATE INDEX IF NOT EXISTS idx_cache_policies_organization_id ON cache_policies(organization_id);
CREATE INDEX IF NOT EXISTS idx_purge_requests_organization_id ON purge_requests(organization_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_organization_id ON request_logs(organization_id);

-- Create updated_at triggers for new tables
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_organizations_updated_at BEFORE UPDATE ON organizations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_organization_members_updated_at BEFORE UPDATE ON organization_members FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data for development
INSERT INTO organizations (name, slug, description, plan) VALUES 
('NaijCloud Demo', 'naijcloud-demo', 'Default demo organization', 'pro'),
('Test Company', 'test-company', 'Test organization for development', 'free')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO users (email, full_name, password_hash, is_verified) VALUES 
('admin@naijcloud.com', 'Admin User', '$2a$10$dummy.hash.for.development', true),
('user@testcompany.com', 'Test User', '$2a$10$dummy.hash.for.development', true)
ON CONFLICT (email) DO NOTHING;

-- Create organization memberships
INSERT INTO organization_members (organization_id, user_id, role) 
SELECT 
    o.id,
    u.id,
    'owner'
FROM organizations o, users u 
WHERE o.slug = 'naijcloud-demo' AND u.email = 'admin@naijcloud.com'
ON CONFLICT (organization_id, user_id) DO NOTHING;

INSERT INTO organization_members (organization_id, user_id, role) 
SELECT 
    o.id,
    u.id,
    'admin'
FROM organizations o, users u 
WHERE o.slug = 'test-company' AND u.email = 'user@testcompany.com'
ON CONFLICT (organization_id, user_id) DO NOTHING;

-- Update existing data to belong to the demo organization
UPDATE domains 
SET organization_id = (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1)
WHERE organization_id IS NULL;

UPDATE edges 
SET organization_id = (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1)
WHERE organization_id IS NULL;

UPDATE cache_policies 
SET organization_id = (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1)
WHERE organization_id IS NULL;

UPDATE purge_requests 
SET organization_id = (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1)
WHERE organization_id IS NULL;

UPDATE request_logs 
SET organization_id = (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1)
WHERE organization_id IS NULL;
