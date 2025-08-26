import { 
  UserIcon, 
  BellIcon, 
  ShieldCheckIcon, 
  GlobeAltIcon,
  KeyIcon,
  CogIcon 
} from '@heroicons/react/24/outline';

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">Settings</h1>
        <p className="text-sm text-gray-600 mt-1">
          Manage your account and CDN configuration
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Settings Navigation */}
        <div className="lg:col-span-1">
          <nav className="space-y-1">
            <a
              href="#profile"
              className="bg-blue-50 border-blue-500 text-blue-700 hover:bg-blue-50 hover:text-blue-700 group border-l-4 px-3 py-2 flex items-center text-sm font-medium"
            >
              <UserIcon className="text-blue-500 mr-3 h-6 w-6" />
              Profile
            </a>
            <a
              href="#notifications"
              className="border-transparent text-gray-900 hover:bg-gray-50 hover:text-gray-900 group border-l-4 px-3 py-2 flex items-center text-sm font-medium"
            >
              <BellIcon className="text-gray-400 group-hover:text-gray-500 mr-3 h-6 w-6" />
              Notifications
            </a>
            <a
              href="#security"
              className="border-transparent text-gray-900 hover:bg-gray-50 hover:text-gray-900 group border-l-4 px-3 py-2 flex items-center text-sm font-medium"
            >
              <ShieldCheckIcon className="text-gray-400 group-hover:text-gray-500 mr-3 h-6 w-6" />
              Security
            </a>
            <a
              href="#api-keys"
              className="border-transparent text-gray-900 hover:bg-gray-50 hover:text-gray-900 group border-l-4 px-3 py-2 flex items-center text-sm font-medium"
            >
              <KeyIcon className="text-gray-400 group-hover:text-gray-500 mr-3 h-6 w-6" />
              API Keys
            </a>
            <a
              href="#cdn-config"
              className="border-transparent text-gray-900 hover:bg-gray-50 hover:text-gray-900 group border-l-4 px-3 py-2 flex items-center text-sm font-medium"
            >
              <GlobeAltIcon className="text-gray-400 group-hover:text-gray-500 mr-3 h-6 w-6" />
              CDN Configuration
            </a>
            <a
              href="#advanced"
              className="border-transparent text-gray-900 hover:bg-gray-50 hover:text-gray-900 group border-l-4 px-3 py-2 flex items-center text-sm font-medium"
            >
              <CogIcon className="text-gray-400 group-hover:text-gray-500 mr-3 h-6 w-6" />
              Advanced
            </a>
          </nav>
        </div>

        {/* Settings Content */}
        <div className="lg:col-span-2">
          <div className="space-y-6">
            {/* Profile Section */}
            <div id="profile" className="bg-white shadow rounded-lg border border-gray-200">
              <div className="px-6 py-4 border-b border-gray-200">
                <h3 className="text-lg font-medium text-gray-900">Profile Information</h3>
                <p className="text-sm text-gray-500 mt-1">
                  Update your account profile information and email address.
                </p>
              </div>
              <div className="px-6 py-4 space-y-4">
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium text-gray-700">
                      First Name
                    </label>
                    <input
                      type="text"
                      className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                      defaultValue="John"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-gray-700">
                      Last Name
                    </label>
                    <input
                      type="text"
                      className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                      defaultValue="Doe"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Email Address
                  </label>
                  <input
                    type="email"
                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    defaultValue="john@example.com"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Company
                  </label>
                  <input
                    type="text"
                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    defaultValue="Example Corp"
                  />
                </div>
                <div className="flex justify-end">
                  <button
                    type="button"
                    className="bg-blue-600 border border-transparent rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  >
                    Save Changes
                  </button>
                </div>
              </div>
            </div>

            {/* CDN Configuration Section */}
            <div id="cdn-config" className="bg-white shadow rounded-lg border border-gray-200">
              <div className="px-6 py-4 border-b border-gray-200">
                <h3 className="text-lg font-medium text-gray-900">CDN Configuration</h3>
                <p className="text-sm text-gray-500 mt-1">
                  Global settings for your CDN infrastructure.
                </p>
              </div>
              <div className="px-6 py-4 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Default Cache TTL (seconds)
                  </label>
                  <input
                    type="number"
                    className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                    defaultValue="3600"
                  />
                  <p className="mt-1 text-sm text-gray-500">
                    Default time-to-live for cached content (1 hour = 3600 seconds)
                  </p>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Gzip Compression
                  </label>
                  <div className="mt-1">
                    <label className="inline-flex items-center">
                      <input
                        type="checkbox"
                        className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                        defaultChecked
                      />
                      <span className="ml-2 text-sm text-gray-700">
                        Enable Gzip compression for text-based content
                      </span>
                    </label>
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    HTTP/2 Support
                  </label>
                  <div className="mt-1">
                    <label className="inline-flex items-center">
                      <input
                        type="checkbox"
                        className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                        defaultChecked
                      />
                      <span className="ml-2 text-sm text-gray-700">
                        Enable HTTP/2 protocol support
                      </span>
                    </label>
                  </div>
                </div>
                <div className="flex justify-end">
                  <button
                    type="button"
                    className="bg-blue-600 border border-transparent rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  >
                    Update Configuration
                  </button>
                </div>
              </div>
            </div>

            {/* API Keys Section */}
            <div id="api-keys" className="bg-white shadow rounded-lg border border-gray-200">
              <div className="px-6 py-4 border-b border-gray-200">
                <h3 className="text-lg font-medium text-gray-900">API Keys</h3>
                <p className="text-sm text-gray-500 mt-1">
                  Manage API keys for programmatic access to your CDN.
                </p>
              </div>
              <div className="px-6 py-4">
                <div className="space-y-4">
                  <div className="flex justify-between items-center p-4 border border-gray-200 rounded-lg">
                    <div>
                      <div className="text-sm font-medium text-gray-900">Production API Key</div>
                      <div className="text-sm text-gray-500">nj_prod_****************************abc123</div>
                      <div className="text-xs text-gray-500 mt-1">Created on Jan 15, 2024 • Last used 2 hours ago</div>
                    </div>
                    <div className="flex space-x-2">
                      <button className="text-blue-600 hover:text-blue-700 text-sm">Copy</button>
                      <button className="text-red-600 hover:text-red-700 text-sm">Revoke</button>
                    </div>
                  </div>
                  <div className="flex justify-end">
                    <button
                      type="button"
                      className="bg-blue-600 border border-transparent rounded-md shadow-sm py-2 px-4 inline-flex justify-center text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    >
                      Generate New Key
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
