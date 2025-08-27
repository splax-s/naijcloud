import NextAuth from 'next-auth';
import CredentialsProvider from 'next-auth/providers/credentials';
import { NextAuthOptions, Session } from 'next-auth';

interface CustomUser {
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
  tokens?: {
    accessToken: string;
    refreshToken: string;
    expiresAt: string;
  };
}

export const authOptions: NextAuthOptions = {
  providers: [
    CredentialsProvider({
      name: 'credentials',
      credentials: {
        email: { label: 'Email', type: 'email' },
        password: { label: 'Password', type: 'password' }
      },
      async authorize(credentials) {
        if (!credentials?.email || !credentials?.password) {
          return null;
        }

        try {
          // Authenticate with our Phase 6 backend API
          const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/auth/login`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({
              email: credentials.email,
              password: credentials.password,
            }),
          });

          if (response.ok) {
            const data = await response.json();
            
            // Phase 6 returns user and organization data
            if (data.user) {
              return {
                id: data.user.id,
                email: data.user.email,
                name: data.user.name,
                role: 'user',
                organization: data.organization,
                emailVerified: data.user.email_verified,
                tokens: data.tokens, // Include tokens if present
              };
            }
          } else {
            const errorData = await response.json();
            console.error('Backend auth error:', errorData);
          }
        } catch (error) {
          console.error('Backend auth failed:', error);
        }

        // Fallback to hardcoded admin for demo (remove in production)
        if (
          credentials.email === 'admin@naijcloud.com' && 
          credentials.password === 'password'
        ) {
          return {
            id: '1',
            email: 'admin@naijcloud.com',
            name: 'NaijCloud Admin',
            role: 'admin',
            emailVerified: true,
            organization: {
              id: '3fbdbdad-dbf5-4ac1-9335-e644302769ad',
              name: 'NaijCloud Demo',
              slug: 'naijcloud-demo',
            },
          };
        }

        return null;
      }
    })
  ],
  session: {
    strategy: 'jwt',
  },
  callbacks: {
    async jwt({ token, user, account }) {
      // Initial sign in
      if (account && user) {
        const customUser = user as CustomUser;
        return {
          ...token,
          accessToken: customUser.tokens?.accessToken,
          refreshToken: customUser.tokens?.refreshToken,
          accessTokenExpires: customUser.tokens?.expiresAt ? new Date(customUser.tokens.expiresAt).getTime() : Date.now() + 30 * 60 * 1000,
          user: {
            id: user.id || '',
            email: user.email || '',
            name: user.name || '',
            role: customUser.role,
            organization: customUser.organization,
            emailVerified: customUser.emailVerified,
          },
        };
      }

      // Return previous token if the access token has not expired yet
      if (token.accessTokenExpires && typeof token.accessTokenExpires === 'number' && Date.now() < token.accessTokenExpires) {
        return token;
      }

      // Access token has expired, try to update it
      if (token.refreshToken) {
        try {
          const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/api/v1/auth/refresh`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token.refreshToken}`,
            },
          });

          if (response.ok) {
            const refreshedTokens = await response.json();
            return {
              ...token,
              accessToken: refreshedTokens.accessToken,
              accessTokenExpires: new Date(refreshedTokens.expiresAt).getTime(),
              refreshToken: refreshedTokens.refreshToken ?? token.refreshToken,
            };
          }
        } catch (error) {
          console.error('Error refreshing access token:', error);
          return { ...token, error: 'RefreshTokenError' };
        }
      }

      return token;
    },
    async session({ session, token }) {
      if (token && token.user) {
        const extendedSession = session as Session & { accessToken?: string; error?: string };
        extendedSession.user = {
          ...session.user,
          id: token.user.id,
          role: token.user.role,
          organization: token.user.organization,
          emailVerified: token.user.emailVerified,
        };
        extendedSession.accessToken = token.accessToken;
        extendedSession.error = token.error;
        return extendedSession;
      }
      return session;
    },
  },
  pages: {
    signIn: '/auth/signin',
    signOut: '/auth/signout',
  },
  secret: process.env.NEXTAUTH_SECRET,
};

const handler = NextAuth(authOptions);
export { handler as GET, handler as POST };
