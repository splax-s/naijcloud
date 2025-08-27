'use client';

import { useState, useEffect } from 'react';
import { useSession } from 'next-auth/react';
import { 
  PlusIcon, 
  KeyIcon, 
  ClipboardIcon, 
  TrashIcon
} from '@heroicons/react/24/outline';

interface APIKey {
  id: string;
  name: string;
  key_prefix: string;
  scopes: string[];
  rate_limit: number;
  last_used_at: string | null;
  created_at: string;
}

export default function APIKeysPage() {
  const { data: session } = useSession();
  const [apiKeys, setApiKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newKeyData, setNewKeyData] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    scopes: ['domains:read'] as string[],
    rate_limit: 1000,
  });

  const availableScopes = [
    { value: 'domains:read', label: 'Read Domains' },
    { value: 'domains:write', label: 'Write Domains' },
    { value: 'edges:read', label: 'Read Edge Nodes' },
    { value: 'edges:write', label: 'Write Edge Nodes' },
    { value: 'analytics:read', label: 'Read Analytics' },
    { value: 'cache:purge', label: 'Purge Cache' },
  ];

  useEffect(() => {
    if (session?.user) {
      fetchAPIKeys();
    }
  }, [session]);

  const fetchAPIKeys = async () => {
    try {
      setLoading(true);
      // This would be the actual API call to fetch API keys
      // For now, we'll simulate with demo data
      const demoKeys: APIKey[] = [
        {
          id: '1',
          name: 'Production API Key',
          key_prefix: 'nj_prod_',
          scopes: ['domains:read', 'domains:write', 'cache:purge'],
          rate_limit: 5000,
          last_used_at: '2025-08-26T18:30:00Z',
          created_at: '2025-01-15T10:00:00Z',
        },
        {
          id: '2',
          name: 'Development Key',
          key_prefix: 'nj_dev_',
          scopes: ['domains:read'],
          rate_limit: 1000,
          last_used_at: null,
          created_at: '2025-08-20T14:20:00Z',
        },
      ];
      setApiKeys(demoKeys);
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAPIKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setCreating(true);

    try {
      // This would be the actual API call to create an API key
      // For now, we'll simulate the response
      const response = {
        api_key: {
          id: Date.now().toString(),
          name: formData.name,
          key_prefix: 'nj_new_',
          scopes: formData.scopes,
          rate_limit: formData.rate_limit,
          last_used_at: null,
          created_at: new Date().toISOString(),
        },
        plain_key: `nj_new_${Math.random().toString(36).substring(2, 15)}${Math.random().toString(36).substring(2, 15)}`,
      };

      setNewKeyData(response.plain_key);
      setApiKeys([...apiKeys, response.api_key]);
      setShowCreateForm(false);
      setFormData({ name: '', scopes: ['domains:read'], rate_limit: 1000 });
    } catch (error) {
      console.error('Failed to create API key:', error);
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteAPIKey = async (keyId: string) => {
    if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
      return;
    }

    try {
      // This would be the actual API call to delete the key
      setApiKeys(apiKeys.filter(key => key.id !== keyId));
    } catch (error) {
      console.error('Failed to delete API key:', error);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    // You could add a toast notification here
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-1/2"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">API Keys</h1>
          <p className="text-sm text-gray-600 mt-1">
            Manage API keys for programmatic access to your CDN infrastructure
          </p>
        </div>
        <button
          onClick={() => setShowCreateForm(true)}
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          <PlusIcon className="w-4 h-4 mr-2" />
          Create API Key
        </button>
      </div>

      {/* New Key Display */}
      {newKeyData && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <div className="flex items-start">
            <KeyIcon className="w-5 h-5 text-green-500 mt-0.5 mr-3" />
            <div className="flex-1">
              <h3 className="text-sm font-medium text-green-800">
                API Key Created Successfully
              </h3>
              <p className="text-sm text-green-700 mt-1">
                Please copy your API key now. You won&apos;t be able to see it again!
              </p>
              <div className="mt-3 p-3 bg-white border border-green-200 rounded-md font-mono text-sm">
                {newKeyData}
              </div>
              <div className="mt-3 flex space-x-3">
                <button
                  onClick={() => copyToClipboard(newKeyData)}
                  className="inline-flex items-center px-3 py-1 border border-green-300 text-sm font-medium rounded-md text-green-700 bg-white hover:bg-green-50"
                >
                  <ClipboardIcon className="w-4 h-4 mr-1" />
                  Copy Key
                </button>
                <button
                  onClick={() => setNewKeyData(null)}
                  className="text-sm text-green-600 hover:text-green-700"
                >
                  Dismiss
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Create Form */}
      {showCreateForm && (
        <div className="bg-white shadow rounded-lg border border-gray-200">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-medium text-gray-900">Create New API Key</h3>
          </div>
          <form onSubmit={handleCreateAPIKey} className="px-6 py-4 space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Name
              </label>
              <input
                type="text"
                required
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                placeholder="e.g., Production API Key"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">
                Scopes
              </label>
              <div className="mt-2 space-y-2">
                {availableScopes.map((scope) => (
                  <label key={scope.value} className="inline-flex items-center mr-6">
                    <input
                      type="checkbox"
                      checked={formData.scopes.includes(scope.value)}
                      onChange={(e) => {
                        if (e.target.checked) {
                          setFormData({
                            ...formData,
                            scopes: [...formData.scopes, scope.value],
                          });
                        } else {
                          setFormData({
                            ...formData,
                            scopes: formData.scopes.filter(s => s !== scope.value),
                          });
                        }
                      }}
                      className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                    />
                    <span className="ml-2 text-sm text-gray-700">{scope.label}</span>
                  </label>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">
                Rate Limit (requests per hour)
              </label>
              <input
                type="number"
                min="100"
                max="10000"
                step="100"
                value={formData.rate_limit}
                onChange={(e) => setFormData({ ...formData, rate_limit: parseInt(e.target.value) })}
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
              />
            </div>

            <div className="flex justify-end space-x-3">
              <button
                type="button"
                onClick={() => setShowCreateForm(false)}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={creating}
                className="px-4 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {creating ? 'Creating...' : 'Create API Key'}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* API Keys List */}
      <div className="bg-white shadow rounded-lg border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Your API Keys</h3>
        </div>
        <div className="divide-y divide-gray-200">
          {apiKeys.length === 0 ? (
            <div className="px-6 py-8 text-center">
              <KeyIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No API keys</h3>
              <p className="mt-1 text-sm text-gray-500">
                Get started by creating a new API key.
              </p>
            </div>
          ) : (
            apiKeys.map((apiKey) => (
              <div key={apiKey.id} className="px-6 py-4">
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center">
                      <h4 className="text-sm font-medium text-gray-900">{apiKey.name}</h4>
                      <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                        {apiKey.key_prefix}••••
                      </span>
                    </div>
                    <div className="mt-1 flex items-center space-x-4 text-sm text-gray-500">
                      <span>Scopes: {apiKey.scopes.join(', ')}</span>
                      <span>Rate limit: {apiKey.rate_limit.toLocaleString()}/hour</span>
                    </div>
                    <div className="mt-1 text-xs text-gray-500">
                      Created {formatDate(apiKey.created_at)}
                      {apiKey.last_used_at && (
                        <span> • Last used {formatDate(apiKey.last_used_at)}</span>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => copyToClipboard(apiKey.key_prefix)}
                      className="p-2 text-gray-400 hover:text-gray-600"
                      title="Copy key prefix"
                    >
                      <ClipboardIcon className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => handleDeleteAPIKey(apiKey.id)}
                      className="p-2 text-red-400 hover:text-red-600"
                      title="Delete API key"
                    >
                      <TrashIcon className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Usage Information */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h4 className="text-sm font-medium text-blue-800">API Usage</h4>
        <div className="mt-2 text-sm text-blue-700">
          <p>Use your API keys to authenticate requests to our REST API:</p>
          <code className="mt-2 block p-2 bg-white border border-blue-200 rounded text-xs">
            curl -H &quot;Authorization: Bearer YOUR_API_KEY&quot; https://api.naijcloud.com/v1/domains
          </code>
        </div>
      </div>
    </div>
  );
}
