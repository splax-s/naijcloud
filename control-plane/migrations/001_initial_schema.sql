-- Migration 001: Initial schema
-- Creates the basic domain and edge infrastructure tables

-- Domains table - stores registered domains and their configurations
CREATE TABLE domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain VARCHAR(255) UNIQUE NOT NULL,
    origin_url VARCHAR(512) NOT NULL,
    cache_ttl INTEGER DEFAULT 3600, -- seconds
    rate_limit INTEGER DEFAULT 1000, -- requests per minute
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Edge nodes table - tracks all edge proxy instances
CREATE TABLE edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region VARCHAR(50) NOT NULL,
    ip_address INET NOT NULL,
    hostname VARCHAR(255),
    capacity INTEGER DEFAULT 1000, -- max requests per second
    status VARCHAR(20) DEFAULT 'healthy' CHECK (status IN ('healthy', 'degraded', 'unhealthy')),
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    metadata JSONB DEFAULT '{}' -- flexible metadata storage
);

-- Cache policies table - fine-grained cache rules per domain
CREATE TABLE cache_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    path_pattern VARCHAR(512) NOT NULL, -- e.g., "/*.jpg", "/api/*"
    cache_ttl INTEGER NOT NULL, -- seconds
    cache_key_template VARCHAR(512), -- custom cache key generation
    headers_to_vary TEXT[], -- headers that affect caching
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Request logs table - stores request analytics (partitioned by date)
CREATE TABLE request_logs (
    id UUID DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id),
    edge_id UUID NOT NULL REFERENCES edges(id),
    request_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    method VARCHAR(10) NOT NULL,
    path VARCHAR(2048) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER NOT NULL,
    bytes_sent BIGINT DEFAULT 0,
    cache_status VARCHAR(20) NOT NULL CHECK (cache_status IN ('hit', 'miss', 'stale', 'bypass')),
    client_ip INET,
    user_agent TEXT,
    referer TEXT
) PARTITION BY RANGE (request_time);

-- Create monthly partitions for request_logs (example for 2025)
CREATE TABLE request_logs_2025_01 PARTITION OF request_logs
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE request_logs_2025_02 PARTITION OF request_logs
    FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');

-- Purge requests table - tracks cache purge operations
CREATE TABLE purge_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id UUID NOT NULL REFERENCES domains(id),
    paths TEXT[] NOT NULL, -- array of path patterns to purge
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed')),
    requested_by VARCHAR(255), -- user/system that requested purge
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Basic users table for authentication
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_domains_domain ON domains(domain);
CREATE INDEX idx_domains_status ON domains(status);

CREATE INDEX idx_edges_region ON edges(region);
CREATE INDEX idx_edges_status ON edges(status);
CREATE INDEX idx_edges_last_heartbeat ON edges(last_heartbeat);

CREATE INDEX idx_cache_policies_domain_id ON cache_policies(domain_id);
CREATE INDEX idx_cache_policies_path_pattern ON cache_policies(path_pattern);

CREATE INDEX idx_request_logs_domain_time ON request_logs(domain_id, request_time);
CREATE INDEX idx_request_logs_edge_time ON request_logs(edge_id, request_time);
CREATE INDEX idx_request_logs_cache_status ON request_logs(cache_status);

CREATE INDEX idx_purge_requests_domain_id ON purge_requests(domain_id);
CREATE INDEX idx_purge_requests_status ON purge_requests(status);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);
