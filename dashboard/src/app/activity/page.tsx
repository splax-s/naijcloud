'use client';

import { useState, useEffect } from 'react';
import {
  BellIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  XCircleIcon,
  ClockIcon,
  BuildingOfficeIcon,
} from '@heroicons/react/24/outline';
import { useOrganization } from '@/components/providers/OrganizationProvider';
import { useAuth } from '@/hooks/useAuth';
import { apiClient } from '@/lib/api-client';

interface ActivityApiResponse {
  id: string;
  type: string;
  action: string;
  target?: string;
  resource_id?: string;
  timestamp: string;
  created_at?: string;
  user?: {
    email: string;
  };
  ip_address?: string;
  metadata?: Record<string, unknown>;
}

interface ActivityItem {
  id: string;
  type: 'domain' | 'api_key' | 'organization' | 'user' | 'cache' | 'ssl' | 'success' | 'info' | 'warning' | 'error' | 'edge' | 'system' | 'security' | 'database' | 'monitoring';
  action?: string;
  target?: string;
  timestamp: string;
  user_email?: string;
  ip_address?: string;
  metadata?: Record<string, unknown>;
  title?: string;
  description?: string;
}

export default function ActivityPage() {
  const { organization } = useOrganization();
  const { isAuthenticated } = useAuth();
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<string>('all');

  // Map API activity types to our display types
  const mapActivityType = (apiType: string): ActivityItem['type'] => {
    switch (apiType) {
      case 'edge':
      case 'system':
        return 'info';
      case 'security':
        return 'success';
      case 'database':
      case 'cache':
        return 'info';
      case 'monitoring':
        return 'warning';
      case 'domain':
        return 'success';
      default:
        return 'info';
    }
  };

  useEffect(() => {
    const loadActivities = async () => {
      if (!organization || !isAuthenticated) {
        setLoading(false);
        return;
      }

      try {
        setLoading(true);
        
        // Try to fetch real data from Phase 6 backend using our API client
        try {
          const response = await apiClient.getRecentActivity(50); // Get more activities for the full page
          
          // Transform API response to match our interface
          const transformedActivities: ActivityItem[] = (response || []).map((item: ActivityApiResponse) => ({
            id: item.id || Math.random().toString(),
            type: mapActivityType(item.type || 'info'),
            action: item.action,
            target: item.target || item.resource_id,
            timestamp: item.created_at || item.timestamp || new Date().toISOString(),
            title: item.action || 'Activity',
            description: `${item.action || 'Activity'} for ${item.target || item.resource_id || 'resource'}`,
            user_email: item.user?.email,
            ip_address: item.ip_address,
            metadata: item.metadata || {}
          }));
          setActivities(transformedActivities);
        } catch (apiError) {
          console.warn('Real API failed, using mock data:', apiError);
          
          // Fallback to mock data if Phase 6 API is not available
          const mockActivities: ActivityItem[] = [
            {
              id: '1',
              type: 'success',
              action: 'domain_added',
              target: 'example.com',
              title: 'Domain added successfully',
              description: 'example.com has been configured and is now active',
              timestamp: new Date(Date.now() - 5 * 60000).toISOString(),
              metadata: { domain: 'example.com' }
            },
            {
              id: '2', 
              type: 'info',
              action: 'edge_node_connected',
              target: 'us-east-1',
              title: 'Edge node connected',
              description: 'New edge node in us-east-1 region came online',
              timestamp: new Date(Date.now() - 15 * 60000).toISOString(),
              metadata: { region: 'us-east-1', edgeId: 'edge-123' }
            },
            {
              id: '3',
              type: 'warning',
              action: 'ssl_expiry_warning',
              target: 'test.com',
              title: 'SSL certificate expiring soon',
              description: 'SSL certificate for test.com will expire in 7 days',
              timestamp: new Date(Date.now() - 30 * 60000).toISOString(),
              metadata: { domain: 'test.com', days_remaining: 7 }
            },
            {
              id: '4',
              type: 'error',
              action: 'cache_miss_rate_high',
              target: 'global',
              title: 'High cache miss rate detected',
              description: 'Cache miss rate exceeded 80% threshold',
              timestamp: new Date(Date.now() - 45 * 60000).toISOString(),
              metadata: { miss_rate: 0.85, threshold: 0.8 }
            }
          ];
          setActivities(mockActivities);
        }
      } catch (error) {
        console.error('Failed to load activities:', error);
        setActivities([]);
      } finally {
        setLoading(false);
      }
    };

    loadActivities();
    const interval = setInterval(loadActivities, 30000);
    return () => clearInterval(interval);
  }, [organization, isAuthenticated]);

  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'success':
        return <CheckCircleIcon className="h-5 w-5 text-green-500" />;
      case 'error':
        return <XCircleIcon className="h-5 w-5 text-red-500" />;
      case 'warning':
        return <ExclamationTriangleIcon className="h-5 w-5 text-yellow-500" />;
      case 'info':
      default:
        return <InformationCircleIcon className="h-5 w-5 text-blue-500" />;
    }
  };

  const getActivityBadgeColor = (type: string) => {
    switch (type) {
      case 'success':
        return 'bg-green-100 text-green-800';
      case 'error':
        return 'bg-red-100 text-red-800';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800';
      case 'info':
      default:
        return 'bg-blue-100 text-blue-800';
    }
  };

  const filteredActivities = filter === 'all' 
    ? activities 
    : activities.filter(activity => activity.type === filter);

  const formatTimeAgo = (timestamp: string) => {
    const now = new Date();
    const time = new Date(timestamp);
    const diffInMinutes = Math.floor((now.getTime() - time.getTime()) / (1000 * 60));
    
    if (diffInMinutes < 1) return 'Just now';
    if (diffInMinutes < 60) return `${diffInMinutes}m ago`;
    
    const diffInHours = Math.floor(diffInMinutes / 60);
    if (diffInHours < 24) return `${diffInHours}h ago`;
    
    const diffInDays = Math.floor(diffInHours / 24);
    return `${diffInDays}d ago`;
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">Activity Feed</h1>
            <p className="text-sm text-gray-600 mt-1">
              Monitor real-time activities across your CDN infrastructure
            </p>
          </div>
        </div>
        <div className="text-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto"></div>
          <p className="mt-2 text-gray-500">Loading activities...</p>
        </div>
      </div>
    );
  }

  // Show organization selection prompt if no organization is selected
  if (!organization) {
    return (
      <div className="space-y-6">
        <div className="text-center py-12">
          <BuildingOfficeIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-lg font-medium text-gray-900">No Organization Selected</h3>
          <p className="mt-1 text-sm text-gray-500">
            Please select an organization from the header to view activity.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Activity Feed</h1>
          <p className="text-sm text-gray-600 mt-1">
            Monitor real-time activities across your CDN infrastructure
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <ClockIcon className="h-5 w-5 text-gray-400" />
          <span className="text-sm text-gray-500">Auto-refresh: 30s</span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {[
          { key: 'error', label: 'Errors', count: activities.filter(a => a.type === 'error').length },
          { key: 'warning', label: 'Warnings', count: activities.filter(a => a.type === 'warning').length },
          { key: 'success', label: 'Success', count: activities.filter(a => a.type === 'success').length },
          { key: 'info', label: 'Info', count: activities.filter(a => a.type === 'info').length },
        ].map((stat) => (
          <div key={stat.key} className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-900">{stat.label}</p>
                <p className="text-2xl font-bold text-gray-600">{stat.count}</p>
              </div>
              <div className={`p-2 rounded-lg ${
                stat.key === 'error' ? 'bg-red-100' :
                stat.key === 'warning' ? 'bg-yellow-100' :
                stat.key === 'success' ? 'bg-green-100' : 'bg-blue-100'
              }`}>
                {getActivityIcon(stat.key)}
              </div>
            </div>
          </div>
        ))}
      </div>

      <div className="bg-white shadow border border-gray-200 rounded-lg">
        <div className="px-6 py-4 border-b border-gray-200">
          <div className="flex items-center justify-between">
            <div className="flex items-center">
              <BellIcon className="h-5 w-5 text-gray-400 mr-2" />
              <h3 className="text-lg font-medium text-gray-900">Recent Activities</h3>
              <span className="ml-2 bg-blue-100 text-blue-800 py-0.5 px-2.5 rounded-full text-xs font-medium">
                {filteredActivities.length}
              </span>
            </div>
            <div className="flex items-center space-x-2">
              <select
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                className="text-sm border border-gray-300 rounded-md px-3 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="all">All Activities</option>
                <option value="success">Success</option>
                <option value="error">Errors</option>
                <option value="warning">Warnings</option>
                <option value="info">Info</option>
                <option value="domain">Domains</option>
                <option value="security">Security</option>
                <option value="system">System</option>
              </select>
            </div>
          </div>
        </div>
        
        <div className="divide-y divide-gray-200">
          {filteredActivities.length > 0 ? (
            filteredActivities.map((activity) => (
              <div key={activity.id} className="p-6 hover:bg-gray-50">
                <div className="flex">
                  <div className="flex-shrink-0">
                    {getActivityIcon(activity.type)}
                  </div>
                  <div className="ml-4 flex-1">
                    <div className="flex items-center justify-between">
                      <h4 className="text-sm font-medium text-gray-900">
                        {activity.title}
                      </h4>
                      <div className="flex items-center space-x-2">
                        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${getActivityBadgeColor(activity.type)}`}>
                          {activity.type}
                        </span>
                        <span className="text-sm text-gray-500">
                          {formatTimeAgo(activity.timestamp)}
                        </span>
                      </div>
                    </div>
                    <p className="mt-1 text-sm text-gray-600">
                      {activity.description}
                    </p>
                    {activity.target && (
                      <div className="mt-2 text-xs text-gray-500">
                        Target: <span className="font-mono">{activity.target}</span>
                      </div>
                    )}
                    {activity.action && (
                      <div className="mt-1 text-xs text-gray-500">
                        Action: <span className="font-mono">{activity.action}</span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))
          ) : (
            <div className="text-center py-12">
              <BellIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No activities found</h3>
              <p className="mt-1 text-sm text-gray-500">
                {filter === 'all' 
                  ? 'No activities to display yet.' 
                  : `No ${filter} activities found.`
                }
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}