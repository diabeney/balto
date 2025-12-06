'use client';

import type { AggregatedMetrics, SystemMetrics } from '@/lib/metrics/types';
import { formatBytes, formatDuration, formatNumber } from '@/lib/metrics/transform';

interface MetricsOverviewProps {
  aggregated: AggregatedMetrics | null;
  system: SystemMetrics | null;
}

interface MetricCardProps {
  label: string;
  value: string | number;
  subValue?: string;
  trend?: 'up' | 'down' | 'neutral';
  color?: 'default' | 'success' | 'warning' | 'error' | 'info';
}

function MetricCard({ label, value, subValue, color = 'default' }: MetricCardProps) {
  const colorClasses = {
    default: 'bg-zinc-50 border-zinc-200 dark:bg-zinc-800 dark:border-zinc-700',
    success: 'bg-emerald-50 border-emerald-200 dark:bg-emerald-900/20 dark:border-emerald-700',
    warning: 'bg-amber-50 border-amber-200 dark:bg-amber-900/20 dark:border-amber-700',
    error: 'bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-700',
    info: 'bg-blue-50 border-blue-200 dark:bg-blue-900/20 dark:border-blue-700',
  };

  const textClasses = {
    default: 'text-zinc-900 dark:text-zinc-50',
    success: 'text-emerald-700 dark:text-emerald-300',
    warning: 'text-amber-700 dark:text-amber-300',
    error: 'text-red-700 dark:text-red-300',
    info: 'text-blue-700 dark:text-blue-300',
  };

  return (
    <div className={`rounded-lg border p-4 ${colorClasses[color]}`}>
      <div className="text-xs text-zinc-500 dark:text-zinc-400">{label}</div>
      <div className={`mt-1 font-mono text-lg font-semibold ${textClasses[color]}`}>{value}</div>
      {subValue && <div className="mt-0.5 text-xs text-zinc-500">{subValue}</div>}
    </div>
  );
}

export function MetricsOverview({ aggregated, system }: MetricsOverviewProps) {
  if (!aggregated) {
    return (
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-6">
        {Array.from({ length: 6 }).map((_, i) => (
          <div
            key={i}
            className="h-[88px] animate-pulse rounded-lg border border-zinc-200 bg-zinc-100 dark:border-zinc-700 dark:bg-zinc-800"
          />
        ))}
      </div>
    );
  }

  const totalRequests = Object.values(aggregated.status_codes || {}).reduce((a, b) => a + b, 0);
  const successRate = aggregated.error_rate > 0 ? 100 - aggregated.error_rate : 100;
  const backendsHealthy = Object.values(aggregated.backends || {}).filter((b) => b.healthy).length;
  const backendsTotal = Object.keys(aggregated.backends || {}).length;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
        <MetricCard
          label="Request Rate"
          value={`${aggregated.request_rate.toFixed(1)}/s`}
          subValue={`${formatNumber(totalRequests)} total`}
          color="info"
        />
        <MetricCard
          label="Success Rate"
          value={`${successRate.toFixed(1)}%`}
          subValue={`${aggregated.error_rate.toFixed(2)}% errors`}
          color={successRate >= 99 ? 'success' : successRate >= 95 ? 'warning' : 'error'}
        />
        <MetricCard
          label="Avg Latency"
          value={formatDuration(aggregated.avg_latency_ms)}
          subValue={`P95: ${formatDuration(aggregated.p95_latency_ms)}`}
        />
        <MetricCard
          label="TTFB"
          value={formatDuration(aggregated.avg_ttfb_ms)}
          subValue="Time to first byte"
        />
        <MetricCard
          label="Backends"
          value={`${backendsHealthy}/${backendsTotal}`}
          subValue={backendsHealthy === backendsTotal ? 'All healthy' : 'Some unhealthy'}
          color={backendsHealthy === backendsTotal ? 'success' : 'warning'}
        />
        <MetricCard
          label="Throughput"
          value={formatBytes(aggregated.bytes_received + aggregated.bytes_sent)}
          subValue={`↓${formatBytes(aggregated.bytes_received)} ↑${formatBytes(aggregated.bytes_sent)}`}
        />
      </div>

      {system && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <MetricCard
            label="Goroutines"
            value={formatNumber(system.goroutines)}
          />
          <MetricCard
            label="Heap Memory"
            value={formatBytes(system.heap_alloc_bytes)}
            subValue={`Sys: ${formatBytes(system.heap_sys_bytes)}`}
          />
          <MetricCard
            label="GC Runs"
            value={formatNumber(system.num_gc)}
            subValue={system.last_gc_pause_ns > 0 ? `Last: ${(system.last_gc_pause_ns / 1e6).toFixed(2)}ms` : undefined}
          />
          <MetricCard
            label="Stack Memory"
            value={formatBytes(system.stack_inuse_bytes)}
          />
        </div>
      )}
    </div>
  );
}

