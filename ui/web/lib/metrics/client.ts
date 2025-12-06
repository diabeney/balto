import type {
  AggregatedMetrics,
  BackendMetrics,
  LatencyBreakdown,
  MetricsAPIResponse,
  RealtimeSnapshot,
  RequestMetric,
  SystemMetrics,
  TimeSeries,
} from './types';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:5500';
const BROWSER_API_BASE = typeof window !== 'undefined' 
  ? (process.env.NEXT_PUBLIC_APP_URL || '') 
  : API_BASE;

function getBaseUrl(): string {
  if (typeof window !== 'undefined') {
    return BROWSER_API_BASE || '';
  }
  return API_BASE;
}

async function fetchJSON<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const baseUrl = getBaseUrl();
  const url = `${baseUrl}${endpoint}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }

  return response.json();
}

export async function fetchAggregatedMetrics(windowSeconds = 60): Promise<AggregatedMetrics> {
  const response = await fetchJSON<MetricsAPIResponse<AggregatedMetrics>>(
    `/api/metrics/aggregated?window=${windowSeconds}`
  );
  return response.data;
}

export async function fetchTimeSeries(name: string, windowMinutes = 60): Promise<TimeSeries> {
  const response = await fetchJSON<MetricsAPIResponse<TimeSeries>>(
    `/api/metrics/timeseries?name=${name}&window=${windowMinutes}`
  );
  return response.data;
}

export async function fetchAllTimeSeries(windowMinutes = 60): Promise<Record<string, TimeSeries>> {
  const response = await fetchJSON<MetricsAPIResponse<Record<string, TimeSeries>>>(
    `/api/metrics/timeseries/all?window=${windowMinutes}`
  );
  return response.data;
}

export async function fetchSnapshot(windowSeconds = 60): Promise<RealtimeSnapshot> {
  const response = await fetchJSON<MetricsAPIResponse<RealtimeSnapshot>>(
    `/api/metrics/snapshot?window=${windowSeconds}`
  );
  return response.data;
}

export async function fetchBackends(): Promise<BackendMetrics[]> {
  const response = await fetchJSON<MetricsAPIResponse<BackendMetrics[]>>(
    '/api/metrics/backends'
  );
  return response.data;
}

export async function fetchLatencyBreakdown(windowSeconds = 60): Promise<LatencyBreakdown> {
  const response = await fetchJSON<MetricsAPIResponse<LatencyBreakdown>>(
    `/api/metrics/latency?window=${windowSeconds}`
  );
  return response.data;
}

export async function fetchSystemMetrics(): Promise<SystemMetrics> {
  const response = await fetchJSON<MetricsAPIResponse<SystemMetrics>>(
    '/api/metrics/system'
  );
  return response.data;
}

export async function fetchRecentRequests(limit = 100): Promise<RequestMetric[]> {
  const response = await fetchJSON<MetricsAPIResponse<RequestMetric[]>>(
    `/api/metrics/requests?limit=${limit}`
  );
  return response.data;
}

export async function fetchPrometheusMetrics(): Promise<string> {
  const response = await fetch(`${API_BASE}/balto/metrics`, {
    cache: 'no-store',
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch metrics: ${response.statusText}`);
  }

  return response.text();
}

export function createMetricsWebSocket(
  onMessage: (data: unknown) => void,
  onError?: (error: Event) => void,
  onClose?: () => void,
  intervalMs = 1000
): WebSocket {
  const wsUrl = API_BASE.replace(/^http/, 'ws');
  const ws = new WebSocket(`${wsUrl}/balto/metrics/stream?interval=${intervalMs}`);

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      onMessage(data);
    } catch (e) {
      console.error('Failed to parse WebSocket message:', e);
    }
  };

  if (onError) {
    ws.onerror = onError;
  }

  if (onClose) {
    ws.onclose = onClose;
  }

  return ws;
}

