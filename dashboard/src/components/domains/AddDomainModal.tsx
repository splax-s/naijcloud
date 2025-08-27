'use client';

import { Fragment, useState } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { 
  XMarkIcon, 
  GlobeAltIcon, 
  CheckCircleIcon,
  ExclamationTriangleIcon 
} from '@heroicons/react/24/outline';
import { AddDomainModalProps, DomainFormData, FormErrors } from '@/lib/types';

export default function AddDomainModal({ isOpen, onClose, onSubmit }: AddDomainModalProps) {
  const [step, setStep] = useState(1);
  const [formData, setFormData] = useState<DomainFormData>({
    domain: '',
    origin: '',
    ssl_enabled: true,
    cache_ttl: 3600,
    compression_enabled: true,
    security_level: 'medium',
  });
  const [errors, setErrors] = useState<FormErrors>({});
  const [isValidating, setIsValidating] = useState(false);

  const validateDomain = (domain: string): boolean => {
    const domainRegex = /^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?\.([a-zA-Z]{2,}\.)*[a-zA-Z]{2,}$/;
    return domainRegex.test(domain);
  };

  const validateOrigin = (origin: string): boolean => {
    try {
      new URL(origin);
      return true;
    } catch {
      return false;
    }
  };

  const handleNext = () => {
    const newErrors: FormErrors = {};
    
    if (!formData.domain) {
      newErrors.domain = 'Domain is required';
    } else if (!validateDomain(formData.domain)) {
      newErrors.domain = 'Please enter a valid domain name';
    }
    
    if (!formData.origin) {
      newErrors.origin = 'Origin server is required';
    } else if (!validateOrigin(formData.origin)) {
      newErrors.origin = 'Please enter a valid URL';
    }
    
    setErrors(newErrors);
    
    if (Object.keys(newErrors).length === 0) {
      setStep(2);
      // Simulate domain validation
      setIsValidating(true);
      setTimeout(() => setIsValidating(false), 2000);
    }
  };

  const handleSubmit = async () => {
    try {
      await onSubmit(formData);
      onClose();
      setStep(1);
      setFormData({
        domain: '',
        origin: '',
        ssl_enabled: true,
        cache_ttl: 3600,
        compression_enabled: true,
        security_level: 'medium',
      });
      setErrors({});
    } catch {
      setErrors({ submit: 'Failed to add domain. Please try again.' });
    }
  };

  const handleClose = () => {
    onClose();
    setStep(1);
    setErrors({});
  };

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={handleClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/50" />
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
              <Dialog.Panel className="w-full max-w-2xl transform overflow-hidden rounded-2xl bg-white text-left align-middle shadow-xl transition-all">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
                  <div className="flex items-center">
                    <GlobeAltIcon className="h-6 w-6 text-blue-500 mr-3" />
                    <div>
                      <Dialog.Title as="h3" className="text-lg font-medium text-gray-900">
                        Add New Domain
                      </Dialog.Title>
                      <p className="text-sm text-gray-500">
                        Step {step} of 2: {step === 1 ? 'Domain Configuration' : 'Verification & Settings'}
                      </p>
                    </div>
                  </div>
                  <button
                    type="button"
                    className="rounded-md text-gray-400 hover:text-gray-500 focus:outline-none"
                    onClick={handleClose}
                  >
                    <XMarkIcon className="h-6 w-6" />
                  </button>
                </div>

                <div className="p-6">
                  {step === 1 && (
                    <div className="space-y-6">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Domain Name
                        </label>
                        <input
                          type="text"
                          value={formData.domain}
                          onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
                          className={`block w-full border rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 ${
                            errors.domain ? 'border-red-300' : 'border-gray-300'
                          }`}
                          placeholder="example.com"
                        />
                        {errors.domain && (
                          <p className="mt-1 text-sm text-red-600">{errors.domain}</p>
                        )}
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Origin Server
                        </label>
                        <input
                          type="url"
                          value={formData.origin}
                          onChange={(e) => setFormData({ ...formData, origin: e.target.value })}
                          className={`block w-full border rounded-md shadow-sm px-3 py-2 focus:ring-blue-500 focus:border-blue-500 ${
                            errors.origin ? 'border-red-300' : 'border-gray-300'
                          }`}
                          placeholder="https://your-server.com"
                        />
                        {errors.origin && (
                          <p className="mt-1 text-sm text-red-600">{errors.origin}</p>
                        )}
                        <p className="mt-1 text-sm text-gray-500">
                          The server where your content is hosted
                        </p>
                      </div>

                      <div className="bg-blue-50 p-4 rounded-lg">
                        <h4 className="font-medium text-blue-900 mb-2">Quick Setup</h4>
                        <div className="space-y-2">
                          <label className="flex items-center text-sm">
                            <input
                              type="checkbox"
                              checked={formData.ssl_enabled}
                              onChange={(e) => setFormData({ ...formData, ssl_enabled: e.target.checked })}
                              className="rounded border-gray-300 text-blue-600"
                            />
                            <span className="ml-2 text-blue-800">Enable SSL/TLS (Recommended)</span>
                          </label>
                          <label className="flex items-center text-sm">
                            <input
                              type="checkbox"
                              checked={formData.compression_enabled}
                              onChange={(e) => setFormData({ ...formData, compression_enabled: e.target.checked })}
                              className="rounded border-gray-300 text-blue-600"
                            />
                            <span className="ml-2 text-blue-800">Enable compression</span>
                          </label>
                        </div>
                      </div>
                    </div>
                  )}

                  {step === 2 && (
                    <div className="space-y-6">
                      {isValidating ? (
                        <div className="text-center py-8">
                          <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-100 rounded-full mb-4">
                            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                          </div>
                          <h4 className="text-lg font-medium text-gray-900 mb-2">Verifying Domain</h4>
                          <p className="text-gray-600">
                            Checking DNS records and origin server connectivity...
                          </p>
                        </div>
                      ) : (
                        <>
                          <div className="flex items-center space-x-3 p-4 bg-green-50 rounded-lg">
                            <CheckCircleIcon className="h-6 w-6 text-green-600" />
                            <div>
                              <h4 className="font-medium text-green-900">Domain Verified</h4>
                              <p className="text-sm text-green-700">
                                {formData.domain} is ready to be added to your CDN
                              </p>
                            </div>
                          </div>

                          <div className="space-y-4">
                            <h4 className="font-medium text-gray-900">Configuration Summary</h4>
                            <div className="bg-gray-50 p-4 rounded-lg space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-gray-600">Domain:</span>
                                <span className="font-medium">{formData.domain}</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-gray-600">Origin:</span>
                                <span className="font-medium">{formData.origin}</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-gray-600">SSL/TLS:</span>
                                <span className="font-medium">
                                  {formData.ssl_enabled ? 'Enabled' : 'Disabled'}
                                </span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-gray-600">Cache TTL:</span>
                                <span className="font-medium">{formData.cache_ttl}s</span>
                              </div>
                            </div>
                          </div>

                          <div className="bg-yellow-50 p-4 rounded-lg">
                            <div className="flex">
                              <ExclamationTriangleIcon className="h-5 w-5 text-yellow-400 mr-2 mt-0.5" />
                              <div className="text-sm">
                                <h4 className="font-medium text-yellow-800">DNS Configuration Required</h4>
                                <p className="text-yellow-700 mt-1">
                                  Update your DNS records to point {formData.domain} to our CDN:
                                </p>
                                <code className="block mt-2 p-2 bg-yellow-100 text-yellow-900 text-xs rounded">
                                  CNAME {formData.domain} → cdn.naijcloud.com
                                </code>
                              </div>
                            </div>
                          </div>
                        </>
                      )}
                    </div>
                  )}

                  {errors.submit && (
                    <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-md">
                      <p className="text-sm text-red-600">{errors.submit}</p>
                    </div>
                  )}
                </div>

                {/* Footer */}
                <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end space-x-3">
                  {step === 1 ? (
                    <>
                      <button
                        type="button"
                        className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                        onClick={handleClose}
                      >
                        Cancel
                      </button>
                      <button
                        type="button"
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700"
                        onClick={handleNext}
                      >
                        Next
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        type="button"
                        className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50"
                        onClick={() => setStep(1)}
                        disabled={isValidating}
                      >
                        Back
                      </button>
                      <button
                        type="button"
                        className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 disabled:opacity-50"
                        onClick={handleSubmit}
                        disabled={isValidating}
                      >
                        Add Domain
                      </button>
                    </>
                  )}
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
