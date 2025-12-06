package metrics

import (
	"encoding/json"
	"time"
)

type RequestTiming struct {
	DNSLookup        time.Duration `json:"-"`
	TCPConnection    time.Duration `json:"-"`
	TLSHandshake     time.Duration `json:"-"`
	ServerProcessing time.Duration `json:"-"`
	ContentTransfer  time.Duration `json:"-"`
	TTFB             time.Duration `json:"-"`
	Total            time.Duration `json:"-"`
}

type requestTimingJSON struct {
	DNSLookup        float64 `json:"dns_lookup_ms"`
	TCPConnection    float64 `json:"tcp_connection_ms"`
	TLSHandshake     float64 `json:"tls_handshake_ms"`
	ServerProcessing float64 `json:"server_processing_ms"`
	ContentTransfer  float64 `json:"content_transfer_ms"`
	TTFB             float64 `json:"ttfb_ms"`
	Total            float64 `json:"total_ms"`
}

func (rt RequestTiming) MarshalJSON() ([]byte, error) {
	return json.Marshal(requestTimingJSON{
		DNSLookup:        rt.DNSLookup.Seconds() * 1000,
		TCPConnection:    rt.TCPConnection.Seconds() * 1000,
		TLSHandshake:     rt.TLSHandshake.Seconds() * 1000,
		ServerProcessing: rt.ServerProcessing.Seconds() * 1000,
		ContentTransfer:  rt.ContentTransfer.Seconds() * 1000,
		TTFB:             rt.TTFB.Seconds() * 1000,
		Total:            rt.Total.Seconds() * 1000,
	})
}

type RequestMetric struct {
	Timestamp     time.Time     `json:"timestamp"`
	Method        string        `json:"method"`
	Route         string        `json:"route"`
	BackendID     string        `json:"backend_id"`
	StatusCode    int           `json:"status_code"`
	Timing        RequestTiming `json:"timing"`
	BytesSent     int64         `json:"bytes_sent"`
	BytesReceived int64         `json:"bytes_received"`
}

type BackendMetrics struct {
	ID                string    `json:"id"`
	Route             string    `json:"route"`
	Healthy           bool      `json:"healthy"`
	ActiveConnections int64     `json:"active_connections"`
	TotalRequests     int64     `json:"total_requests"`
	SuccessfulReqs    int64     `json:"successful_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	AvgResponseTime   float64   `json:"avg_response_time_ms"`
	P50ResponseTime   float64   `json:"p50_response_time_ms"`
	P95ResponseTime   float64   `json:"p95_response_time_ms"`
	P99ResponseTime   float64   `json:"p99_response_time_ms"`
	LastUpdated       time.Time `json:"last_updated"`
}

type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type TimeSeries struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Points []TimeSeriesPoint `json:"points"`
}

type AggregatedMetrics struct {
	Timestamp           time.Time                 `json:"timestamp"`
	WindowSeconds       int                       `json:"window_seconds"`
	RequestRate         float64                   `json:"request_rate"`
	ErrorRate           float64                   `json:"error_rate"`
	AvgLatency          float64                   `json:"avg_latency_ms"`
	P50Latency          float64                   `json:"p50_latency_ms"`
	P95Latency          float64                   `json:"p95_latency_ms"`
	P99Latency          float64                   `json:"p99_latency_ms"`
	AvgTTFB             float64                   `json:"avg_ttfb_ms"`
	AvgDNSLookup        float64                   `json:"avg_dns_lookup_ms"`
	AvgTCPConnection    float64                   `json:"avg_tcp_connection_ms"`
	AvgTLSHandshake     float64                   `json:"avg_tls_handshake_ms"`
	AvgServerProcessing float64                   `json:"avg_server_processing_ms"`
	AvgContentTransfer  float64                   `json:"avg_content_transfer_ms"`
	StatusCodes         map[int]int64             `json:"status_codes"`
	MethodCounts        map[string]int64          `json:"method_counts"`
	BytesSent           int64                     `json:"bytes_sent"`
	BytesReceived       int64                     `json:"bytes_received"`
	Backends            map[string]BackendMetrics `json:"backends"`
}

type SystemMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	Goroutines      int       `json:"goroutines"`
	HeapAlloc       uint64    `json:"heap_alloc_bytes"`
	HeapSys         uint64    `json:"heap_sys_bytes"`
	HeapInuse       uint64    `json:"heap_inuse_bytes"`
	StackInuse      uint64    `json:"stack_inuse_bytes"`
	NumGC           uint32    `json:"num_gc"`
	GCPauseTotal    uint64    `json:"gc_pause_total_ns"`
	LastGCPause     uint64    `json:"last_gc_pause_ns"`
	CPUUsage        float64   `json:"cpu_usage_percent"`
	OpenFiles       int       `json:"open_files"`
	TotalMemory     uint64    `json:"total_memory_bytes"`
	AvailableMemory uint64    `json:"available_memory_bytes"`
}

type RealtimeSnapshot struct {
	Timestamp  time.Time             `json:"timestamp"`
	System     SystemMetrics         `json:"system"`
	Aggregated AggregatedMetrics     `json:"aggregated"`
	TimeSeries map[string]TimeSeries `json:"time_series"`
}

type MetricsEvent struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}
