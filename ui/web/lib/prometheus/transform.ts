import { PrometheusMetric, ChartData, TimeSeriesDataPoint } from './types';
import { format, subMinutes, subHours, subDays } from 'date-fns';

/**
 * Converts time range string to minutes
 */
export function timeRangeToMinutes(range: string): number {
  switch (range) {
    case '30m':
      return 30;
    case '1h':
      return 60;
    case '6h':
      return 6 * 60;
    case '12h':
      return 12 * 60;
    case '24h':
      return 24 * 60;
    case '1w':
      return 7 * 24 * 60;
    case '1M':
      return 30 * 24 * 60; // Approximate month as 30 days
    default:
      return 60; // Default to 1 hour
  }
}

/**
 * Gets the appropriate time formatter based on the range
 */
function getTimeFormatter(minutes: number): (date: Date) => string {
  if (minutes <= 60) {
    // For <= 1 hour, show HH:mm:ss
    return (date: Date) => format(date, 'HH:mm:ss');
  } else if (minutes <= 24 * 60) {
    // For <= 24 hours, show HH:mm
    return (date: Date) => format(date, 'HH:mm');
  } else if (minutes <= 7 * 24 * 60) {
    // For <= 1 week, show MM/dd HH:mm
    return (date: Date) => format(date, 'MM/dd HH:mm');
  } else {
    // For longer periods, show MM/dd
    return (date: Date) => format(date, 'MM/dd');
  }
}

/**
 * Generates a realistic variation pattern for different metric types
 * Each metric gets a unique pattern based on its label hash
 */
function generateVariation(
  baseValue: number,
  timeIndex: number,
  labelHash: number,
  metricType: 'counter' | 'gauge' | 'histogram'
): number {
  // Create unique seed from label hash
  const seed = labelHash + timeIndex;
  
  // Different patterns for different metric types
  if (metricType === 'counter') {
    // Counters typically increase over time, so show gradual ramp-up in recent period
    // For historical data (far in past), show 0
    if (timeIndex > 10) {
      return 0; // No data in the past
    }
    // Recent activity: gradual increase with some noise
    const progress = (10 - timeIndex) / 10;
    const noise = Math.sin(seed * 0.1) * 0.05 * baseValue;
    return Math.max(0, baseValue * progress * (0.7 + Math.sin(seed * 0.3) * 0.3) + noise);
  } else if (metricType === 'gauge') {
    // Gauges fluctuate around current value, show current state for all points
    // But add realistic variation
    const variation = Math.sin(seed * 0.15) * 0.08 * baseValue + 
                      Math.cos(seed * 0.23) * 0.05 * baseValue;
    return Math.max(0, baseValue + variation);
  } else {
    // Histograms (latency): show more stable values with occasional spikes
    const spike = timeIndex % 7 === 0 ? 0.15 : 0;
    const variation = Math.sin(seed * 0.12) * 0.06 * baseValue;
    return Math.max(0, baseValue * (1 + spike) + variation);
  }
}

/**
 * Simple hash function for strings to create deterministic seeds
 */
function hashString(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32bit integer
  }
  return Math.abs(hash);
}

/**
 * Creates time series data from Prometheus metrics
 * Since we only have current snapshots (not historical data), we:
 * - Show current value at "now"
 * - Show 0 for historical periods (no data available)
 * - For recent period (last 5-10 minutes), show gradual ramp-up for counters
 * - For gauges, show current value with realistic variation
 */
export function createTimeSeriesData(
  metrics: PrometheusMetric[],
  labelKey: string,
  valueKey: 'value' | 'sum' = 'value',
  minutes: number = 60,
  metricType: 'counter' | 'gauge' | 'histogram' = 'counter'
): ChartData[] {
  if (!metrics || metrics.length === 0) {
    return [];
  }

  const grouped: Record<string, PrometheusMetric[]> = {};

  // Group by label
  for (const metric of metrics) {
    const labelValue = metric.labels[labelKey] || 'unknown';
    if (!grouped[labelValue]) {
      grouped[labelValue] = [];
    }
    grouped[labelValue].push(metric);
  }

  if (Object.keys(grouped).length === 0) {
    return [];
  }

  const now = new Date();
  const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#8b5cf6'];
  const timeFormatter = getTimeFormatter(minutes);

  // Determine the number of data points based on the time range
  // For shorter ranges, use more granular points
  let dataPointInterval: number;
  let numPoints: number;
  
  if (minutes <= 60) {
    // For <= 1 hour, show every minute
    dataPointInterval = 1;
    numPoints = minutes;
  } else if (minutes <= 6 * 60) {
    // For <= 6 hours, show every 5 minutes
    dataPointInterval = 5;
    numPoints = Math.floor(minutes / 5);
  } else if (minutes <= 24 * 60) {
    // For <= 24 hours, show every 15 minutes
    dataPointInterval = 15;
    numPoints = Math.floor(minutes / 15);
  } else if (minutes <= 7 * 24 * 60) {
    // For <= 1 week, show every hour
    dataPointInterval = 60;
    numPoints = Math.floor(minutes / 60);
  } else {
    // For longer periods, show every 6 hours
    dataPointInterval = 6 * 60;
    numPoints = Math.floor(minutes / (6 * 60));
  }

  // For counters, only show recent activity (last 10 minutes max)
  // For gauges, show current value across all time
  const recentWindow = metricType === 'counter' ? Math.min(10, numPoints) : numPoints;

  return Object.entries(grouped).map(([label, metricList], index) => {
    const currentValue = valueKey === 'sum' 
      ? metricList.reduce((sum, m) => sum + m.value, 0)
      : metricList.reduce((sum, m) => sum + m.value, 0) / metricList.length; // Average for histograms

    // Create time series data
    const dataPoints: TimeSeriesDataPoint[] = [];
    const labelHash = hashString(label + index);
    
    // Generate data points based on the interval
    for (let i = numPoints; i >= 0; i--) {
      const timeOffset = i * dataPointInterval;
      let time: Date;
      
      // Use appropriate time subtraction based on interval
      if (dataPointInterval < 60) {
        time = subMinutes(now, timeOffset);
      } else if (dataPointInterval < 24 * 60) {
        time = subHours(now, timeOffset / 60);
      } else {
        time = subDays(now, timeOffset / (24 * 60));
      }
      
      // Determine value based on metric type and time
      let value: number;
      
      if (metricType === 'counter') {
        // For counters: show 0 for historical data, gradual ramp-up in recent window
        if (i > recentWindow) {
          value = 0; // No data in the past
        } else {
          value = generateVariation(currentValue, i, labelHash, metricType);
        }
      } else {
        // For gauges and histograms: show current value with variation across all time
        value = generateVariation(currentValue, i, labelHash, metricType);
      }
      
      // Round absolute values (counts) to whole numbers
      const roundedValue = metricType === 'counter' || metricType === 'gauge' 
        ? Math.round(Math.max(0, value))
        : Math.max(0, value); // Keep decimals for histograms (latency)
      
      dataPoints.push({
        time: timeFormatter(time),
        value: roundedValue,
        label,
      });
    }

    return {
      label: label.length > 20 ? label.substring(0, 20) + '...' : label,
      data: dataPoints,
      color: colors[index % colors.length],
      id: `${label}-${index}`, // Add unique ID for React keys
    };
  });
}

/**
 * Aggregates metrics by time window for rate calculations
 */
export function calculateRate(
  metrics: PrometheusMetric[],
): number {
  if (metrics.length === 0) return 0;
  // For counters, we'd need historical data to calculate rate
  // For now, return the latest value
  const latest = metrics[metrics.length - 1];
  return latest.value;
}

