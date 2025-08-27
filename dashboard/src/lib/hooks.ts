import useSWR from 'swr';
import { apiClient } from './api-client';
import { Domain, EdgeNode, CacheEntry, Analytics } from './types';

// Hook to create organization-scoped API client
function useOrganizationApiClient(organizationSlug?: string | null) {
  if (!organizationSlug) return apiClient;
  return apiClient.forOrganization(organizationSlug);
}

// Domain hooks
export function useDomains(organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR<Domain[]>(
    organizationSlug ? `/orgs/${organizationSlug}/domains` : '/api/domains', 
    () => client.getDomains()
  );
  
  return {
    domains: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useDomain(id: string, organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR<Domain>(
    id && organizationSlug ? `/orgs/${organizationSlug}/domains/${id}` : null,
    () => client.getDomain(id)
  );
  
  return {
    domain: data,
    isLoading,
    isError: error,
    mutate,
  };
}

// Organization hooks
export function useUserOrganizations() {
  const { data, error, isLoading, mutate } = useSWR('/user/organizations', () => 
    apiClient.getUserOrganizations()
  );
  
  return {
    organizations: data?.organizations || [],
    isLoading,
    isError: error,
    mutate,
  };
}

export function useOrganization(slug: string) {
  const { data, error, isLoading, mutate } = useSWR(
    slug ? `/orgs/${slug}` : null,
    () => apiClient.getOrganization(slug)
  );
  
  return {
    organization: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useOrganizationMembers(slug: string) {
  const { data, error, isLoading, mutate } = useSWR(
    slug ? `/orgs/${slug}/members` : null,
    () => apiClient.getOrganizationMembers(slug)
  );
  
  return {
    members: data?.members || [],
    isLoading,
    isError: error,
    mutate,
  };
}

// Edge Node hooks
export function useEdgeNodes(organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR<EdgeNode[]>(
    organizationSlug ? `/orgs/${organizationSlug}/edges` : '/api/edges',
    () => client.getEdgeNodes()
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
export function useDashboardMetrics(organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR(
    organizationSlug ? `/orgs/${organizationSlug}/analytics/overview` : '/api/metrics/dashboard',
    () => client.getDashboardMetrics(),
    { refreshInterval: 30000 } // Refresh every 30 seconds
  );
  
  return {
    metrics: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useTrafficData(hours: number = 24, organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR(
    organizationSlug ? `/orgs/${organizationSlug}/analytics/traffic?hours=${hours}` : `/api/metrics/traffic?hours=${hours}`,
    () => client.getTrafficData(hours),
    { refreshInterval: 60000 } // Refresh every minute
  );
  
  return {
    trafficData: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useTopDomains(limit: number = 10, organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR(
    organizationSlug ? `/orgs/${organizationSlug}/analytics/top-domains?limit=${limit}` : `/api/metrics/top-domains?limit=${limit}`,
    () => client.getTopDomains(limit),
    { refreshInterval: 60000 }
  );
  
  return {
    topDomains: data,
    isLoading,
    isError: error,
    mutate,
  };
}

export function useRecentActivity(limit: number = 10, organizationSlug?: string | null) {
  const client = useOrganizationApiClient(organizationSlug);
  const { data, error, isLoading, mutate } = useSWR(
    organizationSlug ? `/orgs/${organizationSlug}/activity?limit=${limit}` : `/api/activity?limit=${limit}`,
    () => client.getRecentActivity(limit),
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
