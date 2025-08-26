'use client';

import { SWRConfig } from 'swr';
import { ReactNode } from 'react';

interface SWRProviderProps {
  children: ReactNode;
}

export function SWRProvider({ children }: SWRProviderProps) {
  return (
    <SWRConfig
      value={{
        refreshInterval: 0, // Disable automatic refresh by default
        revalidateOnFocus: false,
        revalidateOnReconnect: true,
        dedupingInterval: 5000,
        errorRetryCount: 3,
        errorRetryInterval: 1000,
        onError: (error) => {
          console.error('SWR Error:', error);
          // Here you could add error tracking like Sentry
        },
      }}
    >
      {children}
    </SWRConfig>
  );
}
