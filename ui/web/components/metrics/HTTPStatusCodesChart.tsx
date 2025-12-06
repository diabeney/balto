'use client';

import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { ChartData } from '@/lib/prometheus/types';

interface HTTPStatusCodesChartProps {
  data: ChartData[];
  title?: string;
}

export function HTTPStatusCodesChart({ data, title = 'HTTP Status Codes' }: HTTPStatusCodesChartProps) {
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

  // Transform data for Recharts
  const allTimes = new Set<string>();
  data.forEach(series => {
    series.data.forEach(point => allTimes.add(point.time));
  });
  
  const sortedTimes = Array.from(allTimes).sort();
  const chartData = sortedTimes.map(time => {
    const point: Record<string, string | number> = { time };
    data.forEach(series => {
      const matchingPoint = series.data.find(p => p.time === time);
      point[series.label] = Math.round(matchingPoint?.value || 0);
    });
    return point;
  });

  // Color mapping for status codes
  const getColorForStatus = (status: string): string => {
    if (status.startsWith('2')) return '#10b981'; // green
    if (status.startsWith('3')) return '#3b82f6'; // blue
    if (status.startsWith('4')) return '#f59e0b'; // orange
    if (status.startsWith('5')) return '#ef4444'; // red
    return '#6b7280'; // gray
  };
  
  return (
    <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-6 dark:border-dashed dark:border-zinc-700 dark:bg-zinc-950">
      <h3 className="mb-2 text-xs font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <defs>
            {data.map((series, index) => {
              const color = series.color || getColorForStatus(series.label);
              return (
                <linearGradient key={series.id || `gradient-${index}`} id={`statusGradient${index}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={color} stopOpacity={0.3} />
                  <stop offset="95%" stopColor={color} stopOpacity={0} />
                </linearGradient>
              );
            })}
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
            label={{ value: 'Count', angle: -90, position: 'insideLeft', style: { fontSize: 10, fontFamily: 'var(--font-mono)' } }}
            tickFormatter={(value) => Math.round(value).toString()}
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
            formatter={(value: number) => Math.round(value)}
          />
          <Legend wrapperStyle={{ fontSize: '10px', fontFamily: 'var(--font-mono)' }} />
          {data.map((series, index) => {
            const color = series.color || getColorForStatus(series.label);
            return (
              <Area
                key={series.id || `area-${index}`}
                type="monotone"
                dataKey={series.label}
                stroke={color}
                fill={`url(#statusGradient${index})`}
                strokeWidth={2}
              />
            );
          })}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

