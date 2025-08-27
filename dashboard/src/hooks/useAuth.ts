import { useSession } from 'next-auth/react';
import { Session } from 'next-auth';

interface ExtendedSession extends Session {
  accessToken?: string;
  error?: string;
}

export function useAuth() {
  const { data: session, status, update } = useSession();
  const extendedSession = session as ExtendedSession;

  return {
    session: extendedSession,
    user: session?.user,
    status,
    update,
    isLoading: status === 'loading',
    isAuthenticated: status === 'authenticated',
    accessToken: extendedSession?.accessToken,
    error: extendedSession?.error,
  };
}

export type { ExtendedSession };
