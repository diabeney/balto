'use client';

import { format } from 'date-fns';
import type { RequestMetric } from '@/lib/metrics/types';
import { useRecentRequests } from '@/lib/metrics/hooks';

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDuration(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

const methodColors: Record<string, string> = {
  GET: '#3b82f6',
  POST: '#10b981',
  PUT: '#f59e0b',
  DELETE: '#ef4444',
  PATCH: '#8b5cf6',
};

const statusColors: Record<string, string> = {
  '2xx': '#10b981',
  '3xx': '#3b82f6',
  '4xx': '#f59e0b',
  '5xx': '#ef4444',
};

function getStatusGroup(code: number): string {
  if (code >= 200 && code < 300) return '2xx';
  if (code >= 300 && code < 400) return '3xx';
  if (code >= 400 && code < 500) return '4xx';
  return '5xx';
}

interface RequestsTableProps {
  limit?: number;
  title?: string;
}

export function RequestsTable({ limit = 50, title = 'Recent Requests' }: RequestsTableProps) {
  const { data: requests, isLoading } = useRecentRequests(limit);

  if (isLoading && !requests) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-5">
        <h3 className="text-sm font-medium text-zinc-400">{title}</h3>
        <div className="flex h-[300px] items-center justify-center">
          <span className="text-zinc-600">Loading...</span>
        </div>
      </div>
    );
  }

  const displayRequests = Array.isArray(requests) ? requests : [];

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-5">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium text-zinc-400">{title}</h3>
          <p className="mt-0.5 text-xs text-zinc-600">
            End-to-end request latency and status
          </p>
        </div>
        <span className="rounded-full bg-zinc-800 px-2 py-0.5 text-xs text-zinc-400">
          {displayRequests.length} requests
        </span>
      </div>

      <div className="overflow-hidden rounded-lg border border-zinc-800">
        <table className="w-full text-left text-sm">
          <thead className="bg-zinc-900/50">
            <tr className="text-xs text-zinc-500">
              <th className="px-4 py-3 font-medium">Timestamp</th>
              <th className="px-4 py-3 font-medium">Method</th>
              <th className="px-4 py-3 font-medium">Backend</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium">Latency</th>
              <th className="px-4 py-3 font-medium">Size</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800/50">
            {displayRequests.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-zinc-600">
                  No requests yet
                </td>
              </tr>
            ) : (
              displayRequests.map((req, i) => {
                const statusGroup = getStatusGroup(req.status_code);
                const statusColor = statusColors[statusGroup];
                const methodColor = methodColors[req.method] || '#6b7280';
                const latencyMs = req.timing?.total_ms || 0;
                const size = (req.bytes_sent || 0) + (req.bytes_received || 0);

                return (
                  <tr key={`${req.timestamp}-${i}`} className="hover:bg-zinc-900/30">
                    <td className="px-4 py-2.5">
                      <span className="font-mono text-xs text-zinc-400">
                        {format(new Date(req.timestamp), 'MMM dd HH:mm:ss')}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      <span
                        className="rounded px-1.5 py-0.5 text-xs font-medium"
                        style={{
                          backgroundColor: `${methodColor}15`,
                          color: methodColor,
                        }}
                      >
                        {req.method}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      <span className="font-mono text-xs text-zinc-300">
                        {req.backend_id || 'unknown'}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex items-center gap-1.5">
                        <div
                          className="h-1.5 w-1.5 rounded-full"
                          style={{ backgroundColor: statusColor }}
                        />
                        <span
                          className="font-mono text-xs font-medium"
                          style={{ color: statusColor }}
                        >
                          {req.status_code}
                        </span>
                      </div>
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex items-center gap-1.5">
                        {(() => {
                          let latencyColor = '#8b5cf6';
                          if (latencyMs < 50) latencyColor = '#8b5cf6';
                          else if (latencyMs < 100) latencyColor = '#3b82f6';
                          else if (latencyMs < 200) latencyColor = '#eab308';
                          else if (latencyMs < 500) latencyColor = '#f97316';
                          else latencyColor = '#ef4444';
                          return (
                            <>
                              <div
                                className="h-1.5 w-1.5 rounded-full"
                                style={{ backgroundColor: latencyColor }}
                              />
                              <span className="font-mono text-xs text-zinc-300">
                                {formatDuration(latencyMs)}
                              </span>
                            </>
                          );
                        })()}
                      </div>
                    </td>
                    <td className="px-4 py-2.5">
                      <div className="flex items-center gap-1.5">
                        <div className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                        <span className="font-mono text-xs text-zinc-300">
                          {formatBytes(size)}
                        </span>
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

