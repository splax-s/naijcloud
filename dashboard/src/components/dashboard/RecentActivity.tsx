'use client';

import { GlobeAltIcon, ClockIcon, CheckCircleIcon } from '@heroicons/react/24/outline';
import { useRecentActivity } from '@/lib/hooks';
import { LoadingCard, ErrorCard } from '@/components/ui/Loading';

export function RecentActivity() {
  const { recentActivity, isLoading, isError, mutate } = useRecentActivity(5);

  if (isLoading) {
    return (
      <LoadingCard 
        title="Loading recent activity..." 
        description="Fetching the latest system events"
      />
    );
  }

  if (isError) {
    return (
      <ErrorCard 
        title="Failed to load activity" 
        description="Unable to fetch recent activity. Please try again."
        onRetry={() => mutate()}
      />
    );
  }

  const activities = recentActivity?.map(activity => ({
    ...activity,
    icon: getIconForActivityType(activity.type),
    iconBackground: getIconBackgroundForActivityType(activity.type),
  })) || [];

  function getIconForActivityType(type: string) {
    switch (type) {
      case 'domain':
        return GlobeAltIcon;
      case 'cache':
        return ClockIcon;
      case 'health':
        return CheckCircleIcon;
      default:
        return CheckCircleIcon;
    }
  }

  function getIconBackgroundForActivityType(type: string) {
    switch (type) {
      case 'domain':
        return 'bg-blue-500';
      case 'cache':
        return 'bg-yellow-500';
      case 'health':
        return 'bg-green-500';
      default:
        return 'bg-gray-500';
    }
  }
  return (
    <div className="bg-white shadow rounded-lg border border-gray-200">
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="text-lg font-medium text-gray-900">Recent Activity</h3>
      </div>
      <div className="divide-y divide-gray-200">
        {activities.map((activity) => (
          <div key={activity.id} className="px-6 py-4">
            <div className="flex items-center space-x-3">
              <div
                className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${activity.iconBackground}`}
              >
                <activity.icon className="w-4 h-4 text-white" />
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900">
                  {activity.action}
                </p>
                <p className="text-sm text-gray-500">{activity.target}</p>
              </div>
              <div className="flex-shrink-0 text-sm text-gray-500">
                {activity.timestamp}
              </div>
            </div>
          </div>
        ))}
      </div>
      <div className="px-6 py-3 bg-gray-50 border-t border-gray-200">
        <a
          href="#"
          className="text-sm font-medium text-blue-600 hover:text-blue-500"
        >
          View all activity →
        </a>
      </div>
    </div>
  );
}
