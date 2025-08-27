'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { apiClient } from '@/lib/api-client';
import { LoadingSpinner } from '@/components/ui/Loading';
import { RecentActivity } from '@/lib/types';
import {
  UserIcon,
  KeyIcon,
  BellIcon,
  ClockIcon,
  BuildingOfficeIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline';

interface Notification {
  id: string;
  type: string;
  title: string;
  message: string;
  read: boolean;
  data: Record<string, unknown>;
  created_at: string;
}

export function RealTimeActivity() {
  const { isAuthenticated } = useAuth();
  const [activities, setActivities] = useState<RecentActivity[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

    const fetchActivities = async () => {
    try {
      const response = await apiClient.getRecentActivity(10);
      if (response && Array.isArray(response)) {
        setActivities(response);
      } else {
        // Fallback to empty array
        setActivities([]);
      }
    } catch (err) {
      console.error('Failed to fetch activities:', err);
      setError('Failed to load activity logs');
      setActivities([]);
    }
  };

  const fetchNotifications = async () => {
    try {
      // For now, use mock data as notifications endpoint is not implemented yet
      const mockNotifications: Notification[] = [
        {
          id: '1',
          type: 'info',
          title: 'System Update',
          message: 'Platform maintenance scheduled for tonight',
          read: false,
          data: {},
          created_at: new Date(Date.now() - 5 * 60000).toISOString(),
        },
        {
          id: '2',
          type: 'warning',
          title: 'High Traffic Alert',
          message: 'Traffic spike detected on CDN edge nodes',
          read: true,
          data: {},
          created_at: new Date(Date.now() - 15 * 60000).toISOString(),
        }
      ];
      setNotifications(mockNotifications);
    } catch (err) {
      console.error('Failed to fetch notifications:', err);
      setError('Failed to load notifications');
      setNotifications([]);
    }
  };

  useEffect(() => {
    if (!isAuthenticated) return;

    const loadData = async () => {
      setIsLoading(true);
      await Promise.all([fetchActivities(), fetchNotifications()]);
      setIsLoading(false);
    };

    loadData();

    // Set up polling for real-time updates
    const interval = setInterval(() => {
      fetchActivities();
      fetchNotifications();
    }, 10000); // Poll every 10 seconds

    return () => clearInterval(interval);
  }, [isAuthenticated]);

  const getActivityIcon = (action: string, type: string) => {
    if (action.includes('login') || action.includes('auth')) return UserIcon;
    if (type === 'api_key' || action.includes('key')) return KeyIcon;
    if (type === 'organization' || action.includes('org')) return BuildingOfficeIcon;
    return ClockIcon;
  };

  const getNotificationIcon = (type: string) => {
    if (type === 'warning') return ExclamationTriangleIcon;
    return BellIcon;
  };

  const formatTimeAgo = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
    return `${Math.floor(diffMins / 1440)}d ago`;
  };

  if (!isAuthenticated) {
    return (
      <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
        <div className="p-8 text-center">
          <UserIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-lg font-medium text-gray-900">Authentication Required</h3>
          <p className="mt-1 text-sm text-gray-500">
            Please sign in to view real-time activity.
          </p>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
        <div className="p-8 flex items-center justify-center">
          <LoadingSpinner size="md" />
          <span className="ml-2 text-sm text-gray-500">Loading real-time data...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Recent Activity */}
      <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            Recent Activity
          </h3>
          
          {error && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}

          {activities.length === 0 ? (
            <div className="text-center py-6">
              <ClockIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No recent activity</h3>
              <p className="mt-1 text-sm text-gray-500">
                Activity logs will appear here as you use the platform.
              </p>
            </div>
          ) : (
            <div className="flow-root">
              <ul className="-mb-8">
                {activities.slice(0, 5).map((activity, index) => {
                  const Icon = getActivityIcon(activity.action, activity.type);
                  
                  return (
                    <li key={activity.id}>
                      <div className="relative pb-8">
                        {index !== activities.length - 1 && index !== 4 ? (
                          <span
                            className="absolute top-4 left-4 -ml-px h-full w-0.5 bg-gray-200"
                            aria-hidden="true"
                          />
                        ) : null}
                        <div className="relative flex space-x-3">
                          <div>
                            <span className="h-8 w-8 rounded-full bg-blue-500 flex items-center justify-center ring-8 ring-white">
                              <Icon className="w-4 h-4 text-white" aria-hidden="true" />
                            </span>
                          </div>
                          <div className="min-w-0 flex-1 pt-1.5 flex justify-between space-x-4">
                            <div>
                              <p className="text-sm text-gray-500">
                                <span className="font-medium text-gray-900">System</span>{' '}
                                {activity.action} {activity.type}
                                {activity.target && (
                                  <span className="font-medium"> &ldquo;{activity.target}&rdquo;</span>
                                )}
                              </p>
                            </div>
                            <div className="text-right text-sm whitespace-nowrap text-gray-500">
                              {formatTimeAgo(activity.timestamp)}
                            </div>
                          </div>
                        </div>
                      </div>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
        </div>
      </div>

      {/* Notifications */}
      <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            Notifications
          </h3>

          {notifications.length === 0 ? (
            <div className="text-center py-6">
              <BellIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No notifications</h3>
              <p className="mt-1 text-sm text-gray-500">
                You&apos;re all caught up! Notifications will appear here.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {notifications.slice(0, 5).map((notification) => {
                const Icon = getNotificationIcon(notification.type);
                
                return (
                  <div
                    key={notification.id}
                    className={`p-4 border rounded-lg ${
                      notification.read 
                        ? 'bg-gray-50 border-gray-200' 
                        : 'bg-blue-50 border-blue-200'
                    }`}
                  >
                    <div className="flex">
                      <div className="flex-shrink-0">
                        <Icon className={`h-5 w-5 ${
                          notification.type === 'warning' ? 'text-yellow-400' : 'text-blue-400'
                        }`} />
                      </div>
                      <div className="ml-3 flex-1">
                        <p className="text-sm font-medium text-gray-900">
                          {notification.title}
                        </p>
                        <p className="mt-1 text-sm text-gray-500">
                          {notification.message}
                        </p>
                        <p className="mt-2 text-xs text-gray-400">
                          {formatTimeAgo(notification.created_at)}
                        </p>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
