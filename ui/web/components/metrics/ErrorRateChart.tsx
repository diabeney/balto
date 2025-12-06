'use client';

import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import { ChartData } from '@/lib/prometheus/types';

interface ErrorRateChartProps {
  data: ChartData[];
  title?: string;
}

export function ErrorRateChart({ data, title = 'Error Rate' }: ErrorRateChartProps) {
  const chartData = data.flatMap((series) =>
    series.data.map((point) => ({
      time: point.time,
      [series.label]: point.value,
    }))
  );

  const colors = ['#ef4444', '#f59e0b', '#8b5cf6'];
  
  return (
    <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-6 dark:border-dashed dark:border-zinc-700 dark:bg-zinc-950">
      <h3 className="mb-2 text-xs font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <AreaChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <defs>
            {data.map((series, index) => (
              <linearGradient key={series.id || `gradient-${index}`} id={`errorRateGradient${index}`} x1="0" y1="0" x2="0" y2="1">
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
            label={{ value: '%', angle: -90, position: 'insideLeft', style: { fontSize: 10, fontFamily: 'var(--font-mono)' } }}
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
            formatter={(value: number) => `${value.toFixed(2)}%`}
          />
          <Legend wrapperStyle={{ fontSize: '10px', fontFamily: 'var(--font-mono)' }} />
          {data.map((series, index) => (
            <Area
              key={series.id || `area-${index}`}
              type="monotone"
              dataKey={series.label}
              stroke={series.color || colors[index % colors.length]}
              fill={`url(#errorRateGradient${index})`}
              strokeWidth={2}
            />
          ))}
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

