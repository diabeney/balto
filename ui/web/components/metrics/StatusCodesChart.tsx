'use client';

import { useEffect, useState } from 'react';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from 'recharts';
import { format } from 'date-fns';
import type { AggregatedMetrics } from '@/lib/metrics/types';
import { STATUS_COLORS, formatNumber } from '@/lib/metrics/transform';

interface StatusData {
  name: string;
  value: number;
  color: string;
  [key: string]: string | number;
}

interface StatusCodesChartProps {
  data: AggregatedMetrics | null;
  title?: string;
}

export function StatusCodesChart({ data, title = 'HTTP Status Codes' }: StatusCodesChartProps) {
  const [chartData, setChartData] = useState<StatusData[]>([]);

  useEffect(() => {
    if (!data?.status_codes) return;

    const groups: Record<string, number> = {
      '2xx': 0,
      '3xx': 0,
      '4xx': 0,
      '5xx': 0,
    };

    for (const [code, count] of Object.entries(data.status_codes)) {
      const codeNum = parseInt(code, 10);
      if (codeNum >= 200 && codeNum < 300) groups['2xx'] += count;
      else if (codeNum >= 300 && codeNum < 400) groups['3xx'] += count;
      else if (codeNum >= 400 && codeNum < 500) groups['4xx'] += count;
      else if (codeNum >= 500) groups['5xx'] += count;
    }

    setChartData(
      Object.entries(groups)
        .filter(([, value]) => value > 0)
        .map(([name, value]) => ({
          name,
          value,
          color: STATUS_COLORS[name as keyof typeof STATUS_COLORS] || STATUS_COLORS.other,
        }))
    );
  }, [data]);

  const total = chartData.reduce((sum, item) => sum + item.value, 0);

  if (!data || chartData.length === 0) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h3 className="mb-4 font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[250px] items-center justify-center text-zinc-500">
          No data available
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="text-xs text-zinc-500">
          Total: <span className="font-mono font-medium">{formatNumber(total)}</span>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <ResponsiveContainer width={200} height={200}>
          <PieChart>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              innerRadius={50}
              outerRadius={80}
              paddingAngle={2}
              dataKey="value"
            >
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color} stroke={entry.color} strokeWidth={2} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--background)',
                border: '1px solid var(--border)',
                borderRadius: '8px',
                fontFamily: 'var(--font-mono)',
                fontSize: '12px',
              }}
              formatter={(value: number) => [formatNumber(value), 'Requests']}
            />
          </PieChart>
        </ResponsiveContainer>

        <div className="flex-1 space-y-3">
          {chartData.map((item) => {
            const percentage = total > 0 ? ((item.value / total) * 100).toFixed(1) : '0';
            return (
              <div key={item.name} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div
                    className="h-3 w-3 rounded-full"
                    style={{ backgroundColor: item.color }}
                  />
                  <span className="font-mono text-sm font-medium text-zinc-900 dark:text-zinc-50">
                    {item.name}
                  </span>
                </div>
                <div className="text-right">
                  <div className="font-mono text-sm font-semibold text-zinc-900 dark:text-zinc-50">
                    {formatNumber(item.value)}
                  </div>
                  <div className="text-xs text-zinc-500">{percentage}%</div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="mt-4 grid grid-cols-4 gap-2 border-t border-zinc-200 pt-4 dark:border-zinc-700">
        {Object.entries(data.status_codes || {})
          .slice(0, 8)
          .map(([code, count]) => (
            <div key={code} className="text-center">
              <div className="font-mono text-xs font-medium text-zinc-500">{code}</div>
              <div className="font-mono text-sm font-semibold text-zinc-900 dark:text-zinc-50">
                {formatNumber(count)}
              </div>
            </div>
          ))}
      </div>
    </div>
  );
}

