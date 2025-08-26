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

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string = API_BASE_URL) {
    this.baseUrl = baseUrl + '/v1'; // Add v1 prefix for the Go API
  }

  // Endpoints that exist in the Go backend
  private readonly realEndpoints = [
    '/domains',
    '/edges', 
    '/analytics'
  ];

  private isRealEndpoint(endpoint: string): boolean {
    return this.realEndpoints.some(real => endpoint.startsWith(real));
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<T> {
    // Check if this endpoint exists in the real API
    const useRealAPI = this.isRealEndpoint(endpoint);
    
    if (useRealAPI) {
      // Try real API for endpoints that exist in Go backend
      try {
        const url = `${this.baseUrl}${endpoint}`;
        const response = await fetch(url, {
          headers: {
            'Content-Type': 'application/json',
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
