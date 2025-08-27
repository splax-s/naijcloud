'use client';

import { Fragment, useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { 
  XMarkIcon, 
  ShieldCheckIcon, 
  GlobeAltIcon, 
  CogIcon,
  ChartBarIcon,
  ClockIcon,
  ExclamationTriangleIcon 
} from '@heroicons/react/24/outline';
import { DomainModalProps } from '@/lib/types';

function classNames(...classes: string[]): string {
  return classes.filter(Boolean).join(' ');
}

export default function DomainConfigModal({ domain, isOpen, onClose }: DomainModalProps) {
  const [activeTab, setActiveTab] = useState(0);
  const [formData, setFormData] = useState({
    origin: domain?.origin || '',
    cache_ttl: domain?.cache_ttl || 3600,
    ssl_enabled: domain?.ssl_enabled || false,
    compression_enabled: domain?.compression_enabled !== undefined ? domain.compression_enabled : true,
    security_level: domain?.security_level || ('medium' as const),
    custom_headers: domain?.custom_headers || {},
  });

  const tabs = [
    { name: 'General', icon: CogIcon },
    { name: 'SSL/TLS', icon: ShieldCheckIcon },
    { name: 'Caching', icon: ClockIcon },
    { name: 'Security', icon: ExclamationTriangleIcon },
    { name: 'Analytics', icon: ChartBarIcon },
  ];

  const handleSave = async () => {
    if (!domain) return;
    
    try {
      // Save domain configuration
      // For now, simulate the API call since the endpoint may not exist yet
      console.log('Saving domain configuration for:', domain.domain, formData);
      
      // In production, this would be:
      // await apiClient.put(`/domains/${domain.id}/config`, formData);
      
      onClose();
    } catch (error) {
      console.error('Failed to save domain configuration:', error);
      // Show error notification in production
    }
  };

  if (!domain) return null;

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/10" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-4xl transform overflow-hidden rounded-2xl bg-white text-left align-middle shadow-xl transition-all">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
                  <div className="flex items-center">
                    <GlobeAltIcon className="h-6 w-6 text-blue-500 mr-3" />
                    <div>
                      <Dialog.Title as="h3" className="text-lg font-medium text-gray-900">
                        Configure {domain.domain}
                      </Dialog.Title>
                      <p className="text-sm text-gray-500">
                        Manage SSL, caching, security, and performance settings
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    className="rounded-md text-gray-400 hover:text-gray-500 focus:outline-none"
                    onClick={onClose}
                  >
                    <XMarkIcon className="h-6 w-6" />
                  </button>
                </div>

                <div className="flex">
                  {/* Tab Navigation */}
                  <div className="w-48 border-r border-gray-200 bg-gray-50">
                    <nav className="space-y-1 p-4">
                      {tabs.map((tab, index) => (
                        <button
                          key={tab.name}
                          onClick={() => setActiveTab(index)}
                          className={classNames(
                            activeTab === index
                              ? 'bg-blue-100 text-blue-700 border-blue-500'
                              : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100 border-transparent',
                            'group flex items-center px-3 py-2 text-sm font-medium border-l-4 w-full text-left'
                          )}
                        >
                          <tab.icon className="mr-3 h-5 w-5" />
                          {tab.name}
                        </button>
                      ))}
                    </nav>
                  </div>

                  {/* Tab Content */}
                  <div className="flex-1 p-6">
                    {activeTab === 0 && (
                      <div className="space-y-6">
                        <h4 className="text-lg font-medium text-gray-900">General Settings</h4>
                        <div className="grid grid-cols-1 gap-6">
                          <div>
                            <label className="block text-sm font-medium text-gray-700">
                              Origin Server
                            </label>
                            <input
                              type="url"
                              value={formData.origin}
                              onChange={(e) => setFormData({ ...formData, origin: e.target.value })}
                              className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                              placeholder="https://your-origin.com"
                            />
                          </div>
                          <div>
                            <label className="flex items-center">
                              <input
                                type="checkbox"
                                checked={formData.compression_enabled}
                                onChange={(e) => setFormData({ ...formData, compression_enabled: e.target.checked })}
                                className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                              />
                              <span className="ml-2 text-sm text-gray-700">Enable compression (gzip/brotli)</span>
                            </label>
                          </div>
                        </div>
                      </div>
                    )}

                    {activeTab === 1 && (
                      <div className="space-y-6">
                        <h4 className="text-lg font-medium text-gray-900">SSL/TLS Configuration</h4>
                        <div className="space-y-4">
                          <div>
                            <label className="flex items-center">
                              <input
                                type="checkbox"
                                checked={formData.ssl_enabled}
                                onChange={(e) => setFormData({ ...formData, ssl_enabled: e.target.checked })}
                                className="rounded border-gray-300 text-blue-600 shadow-sm focus:border-blue-300 focus:ring focus:ring-blue-200 focus:ring-opacity-50"
                              />
                              <span className="ml-2 text-sm text-gray-700">Enable SSL/TLS</span>
                            </label>
                          </div>
                          
                          {domain.ssl_certificate && (
                            <div className="bg-gray-50 p-4 rounded-lg">
                              <h5 className="font-medium text-gray-900 mb-2">Current Certificate</h5>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Issuer:</span>
                                  <span className="text-gray-900">{domain.ssl_certificate.issuer}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Expires:</span>
                                  <span className="text-gray-900">
                                    {new Date(domain.ssl_certificate.expires_at).toLocaleDateString()}
                                  </span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-600">Status:</span>
                                  <span className={`px-2 py-1 text-xs rounded-full ${
                                    domain.ssl_certificate.status === 'valid' 
                                      ? 'bg-green-100 text-green-800'
                                      : 'bg-red-100 text-red-800'
                                  }`}>
                                    {domain.ssl_certificate.status}
                                  </span>
                                </div>
                              </div>
                              <button className="mt-3 text-blue-600 text-sm hover:text-blue-800">
                                Renew Certificate
                              </button>
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {activeTab === 2 && (
                      <div className="space-y-6">
                        <h4 className="text-lg font-medium text-gray-900">Caching Configuration</h4>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-700">
                              Cache TTL (seconds)
                            </label>
                            <select
                              value={formData.cache_ttl}
                              onChange={(e) => setFormData({ ...formData, cache_ttl: parseInt(e.target.value) })}
                              className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                            >
                              <option value={300}>5 minutes</option>
                              <option value={900}>15 minutes</option>
                              <option value={1800}>30 minutes</option>
                              <option value={3600}>1 hour</option>
                              <option value={7200}>2 hours</option>
                              <option value={14400}>4 hours</option>
                              <option value={86400}>24 hours</option>
                            </select>
                          </div>
                          
                          <div className="bg-blue-50 p-4 rounded-lg">
                            <h5 className="font-medium text-blue-900 mb-2">Cache Performance</h5>
                            <div className="grid grid-cols-2 gap-4 text-sm">
                              <div>
                                <span className="text-blue-700">Hit Rate:</span>
                                <div className="text-lg font-semibold text-blue-900">94.2%</div>
                              </div>
                              <div>
                                <span className="text-blue-700">Bandwidth Saved:</span>
                                <div className="text-lg font-semibold text-blue-900">1.2 TB</div>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    )}

                    {activeTab === 3 && (
                      <div className="space-y-6">
                        <h4 className="text-lg font-medium text-gray-900">Security Settings</h4>
                        <div className="space-y-4">
                          <div>
                            <label className="block text-sm font-medium text-gray-700">
                              Security Level
                            </label>
                            <select
                              value={formData.security_level}
                              onChange={(e) => setFormData({ ...formData, security_level: e.target.value as 'off' | 'low' | 'medium' | 'high' | 'under_attack' })}
                              className="mt-1 block w-full border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                            >
                              <option value="off">Off</option>
                              <option value="low">Low</option>
                              <option value="medium">Medium</option>
                              <option value="high">High</option>
                              <option value="under_attack">Under Attack</option>
                            </select>
                          </div>
                          
                          <div className="bg-yellow-50 p-4 rounded-lg">
                            <h5 className="font-medium text-yellow-900 mb-2">Threat Intelligence</h5>
                            <div className="space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-yellow-700">Blocked Requests (24h):</span>
                                <span className="font-semibold text-yellow-900">1,247</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-yellow-700">Bot Traffic:</span>
                                <span className="font-semibold text-yellow-900">12.3%</span>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    )}

                    {activeTab === 4 && (
                      <div className="space-y-6">
                        <h4 className="text-lg font-medium text-gray-900">Analytics & Performance</h4>
                        <div className="grid grid-cols-2 gap-6">
                          <div className="bg-green-50 p-4 rounded-lg">
                            <h5 className="font-medium text-green-900 mb-2">Response Times</h5>
                            <div className="text-2xl font-bold text-green-900">98ms</div>
                            <div className="text-sm text-green-700">Average response time</div>
                          </div>
                          <div className="bg-blue-50 p-4 rounded-lg">
                            <h5 className="font-medium text-blue-900 mb-2">Bandwidth Usage</h5>
                            <div className="text-2xl font-bold text-blue-900">2.4 TB</div>
                            <div className="text-sm text-blue-700">This month</div>
                          </div>
                        </div>
                        
                        <div className="bg-gray-50 p-4 rounded-lg">
                          <h5 className="font-medium text-gray-900 mb-2">Geographic Distribution</h5>
                          <div className="space-y-2 text-sm">
                            <div className="flex justify-between">
                              <span>North America:</span>
                              <span className="font-semibold">45.2%</span>
                            </div>
                            <div className="flex justify-between">
                              <span>Europe:</span>
                              <span className="font-semibold">32.1%</span>
                            </div>
                            <div className="flex justify-between">
                              <span>Asia Pacific:</span>
                              <span className="font-semibold">18.7%</span>
                            </div>
                            <div className="flex justify-between">
                              <span>Others:</span>
                              <span className="font-semibold">4.0%</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>

                {/* Footer */}
                <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end space-x-3">
                  <button
                    type="button"
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    onClick={onClose}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
                    onClick={handleSave}
                  >
                    Save Changes
                  </button>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
