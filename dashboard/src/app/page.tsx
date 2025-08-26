'use client';

import { StatsCards } from '@/components/dashboard/StatsCards';
import { RecentActivity } from '@/components/dashboard/RecentActivity';
import { TrafficChart } from '@/components/dashboard/TrafficChart';
import { TopDomains } from '@/components/dashboard/TopDomains';
import { ErrorBoundary } from '@/components/ErrorBoundary';

export default function DashboardPage() {
  return (
    <ErrorBoundary>
      <div className="space-y-6">
        {/* Welcome Header */}
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">Dashboard</h1>
          <p className="text-sm text-gray-600 mt-1">
            Welcome back! Here&apos;s what&apos;s happening with your CDN.
          </p>
        </div>

        {/* Stats Cards */}
        <StatsCards />

        {/* Main Content Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Traffic Chart */}
          <TrafficChart />

          {/* Recent Activity */}
          <RecentActivity />
        </div>

        {/* Top Domains */}
        <TopDomains />
      </div>
    </ErrorBoundary>
  );
}
