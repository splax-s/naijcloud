import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { ConditionalLayout } from '@/components/layout/ConditionalLayout';
import { SWRProvider } from '@/components/providers/SWRProvider';
import { AuthProvider } from '@/components/providers/AuthProvider';

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "NaijCloud CDN Dashboard",
  description: "Manage your CDN domains, analytics, and cache policies",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <AuthProvider>
          <SWRProvider>
            <ConditionalLayout>
              {children}
            </ConditionalLayout>
          </SWRProvider>
        </AuthProvider>
      </body>
    </html>
  );
}