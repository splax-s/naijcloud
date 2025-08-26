import { TrashIcon, ClockIcon, CircleStackIcon } from '@heroicons/react/24/outline';

const cacheEntries = [
  {
    id: '1',
    path: '/api/v1/users',
    domain: 'api.example.com',
    size: '2.4 MB',
    hits: 1250,
    lastAccessed: '2 minutes ago',
    expires: 'in 4 hours',
    type: 'API Response',
  },
  {
    id: '2',
    path: '/static/css/main.css',
    domain: 'static.example.com',
    size: '125 KB',
    hits: 8900,
    lastAccessed: '30 seconds ago',
    expires: 'in 23 hours',
    type: 'Static Asset',
  },
  {
    id: '3',
    path: '/images/hero-banner.jpg',
    domain: 'cdn.example.com',
    size: '850 KB',
    hits: 3400,
    lastAccessed: '1 minute ago',
    expires: 'in 6 days',
    type: 'Image',
  },
  {
    id: '4',
    path: '/api/v1/products',
    domain: 'api.example.com',
    size: '1.8 MB',
    hits: 890,
    lastAccessed: '5 minutes ago',
    expires: 'in 2 hours',
    type: 'API Response',
  },
];

const purgeHistory = [
  {
    id: '1',
    path: '/api/v1/inventory',
    domain: 'api.example.com',
    timestamp: '10 minutes ago',
    reason: 'Content Update',
    user: 'admin@example.com',
  },
  {
    id: '2',
    path: '/static/js/*',
    domain: 'static.example.com',
    timestamp: '1 hour ago',
    reason: 'Deployment',
    user: 'deploy@example.com',
  },
  {
    id: '3',
    path: '/images/product-*',
    domain: 'cdn.example.com',
    timestamp: '3 hours ago',
    reason: 'Manual Purge',
    user: 'admin@example.com',
  },
];

export default function CacheManagementPage() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Cache Management</h1>
          <p className="text-sm text-gray-600 mt-1">
            Monitor cache performance and manage cache entries
          </p>
        </div>
        <button
          type="button"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-red-600 hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-red-500"
        >
          <TrashIcon className="-ml-1 mr-2 h-5 w-5" />
          Purge Cache
        </button>
      </div>

      {/* Cache Stats */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <div className="bg-white overflow-hidden shadow rounded-lg border border-gray-200">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <CircleStackIcon className="h-8 w-8 text-blue-600" />
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">
                    Cache Size
                  </dt>
                  <dd className="text-2xl font-semibold text-gray-900">
                    2.3 GB
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
                <ClockIcon className="h-8 w-8 text-green-600" />
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">
                    Hit Ratio
                  </dt>
                  <dd className="text-2xl font-semibold text-gray-900">
                    94.2%
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
                <TrashIcon className="h-8 w-8 text-purple-600" />
              </div>
              <div className="ml-5 w-0 flex-1">
                <dl>
                  <dt className="text-sm font-medium text-gray-500 truncate">
                    Entries
                  </dt>
                  <dd className="text-2xl font-semibold text-gray-900">
                    {cacheEntries.length.toLocaleString()}
                  </dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Cache Entries */}
      <div className="bg-white shadow border border-gray-200 rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Active Cache Entries</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Path
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Domain
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Size
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Hits
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Last Accessed
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Expires
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {cacheEntries.map((entry) => (
                <tr key={entry.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">{entry.path}</div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {entry.domain}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-800">
                      {entry.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {entry.size}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {entry.hits.toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {entry.lastAccessed}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {entry.expires}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button className="text-red-600 hover:text-red-900">Purge</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Purge History */}
      <div className="bg-white shadow border border-gray-200 rounded-lg overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Recent Purge History</h3>
        </div>
        <div className="divide-y divide-gray-200">
          {purgeHistory.map((purge) => (
            <div key={purge.id} className="px-6 py-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm font-medium text-gray-900">
                    {purge.path} on {purge.domain}
                  </div>
                  <div className="text-sm text-gray-500">
                    {purge.reason} • by {purge.user}
                  </div>
                </div>
                <div className="text-sm text-gray-500">
                  {purge.timestamp}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
