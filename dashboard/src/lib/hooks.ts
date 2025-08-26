import useSWR from 'swr';
import { apiClient } from './api-client';
import { Domain, EdgeNode, CacheEntry, Analytics } from './types';

// Domain hooks
export function useDomains() {
  const { data, error, isLoading, mutate } = useSWR<Domain[]>('/api/domains', () => 
    apiClient.getDomains()
  );
  
  return {
    domains: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useDomain(id: string) {
  const { data, error, isLoading, mutate } = useSWR<Domain>(
    id ? `/api/domains/${id}` : null,
    () => apiClient.getDomain(id)
  );
  
  return {
    domain: data,
    isLoading,
    isError: error,
    mutate,
  };
}

// Edge Node hooks
export function useEdgeNodes() {
  const { data, error, isLoading, mutate } = useSWR<EdgeNode[]>('/api/edges', () => 
    apiClient.getEdgeNodes()
  );
  
  return {
    edgeNodes: data,
    isLoading,
    isError: error,
    mutate,
  };
}

// Cache hooks
export function useCacheEntries(domain?: string) {
  const { data, error, isLoading, mutate } = useSWR<CacheEntry[]>(
    `/api/cache${domain ? `?domain=${domain}` : ''}`,
    () => apiClient.getCacheEntries(domain)
  );
  
  return {
    cacheEntries: data,
    isLoading,
    isError: error,
    mutate,
  };
}

// Analytics hooks
export function useAnalytics(params: {
  domain_id?: string;
  start_time?: string;
  end_time?: string;
  granularity?: 'hour' | 'day' | 'week';
} = {}) {
  const key = `/api/analytics?${new URLSearchParams(params as Record<string, string>).toString()}`;
  
  const { data, error, isLoading, mutate } = useSWR<Analytics[]>(key, () => 
    apiClient.getAnalytics(params)
  );
  
  return {
    analytics: data,
    isLoading,
    isError: error,
    mutate,
  };
}

// Dashboard specific hooks
export function useDashboardMetrics() {
  const { data, error, isLoading, mutate } = useSWR('/api/metrics/dashboard', () => 
    apiClient.getDashboardMetrics(),
    { refreshInterval: 30000 } // Refresh every 30 seconds
  );
  
  return {
    metrics: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useTrafficData(hours: number = 24) {
  const { data, error, isLoading, mutate } = useSWR(`/api/metrics/traffic?hours=${hours}`, () => 
    apiClient.getTrafficData(hours),
    { refreshInterval: 60000 } // Refresh every minute
  );
  
  return {
    trafficData: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useTopDomains(limit: number = 10) {
  const { data, error, isLoading, mutate } = useSWR(`/api/metrics/top-domains?limit=${limit}`, () => 
    apiClient.getTopDomains(limit),
    { refreshInterval: 60000 }
  );
  
  return {
    topDomains: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useRecentActivity(limit: number = 10) {
  const { data, error, isLoading, mutate } = useSWR(`/api/activity?limit=${limit}`, () => 
    apiClient.getRecentActivity(limit),
    { refreshInterval: 30000 }
  );
  
  return {
    recentActivity: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useSystemHealth() {
  const { data, error, isLoading, mutate } = useSWR('/api/health', () => 
    apiClient.getSystemHealth(),
    { refreshInterval: 10000 } // Refresh every 10 seconds
  );
  
  return {
    health: data,
    isLoading,
    isError: error,
    mutate,
  };
}
