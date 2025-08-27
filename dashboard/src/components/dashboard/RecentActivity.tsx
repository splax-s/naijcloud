'use client';

import { 
  GlobeAltIcon, 
  ClockIcon, 
  CheckCircleIcon,
  ServerIcon,
  ShieldCheckIcon,
  CircleStackIcon,
  ExclamationTriangleIcon,
  CpuChipIcon
} from '@heroicons/react/24/outline';
import { useRecentActivity } from '@/lib/hooks';
import { LoadingCard, ErrorCard } from '@/components/ui/Loading';
import { formatDistanceToNow } from 'date-fns';
import { useOrganization } from '@/components/providers/OrganizationProvider';

export function RecentActivity() {
  const { organization } = useOrganization();
  const { recentActivity, isLoading, isError, mutate } = useRecentActivity(6, organization?.slug);

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
    formattedTime: formatDistanceToNow(new Date(activity.timestamp), { addSuffix: true }),
  })) || [];

  function getIconForActivityType(type: string) {
    switch (type) {
      case 'domain':
        return GlobeAltIcon;
      case 'cache':
        return ClockIcon;
      case 'edge':
        return ServerIcon;
      case 'system':
        return CpuChipIcon;
      case 'security':
        return ShieldCheckIcon;
      case 'database':
        return CircleStackIcon;
      case 'monitoring':
        return ExclamationTriangleIcon;
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
      case 'edge':
        return 'bg-green-500';
      case 'system':
        return 'bg-purple-500';
      case 'security':
        return 'bg-indigo-500';
      case 'database':
        return 'bg-cyan-500';
      case 'monitoring':
        return 'bg-orange-500';
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
                {activity.formattedTime}
              </div>
            </div>
          </div>
        ))}
      </div>
      <div className="px-6 py-3 bg-gray-50 border-t border-gray-200">
        <a
          href="/activity"
          className="text-sm font-medium text-blue-600 hover:text-blue-500"
        >
          View all activity →
        </a>
      </div>
    </div>
  );
}
