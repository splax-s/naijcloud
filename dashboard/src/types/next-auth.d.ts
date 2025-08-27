import { DefaultSession, DefaultUser } from 'next-auth';
import { DefaultJWT } from 'next-auth/jwt';

declare module 'next-auth' {
  interface Session {
    user: {
      id: string;
      role: string;
      organization?: {
        id: string;
        name: string;
        slug: string;
      };
      emailVerified?: boolean;
    } & DefaultSession['user'];
    accessToken?: string;
    error?: string;
  }

  interface User extends DefaultUser {
    role: string;
    organization?: {
      id: string;
      name: string;
      slug: string;
    };
    emailVerified?: boolean;
    tokens?: {
      accessToken: string;
      refreshToken: string;
      expiresAt: string;
    };
  }
}

declare module 'next-auth/jwt' {
  interface JWT extends DefaultJWT {
    accessToken?: string;
    refreshToken?: string;
    accessTokenExpires?: number;
    user?: {
      id: string;
      email: string;
      name: string;
      role: string;
      organization?: {
        id: string;
        name: string;
        slug: string;
      };
      emailVerified?: boolean;
    };
    error?: string;
  }
}
