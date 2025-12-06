'use client';

import { useRouter, useSearchParams } from 'next/navigation';

export type TimeRange = '30m' | '1h' | '6h' | '12h' | '24h' | '1w' | '1M';

const timeRanges: { value: TimeRange; label: string }[] = [
  { value: '30m', label: '30m' },
  { value: '1h', label: '1h' },
  { value: '6h', label: '6h' },
  { value: '12h', label: '12h' },
  { value: '24h', label: '24h' },
  { value: '1w', label: '1w' },
  { value: '1M', label: '1M' },
];

export function TimeRangeSelector() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const currentRange = (searchParams.get('range') as TimeRange) || '1h';

  const handleRangeChange = (range: TimeRange) => {
    const params = new URLSearchParams(searchParams.toString());
    params.set('range', range);
    router.push(`/dashboard?${params.toString()}`);
    router.refresh(); // Force server component to re-render with new params
  };

  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-zinc-600 dark:text-zinc-400">Time Range:</span>
      <div className="flex gap-1 rounded-lg border border-dashed border-zinc-300 bg-zinc-50 p-1 dark:border-zinc-700 dark:bg-zinc-950">
        {timeRanges.map((range) => (
          <button
            key={range.value}
            onClick={() => handleRangeChange(range.value)}
            className={`rounded px-2 py-1 text-xs font-medium transition-colors ${
              currentRange === range.value
                ? 'bg-zinc-200 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-50'
                : 'text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900'
            }`}
          >
            {range.label}
          </button>
        ))}
      </div>
    </div>
  );
}

