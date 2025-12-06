'use client';

import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { ChartData } from '@/lib/prometheus/types';

interface LatencyChartProps {
  data: ChartData[];
  title?: string;
}

export function LatencyChart({ data, title = 'Request Latency' }: LatencyChartProps) {
  // Handle empty data
  if (!data || data.length === 0 || data.every(s => !s.data || s.data.length === 0)) {
    return (
      <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-6 dark:border-dashed dark:border-zinc-700 dark:bg-zinc-950">
        <h3 className="mb-2 text-xs font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[300px] items-center justify-center text-zinc-500 dark:text-zinc-400">
          No data available
        </div>
      </div>
    );
  }

  // Transform data for Recharts - merge all series into a single dataset
  const allTimes = new Set<string>();
  data.forEach(series => {
    series.data.forEach(point => allTimes.add(point.time));
  });
  
  const sortedTimes = Array.from(allTimes).sort();
  const chartData = sortedTimes.map(time => {
    const point: Record<string, string | number> = { time };
    data.forEach(series => {
      const matchingPoint = series.data.find(p => p.time === time);
      point[series.label] = (matchingPoint?.value || 0) * 1000; // Convert to milliseconds
    });
    return point;
  });

  const colors = ['#3b82f6', '#10b981', '#f59e0b'];
  
  return (
    <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-6 dark:border-dashed dark:border-zinc-700 dark:bg-zinc-950">
      <h3 className="mb-2 text-xs font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <defs>
            {data.map((series, index) => (
              <linearGradient key={series.id || `gradient-${index}`} id={`color${index}`} x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={series.color || colors[index % colors.length]} stopOpacity={0.3} />
                <stop offset="95%" stopColor={series.color || colors[index % colors.length]} stopOpacity={0} />
              </linearGradient>
            ))}
          </defs>
          <CartesianGrid strokeDasharray="3 3" className="stroke-zinc-200 dark:stroke-zinc-700" />
          <XAxis 
            dataKey="time" 
            className="text-xs text-zinc-600 dark:text-zinc-400"
            tick={{ fill: 'currentColor', fontFamily: 'var(--font-mono)', fontSize: 10 }}
            label={{ value: 'Time', position: 'insideBottom', offset: -5, style: { fontSize: 10, fontFamily: 'var(--font-mono)' } }}
          />
          <YAxis 
            className="text-xs text-zinc-600 dark:text-zinc-400"
            tick={{ fill: 'currentColor', fontFamily: 'var(--font-mono)', fontSize: 10 }}
            label={{ value: 'ms', angle: -90, position: 'insideLeft', style: { fontSize: 10, fontFamily: 'var(--font-mono)' } }}
          />
          <Tooltip 
            contentStyle={{
              backgroundColor: 'var(--background)',
              border: '1px solid var(--foreground)',
              borderRadius: '6px',
              fontFamily: 'var(--font-mono)',
              fontSize: '10px',
            }}
            labelStyle={{ color: 'var(--foreground)', fontFamily: 'var(--font-mono)', fontSize: '10px' }}
            formatter={(value: number) => `${value.toFixed(2)} ms`}
          />
          <Legend wrapperStyle={{ fontSize: '10px', fontFamily: 'var(--font-mono)' }} />
          {data.map((series, index) => (
            <Area
              key={series.id || `area-${index}`}
              type="monotone"
              dataKey={series.label}
              stroke={series.color || colors[index % colors.length]}
              fill={`url(#color${index})`}
              strokeWidth={2}
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

