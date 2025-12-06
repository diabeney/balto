package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/diabeney/balto/internal/metrics"
	"github.com/diabeney/balto/internal/router"
)

type Proxy struct {
	router    *atomic.Pointer[router.Router]
	client    *http.Client
	transport *TimedTransport
	metrics   *metrics.Collector
}

func New(r *router.Router) *Proxy {
	return NewWithMetrics(r, nil)
}

func NewWithMetrics(r *router.Router, m *metrics.Collector) *Proxy {
	transport := NewTimedTransport()

	p := &Proxy{
		router:    &atomic.Pointer[router.Router]{},
		transport: transport,
		metrics:   m,
		client: &http.Client{
			Transport: transport.Base,
			Timeout:   30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	p.router.Store(r)
	return p
}

func (p *Proxy) UpdateRouter(r *router.Router) {
	p.router.Store(r)
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	timing := NewTimingInfo()
	ctx := req.Context()
	rt := p.router.Load()

	if rt == nil {
		p.recordErrorMetrics(req, "", "", timing, http.StatusServiceUnavailable)
		http.Error(w, "router not initialized", http.StatusServiceUnavailable)
		return
	}

	route, params, ok := rt.Lookup(router.Host(req.Host), req.URL.Path)
	if !ok {
		p.recordErrorMetrics(req, "", "", timing, http.StatusNotFound)
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	backend, err := route.NextBackend()
	if err != nil {
		p.recordErrorMetrics(req, route.Prefix, "", timing, http.StatusServiceUnavailable)
		http.Error(w, "no backend available", http.StatusServiceUnavailable)
		return
	}

	backend.Meta.IncrActive()
	if p.metrics != nil {
		p.metrics.SetBackendActiveConnections(backend.ID, backend.Meta.Active())
	}
	defer func() {
		backend.Meta.DecrActive()
		if p.metrics != nil {
			p.metrics.SetBackendActiveConnections(backend.ID, backend.Meta.Active())
		}
	}()

	outURL := *backend.URL
	outURL.Path = stripPrefix(req.URL.Path, route.Prefix)
	if outURL.Path == "" {
		outURL.Path = "/"
	}
	outURL.RawQuery = req.URL.RawQuery

	tracedCtx := WithClientTrace(ctx, timing)
	outReq, err := http.NewRequestWithContext(tracedCtx, req.Method, outURL.String(), req.Body)
	if err != nil {
		p.recordErrorMetrics(req, route.Prefix, backend.ID, timing, http.StatusInternalServerError)
		http.Error(w, "failed to create outbound request", http.StatusInternalServerError)
		return
	}

	copyHeaders(req.Header, outReq.Header)
	removeHopHeaders(outReq.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHeader(outReq.Header, "X-Forwarded-For", clientIP)
	}
	appendHeader(outReq.Header, "X-Forwarded-Proto", schemeOf(req))
	appendHeader(outReq.Header, "X-Forwarded-Host", req.Host)

	if len(params) > 0 {
		for k, v := range params {
			appendHeader(outReq.Header, "X-Param-"+k, v)
		}
	}

	outReq.Host = backend.URL.Host
	outReq.ContentLength = req.ContentLength

	var bytesSent int64
	if req.ContentLength > 0 {
		bytesSent = req.ContentLength
	}

	resp, err := p.client.Do(outReq)
	timing.RequestDone = time.Now()
	timing.Calculate()

	if err != nil {
		route.Pool.RecordFailure(backend)
		if p.metrics != nil {
			p.metrics.RecordBackendFailure(backend.ID, "connection_error")
			p.recordDetailedMetrics(req, route.Prefix, backend.ID, timing, http.StatusBadGateway, bytesSent, 0)
		}
		http.Error(w, "bad gateway", http.StatusBadGateway)
		fmt.Printf("[PROXY] %s %s -> failed: %v\n", req.Host, req.URL.Path, err)
		return
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if statusCode >= 200 && statusCode < 400 {
		route.Pool.RecordSuccess(backend)
	} else {
		route.Pool.RecordFailure(backend)
		if p.metrics != nil {
			p.metrics.RecordBackendFailure(backend.ID, "http_error")
		}
	}

	copyHeaders(resp.Header, w.Header())
	removeHopHeaders(w.Header())
	w.WriteHeader(resp.StatusCode)

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			resp.Body.Close()
		case <-done:
		}
	}()

	var bytesReceived int64
	if flusher, ok := w.(http.Flusher); ok {
		bytesReceived, _ = io.Copy(countingWriter{flushWriter{w, flusher}, &bytesReceived}, resp.Body)
	} else {
		bytesReceived, _ = io.Copy(countingWriter{w, &bytesReceived}, resp.Body)
	}
	close(done)

	timing.RequestDone = time.Now()
	timing.Calculate()

	if p.metrics != nil {
		p.recordDetailedMetrics(req, route.Prefix, backend.ID, timing, statusCode, bytesSent, bytesReceived)
	}
}

func (p *Proxy) recordDetailedMetrics(req *http.Request, routePrefix string, backendID string, timing *TimingInfo, statusCode int, bytesSent, bytesReceived int64) {
	if p.metrics == nil {
		return
	}

	metric := metrics.RequestMetric{
		Timestamp:     time.Now(),
		Method:        req.Method,
		Route:         routePrefix,
		BackendID:     backendID,
		StatusCode:    statusCode,
		BytesSent:     bytesSent,
		BytesReceived: bytesReceived,
		Timing: metrics.RequestTiming{
			DNSLookup:        timing.DNSLookup,
			TCPConnection:    timing.TCPConnection,
			TLSHandshake:     timing.TLSHandshake,
			ServerProcessing: timing.ServerProcessing,
			ContentTransfer:  timing.ContentTransfer,
			TTFB:             timing.TTFB,
			Total:            timing.Total,
		},
	}

	p.metrics.RecordDetailedRequest(metric)
}

func (p *Proxy) recordErrorMetrics(req *http.Request, routePrefix string, backendID string, timing *TimingInfo, statusCode int) {
	if p.metrics == nil {
		return
	}

	timing.RequestDone = time.Now()
	timing.Calculate()

	p.recordDetailedMetrics(req, routePrefix, backendID, timing, statusCode, 0, 0)
}

func stripPrefix(path, prefix string) string {
	if strings.HasSuffix(prefix, "/*") {
		basePrefix := strings.TrimSuffix(prefix, "/*")
		stripped := strings.TrimPrefix(path, basePrefix)
		if stripped == "" || stripped == "/" {
			return "/"
		}
		if !strings.HasPrefix(stripped, "/") {
			return "/" + stripped
		}
		return stripped
	}
	return strings.TrimPrefix(path, prefix)
}

func schemeOf(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

func copyHeaders(src, dst http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func removeHopHeaders(h http.Header) {
	hop := []string{
		"Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailers",
		"Transfer-Encoding", "Upgrade",
	}
	for _, k := range hop {
		h.Del(k)
	}
}

func appendHeader(h http.Header, key, val string) {
	if existing := h.Get(key); existing != "" {
		h.Set(key, existing+", "+val)
	} else {
		h.Set(key, val)
	}
}

type flushWriter struct {
	http.ResponseWriter
	flusher http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.ResponseWriter.Write(p)
	fw.flusher.Flush()
	return n, err
}

type countingWriter struct {
	w     io.Writer
	count *int64
}

func (cw countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	*cw.count += int64(n)
	return n, err
}
