package metrics

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	globalCollector *Collector
	globalMu        sync.RWMutex
)

type Collector struct {
	requestsTotal        *prometheus.CounterVec
	backendFailuresTotal *prometheus.CounterVec
	probeFailuresTotal   *prometheus.CounterVec

	backendActiveConnections *prometheus.GaugeVec
	backendHealthy           *prometheus.GaugeVec
	systemMemoryBytes        *prometheus.GaugeVec
	systemGoroutines         prometheus.Gauge

	requestDurationSeconds  *prometheus.HistogramVec
	ttfbSeconds             *prometheus.HistogramVec
	dnsLookupSeconds        *prometheus.HistogramVec
	tcpConnectionSeconds    *prometheus.HistogramVec
	tlsHandshakeSeconds     *prometheus.HistogramVec
	serverProcessingSeconds *prometheus.HistogramVec
	contentTransferSeconds  *prometheus.HistogramVec

	requestBytesTotal  *prometheus.CounterVec
	responseBytesTotal *prometheus.CounterVec

	store        *TimeSeriesStore
	startTime    time.Time
	systemTicker *time.Ticker
	stopCh       chan struct{}
}

func NewCollector() *Collector {
	latencyBuckets := []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

	c := &Collector{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "balto_requests_total",
				Help: "Total number of requests processed",
			},
			[]string{"method", "route", "backend_id", "status_code"},
		),
		backendFailuresTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "balto_backend_failures_total",
				Help: "Total number of backend failures",
			},
			[]string{"backend_id", "failure_type"},
		),
		probeFailuresTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "balto_probe_failures_total",
				Help: "Total number of health probe failures",
			},
			[]string{"backend_id"},
		),
		backendActiveConnections: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "balto_backend_active_connections",
				Help: "Number of active connections per backend",
			},
			[]string{"backend_id"},
		),
		backendHealthy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "balto_backend_healthy",
				Help: "Backend health status (1 = healthy, 0 = unhealthy)",
			},
			[]string{"backend_id", "route"},
		),
		systemMemoryBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "balto_system_memory_bytes",
				Help: "System memory usage in bytes",
			},
			[]string{"type"},
		),
		systemGoroutines: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "balto_system_goroutines",
				Help: "Number of goroutines",
			},
		),
		requestDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_request_duration_seconds",
				Help:    "Total request duration in seconds",
				Buckets: latencyBuckets,
			},
			[]string{"method", "route", "backend_id", "status_code"},
		),
		ttfbSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_ttfb_seconds",
				Help:    "Time to first byte in seconds",
				Buckets: latencyBuckets,
			},
			[]string{"method", "route", "backend_id"},
		),
		dnsLookupSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_dns_lookup_seconds",
				Help:    "DNS lookup time in seconds",
				Buckets: []float64{.0001, .0005, .001, .005, .01, .025, .05, .1},
			},
			[]string{"backend_id"},
		),
		tcpConnectionSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_tcp_connection_seconds",
				Help:    "TCP connection establishment time in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
			},
			[]string{"backend_id"},
		),
		tlsHandshakeSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_tls_handshake_seconds",
				Help:    "TLS handshake time in seconds",
				Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2},
			},
			[]string{"backend_id"},
		),
		serverProcessingSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_server_processing_seconds",
				Help:    "Backend server processing time in seconds",
				Buckets: latencyBuckets,
			},
			[]string{"method", "route", "backend_id"},
		),
		contentTransferSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "balto_content_transfer_seconds",
				Help:    "Response content transfer time in seconds",
				Buckets: latencyBuckets,
			},
			[]string{"backend_id"},
		),
		requestBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "balto_request_bytes_total",
				Help: "Total bytes received in requests",
			},
			[]string{"method", "route", "backend_id"},
		),
		responseBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "balto_response_bytes_total",
				Help: "Total bytes sent in responses",
			},
			[]string{"method", "route", "backend_id"},
		),
		store:     NewTimeSeriesStore(1, 60),
		startTime: time.Now(),
		stopCh:    make(chan struct{}),
	}

	return c
}

func (c *Collector) Register() {
	prometheus.MustRegister(
		c.requestsTotal,
		c.backendFailuresTotal,
		c.probeFailuresTotal,
		c.backendActiveConnections,
		c.backendHealthy,
		c.systemMemoryBytes,
		c.systemGoroutines,
		c.requestDurationSeconds,
		c.ttfbSeconds,
		c.dnsLookupSeconds,
		c.tcpConnectionSeconds,
		c.tlsHandshakeSeconds,
		c.serverProcessingSeconds,
		c.contentTransferSeconds,
		c.requestBytesTotal,
		c.responseBytesTotal,
	)

	globalMu.Lock()
	globalCollector = c
	globalMu.Unlock()

	c.startSystemMetricsCollection()
}

func (c *Collector) startSystemMetricsCollection() {
	c.systemTicker = time.NewTicker(time.Second)
	go func() {
		for {
			select {
			case <-c.systemTicker.C:
				c.collectSystemMetrics()
			case <-c.stopCh:
				return
			}
		}
	}()
}

func (c *Collector) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.systemMemoryBytes.WithLabelValues("alloc").Set(float64(m.Alloc))
	c.systemMemoryBytes.WithLabelValues("sys").Set(float64(m.Sys))
	c.systemMemoryBytes.WithLabelValues("heap_alloc").Set(float64(m.HeapAlloc))
	c.systemMemoryBytes.WithLabelValues("heap_sys").Set(float64(m.HeapSys))
	c.systemMemoryBytes.WithLabelValues("heap_inuse").Set(float64(m.HeapInuse))
	c.systemMemoryBytes.WithLabelValues("stack_inuse").Set(float64(m.StackInuse))
	c.systemGoroutines.Set(float64(runtime.NumGoroutine()))

	if c.store != nil {
		c.store.RecordSystemMetrics(SystemMetrics{
			Timestamp:    time.Now(),
			Goroutines:   runtime.NumGoroutine(),
			HeapAlloc:    m.HeapAlloc,
			HeapSys:      m.HeapSys,
			HeapInuse:    m.HeapInuse,
			StackInuse:   m.StackInuse,
			NumGC:        m.NumGC,
			GCPauseTotal: m.PauseTotalNs,
			LastGCPause:  m.PauseNs[(m.NumGC+255)%256],
		})
	}
}

func GetGlobal() *Collector {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalCollector
}

func (c *Collector) Handler() http.Handler {
	return promhttp.Handler()
}

func (c *Collector) Store() *TimeSeriesStore {
	return c.store
}

func (c *Collector) RecordRequest(method, route, backendID, statusCode string, duration time.Duration) {
	labels := prometheus.Labels{
		"method":      method,
		"route":       route,
		"backend_id":  backendID,
		"status_code": statusCode,
	}
	c.requestsTotal.With(labels).Inc()
	c.requestDurationSeconds.With(labels).Observe(duration.Seconds())
}

func (c *Collector) RecordDetailedRequest(metric RequestMetric) {
	labels := prometheus.Labels{
		"method":      metric.Method,
		"route":       metric.Route,
		"backend_id":  metric.BackendID,
		"status_code": string(rune(metric.StatusCode + '0')),
	}

	statusStr := intToStatusStr(metric.StatusCode)
	labels["status_code"] = statusStr

	c.requestsTotal.With(labels).Inc()
	c.requestDurationSeconds.With(labels).Observe(metric.Timing.Total.Seconds())

	timingLabels := prometheus.Labels{
		"method":     metric.Method,
		"route":      metric.Route,
		"backend_id": metric.BackendID,
	}

	c.ttfbSeconds.With(timingLabels).Observe(metric.Timing.TTFB.Seconds())
	c.serverProcessingSeconds.With(timingLabels).Observe(metric.Timing.ServerProcessing.Seconds())

	backendLabels := prometheus.Labels{"backend_id": metric.BackendID}
	c.dnsLookupSeconds.With(backendLabels).Observe(metric.Timing.DNSLookup.Seconds())
	c.tcpConnectionSeconds.With(backendLabels).Observe(metric.Timing.TCPConnection.Seconds())
	c.tlsHandshakeSeconds.With(backendLabels).Observe(metric.Timing.TLSHandshake.Seconds())
	c.contentTransferSeconds.With(backendLabels).Observe(metric.Timing.ContentTransfer.Seconds())

	c.requestBytesTotal.With(timingLabels).Add(float64(metric.BytesSent))
	c.responseBytesTotal.With(timingLabels).Add(float64(metric.BytesReceived))

	if c.store != nil {
		c.store.RecordRequest(metric)
	}
}

func intToStatusStr(code int) string {
	if code < 100 || code > 599 {
		return "unknown"
	}
	return string([]byte{
		byte('0' + code/100),
		byte('0' + (code/10)%10),
		byte('0' + code%10),
	})
}

func (c *Collector) RecordBackendFailure(backendID, failureType string) {
	c.backendFailuresTotal.WithLabelValues(backendID, failureType).Inc()
}

func (c *Collector) RecordProbeFailure(backendID string) {
	c.probeFailuresTotal.WithLabelValues(backendID).Inc()
}

func (c *Collector) SetBackendActiveConnections(backendID string, count uint64) {
	c.backendActiveConnections.WithLabelValues(backendID).Set(float64(count))
	if c.store != nil {
		c.store.SetBackendActiveConnections(backendID, "", int64(count))
	}
}

func (c *Collector) SetBackendHealthy(backendID, route string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	c.backendHealthy.WithLabelValues(backendID, route).Set(value)
	if c.store != nil {
		c.store.SetBackendHealth(backendID, route, healthy)
	}
}

func (c *Collector) RecordSystemMetrics() {
	c.collectSystemMetrics()
}

func (c *Collector) GetAggregatedMetrics(windowSeconds int) AggregatedMetrics {
	if c.store == nil {
		return AggregatedMetrics{}
	}
	return c.store.GetAggregated(windowSeconds)
}

func (c *Collector) GetTimeSeries(name string, windowMinutes int) TimeSeries {
	if c.store == nil {
		return TimeSeries{}
	}
	return c.store.GetTimeSeries(name, windowMinutes)
}

func (c *Collector) Subscribe() chan MetricsEvent {
	if c.store == nil {
		return nil
	}
	return c.store.Subscribe()
}

func (c *Collector) Unsubscribe(ch chan MetricsEvent) {
	if c.store != nil {
		c.store.Unsubscribe(ch)
	}
}

func (c *Collector) Stop() {
	close(c.stopCh)
	if c.systemTicker != nil {
		c.systemTicker.Stop()
	}
}

func (c *Collector) Uptime() time.Duration {
	return time.Since(c.startTime)
}
