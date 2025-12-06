'use client';

interface MetricCardProps {
  label: string;
  value: string | number;
  subValue?: string;
  trend?: 'up' | 'down' | 'neutral';
  trendValue?: string;
  color?: 'default' | 'success' | 'warning' | 'error';
}

const colorStyles = {
  default: {
    border: 'border-zinc-800',
    bg: 'bg-zinc-950',
    value: 'text-zinc-100',
  },
  success: {
    border: 'border-emerald-900/50',
    bg: 'bg-emerald-950/30',
    value: 'text-emerald-400',
  },
  warning: {
    border: 'border-amber-900/50',
    bg: 'bg-amber-950/30',
    value: 'text-amber-400',
  },
  error: {
    border: 'border-red-900/50',
    bg: 'bg-red-950/30',
    value: 'text-red-400',
  },
};

export function MetricCard({
  label,
  value,
  subValue,
  trend,
  trendValue,
  color = 'default',
}: MetricCardProps) {
  const styles = colorStyles[color];

  return (
    <div className={`rounded-xl border ${styles.border} ${styles.bg} p-4`}>
      <div className="text-xs text-zinc-500">{label}</div>
      <div className="mt-1 flex items-baseline gap-2">
        <span className={`font-mono text-xl font-semibold ${styles.value}`}>
          {value}
        </span>
        {trend && trendValue && (
          <span
            className={`text-xs ${
              trend === 'up'
                ? 'text-emerald-500'
                : trend === 'down'
                ? 'text-red-500'
                : 'text-zinc-500'
            }`}
          >
            {trend === 'up' ? '↑' : trend === 'down' ? '↓' : '→'} {trendValue}
          </span>
        )}
      </div>
      {subValue && <div className="mt-0.5 text-xs text-zinc-600">{subValue}</div>}
    </div>
  );
}

interface MetricGridProps {
  children: React.ReactNode;
  columns?: 2 | 3 | 4 | 5 | 6;
}

export function MetricGrid({ children, columns = 4 }: MetricGridProps) {
  const colClass = {
    2: 'grid-cols-2',
    3: 'grid-cols-3',
    4: 'grid-cols-2 sm:grid-cols-4',
    5: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-5',
    6: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-6',
  }[columns];

  return <div className={`grid gap-3 ${colClass}`}>{children}</div>;
}

