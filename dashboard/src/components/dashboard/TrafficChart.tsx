'use client';

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import { useTrafficData } from '@/lib/hooks';
import { LoadingCard, ErrorCard } from '@/components/ui/Loading';
import { useOrganization } from '@/components/providers/OrganizationProvider';

export function TrafficChart() {
  const { organization } = useOrganization();
  const { trafficData, isLoading, isError, mutate } = useTrafficData(24, organization?.slug);

  if (isLoading) {
    return (
      <LoadingCard 
        title="Loading traffic data..." 
        description="Fetching the latest 24 hours of traffic metrics"
      />
    );
  }

  if (isError) {
    return (
      <ErrorCard 
        title="Failed to load traffic data" 
        description="Unable to fetch traffic metrics. Please try again."
        onRetry={() => mutate()}
      />
    );
  }

  // Transform the data for the chart
  const chartData = trafficData?.map(item => ({
    time: new Date(item.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    requests: item.requests,
    bandwidth: Math.round(item.bandwidth / (1024 * 1024)), // Convert to MB
  })) || [];
  return (
    <div className="bg-white shadow rounded-lg border border-gray-200">
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="text-lg font-medium text-gray-900">Traffic Overview</h3>
        <p className="text-sm text-gray-500 mt-1">Last 24 hours</p>
      </div>
      <div className="p-6">
        <div className="h-80">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis 
                dataKey="time" 
                stroke="#6B7280"
                fontSize={12}
              />
              <YAxis 
                yAxisId="requests"
                orientation="left"
                stroke="#3B82F6"
                fontSize={12}
              />
              <YAxis 
                yAxisId="bandwidth"
                orientation="right"
                stroke="#10B981"
                fontSize={12}
              />
              <Tooltip 
                contentStyle={{
                  backgroundColor: '#FFFFFF',
                  border: '1px solid #E5E7EB',
                  borderRadius: '8px',
                }}
              />
              <Line
                yAxisId="requests"
                type="monotone"
                dataKey="requests"
                stroke="#3B82F6"
                strokeWidth={2}
                dot={false}
                name="Requests/hour"
              />
              <Line
                yAxisId="bandwidth"
                type="monotone"
                dataKey="bandwidth"
                stroke="#10B981"
                strokeWidth={2}
                dot={false}
                name="Bandwidth (MB/s)"
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
        <div className="mt-4 flex items-center justify-center space-x-6">
          <div className="flex items-center">
            <div className="w-3 h-3 bg-blue-500 rounded-full mr-2"></div>
            <span className="text-sm text-gray-600">Requests/hour</span>
          </div>
          <div className="flex items-center">
            <div className="w-3 h-3 bg-green-500 rounded-full mr-2"></div>
            <span className="text-sm text-gray-600">Bandwidth (MB/s)</span>
          </div>
        </div>
      </div>
    </div>
  );
}
