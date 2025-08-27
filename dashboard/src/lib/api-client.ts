import { 
  Domain, 
  EdgeNode, 
  CacheEntry, 
  PurgeRequest, 
  Analytics, 
  SystemHealth,
  DashboardMetrics,
  TrafficData,
  TopDomain,
  RecentActivity 
} from './types';
import { mockData } from './mock-data';
import { getSession } from 'next-auth/react';
import { ExtendedSession } from '../hooks/useAuth';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface ApiClientOptions {
  organizationSlug?: string;
}

interface OrganizationSettings {
  [key: string]: unknown;
}

interface User {
  id: string;
  name: string;
  email: string;
  avatar_url?: string;
}

interface OrganizationMember {
  user: User;
  role: string;
  permissions: string[];
}

class ApiClient {
  private baseUrl: string;
  private organizationSlug?: string;

  constructor(baseUrl: string = API_BASE_URL, options?: ApiClientOptions) {
    this.baseUrl = baseUrl + '/api/v1'; // Add api/v1 prefix for the Go API
    this.organizationSlug = options?.organizationSlug;
  }

  // Create organization-scoped client
  forOrganization(organizationSlug: string): ApiClient {
    return new ApiClient(API_BASE_URL, { organizationSlug });
  }

  // Endpoints that exist in the Go backend (Phase 6)
  private readonly realEndpoints = [
    '/auth', // Authentication endpoints
    '/users', // User management
    '/organizations', // Organization management  
    '/api-keys', // API key management
    '/activity', // Activity logs
    '/notifications', // Notifications
    '/health', // Health check
    '/metrics', // Basic metrics
    '/admin', // Admin endpoints
    // Future Phase 7 endpoints:
    '/domains',
    '/edges', 
    '/analytics',
    '/cache'
  ];

  private isRealEndpoint(endpoint: string): boolean {
    return this.realEndpoints.some(real => endpoint.startsWith(real));
  }

  private async getAuthHeaders(): Promise<Record<string, string>> {
    const session = await getSession() as ExtendedSession | null;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    // Use JWT token if available (Phase 6 backend)
    if (session?.accessToken) {
      headers['Authorization'] = `Bearer ${session.accessToken}`;
    }

    // Fallback to user ID headers for mock/testing
    if (session?.user) {
      headers['X-User-ID'] = session.user.id;
      headers['X-User-Email'] = session.user.email || '';
    }

    if (this.organizationSlug) {
      headers['X-Organization-Slug'] = this.organizationSlug;
    }

    return headers;
  }

  private getEndpointUrl(endpoint: string): string {
    // For organization-scoped endpoints, use the multi-tenant API structure
    if (this.organizationSlug && (endpoint.startsWith('/domains') || endpoint.startsWith('/edges') || endpoint.startsWith('/analytics'))) {
      return `${this.baseUrl}/orgs/${this.organizationSlug}${endpoint}`;
    }
    
    return `${this.baseUrl}${endpoint}`;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    // Check if this endpoint exists in the real API
    const useRealAPI = this.isRealEndpoint(endpoint) || endpoint.startsWith('/orgs');
    
    if (useRealAPI) {
      // Try real API for endpoints that exist in Go backend
      try {
        const url = this.getEndpointUrl(endpoint);
        const authHeaders = await this.getAuthHeaders();
        
        const response = await fetch(url, {
          headers: {
            ...authHeaders,
            ...options.headers,
          },
          ...options,
        });

        if (!response.ok) {
          throw new Error(`API request failed: ${response.status} ${response.statusText}`);
        }

        return response.json();
      } catch (error) {
        console.warn(`Real API call failed for ${endpoint}, falling back to mock data:`, error);
        return this.getMockData(endpoint) as T;
      }
    } else {
      // Use mock data for dashboard-specific endpoints that don't exist yet
      if (process.env.NODE_ENV === 'development') {
        console.debug(`Using mock data for ${endpoint} (endpoint not implemented in backend)`);
      }
      return this.getMockData(endpoint) as T;
    }
  }

  private getMockData(endpoint: string): Domain[] | EdgeNode[] | Analytics[] | SystemHealth | DashboardMetrics | TrafficData[] | TopDomain[] | RecentActivity[] {
    // Return appropriate mock data based on endpoint
    if (endpoint === '/metrics/dashboard') {
      return mockData.dashboardMetrics;
    }
    if (endpoint.startsWith('/metrics/traffic')) {
      return mockData.trafficData;
    }
    if (endpoint.startsWith('/metrics/top-domains')) {
      return mockData.topDomains;
    }
    if (endpoint.startsWith('/activity')) {
      return mockData.recentActivity;
    }
    if (endpoint === '/domains') {
      return mockData.domains;
    }
    if (endpoint === '/edges') {
      return mockData.edgeNodes;
    }
    
    // Default empty response
    return [];
  }

  // Domain Management
  async getDomains(): Promise<Domain[]> {
    const response = await this.request<{ domains: Domain[] | null }>('/domains');
    return response.domains || [];
  }

  async getDomain(id: string): Promise<Domain> {
    return this.request<Domain>(`/domains/${id}`);
  }

  async createDomain(domain: Omit<Domain, 'id' | 'created_at' | 'updated_at'>): Promise<Domain> {
    return this.request<Domain>('/domains', {
      method: 'POST',
      body: JSON.stringify(domain),
    });
  }

  async updateDomain(id: string, domain: Partial<Domain>): Promise<Domain> {
    return this.request<Domain>(`/domains/${id}`, {
      method: 'PUT',
      body: JSON.stringify(domain),
    });
  }

  async deleteDomain(id: string): Promise<void> {
    await this.request(`/domains/${id}`, { method: 'DELETE' });
  }

  // Organization Management
  async getUserOrganizations(): Promise<{ organizations: Array<{ id: string; name: string; slug: string; role: string }> }> {
    return this.request<{ organizations: Array<{ id: string; name: string; slug: string; role: string }> }>('/user/organizations');
  }

  async getOrganization(slug: string): Promise<{ id: string; name: string; slug: string; plan: string; settings: OrganizationSettings }> {
    return this.request<{ id: string; name: string; slug: string; plan: string; settings: OrganizationSettings }>(`/orgs/${slug}`);
  }

  async getOrganizationMembers(slug: string): Promise<{ members: OrganizationMember[] }> {
    return this.request<{ members: OrganizationMember[] }>(`/orgs/${slug}/members`);
  }

  // Edge Node Management
  async getEdgeNodes(): Promise<EdgeNode[]> {
    const response = await this.request<{ edges: EdgeNode[] | null }>('/edges');
    return response.edges || [];
  }

  async getEdgeNode(id: string): Promise<EdgeNode> {
    return this.request<EdgeNode>(`/edges/${id}`);
  }

  async deleteEdgeNode(id: string): Promise<void> {
    await this.request(`/edges/${id}`, { method: 'DELETE' });
  }

  // Cache Management
  async getCacheEntries(domain?: string): Promise<CacheEntry[]> {
    const params = domain ? `?domain=${encodeURIComponent(domain)}` : '';
    return this.request<CacheEntry[]>(`/cache${params}`);
  }

  async purgeCacheByPath(domainId: string, paths: string[]): Promise<PurgeRequest> {
    return this.request<PurgeRequest>('/cache/purge', {
      method: 'POST',
      body: JSON.stringify({ domain_id: domainId, paths }),
    });
  }

  async purgeCacheByDomain(domainId: string): Promise<PurgeRequest> {
    return this.request<PurgeRequest>('/cache/purge', {
      method: 'POST',
      body: JSON.stringify({ domain_id: domainId, paths: ['/*'] }),
    });
  }

  async getPurgeRequests(): Promise<PurgeRequest[]> {
    return this.request<PurgeRequest[]>('/cache/purge');
  }

  // Analytics
  async getAnalytics(params: {
    domain_id?: string;
    start_time?: string;
    end_time?: string;
    granularity?: 'hour' | 'day' | 'week';
  }): Promise<Analytics[]> {
    const searchParams = new URLSearchParams();
    
    if (params.domain_id) searchParams.set('domain_id', params.domain_id);
    if (params.start_time) searchParams.set('start_time', params.start_time);
    if (params.end_time) searchParams.set('end_time', params.end_time);
    if (params.granularity) searchParams.set('granularity', params.granularity);

    const query = searchParams.toString();
    return this.request<Analytics[]>(`/analytics${query ? `?${query}` : ''}`);
  }

  // System Health
  async getSystemHealth(): Promise<SystemHealth> {
    return this.request<SystemHealth>('/health');
  }

  // Real-time metrics for dashboard
  async getDashboardMetrics(): Promise<DashboardMetrics> {
    return this.request('/metrics/dashboard');
  }

  async getTrafficData(hours: number = 24): Promise<TrafficData[]> {
    return this.request(`/metrics/traffic?hours=${hours}`);
  }

  async getTopDomains(limit: number = 10): Promise<TopDomain[]> {
    return this.request(`/metrics/top-domains?limit=${limit}`);
  }

  async getRecentActivity(limit: number = 10): Promise<RecentActivity[]> {
    return this.request(`/activity?limit=${limit}`);
  }
}

// Export singleton instance
export const apiClient = new ApiClient();
export { ApiClient };
