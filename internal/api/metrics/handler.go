package metrics

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/diabeney/balto/internal/metrics"
	"github.com/diabeney/balto/pkg/utils"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type MetricsAPIResponse struct {
	Success   bool        `json:"success"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

type TimeSeriesResponse struct {
	Name   string                    `json:"name"`
	Labels map[string]string         `json:"labels"`
	Points []metrics.TimeSeriesPoint `json:"points"`
}

type RealtimeResponse struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

func AggregatedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	windowStr := r.URL.Query().Get("window")
	window := 60
	if windowStr != "" {
		if parsed, err := strconv.Atoi(windowStr); err == nil && parsed > 0 {
			window = parsed
		}
	}

	aggregated := collector.GetAggregatedMetrics(window)

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      aggregated,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func TimeSeriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	seriesName := r.URL.Query().Get("name")
	if seriesName == "" {
		seriesName = "requests"
	}

	windowStr := r.URL.Query().Get("window")
	window := 60
	if windowStr != "" {
		if parsed, err := strconv.Atoi(windowStr); err == nil && parsed > 0 {
			window = parsed
		}
	}

	series := collector.GetTimeSeries(seriesName, window)

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      series,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func SnapshotHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	windowStr := r.URL.Query().Get("window")
	window := 60
	if windowStr != "" {
		if parsed, err := strconv.Atoi(windowStr); err == nil && parsed > 0 {
			window = parsed
		}
	}

	snapshot := metrics.RealtimeSnapshot{
		Timestamp:  time.Now(),
		Aggregated: collector.GetAggregatedMetrics(window),
		TimeSeries: make(map[string]metrics.TimeSeries),
	}

	seriesNames := []string{
		"requests",
		"ttfb",
		"dns_lookup",
		"tcp_connection",
		"tls_handshake",
		"server_processing",
		"content_transfer",
	}

	for _, name := range seriesNames {
		snapshot.TimeSeries[name] = collector.GetTimeSeries(name, window)
	}

	store := collector.Store()
	if store != nil {
		if sysMetrics := store.GetLatestSystemMetrics(); sysMetrics != nil {
			snapshot.System = *sysMetrics
		}
	}

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      snapshot,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

type WSClient struct {
	conn   *websocket.Conn
	send   chan RealtimeResponse
	mu     sync.Mutex
	closed bool
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *WSClient) readPump(done chan struct{}) {
	defer close(done)
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WSClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

func StreamHandler(w http.ResponseWriter, r *http.Request) {
	collector := metrics.GetGlobal()
	if collector == nil {
		http.Error(w, "metrics collector not initialized", http.StatusServiceUnavailable)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &WSClient{
		conn: conn,
		send: make(chan RealtimeResponse, 256),
	}

	eventCh := collector.Subscribe()
	if eventCh == nil {
		conn.Close()
		return
	}

	done := make(chan struct{})
	go client.writePump()
	go client.readPump(done)

	intervalStr := r.URL.Query().Get("interval")
	interval := 1000
	if intervalStr != "" {
		if parsed, err := strconv.Atoi(intervalStr); err == nil && parsed >= 100 {
			interval = parsed
		}
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer func() {
		ticker.Stop()
		collector.Unsubscribe(eventCh)
		client.close()
		conn.Close()
	}()

	for {
		select {
		case <-done:
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			select {
			case client.send <- RealtimeResponse{
				Type:      event.Type,
				Timestamp: event.Timestamp,
				Data:      event.Data,
			}:
			default:
			}
		case <-ticker.C:
			aggregated := collector.GetAggregatedMetrics(60)
			select {
			case client.send <- RealtimeResponse{
				Type:      "snapshot",
				Timestamp: time.Now(),
				Data:      aggregated,
			}:
			default:
			}
		}
	}
}

func BackendsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	aggregated := collector.GetAggregatedMetrics(60)

	backends := make([]metrics.BackendMetrics, 0, len(aggregated.Backends))
	for _, b := range aggregated.Backends {
		backends = append(backends, b)
	}

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      backends,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func LatencyBreakdownHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	windowStr := r.URL.Query().Get("window")
	window := 60
	if windowStr != "" {
		if parsed, err := strconv.Atoi(windowStr); err == nil && parsed > 0 {
			window = parsed
		}
	}

	aggregated := collector.GetAggregatedMetrics(window)

	breakdown := map[string]interface{}{
		"window_seconds":           window,
		"avg_total_ms":             aggregated.AvgLatency,
		"p50_ms":                   aggregated.P50Latency,
		"p95_ms":                   aggregated.P95Latency,
		"p99_ms":                   aggregated.P99Latency,
		"avg_ttfb_ms":              aggregated.AvgTTFB,
		"avg_dns_lookup_ms":        aggregated.AvgDNSLookup,
		"avg_tcp_connection_ms":    aggregated.AvgTCPConnection,
		"avg_tls_handshake_ms":     aggregated.AvgTLSHandshake,
		"avg_server_processing_ms": aggregated.AvgServerProcessing,
		"avg_content_transfer_ms":  aggregated.AvgContentTransfer,
	}

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      breakdown,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func SystemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	store := collector.Store()
	if store == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics store not initialized")
		return
	}

	sysMetrics := store.GetLatestSystemMetrics()
	if sysMetrics == nil {
		sysMetrics = &metrics.SystemMetrics{
			Timestamp: time.Now(),
		}
	}

	sysMetrics.Timestamp = time.Now()

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      sysMetrics,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}

func AllTimeSeriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	windowStr := r.URL.Query().Get("window")
	window := 60
	if windowStr != "" {
		if parsed, err := strconv.Atoi(windowStr); err == nil && parsed > 0 {
			window = parsed
		}
	}

	seriesNames := []string{
		"requests",
		"ttfb",
		"dns_lookup",
		"tcp_connection",
		"tls_handshake",
		"server_processing",
		"content_transfer",
	}

	result := make(map[string]metrics.TimeSeries)
	for _, name := range seriesNames {
		result[name] = collector.GetTimeSeries(name, window)
	}

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      result,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func RecentRequestsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	collector := metrics.GetGlobal()
	if collector == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
		return
	}

	store := collector.Store()
	if store == nil {
		utils.WriteError(w, http.StatusServiceUnavailable, "metrics store not initialized")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	requests := store.GetRecentRequests(limit)

	response := MetricsAPIResponse{
		Success:   true,
		Timestamp: time.Now(),
		Data:      requests,
	}

	utils.WriteJSON(w, http.StatusOK, response)
}
