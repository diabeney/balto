'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
  Legend,
} from 'recharts';
import type { LatencyBreakdown } from '@/lib/metrics/types';
import { LATENCY_COLORS, formatDuration } from '@/lib/metrics/transform';

interface LatencyBreakdownChartProps {
  data: LatencyBreakdown | null;
  title?: string;
}

export function LatencyBreakdownChart({
  data,
  title = 'Latency Breakdown',
}: LatencyBreakdownChartProps) {
  if (!data) {
    return (
      <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
        <h3 className="mb-4 font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex h-[300px] items-center justify-center text-zinc-500">
          Loading...
        </div>
      </div>
    );
  }

  const chartData = [
    {
      name: 'DNS',
      fullName: 'DNS Lookup',
      value: data.avg_dns_lookup_ms,
      color: LATENCY_COLORS.dns_lookup,
    },
    {
      name: 'TCP',
      fullName: 'TCP Connection',
      value: data.avg_tcp_connection_ms,
      color: LATENCY_COLORS.tcp_connection,
    },
    {
      name: 'TLS',
      fullName: 'TLS Handshake',
      value: data.avg_tls_handshake_ms,
      color: LATENCY_COLORS.tls_handshake,
    },
    {
      name: 'Server',
      fullName: 'Server Processing',
      value: data.avg_server_processing_ms,
      color: LATENCY_COLORS.server_processing,
    },
    {
      name: 'Transfer',
      fullName: 'Content Transfer',
      value: data.avg_content_transfer_ms,
      color: LATENCY_COLORS.content_transfer,
    },
  ];

  const totalLatency = data.avg_total_ms;

  return (
    <div className="rounded-xl border border-zinc-200 bg-white p-6 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-semibold text-zinc-900 dark:text-zinc-50">{title}</h3>
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">TTFB:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {formatDuration(data.avg_ttfb_ms)}
            </span>
          </div>
          <div className="flex items-center gap-1">
            <span className="text-zinc-500">Total:</span>
            <span className="font-mono font-medium text-zinc-900 dark:text-zinc-50">
              {formatDuration(totalLatency)}
            </span>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-5 gap-2 mb-4">
        {chartData.map((item) => (
          <div
            key={item.name}
            className="rounded-lg p-3 text-center"
            style={{ backgroundColor: `${item.color}15` }}
          >
            <div className="text-xs text-zinc-500 dark:text-zinc-400">{item.fullName}</div>
            <div
              className="font-mono text-sm font-semibold"
              style={{ color: item.color }}
            >
              {formatDuration(item.value)}
            </div>
          </div>
        ))}
      </div>

      <ResponsiveContainer width="100%" height={200}>
        <BarChart
          data={chartData}
          layout="vertical"
          margin={{ top: 5, right: 30, left: 60, bottom: 5 }}
        >
          <CartesianGrid strokeDasharray="3 3" className="stroke-zinc-200 dark:stroke-zinc-700" />
          <XAxis
            type="number"
            tick={{ fontSize: 10, fontFamily: 'var(--font-mono)' }}
            tickFormatter={(value) => `${value.toFixed(1)}ms`}
          />
          <YAxis
            type="category"
            dataKey="name"
            tick={{ fontSize: 10, fontFamily: 'var(--font-mono)' }}
            width={50}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'var(--background)',
              border: '1px solid var(--border)',
              borderRadius: '8px',
              fontFamily: 'var(--font-mono)',
              fontSize: '12px',
            }}
            formatter={(value: number, name: string, props: any) => {
              const fullName = props?.payload?.fullName || name;
              return [formatDuration(value), fullName];
            }}
          />
          <Bar dataKey="value" radius={[0, 4, 4, 0]}>
            {chartData.map((entry, index) => (
              <Cell key={`cell-${index}`} fill={entry.color} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>

      <div className="mt-4 grid grid-cols-3 gap-4 border-t border-zinc-200 pt-4 dark:border-zinc-700">
        <div className="text-center">
          <div className="text-xs text-zinc-500">P50</div>
          <div className="font-mono text-sm font-semibold text-zinc-900 dark:text-zinc-50">
            {formatDuration(data.p50_ms)}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs text-zinc-500">P95</div>
          <div className="font-mono text-sm font-semibold text-zinc-900 dark:text-zinc-50">
            {formatDuration(data.p95_ms)}
          </div>
        </div>
        <div className="text-center">
          <div className="text-xs text-zinc-500">P99</div>
          <div className="font-mono text-sm font-semibold text-zinc-900 dark:text-zinc-50">
            {formatDuration(data.p99_ms)}
          </div>
        </div>
      </div>
    </div>
  );
}

