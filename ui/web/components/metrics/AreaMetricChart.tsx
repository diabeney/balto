'use client';

import { useMemo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface DataPoint {
  time: string;
  value: number;
  [key: string]: string | number;
}

interface AreaMetricChartProps {
  data: DataPoint[];
  title: string;
  subtitle?: string;
  dataKey?: string;
  color?: string;
  gradientId?: string;
  valueFormatter?: (value: number) => string;
  yAxisFormatter?: (value: number) => string;
  showYAxis?: boolean;
  height?: number;
  currentValue?: string | number;
  unit?: string;
}

export function AreaMetricChart({
  data,
  title,
  subtitle,
  dataKey = 'value',
  color = '#10b981',
  gradientId = 'areaGradient',
  valueFormatter = (v) => v.toFixed(2),
  yAxisFormatter,
  showYAxis = false,
  height = 200,
  currentValue,
  unit,
}: AreaMetricChartProps) {
  const chartData = useMemo(() => data, [data]);

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-5">
      <div className="mb-4 flex items-start justify-between">
        <div>
          <h3 className="text-sm font-medium text-zinc-400">{title}</h3>
          {subtitle && (
            <p className="mt-0.5 text-xs text-zinc-600">{subtitle}</p>
          )}
        </div>
        {currentValue !== undefined && (
          <div className="text-right">
            <span className="font-mono text-2xl font-semibold text-zinc-100">
              {currentValue}
            </span>
            {unit && <span className="ml-1 text-sm text-zinc-500">{unit}</span>}
          </div>
        )}
      </div>

      <div style={{ height, minHeight: height }}>
        <ResponsiveContainer width="100%" height={height} minHeight={height}>
          <AreaChart
            data={chartData}
            margin={{ top: 0, right: 0, left: showYAxis ? -20 : 0, bottom: 0 }}
          >
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis
              dataKey="time"
              axisLine={false}
              tickLine={false}
              tick={{ fontSize: 10, fill: '#52525b', fontFamily: 'var(--font-mono)' }}
              interval="preserveStartEnd"
              minTickGap={50}
            />
            {showYAxis && (
              <YAxis
                axisLine={false}
                tickLine={false}
                tick={{ fontSize: 10, fill: '#52525b', fontFamily: 'var(--font-mono)' }}
                tickFormatter={yAxisFormatter}
                width={40}
              />
            )}
            <Tooltip
              contentStyle={{
                backgroundColor: '#18181b',
                border: '1px solid #27272a',
                borderRadius: '8px',
                padding: '8px 12px',
                fontSize: '12px',
              }}
              labelStyle={{ color: '#71717a', fontSize: '11px', marginBottom: '4px' }}
              formatter={(value: number) => [
                <span key="value" className="font-mono text-xs" style={{ color }}>
                  {valueFormatter(value)}{unit ? ` ${unit}` : ''}
                </span>,
                null,
              ]}
              cursor={{ stroke: '#3f3f46', strokeWidth: 1 }}
            />
            <Area
              type="monotone"
              dataKey={dataKey}
              stroke={color}
              strokeWidth={1.5}
              fill={`url(#${gradientId})`}
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

