'use client';

import { ParsedMetrics } from '@/lib/prometheus/types';
import { aggregateMetrics } from '@/lib/prometheus';

interface OverviewCardsProps {
  metrics: ParsedMetrics;
}

export function OverviewCards({ metrics }: OverviewCardsProps) {
  const totalRequests = aggregateMetrics(metrics.requestsTotal);
  const totalFailures = aggregateMetrics(metrics.backendFailures);
  const successRate = totalRequests > 0 ? ((totalRequests - totalFailures) / totalRequests) * 100 : 100;
  const activeConnections = aggregateMetrics(metrics.backendActiveConnections);
  const healthyBackends = metrics.backendHealthy.filter(m => m.value === 1).length;
  const totalBackends = metrics.backendHealthy.length;

  // Calculate requests by status code
  const requestsByStatus = metrics.requestsTotal.reduce((acc, m) => {
    const status = m.labels.status_code || 'unknown';
    acc[status] = (acc[status] || 0) + m.value;
    return acc;
  }, {} as Record<string, number>);

  const http2xx = Math.round(Object.entries(requestsByStatus)
    .filter(([code]) => code.startsWith('2'))
    .reduce((sum, [, count]) => sum + count, 0));
  const http3xx = Math.round(Object.entries(requestsByStatus)
    .filter(([code]) => code.startsWith('3'))
    .reduce((sum, [, count]) => sum + count, 0));
  const http4xx = Math.round(Object.entries(requestsByStatus)
    .filter(([code]) => code.startsWith('4'))
    .reduce((sum, [, count]) => sum + count, 0));
  const http5xx = Math.round(Object.entries(requestsByStatus)
    .filter(([code]) => code.startsWith('5'))
    .reduce((sum, [, count]) => sum + count, 0));

  const successRate2m = successRate; // In a real implementation, this would be calculated over 2 minutes

  return (
    <div className="grid grid-cols-2 gap-4 md:grid-cols-4 lg:grid-cols-9">
      <div className="rounded-lg border border-zinc-200 bg-zinc-100 p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="text-xs text-zinc-600 dark:text-zinc-400">Requests (period)</div>
        <div className="mt-1 text-xs font-semibold text-zinc-900 dark:text-zinc-50">
          {(totalRequests / 1000).toFixed(1)} K
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-zinc-100 p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="text-xs text-zinc-600 dark:text-zinc-400">% Success...</div>
        <div className="mt-1 text-xs font-semibold text-zinc-900 dark:text-zinc-50">
          {successRate.toFixed(1)}%
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-zinc-100 p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="text-xs text-zinc-600 dark:text-zinc-400">Conns (...)</div>
        <div className="mt-1 text-xs font-semibold text-zinc-900 dark:text-zinc-50">
          {activeConnections}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-zinc-100 p-4 dark:border-zinc-800 dark:bg-zinc-950">
        <div className="text-xs text-zinc-600 dark:text-zinc-400">Backends</div>
        <div className="mt-1 text-xs font-semibold text-zinc-900 dark:text-zinc-50">
          {healthyBackends}/{totalBackends}
        </div>
      </div>
      <div className="rounded-lg border border-green-200 bg-green-100 p-4 dark:border-green-800 dark:bg-green-950/30">
        <div className="text-xs text-green-600 dark:text-green-400">% Success (2m)</div>
        <div className="mt-1 text-xs font-semibold text-green-700 dark:text-green-300">
          {successRate2m.toFixed(1)}%
        </div>
      </div>
      <div className="rounded-lg border border-green-200 bg-green-100 p-4 dark:border-green-800 dark:bg-green-950/30">
        <div className="text-xs text-green-600 dark:text-green-400">HTTP 1/2xx (2m)</div>
        <div className="mt-1 text-xs font-semibold text-green-700 dark:text-green-300">
          {http2xx}
        </div>
      </div>
      <div className="rounded-lg border border-blue-200 bg-blue-100 p-4 dark:border-blue-800 dark:bg-blue-950/30">
        <div className="text-xs text-blue-600 dark:text-blue-400">HTTP 3xx (2m)</div>
        <div className="mt-1 text-xs font-semibold text-blue-700 dark:text-blue-300">
          {http3xx}
        </div>
      </div>
      <div className="rounded-lg border border-orange-200 bg-orange-100 p-4 dark:border-orange-800 dark:bg-orange-950/30">
        <div className="text-xs text-orange-600 dark:text-orange-400">HTTP 4xx (2m)</div>
        <div className="mt-1 text-xs font-semibold text-orange-700 dark:text-orange-300">
          {http4xx}
        </div>
      </div>
      <div className="rounded-lg border border-red-200 bg-red-100 p-4 dark:border-red-800 dark:bg-red-950/30">
        <div className="text-xs text-red-600 dark:text-red-400">HTTP 5xx (2m)</div>
        <div className="mt-1 text-xs font-semibold text-red-700 dark:text-red-300">
          {http5xx}
        </div>
      </div>
    </div>
  );
}

