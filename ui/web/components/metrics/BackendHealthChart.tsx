'use client';

import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, Cell } from 'recharts';
import { PrometheusMetric } from '@/lib/prometheus/types';

interface BackendHealthChartProps {
  data: PrometheusMetric[];
  title?: string;
}

export function BackendHealthChart({ data, title = 'Backend Health Status' }: BackendHealthChartProps) {
  const chartData = data.map((metric) => ({
    backend: metric.labels.backend_id || 'unknown',
    healthy: metric.value === 1 ? 'Healthy' : 'Unhealthy',
    value: metric.value,
  }));

  return (
    <div className="rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-6 dark:border-dashed dark:border-zinc-700 dark:bg-zinc-950">
      <h3 className="mb-2 text-xs font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
      <ResponsiveContainer width="100%" height={300}>
        <BarChart data={chartData} margin={{ top: 5, right: 30, left: 20, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" className="stroke-zinc-200 dark:stroke-zinc-700" />
          <XAxis 
            dataKey="backend" 
            className="text-xs text-zinc-600 dark:text-zinc-400"
            tick={{ fill: 'currentColor', fontFamily: 'var(--font-mono)', fontSize: 10 }}
            angle={-45}
            textAnchor="end"
            height={80}
            label={{ value: 'Backend', position: 'insideBottom', offset: -5, style: { fontSize: 10, fontFamily: 'var(--font-mono)' } }}
          />
          <YAxis 
            className="text-xs text-zinc-600 dark:text-zinc-400"
            tick={{ fill: 'currentColor', fontFamily: 'var(--font-mono)', fontSize: 10 }}
            domain={[0, 1]}
            label={{ value: 'Health', angle: -90, position: 'insideLeft', style: { fontSize: 10, fontFamily: 'var(--font-mono)' } }}
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
          />
          <Legend wrapperStyle={{ fontSize: '10px', fontFamily: 'var(--font-mono)' }} />
          <Bar dataKey="value" name="Health Status">
            {chartData.map((entry, index) => (
              <Cell 
                key={`cell-${index}`} 
                fill={entry.value === 1 ? '#10b981' : '#ef4444'} 
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

