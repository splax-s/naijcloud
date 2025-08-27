'use client';

import { useState, useEffect } from 'react';
import { useSession } from 'next-auth/react';
import { Fragment } from 'react';
import { Listbox, Transition } from '@headlessui/react';
import {
  ChevronUpDownIcon,
  CheckIcon,
  BuildingOfficeIcon,
  PlusIcon,
} from '@heroicons/react/24/outline';

interface Organization {
  id: string;
  name: string;
  slug: string;
}

interface OrganizationSwitcherProps {
  onOrganizationChange?: (organization: Organization) => void;
}

function classNames(...classes: string[]) {
  return classes.filter(Boolean).join(' ');
}

export function OrganizationSwitcher({ onOrganizationChange }: OrganizationSwitcherProps) {
  const { data: session } = useSession();
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [selectedOrganization, setSelectedOrganization] = useState<Organization | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  // Load user's organizations
  useEffect(() => {
    const loadOrganizations = async () => {
      if (!session?.user?.id) return;

      setIsLoading(true);
      try {
        const response = await fetch(`/api/v1/user/organizations`, {
          headers: {
            'X-User-ID': session.user.id,
            'X-User-Email': session.user.email!,
          },
        });

        if (response.ok) {
          const data = await response.json();
          setOrganizations(data.organizations || []);
          
          // Set the current organization from session, or first available
          const currentOrg = session.user.organization || (data.organizations && data.organizations[0]);
          if (currentOrg) {
            setSelectedOrganization(currentOrg);
            onOrganizationChange?.(currentOrg);
          }
        }
      } catch (error) {
        console.error('Failed to load organizations:', error);
        // Fallback to session organization
        if (session.user.organization) {
          setOrganizations([session.user.organization]);
          setSelectedOrganization(session.user.organization);
          onOrganizationChange?.(session.user.organization);
        }
      } finally {
        setIsLoading(false);
      }
    };

    loadOrganizations();
  }, [session, onOrganizationChange]);

  const handleOrganizationChange = (organization: Organization) => {
    setSelectedOrganization(organization);
    onOrganizationChange?.(organization);
    
    // Store selected organization in localStorage for persistence
    localStorage.setItem('selectedOrganization', JSON.stringify(organization));
  };

  // Load persisted organization on mount
  useEffect(() => {
    const stored = localStorage.getItem('selectedOrganization');
    if (stored) {
      try {
        const org = JSON.parse(stored);
        setSelectedOrganization(org);
        onOrganizationChange?.(org);
      } catch (error) {
        console.error('Failed to parse stored organization:', error);
      }
    }
  }, [onOrganizationChange]);

  if (!session?.user) {
    return null;
  }

  if (isLoading) {
    return (
      <div className="flex items-center space-x-2 animate-pulse">
        <div className="w-8 h-8 bg-gray-200 rounded-lg"></div>
        <div className="w-32 h-4 bg-gray-200 rounded"></div>
      </div>
    );
  }

  if (!selectedOrganization) {
    return (
      <div className="flex items-center space-x-2 text-gray-500">
        <BuildingOfficeIcon className="w-5 h-5" />
        <span className="text-sm">No organization</span>
      </div>
    );
  }

  return (
    <Listbox value={selectedOrganization} onChange={handleOrganizationChange}>
      {({ open }) => (
        <div className="relative">
          <Listbox.Button className="relative w-full cursor-default rounded-lg bg-white py-2 pl-3 pr-10 text-left shadow-sm border border-gray-300 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 sm:text-sm">
            <div className="flex items-center">
              <BuildingOfficeIcon className="h-5 w-5 text-gray-400 mr-2" />
              <span className="block truncate font-medium text-gray-900">
                {selectedOrganization.name}
              </span>
            </div>
            <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
              <ChevronUpDownIcon className="h-5 w-5 text-gray-400" aria-hidden="true" />
            </span>
          </Listbox.Button>

          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options className="absolute z-10 mt-1 max-h-60 w-full overflow-auto rounded-md bg-white py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none sm:text-sm">
              {organizations.map((organization) => (
                <Listbox.Option
                  key={organization.id}
                  className={({ active }) =>
                    classNames(
                      active ? 'bg-blue-600 text-white' : 'text-gray-900',
                      'relative cursor-default select-none py-2 pl-3 pr-9'
                    )
                  }
                  value={organization}
                >
                  {({ selected, active }) => (
                    <>
                      <div className="flex items-center">
                        <BuildingOfficeIcon className={classNames(
                          'h-5 w-5 mr-2',
                          active ? 'text-white' : 'text-gray-400'
                        )} />
                        <span className={classNames(
                          selected ? 'font-semibold' : 'font-normal',
                          'block truncate'
                        )}>
                          {organization.name}
                        </span>
                      </div>

                      {selected ? (
                        <span className={classNames(
                          active ? 'text-white' : 'text-blue-600',
                          'absolute inset-y-0 right-0 flex items-center pr-4'
                        )}>
                          <CheckIcon className="h-5 w-5" aria-hidden="true" />
                        </span>
                      ) : null}
                    </>
                  )}
                </Listbox.Option>
              ))}
              
              <div className="border-t border-gray-200 mt-1 pt-1">
                <button className="flex items-center w-full py-2 pl-3 pr-9 text-sm text-gray-700 hover:bg-gray-100">
                  <PlusIcon className="h-5 w-5 text-gray-400 mr-2" />
                  Create new organization
                </button>
              </div>
            </Listbox.Options>
          </Transition>
        </div>
      )}
    </Listbox>
  );
}
