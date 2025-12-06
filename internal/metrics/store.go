package metrics

import (
	"container/ring"
	"sort"
	"sync"
	"time"
)

const (
	DefaultRetentionMinutes = 60
	DefaultBucketSeconds    = 1
	MaxBucketsPerSeries     = 3600
)

type MetricBucket struct {
	Timestamp   time.Time
	Count       int64
	Sum         float64
	Min         float64
	Max         float64
	Values      []float64
	StatusCodes map[int]int64
	Methods     map[string]int64
	BytesSent   int64
	BytesRecv   int64
}

func newMetricBucket(ts time.Time) *MetricBucket {
	return &MetricBucket{
		Timestamp:   ts,
		Min:         float64(^uint64(0) >> 1),
		StatusCodes: make(map[int]int64),
		Methods:     make(map[string]int64),
		Values:      make([]float64, 0, 100),
	}
}

type BackendState struct {
	ID                string
	Route             string
	Healthy           bool
	ActiveConnections int64
	TotalRequests     int64
	SuccessCount      int64
	FailureCount      int64
	LatencySum        float64
	Latencies         *ring.Ring
	LastUpdated       time.Time
	mu                sync.RWMutex
}

func newBackendState(id, route string) *BackendState {
	return &BackendState{
		ID:        id,
		Route:     route,
		Healthy:   true,
		Latencies: ring.New(1000),
	}
}

func (b *BackendState) AddLatency(latency float64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.TotalRequests++
	b.LatencySum += latency
	b.Latencies.Value = latency
	b.Latencies = b.Latencies.Next()
	b.LastUpdated = time.Now()
}

func (b *BackendState) GetPercentile(p float64) float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var values []float64
	b.Latencies.Do(func(v interface{}) {
		if v != nil {
			values = append(values, v.(float64))
		}
	})

	if len(values) == 0 {
		return 0
	}

	sort.Float64s(values)
	idx := int(float64(len(values)) * p)
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

type TimeSeriesStore struct {
	mu              sync.RWMutex
	requestBuckets  map[string]*ring.Ring
	timingBuckets   map[string]*ring.Ring
	backends        map[string]*BackendState
	systemMetrics   *ring.Ring
	recentRequests  *ring.Ring
	bucketDuration  time.Duration
	retentionPeriod time.Duration
	subscribers     []chan MetricsEvent
	subMu           sync.RWMutex
}

func NewTimeSeriesStore(bucketSeconds, retentionMinutes int) *TimeSeriesStore {
	if bucketSeconds <= 0 {
		bucketSeconds = DefaultBucketSeconds
	}
	if retentionMinutes <= 0 {
		retentionMinutes = DefaultRetentionMinutes
	}

	maxBuckets := (retentionMinutes * 60) / bucketSeconds
	if maxBuckets > MaxBucketsPerSeries {
		maxBuckets = MaxBucketsPerSeries
	}

	store := &TimeSeriesStore{
		requestBuckets:  make(map[string]*ring.Ring),
		timingBuckets:   make(map[string]*ring.Ring),
		backends:        make(map[string]*BackendState),
		systemMetrics:   ring.New(maxBuckets),
		recentRequests:  ring.New(1000),
		bucketDuration:  time.Duration(bucketSeconds) * time.Second,
		retentionPeriod: time.Duration(retentionMinutes) * time.Minute,
		subscribers:     make([]chan MetricsEvent, 0),
	}

	go store.cleanupLoop()
	return store
}

func (s *TimeSeriesStore) getBucketKey(ts time.Time) time.Time {
	return ts.Truncate(s.bucketDuration)
}

func (s *TimeSeriesStore) getOrCreateBucket(series string, ts time.Time) *MetricBucket {
	s.mu.Lock()
	defer s.mu.Unlock()

	bucketKey := s.getBucketKey(ts)

	r, exists := s.requestBuckets[series]
	if !exists {
		maxBuckets := int(s.retentionPeriod / s.bucketDuration)
		r = ring.New(maxBuckets)
		s.requestBuckets[series] = r
	}

	if r.Value != nil {
		bucket := r.Value.(*MetricBucket)
		if bucket.Timestamp.Equal(bucketKey) {
			return bucket
		}
	}

	bucket := newMetricBucket(bucketKey)
	r.Value = bucket
	s.requestBuckets[series] = r.Next()
	return bucket
}

func (s *TimeSeriesStore) RecordRequest(metric RequestMetric) {
	series := "requests"
	bucket := s.getOrCreateBucket(series, metric.Timestamp)

	s.mu.Lock()
	bucket.Count++
	bucket.Sum += float64(metric.Timing.Total.Milliseconds())
	bucket.BytesSent += metric.BytesSent
	bucket.BytesRecv += metric.BytesReceived

	latency := float64(metric.Timing.Total.Milliseconds())
	if latency < bucket.Min {
		bucket.Min = latency
	}
	if latency > bucket.Max {
		bucket.Max = latency
	}
	bucket.Values = append(bucket.Values, latency)
	bucket.StatusCodes[metric.StatusCode]++
	bucket.Methods[metric.Method]++
	s.mu.Unlock()

	s.recordTimingDetails(metric)
	s.updateBackendMetrics(metric)

	s.mu.Lock()
	s.recentRequests.Value = metric
	s.recentRequests = s.recentRequests.Next()
	s.mu.Unlock()

	s.publish(MetricsEvent{
		Type:      "request",
		Timestamp: metric.Timestamp,
		Data:      metric,
	})
}

func (s *TimeSeriesStore) recordTimingDetails(metric RequestMetric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ts := s.getBucketKey(metric.Timestamp)

	timings := map[string]float64{
		"ttfb":              float64(metric.Timing.TTFB.Milliseconds()),
		"dns_lookup":        float64(metric.Timing.DNSLookup.Milliseconds()),
		"tcp_connection":    float64(metric.Timing.TCPConnection.Milliseconds()),
		"tls_handshake":     float64(metric.Timing.TLSHandshake.Milliseconds()),
		"server_processing": float64(metric.Timing.ServerProcessing.Milliseconds()),
		"content_transfer":  float64(metric.Timing.ContentTransfer.Milliseconds()),
	}

	for name, value := range timings {
		r, exists := s.timingBuckets[name]
		if !exists {
			maxBuckets := int(s.retentionPeriod / s.bucketDuration)
			r = ring.New(maxBuckets)
			s.timingBuckets[name] = r
		}

		var bucket *MetricBucket
		if r.Value != nil {
			existing := r.Value.(*MetricBucket)
			if existing.Timestamp.Equal(ts) {
				bucket = existing
			}
		}

		if bucket == nil {
			bucket = newMetricBucket(ts)
			r.Value = bucket
			s.timingBuckets[name] = r.Next()
		}

		bucket.Count++
		bucket.Sum += value
		bucket.Values = append(bucket.Values, value)
	}
}

func (s *TimeSeriesStore) updateBackendMetrics(metric RequestMetric) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := metric.BackendID
	backend, exists := s.backends[key]
	if !exists {
		backend = newBackendState(metric.BackendID, metric.Route)
		s.backends[key] = backend
	}

	backend.AddLatency(float64(metric.Timing.Total.Milliseconds()))

	if metric.StatusCode >= 200 && metric.StatusCode < 400 {
		backend.SuccessCount++
	} else {
		backend.FailureCount++
	}
}

func (s *TimeSeriesStore) SetBackendHealth(backendID, route string, healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	backend, exists := s.backends[backendID]
	if !exists {
		backend = newBackendState(backendID, route)
		s.backends[backendID] = backend
	}

	backend.mu.Lock()
	backend.Healthy = healthy
	backend.LastUpdated = time.Now()
	backend.mu.Unlock()

	s.publish(MetricsEvent{
		Type:      "backend_health",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"backend_id": backendID,
			"route":      route,
			"healthy":    healthy,
		},
	})
}

func (s *TimeSeriesStore) SetBackendActiveConnections(backendID, route string, count int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	backend, exists := s.backends[backendID]
	if !exists {
		backend = newBackendState(backendID, route)
		s.backends[backendID] = backend
	}

	backend.mu.Lock()
	backend.ActiveConnections = count
	backend.LastUpdated = time.Now()
	backend.mu.Unlock()
}

func (s *TimeSeriesStore) RecordSystemMetrics(metrics SystemMetrics) {
	s.mu.Lock()
	s.systemMetrics.Value = &metrics
	s.systemMetrics = s.systemMetrics.Next()
	s.mu.Unlock()

	s.publish(MetricsEvent{
		Type:      "system",
		Timestamp: metrics.Timestamp,
		Data:      metrics,
	})
}

func (s *TimeSeriesStore) GetAggregated(windowSeconds int) AggregatedMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	windowStart := now.Add(-time.Duration(windowSeconds) * time.Second)

	result := AggregatedMetrics{
		Timestamp:     now,
		WindowSeconds: windowSeconds,
		StatusCodes:   make(map[int]int64),
		MethodCounts:  make(map[string]int64),
		Backends:      make(map[string]BackendMetrics),
	}

	var allLatencies []float64
	var allTTFB []float64
	var totalRequests int64
	var totalErrors int64

	if r, exists := s.requestBuckets["requests"]; exists {
		r.Do(func(v interface{}) {
			if v == nil {
				return
			}
			bucket := v.(*MetricBucket)
			if bucket.Timestamp.Before(windowStart) {
				return
			}

			totalRequests += bucket.Count
			allLatencies = append(allLatencies, bucket.Values...)

			for code, count := range bucket.StatusCodes {
				result.StatusCodes[code] += count
				if code >= 400 {
					totalErrors += count
				}
			}

			for method, count := range bucket.Methods {
				result.MethodCounts[method] += count
			}

			result.BytesSent += bucket.BytesSent
			result.BytesReceived += bucket.BytesRecv
		})
	}

	if r, exists := s.timingBuckets["ttfb"]; exists {
		r.Do(func(v interface{}) {
			if v == nil {
				return
			}
			bucket := v.(*MetricBucket)
			if bucket.Timestamp.Before(windowStart) {
				return
			}
			allTTFB = append(allTTFB, bucket.Values...)
		})
	}

	if totalRequests > 0 {
		result.RequestRate = float64(totalRequests) / float64(windowSeconds)
		result.ErrorRate = float64(totalErrors) / float64(totalRequests) * 100
	}

	if len(allLatencies) > 0 {
		sort.Float64s(allLatencies)
		sum := 0.0
		for _, v := range allLatencies {
			sum += v
		}
		result.AvgLatency = sum / float64(len(allLatencies))
		result.P50Latency = percentile(allLatencies, 0.50)
		result.P95Latency = percentile(allLatencies, 0.95)
		result.P99Latency = percentile(allLatencies, 0.99)
	}

	if len(allTTFB) > 0 {
		sum := 0.0
		for _, v := range allTTFB {
			sum += v
		}
		result.AvgTTFB = sum / float64(len(allTTFB))
	}

	for name, state := range s.backends {
		state.mu.RLock()
		result.Backends[name] = BackendMetrics{
			ID:                state.ID,
			Route:             state.Route,
			Healthy:           state.Healthy,
			ActiveConnections: state.ActiveConnections,
			TotalRequests:     state.TotalRequests,
			SuccessfulReqs:    state.SuccessCount,
			FailedRequests:    state.FailureCount,
			AvgResponseTime:   0,
			P50ResponseTime:   state.GetPercentile(0.50),
			P95ResponseTime:   state.GetPercentile(0.95),
			P99ResponseTime:   state.GetPercentile(0.99),
			LastUpdated:       state.LastUpdated,
		}
		if state.TotalRequests > 0 {
			result.Backends[name] = BackendMetrics{
				ID:                state.ID,
				Route:             state.Route,
				Healthy:           state.Healthy,
				ActiveConnections: state.ActiveConnections,
				TotalRequests:     state.TotalRequests,
				SuccessfulReqs:    state.SuccessCount,
				FailedRequests:    state.FailureCount,
				AvgResponseTime:   state.LatencySum / float64(state.TotalRequests),
				P50ResponseTime:   state.GetPercentile(0.50),
				P95ResponseTime:   state.GetPercentile(0.95),
				P99ResponseTime:   state.GetPercentile(0.99),
				LastUpdated:       state.LastUpdated,
			}
		}
		state.mu.RUnlock()
	}

	result.AvgDNSLookup = s.getTimingAvg("dns_lookup", windowStart)
	result.AvgTCPConnection = s.getTimingAvg("tcp_connection", windowStart)
	result.AvgTLSHandshake = s.getTimingAvg("tls_handshake", windowStart)
	result.AvgServerProcessing = s.getTimingAvg("server_processing", windowStart)
	result.AvgContentTransfer = s.getTimingAvg("content_transfer", windowStart)

	return result
}

func (s *TimeSeriesStore) getTimingAvg(name string, windowStart time.Time) float64 {
	r, exists := s.timingBuckets[name]
	if !exists {
		return 0
	}

	var sum float64
	var count int
	r.Do(func(v interface{}) {
		if v == nil {
			return
		}
		bucket := v.(*MetricBucket)
		if bucket.Timestamp.Before(windowStart) {
			return
		}
		for _, val := range bucket.Values {
			sum += val
			count++
		}
	})

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func (s *TimeSeriesStore) GetTimeSeries(name string, windowMinutes int) TimeSeries {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := TimeSeries{
		Name:   name,
		Labels: make(map[string]string),
		Points: make([]TimeSeriesPoint, 0),
	}

	windowStart := time.Now().Add(-time.Duration(windowMinutes) * time.Minute)

	var r *ring.Ring
	var exists bool

	switch name {
	case "requests":
		r, exists = s.requestBuckets["requests"]
	case "system":
		r = s.systemMetrics
		exists = true
	default:
		r, exists = s.timingBuckets[name]
	}

	if !exists || r == nil {
		return result
	}

	r.Do(func(v interface{}) {
		if v == nil {
			return
		}

		switch data := v.(type) {
		case *MetricBucket:
			if data.Timestamp.Before(windowStart) {
				return
			}
			avg := 0.0
			if data.Count > 0 {
				avg = data.Sum / float64(data.Count)
			}
			result.Points = append(result.Points, TimeSeriesPoint{
				Timestamp: data.Timestamp,
				Value:     avg,
			})
		case *SystemMetrics:
			if data.Timestamp.Before(windowStart) {
				return
			}
			var value float64
			switch name {
			case "goroutines":
				value = float64(data.Goroutines)
			case "heap_alloc":
				value = float64(data.HeapAlloc)
			case "heap_sys":
				value = float64(data.HeapSys)
			case "cpu":
				value = data.CPUUsage
			default:
				value = float64(data.Goroutines)
			}
			result.Points = append(result.Points, TimeSeriesPoint{
				Timestamp: data.Timestamp,
				Value:     value,
			})
		}
	})

	sort.Slice(result.Points, func(i, j int) bool {
		return result.Points[i].Timestamp.Before(result.Points[j].Timestamp)
	})

	return result
}

func (s *TimeSeriesStore) GetLatestSystemMetrics() *SystemMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prev := s.systemMetrics.Prev()
	if prev.Value != nil {
		return prev.Value.(*SystemMetrics)
	}
	return nil
}

func (s *TimeSeriesStore) Subscribe() chan MetricsEvent {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	ch := make(chan MetricsEvent, 256)
	s.subscribers = append(s.subscribers, ch)
	return ch
}

func (s *TimeSeriesStore) Unsubscribe(ch chan MetricsEvent) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	for i, sub := range s.subscribers {
		if sub == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

func (s *TimeSeriesStore) publish(event MetricsEvent) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *TimeSeriesStore) GetRecentRequests(limit int) []RequestMetric {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var requests []RequestMetric
	s.recentRequests.Do(func(v interface{}) {
		if v != nil {
			requests = append(requests, v.(RequestMetric))
		}
	})

	if len(requests) > limit {
		requests = requests[len(requests)-limit:]
	}

	for i, j := 0, len(requests)-1; i < j; i, j = i+1, j-1 {
		requests[i], requests[j] = requests[j], requests[i]
	}

	return requests
}

func (s *TimeSeriesStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanup()
	}
}

func (s *TimeSeriesStore) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.retentionPeriod)

	for _, r := range s.requestBuckets {
		r.Do(func(v interface{}) {
			if v == nil {
				return
			}
			bucket := v.(*MetricBucket)
			if bucket.Timestamp.Before(cutoff) {
				bucket.Values = nil
				bucket.StatusCodes = nil
				bucket.Methods = nil
			}
		})
	}
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}
