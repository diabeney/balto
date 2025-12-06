'use client';

import type { BackendMetrics } from '@/lib/metrics/types';
import { formatDuration, formatNumber } from '@/lib/metrics/transform';

interface BackendHealthTableProps {
  backends: BackendMetrics[];
  title?: string;
}

export function BackendHealthTable({ backends, title = 'Backend Status' }: BackendHealthTableProps) {
  if (!backends || backends.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h3 className="mb-4 font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[200px] items-center justify-center text-zinc-500">
          No backends configured
        </div>
      </div>
    );
  }

  const healthyCount = backends.filter((b) => b.healthy).length;

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex items-center gap-2">
          <div
            className={`h-2 w-2 rounded-full ${
              healthyCount === backends.length ? 'bg-emerald-500' : 'bg-amber-500'
            }`}
          />
          <span className="text-xs text-zinc-500">
            {healthyCount}/{backends.length} healthy
          </span>
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-zinc-200 dark:border-zinc-700">
              <th className="pb-3 pr-4 font-medium text-zinc-500">Backend</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Status</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Connections</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Requests</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Avg Response</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">P95</th>
              <th className="pb-3 font-medium text-zinc-500">Success Rate</th>
            </tr>
          </thead>
          <tbody>
            {backends.map((backend) => {
              const successRate =
                backend.total_requests > 0
                  ? ((backend.successful_requests / backend.total_requests) * 100).toFixed(1)
                  : '100.0';

              return (
                <tr
                  key={backend.id}
                  className="border-b border-zinc-100 last:border-0 dark:border-zinc-800"
                >
                  <td className="py-3 pr-4">
                    <div className="font-mono text-xs font-medium text-zinc-900 dark:text-zinc-50">
                      {backend.id}
                    </div>
                    <div className="text-xs text-zinc-500">{backend.route || '/'}</div>
                  </td>
                  <td className="py-3 pr-4">
                    <span
                      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${
                        backend.healthy
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                          : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                      }`}
                    >
                      <span
                        className={`h-1.5 w-1.5 rounded-full ${
                          backend.healthy ? 'bg-emerald-500' : 'bg-red-500'
                        }`}
                      />
                      {backend.healthy ? 'Healthy' : 'Unhealthy'}
                    </span>
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs text-zinc-900 dark:text-zinc-50">
                    {backend.active_connections}
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs text-zinc-900 dark:text-zinc-50">
                    {formatNumber(backend.total_requests)}
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs text-zinc-900 dark:text-zinc-50">
                    {formatDuration(backend.avg_response_time_ms)}
                  </td>
                  <td className="py-3 pr-4 font-mono text-xs text-zinc-900 dark:text-zinc-50">
                    {formatDuration(backend.p95_response_time_ms)}
                  </td>
                  <td className="py-3">
                    <span
                      className={`font-mono text-xs font-medium ${
                        parseFloat(successRate) >= 99
                          ? 'text-emerald-600'
                          : parseFloat(successRate) >= 95
                          ? 'text-amber-600'
                          : 'text-red-600'
                      }`}
                    >
                      {successRate}%
                    </span>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

