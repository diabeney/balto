'use client';

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { ChartData } from '@/lib/prometheus/types';

interface TotalRequestsChartProps {
  data: ChartData[];
  title?: string;
}

export function TotalRequestsChart({ data, title = 'Total HTTP Requests' }: TotalRequestsChartProps) {
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

  // Sum all series for each time point
  const allTimes = new Set<string>();
  data.forEach(series => {
    series.data.forEach(point => allTimes.add(point.time));
  });
  
  const sortedTimes = Array.from(allTimes).sort();
  const chartData = sortedTimes.map(time => {
    let total = 0;
    data.forEach(series => {
      const matchingPoint = series.data.find(p => p.time === time);
      total += matchingPoint?.value || 0;
    });
    return { time, total: Math.round(total) };
  });
  
  return (
    <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-6 dark:border-dashed dark:border-zinc-700 dark:bg-zinc-950">
        <h3 className="mb-2 text-xs font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
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
          <Bar dataKey="total" name="Total Requests">
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill="#3b82f6" />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

