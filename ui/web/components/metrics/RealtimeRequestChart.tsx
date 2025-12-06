'use client';

import { useEffect, useState, useCallback } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { format } from 'date-fns';
import type { AggregatedMetrics } from '@/lib/metrics/types';

interface DataPoint {
  time: string;
  timestamp: number;
  requestRate: number;
  errorRate: number;
}

interface RealtimeRequestChartProps {
  data: AggregatedMetrics | null;
  maxPoints?: number;
  title?: string;
}

export function RealtimeRequestChart({
  data,
  maxPoints = 60,
  title = 'Request Rate',
}: RealtimeRequestChartProps) {
  const [chartData, setChartData] = useState<DataPoint[]>([]);

  useEffect(() => {
    if (!data) return;

    const now = Date.now();
    const newPoint: DataPoint = {
      time: format(now, 'HH:mm:ss'),
      timestamp: now,
      requestRate: data.request_rate,
      errorRate: data.error_rate,
    };

    setChartData((prev) => {
      const updated = [...prev, newPoint];
      if (updated.length > maxPoints) {
        return updated.slice(-maxPoints);
      }
      return updated;
    });
  }, [data, maxPoints]);

  if (chartData.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h3 className="mb-4 font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[300px] items-center justify-center text-zinc-500">
          Waiting for data...
        </div>
      </div>
    );
  }

  const currentRate = chartData[chartData.length - 1]?.requestRate || 0;
  const avgRate = chartData.reduce((sum, p) => sum + p.requestRate, 0) / chartData.length;
  const maxRate = Math.max(...chartData.map((p) => p.requestRate));

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">Current:</span>
            <span className="font-mono font-medium text-blue-600">{currentRate.toFixed(1)}/s</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">Avg:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {avgRate.toFixed(1)}/s
            </span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">Max:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {maxRate.toFixed(1)}/s
            </span>
          </div>
        </div>
      </div>

      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <defs>
            <linearGradient id="requestRateGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
            </linearGradient>
            <linearGradient id="errorRateGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#ef4444" stopOpacity={0.3} />
              <stop offset="95%" stopColor="#ef4444" stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" className="stroke-zinc-200 dark:stroke-zinc-700" />
          <XAxis
            dataKey="time"
            tick={{ fontSize: 10, fontFamily: 'var(--font-mono)' }}
            interval="preserveStartEnd"
          />
          <YAxis
            tick={{ fontSize: 10, fontFamily: 'var(--font-mono)' }}
            tickFormatter={(value) => `${value.toFixed(0)}`}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'var(--background)',
              border: '1px solid var(--border)',
              borderRadius: '8px',
              fontFamily: 'var(--font-mono)',
              fontSize: '12px',
            }}
            formatter={(value: number, name: string) => [
              `${value.toFixed(2)}${name === 'errorRate' ? '%' : '/s'}`,
              name === 'requestRate' ? 'Request Rate' : 'Error Rate',
            ]}
          />
          <Legend wrapperStyle={{ fontSize: '12px', fontFamily: 'var(--font-mono)' }} />
          <Area
            type="monotone"
            dataKey="requestRate"
            stroke="#3b82f6"
            fill="url(#requestRateGrad)"
            strokeWidth={2}
            name="Requests/s"
            dot={false}
            animationDuration={300}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

