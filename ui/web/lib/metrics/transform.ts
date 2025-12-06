import { format } from 'date-fns';
import type {
  AggregatedMetrics,
  BackendMetrics,
  ChartDataPoint,
  ChartSeries,
  LatencyBreakdown,
  TimeSeries,
  TimeRange,
} from './types';
import { TIME_RANGE_MINUTES } from './types';

export function formatTimestamp(timestamp: string, range: TimeRange): string {
  const date = new Date(timestamp);
  const minutes = TIME_RANGE_MINUTES[range];

  if (minutes <= 60) {
    return format(date, 'HH:mm:ss');
  } else if (minutes <= 360) {
    return format(date, 'HH:mm');
  } else {
    return format(date, 'MM/dd HH:mm');
  }
}

export function timeSeriestoChartData(
  series: TimeSeries,
  range: TimeRange,
  color: string = '#3b82f6'
): ChartSeries {
  return {
    id: series.name,
    label: formatSeriesName(series.name),
    color,
    data: series.points.map((point) => ({
      time: formatTimestamp(point.timestamp, range),
      value: point.value,
    })),
  };
}

export function formatSeriesName(name: string): string {
  const nameMap: Record<string, string> = {
    requests: 'Requests',
    ttfb: 'TTFB',
    dns_lookup: 'DNS Lookup',
    tcp_connection: 'TCP Connection',
    tls_handshake: 'TLS Handshake',
    server_processing: 'Server Processing',
    content_transfer: 'Content Transfer',
    goroutines: 'Goroutines',
    heap_alloc: 'Heap Alloc',
    heap_sys: 'Heap System',
    cpu: 'CPU Usage',
  };
  return nameMap[name] || name;
}

export function aggregatedToStatusCodeChart(
  aggregated: AggregatedMetrics,
  timestamp: string,
  range: TimeRange
): ChartSeries[] {
  const colors: Record<string, string> = {
    '2xx': '#10b981',
    '3xx': '#3b82f6',
    '4xx': '#f59e0b',
    '5xx': '#ef4444',
  };

  const statusGroups: Record<string, number> = {
    '2xx': 0,
    '3xx': 0,
    '4xx': 0,
    '5xx': 0,
  };

  for (const [code, count] of Object.entries(aggregated.status_codes || {})) {
    const codeNum = parseInt(code, 10);
    if (codeNum >= 200 && codeNum < 300) statusGroups['2xx'] += count;
    else if (codeNum >= 300 && codeNum < 400) statusGroups['3xx'] += count;
    else if (codeNum >= 400 && codeNum < 500) statusGroups['4xx'] += count;
    else if (codeNum >= 500) statusGroups['5xx'] += count;
  }

  return Object.entries(statusGroups).map(([group, value]) => ({
    id: group,
    label: group,
    color: colors[group],
    data: [{ time: formatTimestamp(timestamp, range), value }],
  }));
}

export function aggregatedToMethodChart(
  aggregated: AggregatedMetrics,
  timestamp: string,
  range: TimeRange
): ChartSeries[] {
  const colors: Record<string, string> = {
    GET: '#3b82f6',
    POST: '#10b981',
    PUT: '#f59e0b',
    DELETE: '#ef4444',
    PATCH: '#8b5cf6',
    OPTIONS: '#6b7280',
    HEAD: '#6b7280',
  };

  return Object.entries(aggregated.method_counts || {}).map(([method, count]) => ({
    id: method,
    label: method,
    color: colors[method] || '#6b7280',
    data: [{ time: formatTimestamp(timestamp, range), value: count }],
  }));
}

export function latencyBreakdownToChart(breakdown: LatencyBreakdown): ChartDataPoint[] {
  return [
    { time: 'DNS', value: breakdown.avg_dns_lookup_ms, phase: 'DNS Lookup' },
    { time: 'TCP', value: breakdown.avg_tcp_connection_ms, phase: 'TCP Connection' },
    { time: 'TLS', value: breakdown.avg_tls_handshake_ms, phase: 'TLS Handshake' },
    { time: 'Server', value: breakdown.avg_server_processing_ms, phase: 'Server Processing' },
    { time: 'Transfer', value: breakdown.avg_content_transfer_ms, phase: 'Content Transfer' },
  ];
}

export function backendsToHealthChart(backends: BackendMetrics[]): { healthy: number; unhealthy: number; total: number } {
  const healthy = backends.filter((b) => b.healthy).length;
  return {
    healthy,
    unhealthy: backends.length - healthy,
    total: backends.length,
  };
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function formatDuration(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
  if (ms < 1000) return `${ms.toFixed(2)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toFixed(0);
}

export function calculatePercentChange(current: number, previous: number): number {
  if (previous === 0) return current > 0 ? 100 : 0;
  return ((current - previous) / previous) * 100;
}

export function mergeChartSeries(
  existingSeries: ChartSeries[],
  newPoints: ChartDataPoint[],
  maxPoints = 60
): ChartSeries[] {
  return existingSeries.map((series) => {
    const newPoint = newPoints.find((p) => p.label === series.label);
    if (!newPoint) return series;

    const data = [...series.data, newPoint];
    if (data.length > maxPoints) {
      data.splice(0, data.length - maxPoints);
    }

    return { ...series, data };
  });
}

export const LATENCY_COLORS = {
  dns_lookup: '#6366f1',
  tcp_connection: '#8b5cf6',
  tls_handshake: '#a855f7',
  server_processing: '#d946ef',
  content_transfer: '#ec4899',
  ttfb: '#f43f5e',
  total: '#3b82f6',
};

export const STATUS_COLORS = {
  '2xx': '#10b981',
  '3xx': '#3b82f6',
  '4xx': '#f59e0b',
  '5xx': '#ef4444',
  other: '#6b7280',
};

export const METHOD_COLORS = {
  GET: '#3b82f6',
  POST: '#10b981',
  PUT: '#f59e0b',
  DELETE: '#ef4444',
  PATCH: '#8b5cf6',
  OPTIONS: '#6b7280',
  HEAD: '#64748b',
};

