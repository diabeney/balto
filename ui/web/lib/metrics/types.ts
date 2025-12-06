export interface RequestTiming {
  dns_lookup_ms: number;
  tcp_connection_ms: number;
  tls_handshake_ms: number;
  server_processing_ms: number;
  content_transfer_ms: number;
  ttfb_ms: number;
  total_ms: number;
}

export interface RequestMetric {
  timestamp: string;
  method: string;
  route: string;
  backend_id: string;
  status_code: number;
  timing: RequestTiming;
  bytes_sent: number;
  bytes_received: number;
}

export interface BackendMetrics {
  id: string;
  route: string;
  healthy: boolean;
  active_connections: number;
  total_requests: number;
  successful_requests: number;
  failed_requests: number;
  avg_response_time_ms: number;
  p50_response_time_ms: number;
  p95_response_time_ms: number;
  p99_response_time_ms: number;
  last_updated: string;
}

export interface AggregatedMetrics {
  timestamp: string;
  window_seconds: number;
  request_rate: number;
  error_rate: number;
  avg_latency_ms: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
  avg_ttfb_ms: number;
  avg_dns_lookup_ms: number;
  avg_tcp_connection_ms: number;
  avg_tls_handshake_ms: number;
  avg_server_processing_ms: number;
  avg_content_transfer_ms: number;
  status_codes: Record<string, number>;
  method_counts: Record<string, number>;
  bytes_sent: number;
  bytes_received: number;
  backends: Record<string, BackendMetrics>;
}

export interface SystemMetrics {
  timestamp: string;
  goroutines: number;
  heap_alloc_bytes: number;
  heap_sys_bytes: number;
  heap_inuse_bytes: number;
  stack_inuse_bytes: number;
  num_gc: number;
  gc_pause_total_ns: number;
  last_gc_pause_ns: number;
  cpu_usage_percent: number;
  open_files: number;
  total_memory_bytes: number;
  available_memory_bytes: number;
}

export interface TimeSeriesPoint {
  timestamp: string;
  value: number;
}

export interface TimeSeries {
  name: string;
  labels: Record<string, string>;
  points: TimeSeriesPoint[];
}

export interface RealtimeSnapshot {
  timestamp: string;
  system: SystemMetrics;
  aggregated: AggregatedMetrics;
  time_series: Record<string, TimeSeries>;
}

export interface MetricsAPIResponse<T> {
  success: boolean;
  timestamp: string;
  data: T;
}

export interface RealtimeEvent {
  type: 'request' | 'backend_health' | 'system' | 'snapshot';
  timestamp: string;
  data: unknown;
}

export interface LatencyBreakdown {
  window_seconds: number;
  avg_total_ms: number;
  p50_ms: number;
  p95_ms: number;
  p99_ms: number;
  avg_ttfb_ms: number;
  avg_dns_lookup_ms: number;
  avg_tcp_connection_ms: number;
  avg_tls_handshake_ms: number;
  avg_server_processing_ms: number;
  avg_content_transfer_ms: number;
}

export interface ChartDataPoint {
  time: string;
  value: number;
  [key: string]: string | number;
}

export interface ChartSeries {
  id: string;
  label: string;
  color: string;
  data: ChartDataPoint[];
}

export type TimeRange = '5m' | '15m' | '30m' | '1h' | '6h' | '12h' | '24h';

export const TIME_RANGE_MINUTES: Record<TimeRange, number> = {
  '5m': 5,
  '15m': 15,
  '30m': 30,
  '1h': 60,
  '6h': 360,
  '12h': 720,
  '24h': 1440,
};

export const TIME_RANGE_LABELS: Record<TimeRange, string> = {
  '5m': '5 minutes',
  '15m': '15 minutes',
  '30m': '30 minutes',
  '1h': '1 hour',
  '6h': '6 hours',
  '12h': '12 hours',
  '24h': '24 hours',
};
