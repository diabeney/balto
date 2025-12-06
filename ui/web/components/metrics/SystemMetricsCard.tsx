'use client';

import { PrometheusMetric } from '@/lib/prometheus/types';

interface SystemMetricsCardProps {
  memory: PrometheusMetric[];
  goroutines: number;
}

export function SystemMetricsCard({ memory, goroutines }: SystemMetricsCardProps) {
  const memoryByType = memory.reduce((acc, m) => {
    const type = m.labels.type || 'unknown';
    acc[type] = m.value;
    return acc;
  }, {} as Record<string, number>);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-5">
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Memory Allocated</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {formatBytes(memoryByType.alloc || 0)}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Heap Allocated</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {formatBytes(memoryByType.heap_alloc || 0)}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">System Memory</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {formatBytes(memoryByType.sys || 0)}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Heap System</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {formatBytes(memoryByType.heap_sys || 0)}
        </div>
      </div>
      <div className="rounded-lg border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="text-sm text-zinc-600 dark:text-zinc-400">Goroutines</div>
        <div className="mt-1 text-2xl font-semibold text-zinc-900 dark:text-zinc-50">
          {goroutines}
        </div>
      </div>
    </div>
  );
}

