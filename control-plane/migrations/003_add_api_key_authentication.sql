-- Add API key authentication support to the multi-tenancy system
-- This migration adds API key management tables and updates authentication

-- Create API keys table for programmatic authentication
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    key_prefix VARCHAR(10) NOT NULL,
    permissions JSONB DEFAULT '{}',
    scopes TEXT[] DEFAULT ARRAY['read', 'write'],
    rate_limit INTEGER DEFAULT 1000,
    last_used_at TIMESTAMP WITH TIME ZONE,
    last_used_ip INET,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for efficient API key lookups
CREATE INDEX IF NOT EXISTS idx_api_keys_organization_id ON api_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_last_used ON api_keys(last_used_at);

-- Create API key usage tracking table
CREATE TABLE IF NOT EXISTS api_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER,
    request_size_bytes BIGINT,
    response_size_bytes BIGINT,
    client_ip INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for usage analytics
CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_organization_id ON api_key_usage(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_created_at ON api_key_usage(created_at);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_endpoint ON api_key_usage(endpoint);

-- Create rate limiting table for API keys
CREATE TABLE IF NOT EXISTS api_key_rate_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(api_key_id, window_start)
);

-- Create index for rate limiting lookups
CREATE INDEX IF NOT EXISTS idx_api_key_rate_limits_key_window ON api_key_rate_limits(api_key_id, window_start);

-- Insert sample API keys for testing
INSERT INTO api_keys (
    id, organization_id, user_id, name, key_hash, key_prefix, 
    permissions, scopes, rate_limit
) VALUES (
    gen_random_uuid(),
    (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1), -- NaijCloud Demo org
    (SELECT id FROM users WHERE email = 'admin@naijcloud.com' LIMIT 1), -- Admin user
    'Development API Key',
    '$2a$10$dummy.hash.for.development.key',
    'nj_dev_',
    '{"domains": ["read", "write"], "analytics": ["read"], "edges": ["read"]}',
    ARRAY['domains:read', 'domains:write', 'analytics:read', 'edges:read'],
    5000
), (
    gen_random_uuid(),
    (SELECT id FROM organizations WHERE slug = 'naijcloud-demo' LIMIT 1), -- NaijCloud Demo org
    (SELECT id FROM users WHERE email = 'admin@naijcloud.com' LIMIT 1), -- Admin user
    'Production API Key',
    '$2a$10$production.hash.for.testing',
    'nj_prod_',
    '{"domains": ["read"], "analytics": ["read"]}',
    ARRAY['domains:read', 'analytics:read'],
    10000
);
