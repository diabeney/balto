'use client';

import { useEffect, useRef, useMemo } from 'react';
import { AreaMetricChart } from '@/components/metrics/AreaMetricChart';
import { MultiAreaChart } from '@/components/metrics/MultiAreaChart';
import { LatencyAreaChart } from '@/components/metrics/LatencyAreaChart';
import { MetricCard, MetricGrid } from '@/components/metrics/MetricCard';
import { RequestsTable } from '@/components/metrics/RequestsTable';
import { 
  useAggregatedMetrics,
  useLatencyBreakdown,
  useSystemMetrics,
  useBackends,
  useTimeSeriesBuffer,
  useMultiSeriesBuffer,
} from '@/lib/metrics/hooks';

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDuration(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
  if (ms < 1000) return `${ms.toFixed(1)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export default function DashboardPage() {
  const { data: metrics, error: metricsError, isLoading: metricsLoading } = useAggregatedMetrics(60);
  const { data: latency, error: latencyError } = useLatencyBreakdown(60);
  const { data: system, error: systemError } = useSystemMetrics();
  const { data: backends, error: backendsError } = useBackends();

  const hasError = metricsError || latencyError || systemError || backendsError;
  const isLoading = metricsLoading && !metrics;

  const requestRateBuffer = useTimeSeriesBuffer(60);
  const latencyBuffer = useTimeSeriesBuffer(60);
  const systemBuffer = useMultiSeriesBuffer(['goroutines', 'heapMB'], 60);
  const statusBuffer = useMultiSeriesBuffer(['2xx', '3xx', '4xx', '5xx'], 60);

  const prevMetricsRef = useRef<typeof metrics>(null);
  const prevSystemRef = useRef<typeof system>(null);
  const lastStatusRef = useRef<{ counts: Record<string, number>; timestamp: number } | null>(null);

  useEffect(() => {
    if (!metrics) return;
    if (prevMetricsRef.current?.timestamp === metrics.timestamp) return;
    
    prevMetricsRef.current = metrics;
    
    requestRateBuffer.addPoint(metrics.request_rate || 0);
    latencyBuffer.addPoint(metrics.avg_latency_ms || 0);
    
    const statusCodes = metrics.status_codes || {};
    let s2xx = 0, s3xx = 0, s4xx = 0, s5xx = 0;
    
    Object.entries(statusCodes).forEach(([code, count]) => {
      const codeNum = parseInt(code, 10);
      if (codeNum >= 200 && codeNum < 300) s2xx += count;
      else if (codeNum >= 300 && codeNum < 400) s3xx += count;
      else if (codeNum >= 400 && codeNum < 500) s4xx += count;
      else if (codeNum >= 500) s5xx += count;
    });
    
    const now = Date.now();
    const last = lastStatusRef.current;
    
    if (last) {
      const timeDeltaSec = Math.max(1, (now - last.timestamp) / 1000);
      const rate2xx = Math.max(0, (s2xx - (last.counts['2xx'] || 0)) / timeDeltaSec);
      const rate3xx = Math.max(0, (s3xx - (last.counts['3xx'] || 0)) / timeDeltaSec);
      const rate4xx = Math.max(0, (s4xx - (last.counts['4xx'] || 0)) / timeDeltaSec);
      const rate5xx = Math.max(0, (s5xx - (last.counts['5xx'] || 0)) / timeDeltaSec);
      
      statusBuffer.addPoint({ '2xx': rate2xx, '3xx': rate3xx, '4xx': rate4xx, '5xx': rate5xx });
    }
    
    lastStatusRef.current = { counts: { '2xx': s2xx, '3xx': s3xx, '4xx': s4xx, '5xx': s5xx }, timestamp: now };
  }, [metrics, requestRateBuffer, latencyBuffer, statusBuffer]);

  useEffect(() => {
    if (!system) return;
    if (prevSystemRef.current?.timestamp === system.timestamp) return;
    
    prevSystemRef.current = system;
    
    systemBuffer.addPoint({
      goroutines: system.goroutines,
      heapMB: system.heap_alloc_bytes / (1024 * 1024),
    });
  }, [system, systemBuffer]);

  const totalRequests = useMemo(() => {
    if (!metrics?.status_codes) return 0;
    return Object.values(metrics.status_codes).reduce((a, b) => a + b, 0);
  }, [metrics]);

  const successRate = useMemo(() => {
    if (!metrics) return 100;
    return metrics.error_rate > 0 ? 100 - metrics.error_rate : 100;
  }, [metrics]);

  const healthyBackends = useMemo(() => {
    const backendList = Array.isArray(backends) ? backends : [];
    if (backendList.length === 0) return { healthy: 0, total: 0 };
    const healthy = backendList.filter((b) => b.healthy).length;
    return { healthy, total: backendList.length };
  }, [backends]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center">
        <div className="text-zinc-500">Loading metrics...</div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-zinc-950">
      <div className="mx-auto max-w-7xl px-6 py-8">
        <header className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-zinc-100">Metrics</h1>
            <p className="mt-1 text-sm text-zinc-500">Real-time performance monitoring</p>
          </div>
          <div className="flex items-center gap-2">
            {hasError ? (
              <>
                <div className="h-2 w-2 rounded-full bg-red-500" />
                <span className="text-xs text-red-400">Connection Error</span>
              </>
            ) : (
              <>
                <div className="h-2 w-2 animate-pulse rounded-full bg-emerald-500" />
                <span className="text-xs text-zinc-500">Live</span>
              </>
            )}
          </div>
        </header>

        {hasError && (
          <div className="mb-6 rounded-lg border border-red-800 bg-red-950/30 p-4">
            <p className="text-sm text-red-400">
              Failed to connect to metrics API. Make sure the Go backend is running on port 5500.
            </p>
          </div>
        )}

        <div className="space-y-6">
          <MetricGrid columns={6}>
            <MetricCard
              label="Request Rate"
              value={`${metrics?.request_rate?.toFixed(1) || '0'}`}
              subValue="requests/sec"
            />
            <MetricCard
              label="Total Requests"
              value={totalRequests.toLocaleString()}
              subValue="in window"
            />
            <MetricCard
              label="Success Rate"
              value={`${successRate.toFixed(1)}%`}
              color={successRate >= 99 ? 'success' : successRate >= 95 ? 'warning' : 'error'}
            />
            <MetricCard
              label="Avg Latency"
              value={formatDuration(metrics?.avg_latency_ms || 0)}
              subValue={`p95: ${formatDuration(metrics?.p95_latency_ms || 0)}`}
            />
            <MetricCard
              label="Backends"
              value={`${healthyBackends.healthy}/${healthyBackends.total}`}
              color={healthyBackends.healthy === healthyBackends.total ? 'success' : 'warning'}
            />
            <MetricCard
              label="Throughput"
              value={formatBytes((metrics?.bytes_sent || 0) + (metrics?.bytes_received || 0))}
              subValue="total transfer"
            />
          </MetricGrid>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <AreaMetricChart
              data={requestRateBuffer.data}
              title="Request Rate"
              subtitle="Requests per second"
              color="#10b981"
              gradientId="requestRateGrad"
              valueFormatter={(v) => `${v.toFixed(1)}/s`}
              currentValue={metrics?.request_rate?.toFixed(1) || '0'}
              unit="/s"
            />
            <AreaMetricChart
              data={latencyBuffer.data}
              title="Response Time"
              subtitle="Average latency"
              color="#a855f7"
              gradientId="latencyGrad"
              valueFormatter={(v) => formatDuration(v)}
              currentValue={formatDuration(metrics?.avg_latency_ms || 0)}
            />
          </div>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <MultiAreaChart
              data={statusBuffer.data}
              series={[
                { key: '2xx', label: '2xx Success', color: '#10b981' },
                { key: '3xx', label: '3xx Redirect', color: '#3b82f6' },
                { key: '4xx', label: '4xx Client Error', color: '#f59e0b' },
                { key: '5xx', label: '5xx Server Error', color: '#ef4444' },
              ]}
              title="HTTP Status Codes"
              subtitle="Responses per second by status"
              valueFormatter={(v) => `${v.toFixed(1)}/s`}
              stacked
            />
            <MultiAreaChart
              data={systemBuffer.data}
              series={[
                { key: 'goroutines', label: 'Goroutines', color: '#8b5cf6' },
                { key: 'heapMB', label: 'Heap (MB)', color: '#06b6d4' },
              ]}
              title="System Resources"
              subtitle="Runtime metrics"
              valueFormatter={(v) => v.toFixed(0)}
            />
          </div>

          <LatencyAreaChart data={latency || null} title="Request Latency Breakdown" />

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            {(Array.isArray(backends) ? backends : []).slice(0, 3).map((backend) => (
              <div
                key={backend.id}
                className="rounded-xl border border-zinc-800 bg-zinc-950 p-5"
              >
                <div className="flex items-center justify-between">
                  <div>
                    <div className="flex items-center gap-2">
                      <div
                        className={`h-2 w-2 rounded-full ${
                          backend.healthy ? 'bg-emerald-500' : 'bg-red-500'
                        }`}
                      />
                      <h3 className="font-mono text-sm font-medium text-zinc-200">
                        {backend.id}
                      </h3>
                    </div>
                    <p className="mt-1 text-xs text-zinc-600">{backend.route || '/'}</p>
                  </div>
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      backend.healthy
                        ? 'bg-emerald-950/50 text-emerald-400'
                        : 'bg-red-950/50 text-red-400'
                    }`}
                  >
                    {backend.healthy ? 'Healthy' : 'Unhealthy'}
                  </span>
                </div>
                <div className="mt-4 grid grid-cols-3 gap-4 border-t border-zinc-800 pt-4">
                  <div>
                    <div className="text-xs text-zinc-500">Requests</div>
                    <div className="font-mono text-sm text-zinc-200">
                      {backend.total_requests.toLocaleString()}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-zinc-500">Avg Time</div>
                    <div className="font-mono text-sm text-zinc-200">
                      {formatDuration(backend.avg_response_time_ms)}
                    </div>
                  </div>
                  <div>
                    <div className="text-xs text-zinc-500">p95</div>
                    <div className="font-mono text-sm text-zinc-200">
                      {formatDuration(backend.p95_response_time_ms)}
          </div>
        </div>
        </div>
        </div>
            ))}
        </div>

          <RequestsTable limit={50} title="Recent Requests" />
        </div>
      </div>
    </div>
  );
}
