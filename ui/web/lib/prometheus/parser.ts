import { ParsedMetrics, PrometheusMetric } from './types';

/**
 * Parses Prometheus metrics text format into structured data
 * Handles both exposition format and query result format
 */
export function parsePrometheusMetrics(text: string): ParsedMetrics {
  const lines = text.split('\n');
  const metrics: Record<string, PrometheusMetric[]> = {};

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    
    // Parse metric line: metric_name{labels} value [timestamp]
    // Handle both formats:
    // - Exposition: balto_requests_total{method="GET"} 123
    // - Query result: balto_requests_total{method="GET"} 123 1234567890
    const match = trimmed.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)\s*(?:\{([^}]*)\})?\s+([0-9.+-eE]+)(?:\s+(\d+))?$/);
    if (!match) continue;

    const [, name, labelsStr, valueStr, timestampStr] = match;
    const value = parseFloat(valueStr);
    if (isNaN(value)) continue;
    
    const timestamp = timestampStr ? parseInt(timestampStr, 10) : undefined;

    // Parse labels - handle both quoted and unquoted values
    const labels: Record<string, string> = {};
    if (labelsStr) {
      // Match label="value" or label=value patterns
      const labelPattern = /([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*"([^"]*)"|([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*([^,}]+)/g;
      let labelMatch;
      while ((labelMatch = labelPattern.exec(labelsStr)) !== null) {
        const key = labelMatch[1] || labelMatch[3];
        const value = labelMatch[2] || labelMatch[4];
        if (key && value !== undefined) {
          labels[key.trim()] = value.trim();
        }
      }
    }

    const metric: PrometheusMetric = { name, labels, value, timestamp };
    
    if (!metrics[name]) {
      metrics[name] = [];
    }
    metrics[name].push(metric);
  }

  // Prometheus histograms have _bucket, _sum, and _count suffixes
  // Calculate average latency from _sum / _count
  const durationMetrics: PrometheusMetric[] = [];
  const sumByLabels: Record<string, number> = {};
  const countByLabels: Record<string, number> = {};
  
  for (const [name, metricList] of Object.entries(metrics)) {
    if (name === 'balto_request_duration_seconds_sum') {
      for (const metric of metricList) {
        const labelKey = JSON.stringify(metric.labels);
        sumByLabels[labelKey] = metric.value;
      }
    } else if (name === 'balto_request_duration_seconds_count') {
      for (const metric of metricList) {
        const labelKey = JSON.stringify(metric.labels);
        countByLabels[labelKey] = metric.value;
      }
    }
  }
  
  // Calculate average duration for each unique label set
  for (const labelKey of Object.keys(sumByLabels)) {
    const sum = sumByLabels[labelKey];
    const count = countByLabels[labelKey] || 1;
    const labels = JSON.parse(labelKey);
    durationMetrics.push({
      name: 'balto_request_duration_seconds',
      labels,
      value: count > 0 ? sum / count : 0,
    });
  }

  return {
    requestsTotal: metrics['balto_requests_total'] || [],
    requestDuration: durationMetrics,
    backendFailures: metrics['balto_backend_failures_total'] || [],
    probeFailures: metrics['balto_probe_failures_total'] || [],
    backendActiveConnections: metrics['balto_backend_active_connections'] || [],
    backendHealthy: metrics['balto_backend_healthy'] || [],
    systemMemory: metrics['balto_system_memory_bytes'] || [],
    systemGoroutines: metrics['balto_system_goroutines']?.[0]?.value || 0,
  };
}

/**
 * Groups metrics by label value and creates time series data
 */
export function groupMetricsByLabel(
  metrics: PrometheusMetric[],
  labelKey: string,
  timeWindow: number = 3600 // 1 hour in seconds
): Record<string, { label: string; value: number }[]> {
  const grouped: Record<string, { label: string; value: number }[]> = {};

  for (const metric of metrics) {
    const labelValue = metric.labels[labelKey] || 'unknown';
    if (!grouped[labelValue]) {
      grouped[labelValue] = [];
    }
    grouped[labelValue].push({
      label: labelValue,
      value: metric.value,
    });
  }

  return grouped;
}

/**
 * Aggregates metrics by summing values
 */
export function aggregateMetrics(metrics: PrometheusMetric[]): number {
  return metrics.reduce((sum, m) => sum + m.value, 0);
}

/**
 * Gets latest value for a metric
 */
export function getLatestMetric(metrics: PrometheusMetric[]): number {
  if (metrics.length === 0) return 0;
  return metrics[metrics.length - 1].value;
}

