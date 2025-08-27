import NextAuth from 'next-auth';
import CredentialsProvider from 'next-auth/providers/credentials';
import { NextAuthOptions } from 'next-auth';

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
          // Try authenticating with our backend API first
          const response = await fetch('http://localhost:8080/api/v1/auth/login', {
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
            return {
              id: data.user.id,
              email: data.user.email,
              name: data.user.name,
              role: 'user',
              organization: data.organization,
            };
          }
        } catch (error) {
          console.error('Backend auth failed:', error);
        }

        // Fallback to hardcoded admin for demo
        if (
          credentials.email === 'admin@naijcloud.com' && 
          credentials.password === 'password'
        ) {
          return {
            id: '1',
            email: 'admin@naijcloud.com',
            name: 'NaijCloud Admin',
            role: 'admin',
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
    async jwt({ token, user }) {
      if (user?.role) {
        token.role = user.role;
      }
      if (user?.organization) {
        token.organization = user.organization;
      }
      return token;
    },
    async session({ session, token }) {
      if (token?.sub) {
        session.user.id = token.sub;
        session.user.role = token.role;
        session.user.organization = token.organization;
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
