'use client';

import { useState, useEffect } from 'react';
import { useSession } from 'next-auth/react';
import { 
  BuildingOfficeIcon, 
  PlusIcon, 
  UserGroupIcon,
  CogIcon,
  ChartBarIcon,
} from '@heroicons/react/24/outline';

interface Organization {
  id: string;
  name: string;
  slug: string;
  description: string;
  plan: string;
  created_at: string;
  member_count?: number;
  domain_count?: number;
}

export default function OrganizationsPage() {
  const { data: session } = useSession();
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (session?.user) {
      fetchOrganizations();
    }
  }, [session]);

  const fetchOrganizations = async () => {
    try {
      setLoading(true);
      // This would be the actual API call to fetch organizations
      // For now, we'll simulate with demo data
      const demoOrgs: Organization[] = [
        {
          id: '1',
          name: 'Demo Company',
          slug: 'demo-company', 
          description: 'Main organization for demo purposes',
          plan: 'free',
          created_at: '2025-08-26T21:11:13Z',
          member_count: 1,
          domain_count: 3,
        },
        {
          id: '2',
          name: 'Test Organization',
          slug: 'test-org',
          description: 'Testing environment organization',
          plan: 'pro',
          created_at: '2025-08-26T19:13:33Z',
          member_count: 2,
          domain_count: 8,
        },
      ];
      setOrganizations(demoOrgs);
    } catch (error) {
      console.error('Failed to fetch organizations:', error);
    } finally {
      setLoading(false);
    }
  };

  const getPlanBadgeColor = (plan: string) => {
    switch (plan) {
      case 'free':
        return 'bg-gray-100 text-gray-800';
      case 'pro':
        return 'bg-blue-100 text-blue-800';
      case 'enterprise':
        return 'bg-purple-100 text-purple-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
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
          <h1 className="text-2xl font-semibold text-gray-900">Organizations</h1>
          <p className="text-sm text-gray-600 mt-1">
            Manage your organizations and switch between them
          </p>
        </div>
        <button className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500">
          <PlusIcon className="w-4 h-4 mr-2" />
          Create Organization
        </button>
      </div>

      {/* Current Organization */}
      {session?.user?.organization && (
        <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
          <div className="flex items-center">
            <BuildingOfficeIcon className="w-8 h-8 text-blue-600 mr-4" />
            <div>
              <h3 className="text-lg font-medium text-blue-900">Current Organization</h3>
              <p className="text-blue-700">{session.user.organization.name}</p>
              <p className="text-sm text-blue-600">/{session.user.organization.slug}</p>
            </div>
          </div>
        </div>
      )}

      {/* Organizations Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {organizations.map((org) => (
          <div key={org.id} className="bg-white border border-gray-200 rounded-lg shadow-sm hover:shadow-md transition-shadow">
            <div className="p-6">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center">
                  <BuildingOfficeIcon className="w-8 h-8 text-gray-400 mr-3" />
                  <div>
                    <h3 className="text-lg font-medium text-gray-900">{org.name}</h3>
                    <p className="text-sm text-gray-500">/{org.slug}</p>
                  </div>
                </div>
                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getPlanBadgeColor(org.plan)}`}>
                  {org.plan}
                </span>
              </div>

              {org.description && (
                <p className="text-sm text-gray-600 mb-4">{org.description}</p>
              )}

              <div className="flex justify-between items-center text-sm text-gray-500 mb-4">
                <div className="flex items-center">
                  <UserGroupIcon className="w-4 h-4 mr-1" />
                  {org.member_count} members
                </div>
                <div className="flex items-center">
                  <ChartBarIcon className="w-4 h-4 mr-1" />
                  {org.domain_count} domains
                </div>
              </div>

              <p className="text-xs text-gray-500 mb-4">
                Created {formatDate(org.created_at)}
              </p>

              <div className="flex space-x-2">
                <button className="flex-1 inline-flex items-center justify-center px-3 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50">
                  <CogIcon className="w-4 h-4 mr-2" />
                  Manage
                </button>
                <button className="inline-flex items-center px-3 py-2 border border-transparent text-sm font-medium rounded-md text-blue-700 bg-blue-100 hover:bg-blue-200">
                  Switch
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {organizations.length === 0 && (
        <div className="text-center py-12">
          <BuildingOfficeIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">No organizations</h3>
          <p className="mt-1 text-sm text-gray-500">
            Get started by creating your first organization.
          </p>
          <div className="mt-6">
            <button className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700">
              <PlusIcon className="w-4 h-4 mr-2" />
              Create Organization
            </button>
          </div>
        </div>
      )}

      {/* Organization Stats */}
      <div className="bg-white border border-gray-200 rounded-lg">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-medium text-gray-900">Organization Activity</h3>
        </div>
        <div className="p-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">{organizations.length}</div>
              <div className="text-sm text-gray-500">Total Organizations</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">
                {organizations.reduce((acc, org) => acc + (org.domain_count || 0), 0)}
              </div>
              <div className="text-sm text-gray-500">Total Domains</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-gray-900">
                {organizations.reduce((acc, org) => acc + (org.member_count || 0), 0)}
              </div>
              <div className="text-sm text-gray-500">Total Members</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
