'use client';

import { useEffect, useState } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { format } from 'date-fns';
import type { SystemMetrics } from '@/lib/metrics/types';
import { formatBytes } from '@/lib/metrics/transform';

interface DataPoint {
  time: string;
  timestamp: number;
  goroutines: number;
  heapAlloc: number;
  heapInuse: number;
}

interface SystemMetricsChartProps {
  data: SystemMetrics | null;
  maxPoints?: number;
  title?: string;
}

export function SystemMetricsChart({
  data,
  maxPoints = 60,
  title = 'System Metrics',
}: SystemMetricsChartProps) {
  const [chartData, setChartData] = useState<DataPoint[]>([]);

  useEffect(() => {
    if (!data) return;

    const now = Date.now();
    const newPoint: DataPoint = {
      time: format(now, 'HH:mm:ss'),
      timestamp: now,
      goroutines: data.goroutines,
      heapAlloc: data.heap_alloc_bytes / (1024 * 1024),
      heapInuse: data.heap_inuse_bytes / (1024 * 1024),
    };

    setChartData((prev) => {
      const updated = [...prev, newPoint];
      if (updated.length > maxPoints) {
        return updated.slice(-maxPoints);
      }
      return updated;
    });
  }, [data, maxPoints]);

  if (!data || chartData.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h3 className="mb-4 font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[300px] items-center justify-center text-zinc-500">
          Waiting for data...
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-full bg-violet-500" />
            <span className="text-zinc-500">Goroutines:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {data.goroutines}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <div className="h-2 w-2 rounded-full bg-emerald-500" />
            <span className="text-zinc-500">Heap:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {formatBytes(data.heap_alloc_bytes)}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">GC:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {data.num_gc}
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <div className="mb-2 text-xs font-medium text-zinc-500">Goroutines</div>
          <ResponsiveContainer width="100%" height={150}>
            <LineChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-zinc-200 dark:stroke-zinc-700" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 9, fontFamily: 'var(--font-mono)' }}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={{ fontSize: 9, fontFamily: 'var(--font-mono)' }}
                width={40}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--background)',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: '11px',
                }}
              />
              <Line
                type="monotone"
                dataKey="goroutines"
                stroke="#8b5cf6"
                strokeWidth={2}
                dot={false}
                animationDuration={300}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div>
          <div className="mb-2 text-xs font-medium text-zinc-500">Memory (MB)</div>
          <ResponsiveContainer width="100%" height={150}>
            <LineChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" className="stroke-zinc-200 dark:stroke-zinc-700" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 9, fontFamily: 'var(--font-mono)' }}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={{ fontSize: 9, fontFamily: 'var(--font-mono)' }}
                width={40}
                tickFormatter={(value) => `${value.toFixed(0)}`}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'var(--background)',
                  border: '1px solid var(--border)',
                  borderRadius: '8px',
                  fontFamily: 'var(--font-mono)',
                  fontSize: '11px',
                }}
                formatter={(value: number) => [`${value.toFixed(2)} MB`]}
              />
              <Line
                type="monotone"
                dataKey="heapAlloc"
                stroke="#10b981"
                strokeWidth={2}
                dot={false}
                name="Heap Alloc"
                animationDuration={300}
              />
              <Line
                type="monotone"
                dataKey="heapInuse"
                stroke="#3b82f6"
                strokeWidth={2}
                dot={false}
                name="Heap In Use"
                animationDuration={300}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  );
}

