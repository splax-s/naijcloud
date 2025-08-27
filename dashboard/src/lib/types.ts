export interface SSLCertificate {
  status: 'valid' | 'expired' | 'invalid' | 'expiring';
  issuer: string;
  expires_at: string;
  subject: string;
}

export interface Domain {
  id: string;
  domain: string;
  origin: string;
  enabled: boolean;
  ssl_enabled: boolean;
  cache_ttl: number;
  created_at: string;
  updated_at: string;
  ssl_certificate?: SSLCertificate;
  bandwidth_usage?: number;
  compression_enabled?: boolean;
  security_level?: 'off' | 'low' | 'medium' | 'high' | 'under_attack';
  custom_headers?: Record<string, string>;
}

export interface DomainFormData {
  domain: string;
  origin: string;
  ssl_enabled: boolean;
  cache_ttl: number;
  compression_enabled: boolean;
  security_level: 'off' | 'low' | 'medium' | 'high' | 'under_attack';
}

export interface DomainModalProps {
  domain?: Domain | null;
  isOpen: boolean;
  onClose: () => void;
}

export interface AddDomainModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (domainData: DomainFormData) => Promise<void>;
}

export interface FormErrors {
  domain?: string;
  origin?: string;
  submit?: string;
}

export interface EdgeNode {
  id: string;
  organization_id?: string;
  hostname: string;
  ip_address: string;
  region: string;
  location?: string;
  capacity: number;
  status: 'healthy' | 'degraded' | 'unhealthy' | 'online' | 'offline' | 'warning';
  health_score?: number;
  last_heartbeat: string;
  created_at: string;
  version?: string;
  total_requests?: number;
  cache_hit_ratio?: number;
  avg_response_time?: number;
  metadata?: Record<string, unknown>;
}

export interface CacheEntry {
  key: string;
  domain: string;
  path: string;
  size: number;
  ttl: number;
  created_at: string;
  expires_at: string;
  hit_count: number;
}

export interface PurgeRequest {
  id: string;
  domain_id: string;
  paths: string[];
  status: 'pending' | 'in_progress' | 'completed' | 'failed';
  created_at: string;
  completed_at?: string;
}

export interface Analytics {
  domain_id: string;
  timestamp: string;
  requests: number;
  bytes_transferred: number;
  cache_hits: number;
  cache_misses: number;
  response_time_avg: number;
  status_2xx: number;
  status_3xx: number;
  status_4xx: number;
  status_5xx: number;
}

export interface SystemHealth {
  control_plane: {
    status: 'healthy' | 'degraded' | 'down';
    uptime: string;
    version: string;
  };
  database: {
    status: 'healthy' | 'degraded' | 'down';
    connections: number;
    query_time_avg: number;
  };
  redis: {
    status: 'healthy' | 'degraded' | 'down';
    memory_usage: number;
    connected_clients: number;
  };
}

export interface DashboardMetrics {
  total_domains: number;
  active_edge_nodes: number;
  total_requests_24h: number;
  cache_hit_ratio: number;
  avg_response_time: number;
  bandwidth_24h: number;
}

export interface TrafficData {
  timestamp: string;
  requests: number;
  bandwidth: number;
  cache_hits: number;
  cache_misses: number;
}

export interface TopDomain {
  domain: string;
  requests: number;
  bandwidth: number;
  cache_hit_ratio: number;
}

export interface RecentActivity {
  id: string;
  type: string;
  action: string;
  target: string;
  timestamp: string;
}
