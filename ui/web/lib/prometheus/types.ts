export interface PrometheusMetric {
  name: string;
  labels: Record<string, string>;
  value: number;
  timestamp?: number;
}

export interface ParsedMetrics {
  requestsTotal: PrometheusMetric[];
  requestDuration: PrometheusMetric[];
  backendFailures: PrometheusMetric[];
  probeFailures: PrometheusMetric[];
  backendActiveConnections: PrometheusMetric[];
  backendHealthy: PrometheusMetric[];
  systemMemory: PrometheusMetric[];
  systemGoroutines: number;
}

export interface TimeSeriesDataPoint {
  time: string;
  value: number;
  label?: string;
}

export interface ChartData {
  data: TimeSeriesDataPoint[];
  label: string;
  color?: string;
  id?: string; // Optional unique identifier for React keys
}

