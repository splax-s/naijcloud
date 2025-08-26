'use client';

import { ServerIcon, SignalIcon, ClockIcon } from '@heroicons/react/24/outline';
import { useEdgeNodes } from '@/lib/hooks';
import { LoadingCard, ErrorCard } from '@/components/ui/Loading';

function getStatusColor(status: string) {
  switch (status) {
    case 'online':
      return 'bg-green-100 text-green-800';
    case 'warning':
      return 'bg-yellow-100 text-yellow-800';
    case 'offline':
      return 'bg-red-100 text-red-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
}

function getHealthColor(health: number) {
  if (health >= 95) return 'text-green-600';
  if (health >= 90) return 'text-yellow-600';
  return 'text-red-600';
}

function formatTimestamp(timestamp: string) {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSeconds = Math.floor(diffMs / 1000);
  const diffMinutes = Math.floor(diffSeconds / 60);
  
  if (diffSeconds < 60) {
    return `${diffSeconds} seconds ago`;
  } else if (diffMinutes < 60) {
    return `${diffMinutes} minute${diffMinutes === 1 ? '' : 's'} ago`;
  } else {
    const diffHours = Math.floor(diffMinutes / 60);
    return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
  }
}

export default function EdgeNodesPage() {
  const { edgeNodes, isLoading, isError, mutate } = useEdgeNodes();

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">Edge Nodes</h1>
        <p className="text-sm text-gray-600 mt-1">
          Monitor and manage your edge node infrastructure
        </p>
      </div>

      {/* Summary Stats */}
      {isLoading ? (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="bg-white overflow-hidden shadow rounded-lg border border-gray-200 p-5 flex items-center justify-center">
              <div className="animate-pulse bg-gray-200 h-16 w-full rounded"></div>
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
            <div className="p-5">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <ServerIcon className="h-8 w-8 text-blue-600" />
                </div>
                <div className="ml-5 w-0 flex-1">
                  <dl>
                    <dt className="text-sm font-medium text-gray-500 truncate">
                      Total Nodes
                    </dt>
                    <dd className="text-2xl font-semibold text-gray-900">
                      {edgeNodes?.length || 0}
                    </dd>
                  </dl>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
            <div className="p-5">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <SignalIcon className="h-8 w-8 text-green-600" />
                </div>
                <div className="ml-5 w-0 flex-1">
                  <dl>
                    <dt className="text-sm font-medium text-gray-500 truncate">
                      Online Nodes
                    </dt>
                    <dd className="text-2xl font-semibold text-gray-900">
                      {edgeNodes?.filter(node => node.status === 'online').length || 0}
                    </dd>
                  </dl>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
            <div className="p-5">
              <div className="flex items-center">
                <div className="flex-shrink-0">
                  <ClockIcon className="h-8 w-8 text-purple-600" />
                </div>
                <div className="ml-5 w-0 flex-1">
                  <dl>
                    <dt className="text-sm font-medium text-gray-500 truncate">
                      Avg Latency
                    </dt>
                    <dd className="text-2xl font-semibold text-gray-900">
                      {edgeNodes?.length 
                        ? Math.round(edgeNodes.reduce((sum, node) => sum + node.avg_response_time, 0) / edgeNodes.length)
                        : 0}ms
                    </dd>
                  </dl>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Edge Nodes Table */}
      {isLoading ? (
        <LoadingCard title="Loading edge nodes..." description="Fetching node status and metrics" />
      ) : isError ? (
        <ErrorCard 
          title="Failed to load edge nodes" 
          description="Unable to fetch edge node data. Please try again."
          onRetry={() => mutate()}
        />
      ) : (
        <div className="bg-white shadow border border-gray-200 rounded-lg overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-medium text-gray-900">Edge Node Status</h3>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Node
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Location
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Health
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Latency
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Requests
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Version
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Last Heartbeat
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {(edgeNodes && Array.isArray(edgeNodes) ? edgeNodes : []).map((node) => (
                  <tr key={node.id}>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">{node.hostname}</div>
                      <div className="text-sm text-gray-500">{node.ip_address}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {node.location || node.region}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(node.status)}`}>
                        {node.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`text-sm font-medium ${getHealthColor(node.health_score)}`}>
                        {node.health_score}%
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {node.avg_response_time}ms
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {node.total_requests.toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {node.version}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {formatTimestamp(node.last_heartbeat)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
