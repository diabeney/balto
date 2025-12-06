'use client';

import { useMemo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';

interface DataPoint {
  time: string;
  [key: string]: string | number;
}

interface SeriesConfig {
  key: string;
  label: string;
  color: string;
}

interface MultiAreaChartProps {
  data: DataPoint[];
  series: SeriesConfig[];
  title: string;
  subtitle?: string;
  valueFormatter?: (value: number) => string;
  height?: number;
  stacked?: boolean;
  showLegend?: boolean;
}

export function MultiAreaChart({
  data,
  series,
  title,
  subtitle,
  valueFormatter = (v) => v.toFixed(0),
  height = 200,
  stacked = false,
  showLegend = true,
}: MultiAreaChartProps) {
  const chartData = useMemo(() => data, [data]);

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-5">
      <div className="mb-4">
        <h3 className="text-sm font-medium text-zinc-400">{title}</h3>
        {subtitle && (
          <p className="mt-0.5 text-xs text-zinc-600">{subtitle}</p>
        )}
      </div>

      {showLegend && (
        <div className="mb-3 flex flex-wrap gap-4">
          {series.map((s) => (
            <div key={s.key} className="flex items-center gap-2">
              <div
                className="h-2 w-2 rounded-full"
                style={{ backgroundColor: s.color }}
              />
              <span className="text-xs text-zinc-500">{s.label}</span>
            </div>
          ))}
        </div>
      )}

      <div style={{ height, minHeight: height }}>
        <ResponsiveContainer width="100%" height={height} minHeight={height}>
          <AreaChart
            data={chartData}
            margin={{ top: 0, right: 0, left: 0, bottom: 0 }}
          >
            <defs>
              {series.map((s) => (
                <linearGradient
                  key={`gradient-${s.key}`}
                  id={`gradient-${s.key}`}
                  x1="0"
                  y1="0"
                  x2="0"
                  y2="1"
                >
                  <stop offset="0%" stopColor={s.color} stopOpacity={0.3} />
                  <stop offset="100%" stopColor={s.color} stopOpacity={0} />
                </linearGradient>
              ))}
            </defs>
            <XAxis
              dataKey="time"
              axisLine={false}
              tickLine={false}
              tick={{ fontSize: 10, fill: '#52525b', fontFamily: 'var(--font-mono)' }}
              interval="preserveStartEnd"
              minTickGap={50}
            />
            <YAxis
              axisLine={false}
              tickLine={false}
              tick={{ fontSize: 10, fill: '#52525b', fontFamily: 'var(--font-mono)' }}
              width={35}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#18181b',
                border: '1px solid #27272a',
                borderRadius: '8px',
                padding: '8px 12px',
                fontSize: '12px',
              }}
              labelStyle={{ color: '#71717a', fontSize: '11px', marginBottom: '4px' }}
              itemStyle={{ fontSize: '12px', padding: '2px 0' }}
              formatter={(value: number, name: string) => {
                const seriesConfig = series.find((s) => s.key === name);
                return [
                  <span key={name} className="font-mono text-xs" style={{ color: seriesConfig?.color }}>
                    {valueFormatter(value)}
                  </span>,
                  <span key={`label-${name}`} style={{ color: seriesConfig?.color }}>{seriesConfig?.label || name}</span>,
                ];
              }}
              cursor={{ stroke: '#3f3f46', strokeWidth: 1 }}
            />
            {series.map((s) => (
              <Area
                key={s.key}
                type="monotone"
                dataKey={s.key}
                stroke={s.color}
                strokeWidth={1.5}
                fill={`url(#gradient-${s.key})`}
                stackId={stacked ? 'stack' : undefined}
                isAnimationActive={false}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

