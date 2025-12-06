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
import type { LatencyBreakdown } from '@/lib/metrics/types';

interface LatencyAreaChartProps {
  data: LatencyBreakdown | null;
  title?: string;
}

const PHASES = [
  { key: 'dns', label: 'DNS Lookup', color: '#8b5cf6' },
  { key: 'tcp', label: 'TCP Connect', color: '#3b82f6' },
  { key: 'tls', label: 'TLS Handshake', color: '#eab308' },
  { key: 'server', label: 'Server', color: '#f97316' },
  { key: 'transfer', label: 'Transfer', color: '#ef4444' },
];

function formatMs(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
  if (ms < 1000) return `${ms.toFixed(1)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export function LatencyAreaChart({ data, title = 'Request Latency' }: LatencyAreaChartProps) {
  const chartData = useMemo(() => {
    if (!data) return [];
    return [
      { phase: 'DNS', value: data.avg_dns_lookup_ms, color: '#8b5cf6' },
      { phase: 'TCP', value: data.avg_tcp_connection_ms, color: '#3b82f6' },
      { phase: 'TLS', value: data.avg_tls_handshake_ms, color: '#eab308' },
      { phase: 'Server', value: data.avg_server_processing_ms, color: '#f97316' },
      { phase: 'Transfer', value: data.avg_content_transfer_ms, color: '#ef4444' },
    ];
  }, [data]);

  if (!data) {
    return (
      <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-5">
        <h3 className="text-sm font-medium text-zinc-400">{title}</h3>
        <div className="flex h-[200px] items-center justify-center">
          <span className="text-zinc-600">Loading...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-5">
      <div className="mb-4 flex items-start justify-between">
        <div>
          <h3 className="text-sm font-medium text-zinc-400">{title}</h3>
          <p className="mt-0.5 text-xs text-zinc-600">Average response time breakdown</p>
        </div>
        <div className="text-right">
          <div className="flex items-baseline gap-2">
            <span className="font-mono text-2xl font-semibold text-zinc-100">
              {formatMs(data.avg_total_ms)}
            </span>
            <span className="text-xs text-zinc-500">avg</span>
          </div>
          <div className="mt-1 flex gap-3 text-xs text-zinc-500">
            <span>p50: <span className="font-mono text-zinc-400">{formatMs(data.p50_ms)}</span></span>
            <span>p95: <span className="font-mono text-zinc-400">{formatMs(data.p95_ms)}</span></span>
            <span>p99: <span className="font-mono text-zinc-400">{formatMs(data.p99_ms)}</span></span>
          </div>
        </div>
      </div>

      <div className="mb-4 grid grid-cols-5 gap-2">
        {PHASES.map((phase, i) => {
          const value = chartData[i]?.value || 0;
          return (
            <div
              key={phase.key}
              className="rounded-lg px-3 py-2"
              style={{ backgroundColor: `${phase.color}10` }}
            >
              <div className="text-[10px] text-zinc-500">{phase.label}</div>
              <div className="font-mono text-sm font-medium" style={{ color: phase.color }}>
                {formatMs(value)}
              </div>
            </div>
          );
        })}
      </div>

      <div style={{ height: 120, minHeight: 120 }}>
        <ResponsiveContainer width="100%" height={120} minHeight={120}>
          <AreaChart
            data={chartData}
            margin={{ top: 0, right: 0, left: 0, bottom: 0 }}
          >
            <defs>
              <linearGradient id="latencyGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#8b5cf6" stopOpacity={0.4} />
                <stop offset="100%" stopColor="#8b5cf6" stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis
              dataKey="phase"
              axisLine={false}
              tickLine={false}
              tick={{ fontSize: 10, fill: '#52525b', fontFamily: 'var(--font-mono)' }}
            />
            <YAxis hide />
            <Tooltip
              contentStyle={{
                backgroundColor: '#18181b',
                border: '1px solid #27272a',
                borderRadius: '8px',
                padding: '8px 12px',
                fontSize: '12px',
              }}
              labelStyle={{ color: '#71717a', fontSize: '11px', marginBottom: '4px' }}
              formatter={(value: number, name: string, props: any) => {
                const phaseIndex = chartData.findIndex((d) => d.phase === props.payload?.phase);
                const phaseColor = phaseIndex >= 0 ? chartData[phaseIndex].color : '#8b5cf6';
                return [
                  <span key="val" className="font-mono text-xs" style={{ color: phaseColor }}>
                    {formatMs(value)}
                  </span>,
                  <span key="label" style={{ color: phaseColor }}>Duration</span>,
                ];
              }}
              cursor={{ stroke: '#3f3f46', strokeWidth: 1 }}
            />
            <Area
              type="monotone"
              dataKey="value"
              stroke="#8b5cf6"
              strokeWidth={2}
              fill="url(#latencyGradient)"
              isAnimationActive={false}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-3 flex items-center justify-between border-t border-zinc-800 pt-3">
        <div className="text-xs text-zinc-500">
          TTFB: <span className="font-mono text-zinc-400">{formatMs(data.avg_ttfb_ms)}</span>
        </div>
        <div className="text-xs text-zinc-500">
          Window: <span className="font-mono text-zinc-400">{data.window_seconds}s</span>
        </div>
      </div>
    </div>
  );
}

