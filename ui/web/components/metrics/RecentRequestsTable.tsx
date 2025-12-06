'use client';

import { useEffect, useState } from 'react';
import { format } from 'date-fns';
import type { RequestMetric } from '@/lib/metrics/types';
import { fetchRecentRequests } from '@/lib/metrics/client';
import { formatBytes, formatDuration, METHOD_COLORS, STATUS_COLORS } from '@/lib/metrics/transform';

interface RecentRequestsTableProps {
  limit?: number;
  title?: string;
  autoRefresh?: boolean;
  refreshInterval?: number;
}

export function RecentRequestsTable({
  limit = 100,
  title = 'Recent Requests',
  autoRefresh = true,
  refreshInterval = 2000,
}: RecentRequestsTableProps) {
  const [requests, setRequests] = useState<RequestMetric[]>([]);
  const [loading, setLoading] = useState(true);

  const loadRequests = async () => {
    try {
      const data = await fetchRecentRequests(limit);
      setRequests(data);
    } catch (error) {
      console.error('Failed to fetch recent requests:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRequests();
    if (autoRefresh) {
      const interval = setInterval(loadRequests, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [limit, autoRefresh, refreshInterval]);

  const getStatusColor = (statusCode: number): string => {
    if (statusCode >= 200 && statusCode < 300) return STATUS_COLORS['2xx'];
    if (statusCode >= 300 && statusCode < 400) return STATUS_COLORS['3xx'];
    if (statusCode >= 400 && statusCode < 500) return STATUS_COLORS['4xx'];
    if (statusCode >= 500) return STATUS_COLORS['5xx'];
    return STATUS_COLORS.other;
  };

  const getMethodColor = (method: string): string => {
    return METHOD_COLORS[method as keyof typeof METHOD_COLORS] || METHOD_COLORS.GET;
  };

  const getStatusIcon = (statusCode: number) => {
    if (statusCode >= 200 && statusCode < 300) {
      return (
        <div className="flex h-4 w-4 items-center justify-center rounded-full bg-emerald-500">
          <svg className="h-2.5 w-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
        </div>
      );
    }
    if (statusCode >= 400) {
      return (
        <div className="flex h-4 w-4 items-center justify-center rounded-full bg-red-500">
          <svg className="h-2.5 w-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </div>
      );
    }
    return (
      <div className="flex h-4 w-4 items-center justify-center rounded-full bg-amber-500">
        <svg className="h-2.5 w-2.5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>
    );
  };

  const getMethodIcon = (method: string) => {
    switch (method.toUpperCase()) {
      case 'GET':
        return (
          <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        );
      case 'POST':
        return (
          <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
        );
      case 'PUT':
        return (
          <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        );
      case 'DELETE':
        return (
          <svg className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
        );
      default:
        return null;
    }
  };

  if (loading && requests.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h3 className="mb-4 font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[400px] items-center justify-center text-zinc-500">
          Loading...
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h3 className="font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
          <p className="mt-1 text-xs text-zinc-500">
            Latest HTTP requests with end-to-end latency metrics
          </p>
        </div>
        <div className="text-xs text-zinc-500">
          {requests.length} {requests.length === 1 ? 'request' : 'requests'}
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm">
          <thead>
            <tr className="border-b border-zinc-200 dark:border-zinc-700">
              <th className="pb-3 pr-4 font-medium text-zinc-500">Timestamp</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Method</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Backend</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Status</th>
              <th className="pb-3 pr-4 font-medium text-zinc-500">Latency</th>
              <th className="pb-3 font-medium text-zinc-500">Size</th>
            </tr>
          </thead>
          <tbody>
            {requests.length === 0 ? (
              <tr>
                <td colSpan={6} className="py-8 text-center text-zinc-500">
                  No requests yet
                </td>
              </tr>
            ) : (
              requests.map((request, index) => {
                const totalSize = request.bytes_sent + request.bytes_received;
                const latencyMs = request.timing.total_ms;
                const statusColor = getStatusColor(request.status_code);
                const methodColor = getMethodColor(request.method);

                return (
                  <tr
                    key={`${request.timestamp}-${index}`}
                    className="border-b border-zinc-100 last:border-0 dark:border-zinc-800"
                  >
                    <td className="py-3 pr-4">
                      <div className="font-mono text-xs text-zinc-900 dark:text-zinc-50">
                        {format(new Date(request.timestamp), 'MMM dd HH:mm:ss')}
                      </div>
                    </td>
                    <td className="py-3 pr-4">
                      <span
                        className="inline-flex items-center gap-1.5 rounded px-2 py-0.5 text-xs font-medium"
                        style={{
                          backgroundColor: `${methodColor}15`,
                          color: methodColor,
                        }}
                      >
                        {getMethodIcon(request.method)}
                        {request.method}
                      </span>
                    </td>
                    <td className="py-3 pr-4">
                      <div className="font-mono text-xs text-zinc-900 dark:text-zinc-50">
                        {request.backend_id || 'unknown'}
                      </div>
                      {request.route && (
                        <div className="text-xs text-zinc-500">{request.route}</div>
                      )}
                    </td>
                    <td className="py-3 pr-4">
                      <div className="flex items-center gap-2">
                        {getStatusIcon(request.status_code)}
                        <span
                          className="font-mono text-xs font-medium"
                          style={{ color: statusColor }}
                        >
                          {request.status_code}
                        </span>
                      </div>
                    </td>
                    <td className="py-3 pr-4">
                      <div className="flex items-center gap-1.5">
                        <div
                          className="h-1.5 w-1.5 rounded-full"
                          style={{
                            backgroundColor:
                              latencyMs < 100
                                ? '#10b981'
                                : latencyMs < 500
                                ? '#f59e0b'
                                : '#ef4444',
                          }}
                        />
                        <span className="font-mono text-xs text-zinc-900 dark:text-zinc-50">
                          {formatDuration(latencyMs)}
                        </span>
                      </div>
                    </td>
                    <td className="py-3">
                      <div className="flex items-center gap-1.5">
                        <div className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
                        <span className="font-mono text-xs text-zinc-900 dark:text-zinc-50">
                          {formatBytes(totalSize)}
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

