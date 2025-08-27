'use client';

import { PlusIcon, CogIcon, ShieldCheckIcon, ExclamationTriangleIcon, ClockIcon, BuildingOfficeIcon } from '@heroicons/react/24/outline';
import { useDomains } from '@/lib/hooks';
import { LoadingCard, ErrorCard } from '@/components/ui/Loading';
import { useState } from 'react';
import DomainConfigModal from '@/components/domains/DomainConfigModal';
import AddDomainModal from '@/components/domains/AddDomainModal';
import { Domain, DomainFormData } from '@/lib/types';
import { useOrganization } from '@/components/providers/OrganizationProvider';

export default function DomainsPage() {
  const { organization } = useOrganization();
  const { domains, isLoading, isError, mutate } = useDomains(organization?.slug);
  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);

  const handleAddDomain = async (domainData: DomainFormData) => {
    try {
      // Add domain via API (the apiClient will handle the request)
      // For now, just simulate the API call since the endpoint may not exist yet
      console.log('Would add domain:', domainData);
      setShowAddModal(false);
      // Refresh the domains list
      mutate();
    } catch (error) {
      console.error('Failed to add domain:', error);
      // Show error notification in production
    }
  };

  const getSSLStatusIcon = (sslStatus: string) => {
    switch (sslStatus) {
      case 'valid':
        return <ShieldCheckIcon className="h-4 w-4 text-green-500" />;
      case 'expiring':
        return <ClockIcon className="h-4 w-4 text-yellow-500" />;
      case 'invalid':
      case 'expired':
        return <ExclamationTriangleIcon className="h-4 w-4 text-red-500" />;
      default:
        return <ShieldCheckIcon className="h-4 w-4 text-gray-400" />;
    }
  };

  const getSSLStatusText = (domain: Domain): string => {
    if (!domain.ssl_enabled) return 'Disabled';
    if (domain.ssl_certificate?.status === 'valid') {
      const expiryDate = new Date(domain.ssl_certificate.expires_at);
      const daysUntilExpiry = Math.ceil((expiryDate.getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24));
      if (daysUntilExpiry <= 30) return `Expires in ${daysUntilExpiry} days`;
      return 'Valid';
    }
    return domain.ssl_certificate?.status || 'Unknown';
  };

  const getSSLStatusColor = (domain: Domain): string => {
    if (!domain.ssl_enabled) return 'bg-gray-100 text-gray-800';
    switch (domain.ssl_certificate?.status) {
      case 'valid':
        const expiryDate = new Date(domain.ssl_certificate.expires_at);
        const daysUntilExpiry = Math.ceil((expiryDate.getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24));
        if (daysUntilExpiry <= 30) return 'bg-yellow-100 text-yellow-800';
        return 'bg-green-100 text-green-800';
      case 'expired':
      case 'invalid':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  // Show organization selection prompt if no organization is selected
  if (!organization) {
    return (
      <div className="space-y-6">
        <div className="text-center py-12">
          <BuildingOfficeIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-lg font-medium text-gray-900">No Organization Selected</h3>
          <p className="mt-1 text-sm text-gray-500">
            Please select an organization from the header to view and manage domains.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Domain Management</h1>
          <p className="text-sm text-gray-600 mt-1">
            Manage your domains, SSL certificates, and CDN configurations
          </p>
        </div>
        <div className="flex space-x-3">
          <button
            type="button"
            className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <CogIcon className="-ml-1 mr-2 h-5 w-5" />
            Bulk Actions
          </button>
          <button
            type="button"
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            onClick={() => setShowAddModal(true)}
          >
            <PlusIcon className="-ml-1 mr-2 h-5 w-5" />
            Add Domain
          </button>
        </div>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <div className="w-8 h-8 bg-blue-100 rounded-lg flex items-center justify-center">
                <ShieldCheckIcon className="h-5 w-5 text-blue-600" />
              </div>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-900">Total Domains</p>
              <p className="text-lg font-semibold text-gray-600">
                {(domains && Array.isArray(domains) ? domains.length : 0)}
              </p>
            </div>
          </div>
        </div>
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <div className="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center">
                <ShieldCheckIcon className="h-5 w-5 text-green-600" />
              </div>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-900">SSL Enabled</p>
              <p className="text-lg font-semibold text-gray-600">
                {(domains && Array.isArray(domains) ? domains.filter(d => d.ssl_enabled).length : 0)}
              </p>
            </div>
          </div>
        </div>
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <div className="w-8 h-8 bg-yellow-100 rounded-lg flex items-center justify-center">
                <ClockIcon className="h-5 w-5 text-yellow-600" />
              </div>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-900">Expiring Soon</p>
              <p className="text-lg font-semibold text-gray-600">
                {(domains && Array.isArray(domains) ? domains.filter(d => {
                  if (!d.ssl_certificate?.expires_at) return false;
                  const daysUntilExpiry = Math.ceil((new Date(d.ssl_certificate.expires_at).getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24));
                  return daysUntilExpiry <= 30 && daysUntilExpiry > 0;
                }).length : 0)}
              </p>
            </div>
          </div>
        </div>
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <div className="flex-shrink-0">
              <div className="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center">
                <div className="w-3 h-3 bg-green-500 rounded-full"></div>
              </div>
            </div>
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-900">Active</p>
              <p className="text-lg font-semibold text-gray-600">
                {(domains && Array.isArray(domains) ? domains.filter(d => d.enabled).length : 0)}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Domains Table */}
      {isLoading ? (
        <LoadingCard title="Loading domains..." description="Fetching your domain configurations" />
      ) : isError ? (
        <ErrorCard 
          title="Failed to load domains" 
          description="Unable to fetch domain data. Please try again."
          onRetry={() => mutate()}
        />
      ) : (
        <div className="bg-white shadow border border-gray-200 rounded-lg overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-medium text-gray-900">Domain Configurations</h3>
            <p className="text-sm text-gray-500 mt-1">
              Monitor SSL certificates, origins, and caching policies
            </p>
          </div>
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Domain
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Origin Server
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    SSL Certificate
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Cache Policy
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Performance
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {(domains && Array.isArray(domains) ? domains : []).map((domain) => (
                  <tr key={domain.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div>
                          <div className="text-sm font-medium text-gray-900">{domain.domain}</div>
                          <div className="text-sm text-gray-500">
                            Added {new Date(domain.created_at).toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className={`flex-shrink-0 w-2 h-2 rounded-full mr-2 ${
                          domain.enabled ? 'bg-green-400' : 'bg-red-400'
                        }`}></div>
                        <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
                          domain.enabled 
                            ? 'bg-green-100 text-green-800' 
                            : 'bg-red-100 text-red-800'
                        }`}>
                          {domain.enabled ? 'Active' : 'Disabled'}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">{domain.origin}</div>
                      <div className="text-sm text-gray-500">
                        Health: <span className="text-green-600">Good</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        {getSSLStatusIcon(domain.ssl_certificate?.status || 'unknown')}
                        <span className={`ml-2 inline-flex px-2 py-1 text-xs font-medium rounded-full ${getSSLStatusColor(domain)}`}>
                          {getSSLStatusText(domain)}
                        </span>
                      </div>
                      {domain.ssl_certificate?.issuer && (
                        <div className="text-xs text-gray-500 mt-1">
                          {domain.ssl_certificate.issuer}
                        </div>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">TTL: {domain.cache_ttl}s</div>
                      <div className="text-sm text-gray-500">
                        Hit Rate: <span className="text-green-600">94.2%</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">
                        <span className="text-green-600">98ms</span> avg
                      </div>
                      <div className="text-sm text-gray-500">
                        {domain.bandwidth_usage ? `${(domain.bandwidth_usage / 1024 / 1024).toFixed(1)}MB` : '0MB'} today
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex justify-end space-x-2">
                        <button 
                          className="text-blue-600 hover:text-blue-900"
                          onClick={() => setSelectedDomain(domain)}
                        >
                          Configure
                        </button>
                        <button className="text-green-600 hover:text-green-900">
                          Analytics
                        </button>
                        <button className="text-red-600 hover:text-red-900">
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          
          {(!domains || !Array.isArray(domains) || domains.length === 0) && (
            <div className="text-center py-12">
              <ShieldCheckIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No domains configured</h3>
              <p className="mt-1 text-sm text-gray-500">
                Get started by adding your first domain to the CDN.
              </p>
              <div className="mt-6">
                <button
                  type="button"
                  className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                  onClick={() => setShowAddModal(true)}
                >
                  <PlusIcon className="-ml-1 mr-2 h-5 w-5" />
                  Add your first domain
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Domain Configuration Modal */}
      <DomainConfigModal
        domain={selectedDomain}
        isOpen={selectedDomain !== null}
        onClose={() => setSelectedDomain(null)}
      />

      {/* Add Domain Modal */}
      <AddDomainModal
        isOpen={showAddModal}
        onClose={() => setShowAddModal(false)}
        onSubmit={handleAddDomain}
      />
    </div>
  );
}
