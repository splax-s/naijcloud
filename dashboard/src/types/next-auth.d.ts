import { DefaultSession, DefaultUser } from 'next-auth';

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
    } & DefaultSession['user'];
  }

  interface User extends DefaultUser {
    role: string;
    organization?: {
      id: string;
      name: string;
      slug: string;
    };
  }
}

declare module 'next-auth/jwt' {
  interface JWT {
    role: string;
    organization?: {
      id: string;
      name: string;
      slug: string;
    };
  }
}
