'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { AggregatedMetrics, RealtimeEvent, SystemMetrics } from './types';

function getWsBaseUrl(): string {
  if (typeof window === 'undefined') return 'ws://localhost:5500';
  
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:5500';
  return apiUrl.replace(/^http/, 'ws');
}

interface UseRealtimeMetricsOptions {
  intervalMs?: number;
  enabled?: boolean;
  onSnapshot?: (data: AggregatedMetrics) => void;
  onSystem?: (data: SystemMetrics) => void;
  onRequest?: (data: unknown) => void;
  onBackendHealth?: (data: { backend_id: string; route: string; healthy: boolean }) => void;
}

interface UseRealtimeMetricsReturn {
  connected: boolean;
  lastSnapshot: AggregatedMetrics | null;
  lastSystem: SystemMetrics | null;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
}

export function useRealtimeMetrics(options: UseRealtimeMetricsOptions = {}): UseRealtimeMetricsReturn {
  const {
    intervalMs = 1000,
    enabled = true,
    onSnapshot,
    onSystem,
    onRequest,
    onBackendHealth,
  } = options;

  const [connected, setConnected] = useState(false);
  const [lastSnapshot, setLastSnapshot] = useState<AggregatedMetrics | null>(null);
  const [lastSystem, setLastSystem] = useState<SystemMetrics | null>(null);
  const [error, setError] = useState<string | null>(null);
  
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const data = JSON.parse(event.data) as RealtimeEvent;
      
      switch (data.type) {
        case 'snapshot':
          const snapshotData = data.data as AggregatedMetrics;
          setLastSnapshot(snapshotData);
          onSnapshot?.(snapshotData);
          break;
        case 'system':
          const systemData = data.data as SystemMetrics;
          setLastSystem(systemData);
          onSystem?.(systemData);
          break;
        case 'request':
          onRequest?.(data.data);
          break;
        case 'backend_health':
          const healthData = data.data as { backend_id: string; route: string; healthy: boolean };
          onBackendHealth?.(healthData);
          break;
      }
    } catch (e) {
      console.error('Failed to parse WebSocket message:', e);
    }
  }, [onSnapshot, onSystem, onRequest, onBackendHealth]);

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      const wsBase = getWsBaseUrl();
      const ws = new WebSocket(`${wsBase}/balto/metrics/stream?interval=${intervalMs}`);
      
      ws.onopen = () => {
        setConnected(true);
        setError(null);
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = handleMessage;

      ws.onerror = () => {
        setError('WebSocket connection error');
      };

      ws.onclose = () => {
        setConnected(false);
        wsRef.current = null;

        if (enabled && reconnectAttemptsRef.current < 5) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttemptsRef.current++;
            connect();
          }, delay);
        }
      };

      wsRef.current = ws;
    } catch (e) {
      setError('Failed to create WebSocket connection');
    }
  }, [enabled, intervalMs, handleMessage]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setConnected(false);
  }, []);

  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      disconnect();
    };
  }, [enabled, connect, disconnect]);

  return {
    connected,
    lastSnapshot,
    lastSystem,
    error,
    connect,
    disconnect,
  };
}

export function useMetricsPolling<T>(
  fetcher: () => Promise<T>,
  intervalMs = 5000,
  enabled = true
): { data: T | null; loading: boolean; error: string | null; refetch: () => Promise<void> } {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refetch = useCallback(async () => {
    try {
      setLoading(true);
      const result = await fetcher();
      setData(result);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to fetch data');
    } finally {
      setLoading(false);
    }
  }, [fetcher]);

  useEffect(() => {
    if (!enabled) return;

    refetch();
    const interval = setInterval(refetch, intervalMs);
    return () => clearInterval(interval);
  }, [enabled, intervalMs, refetch]);

  return { data, loading, error, refetch };
}

