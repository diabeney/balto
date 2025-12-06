'use client';

import { ParsedMetrics } from '@/lib/prometheus/types';
import { aggregateMetrics, getLatestMetric } from '@/lib/prometheus/parser';

interface MetricsSummaryProps {
  metrics: ParsedMetrics;
}

export function MetricsSummary({ metrics }: MetricsSummaryProps) {
  const totalRequests = aggregateMetrics(metrics.requestsTotal);
  const totalFailures = aggregateMetrics(metrics.backendFailures);
  const totalProbeFailures = aggregateMetrics(metrics.probeFailures);
  const activeConnections = aggregateMetrics(metrics.backendActiveConnections);
  const healthyBackends = metrics.backendHealthy.filter(m => m.value === 1).length;
  const totalBackends = metrics.backendHealthy.length;

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6">
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Total Requests</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {totalRequests.toLocaleString()}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Backend Failures</div>
        <div className="mt-1 text-2xl font-semibold text-red-600 dark:text-red-400">
          {totalFailures.toLocaleString()}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Probe Failures</div>
        <div className="mt-1 text-2xl font-semibold text-red-600 dark:text-red-400">
          {totalProbeFailures.toLocaleString()}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Active Connections</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {activeConnections.toLocaleString()}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Healthy Backends</div>
        <div className="mt-1 text-2xl font-semibold text-green-600 dark:text-green-400">
          {healthyBackends}/{totalBackends}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Error Rate</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {totalRequests > 0 
            ? ((totalFailures / totalRequests) * 100).toFixed(2) 
            : '0.00'}%
        </div>
      </div>
    </div>
  );
}

