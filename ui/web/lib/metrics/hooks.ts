'use client';

import useSWR from 'swr';
import { useCallback, useEffect, useRef, useState } from 'react';
import type {
  AggregatedMetrics,
  BackendMetrics,
  LatencyBreakdown,
  RealtimeEvent,
  RealtimeSnapshot,
  RequestMetric,
  SystemMetrics,
} from './types';

const GO_API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:5500';

async function fetcher<T>(url: string): Promise<T> {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Failed to fetch: ${res.statusText}`);
  const json = await res.json();
  return json.data ?? json;
}

export function useAggregatedMetrics(windowSeconds = 60) {
  return useSWR<AggregatedMetrics>(
    `/api/metrics/aggregated?window=${windowSeconds}`,
    fetcher,
    { refreshInterval: 2000, revalidateOnFocus: false }
  );
}

export function useLatencyBreakdown(windowSeconds = 60) {
  return useSWR<LatencyBreakdown>(
    `/api/metrics/latency?window=${windowSeconds}`,
    fetcher,
    { refreshInterval: 5000, revalidateOnFocus: false }
  );
}

export function useSystemMetrics() {
  return useSWR<SystemMetrics>(
    '/api/metrics/system',
    fetcher,
    { refreshInterval: 2000, revalidateOnFocus: false }
  );
}

export function useBackends() {
  return useSWR<BackendMetrics[]>(
    '/api/metrics/backends',
    fetcher,
    { refreshInterval: 3000, revalidateOnFocus: false }
  );
}

export function useRecentRequests(limit = 50) {
  return useSWR<RequestMetric[]>(
    `/api/metrics/requests?limit=${limit}`,
    fetcher,
    { refreshInterval: 2000, revalidateOnFocus: false }
  );
}

export function useSnapshot(windowSeconds = 60) {
  return useSWR<RealtimeSnapshot>(
    `/api/metrics/snapshot?window=${windowSeconds}`,
    fetcher,
    { refreshInterval: 2000, revalidateOnFocus: false }
  );
}

export interface TimeSeriesDataPoint {
  time: string;
  value: number;
  timestamp: number;
  [key: string]: string | number;
}

export function useTimeSeriesBuffer(maxPoints = 60) {
  const [data, setData] = useState<TimeSeriesDataPoint[]>([]);
  const dataRef = useRef<TimeSeriesDataPoint[]>([]);

  const addPoint = useCallback((value: number, label?: string) => {
    const now = Date.now();
    const point: TimeSeriesDataPoint = {
      time: new Date(now).toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit' 
      }),
      value,
      timestamp: now,
    };

    dataRef.current = [...dataRef.current, point].slice(-maxPoints);
    setData(dataRef.current);
  }, [maxPoints]);

  const reset = useCallback(() => {
    dataRef.current = [];
    setData([]);
  }, []);

  return { data, addPoint, reset };
}

export interface MultiSeriesDataPoint {
  time: string;
  timestamp: number;
  [key: string]: string | number;
}

export function useMultiSeriesBuffer(series: string[], maxPoints = 60) {
  const [data, setData] = useState<MultiSeriesDataPoint[]>([]);
  const dataRef = useRef<MultiSeriesDataPoint[]>([]);

  const addPoint = useCallback((values: Record<string, number>) => {
    const now = Date.now();
    const point: MultiSeriesDataPoint = {
      time: new Date(now).toLocaleTimeString('en-US', { 
        hour12: false, 
        hour: '2-digit', 
        minute: '2-digit', 
        second: '2-digit' 
      }),
      timestamp: now,
      ...values,
    };

    dataRef.current = [...dataRef.current, point].slice(-maxPoints);
    setData(dataRef.current);
  }, [maxPoints]);

  const reset = useCallback(() => {
    dataRef.current = [];
    setData([]);
  }, []);

  return { data, addPoint, reset };
}

function getWsBaseUrl(): string {
  if (typeof window === 'undefined') return 'ws://localhost:5500';
  return GO_API_BASE.replace(/^http/, 'ws');
}

interface WebSocketOptions {
  intervalMs?: number;
  enabled?: boolean;
  onMessage?: (event: RealtimeEvent) => void;
}

export function useMetricsWebSocket(options: WebSocketOptions = {}) {
  const { intervalMs = 2000, enabled = true, onMessage } = options;
  
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectRef = useRef<NodeJS.Timeout | null>(null);
  const attemptsRef = useRef(0);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    try {
      const ws = new WebSocket(`${getWsBaseUrl()}/balto/metrics/stream?interval=${intervalMs}`);

      ws.onopen = () => {
        setConnected(true);
        setError(null);
        attemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data) as RealtimeEvent;
          onMessage?.(data);
        } catch {}
      };

      ws.onerror = () => setError('Connection error');

      ws.onclose = () => {
        setConnected(false);
        wsRef.current = null;

        if (enabled && attemptsRef.current < 5) {
          reconnectRef.current = setTimeout(() => {
            attemptsRef.current++;
            connect();
          }, Math.min(1000 * Math.pow(2, attemptsRef.current), 30000));
        }
      };

      wsRef.current = ws;
    } catch {
      setError('Failed to connect');
    }
  }, [enabled, intervalMs, onMessage]);

  const disconnect = useCallback(() => {
    if (reconnectRef.current) clearTimeout(reconnectRef.current);
    wsRef.current?.close();
    wsRef.current = null;
    setConnected(false);
  }, []);

  useEffect(() => {
    if (enabled) connect();
    else disconnect();
    return disconnect;
  }, [enabled, connect, disconnect]);

  return { connected, error, connect, disconnect };
}

