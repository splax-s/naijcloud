'use client';

import { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { useSession } from 'next-auth/react';

interface Organization {
  id: string;
  name: string;
  slug: string;
}

interface OrganizationContextType {
  organization: Organization | null;
  setOrganization: (org: Organization | null) => void;
  isLoading: boolean;
}

const OrganizationContext = createContext<OrganizationContextType | undefined>(undefined);

interface OrganizationProviderProps {
  children: ReactNode;
}

export function OrganizationProvider({ children }: OrganizationProviderProps) {
  const [organization, setOrganization] = useState<Organization | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const { data: session } = useSession();

  useEffect(() => {
    // Initialize organization from session or localStorage
    const initializeOrganization = () => {
      setIsLoading(true);
      
      // First, try to get from localStorage
      const stored = localStorage.getItem('selectedOrganization');
      if (stored) {
        try {
          const org = JSON.parse(stored);
          setOrganization(org);
          setIsLoading(false);
          return;
        } catch (error) {
          console.error('Failed to parse stored organization:', error);
        }
      }

      // Fallback to session organization
      if (session?.user?.organization) {
        setOrganization(session.user.organization);
      }
      
      setIsLoading(false);
    };

    if (session !== undefined) {
      initializeOrganization();
    }
  }, [session]);

  const handleSetOrganization = (org: Organization | null) => {
    setOrganization(org);
    if (org) {
      localStorage.setItem('selectedOrganization', JSON.stringify(org));
    } else {
      localStorage.removeItem('selectedOrganization');
    }
  };

  return (
    <OrganizationContext.Provider 
      value={{ 
        organization, 
        setOrganization: handleSetOrganization, 
        isLoading 
      }}
    >
      {children}
    </OrganizationContext.Provider>
  );
}

export function useOrganization() {
  const context = useContext(OrganizationContext);
  if (context === undefined) {
    throw new Error('useOrganization must be used within an OrganizationProvider');
  }
  return context;
}

// Hook to get the current organization slug for API calls
export function useOrganizationSlug(): string | null {
  const { organization } = useOrganization();
  return organization?.slug || null;
}
