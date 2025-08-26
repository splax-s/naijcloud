'use client';

import {
  GlobeAltIcon,
  ServerIcon,
  ChartBarIcon,
  ClockIcon,
} from '@heroicons/react/24/outline';
import { useDashboardMetrics } from '@/lib/hooks';
import { LoadingSpinner, ErrorCard } from '@/components/ui/Loading';

export function StatsCards() {
  const { metrics, isLoading, isError, mutate } = useDashboardMetrics();

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
            <div className="p-5 flex items-center justify-center">
              <LoadingSpinner size="md" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (isError) {
    return (
      <ErrorCard 
        title="Failed to load metrics" 
        description="Unable to fetch dashboard metrics. Please try again."
        onRetry={() => mutate()}
      />
    );
  }

  const stats = [
    {
      name: 'Total Domains',
      value: metrics?.total_domains.toString() || '0',
      change: '+2.1%',
      changeType: 'positive',
      icon: GlobeAltIcon,
    },
    {
      name: 'Active Edge Nodes',
      value: metrics?.active_edge_nodes.toString() || '0',
      change: '+1',
      changeType: 'positive',
      icon: ServerIcon,
    },
    {
      name: 'Cache Hit Ratio',
      value: metrics?.cache_hit_ratio ? `${(metrics.cache_hit_ratio * 100).toFixed(1)}%` : '0%',
      change: '+0.3%',
      changeType: 'positive',
      icon: ChartBarIcon,
    },
    {
      name: 'Avg Response Time',
      value: metrics?.avg_response_time ? `${metrics.avg_response_time}ms` : '0ms',
      change: '-2ms',
      changeType: 'positive',
      icon: ClockIcon,
    },
  ];
  return (
    <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      {stats.map((stat) => (
        <div
          key={stat.name}
          className="bg-white overflow-hidden shadow rounded-lg border border-gray-200"
        >
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <stat.icon className="h-6 w-6 text-gray-400" />
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">
                    {stat.name}
                  </dt>
                  <dd className="flex items-baseline">
                    <div className="text-2xl font-semibold text-gray-900">
                      {stat.value}
                    </div>
                    <div
                      className={`ml-2 flex items-baseline text-sm font-semibold ${
                        stat.changeType === 'positive'
                          ? 'text-green-600'
                          : 'text-red-600'
                      }`}
                    >
                      {stat.change}
                    </div>
                  </dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
