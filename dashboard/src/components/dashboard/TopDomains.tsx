'use client';

import { GlobeAltIcon } from '@heroicons/react/24/outline';
import { useTopDomains } from '@/lib/hooks';
import { LoadingCard, ErrorCard } from '@/components/ui/Loading';

export function TopDomains() {
  const { topDomains, isLoading, isError, mutate } = useTopDomains(5);

  if (isLoading) {
    return (
      <LoadingCard 
        title="Loading top domains..." 
        description="Fetching your highest traffic domains"
      />
    );
  }

  if (isError) {
    return (
      <ErrorCard 
        title="Failed to load domain statistics" 
        description="Unable to fetch top domains data. Please try again."
        onRetry={() => mutate()}
      />
    );
  }

  const domains = topDomains || [];
  return (
    <div className="bg-white shadow rounded-lg border border-gray-200">
      <div className="px-6 py-4 border-b border-gray-200">
        <h3 className="text-lg font-medium text-gray-900">Top Domains</h3>
        <p className="text-sm text-gray-500 mt-1">By request volume</p>
      </div>
      <div className="divide-y divide-gray-200">
        {domains.map((domainData, index) => (
          <div key={domainData.domain} className="px-6 py-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-3">
                <div className="flex-shrink-0 w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                  <span className="text-sm font-medium text-blue-600">
                    {index + 1}
                  </span>
                </div>
                <div className="flex items-center space-x-2">
                  <GlobeAltIcon className="w-4 h-4 text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900">
                      {domainData.domain}
                    </p>
                    <p className="text-xs text-gray-500">
                      {domainData.requests.toLocaleString()} requests
                    </p>
                  </div>
                </div>
              </div>
              <div className="flex items-center space-x-4">
                <div className="text-right">
                  <p className="text-sm font-medium text-gray-900">
                    {(domainData.bandwidth / (1024 * 1024 * 1024)).toFixed(1)} GB
                  </p>
                  <p className="text-xs text-gray-500">
                    {(domainData.cache_hit_ratio * 100).toFixed(1)}% cache hit
                  </p>
                </div>
                <div className="flex-shrink-0">
                  <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800">
                    Active
                  </span>
                </div>
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
          View all domains →
        </a>
      </div>
    </div>
  );
}
