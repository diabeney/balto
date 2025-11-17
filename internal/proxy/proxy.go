package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/diabeney/balto/internal/router"
)

type Proxy struct {
	router *atomic.Pointer[router.Router]
	client *http.Client
}

func New(r *router.Router) *Proxy {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   500, // per backend
		MaxConnsPerHost:       0,   // let the os decide
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	p := &Proxy{
		router: &atomic.Pointer[router.Router]{},
		client: &http.Client{
			Transport: transport,
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
	ctx := req.Context()
	rt := p.router.Load()

	if rt == nil {
		http.Error(w, "router not initialized", http.StatusServiceUnavailable)
		return
	}

	route, params, ok := rt.Lookup(router.Host(req.Host), req.URL.Path)
	if !ok {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	backend, err := route.NextBackend()
	if err != nil {
		http.Error(w, "no backend available", http.StatusServiceUnavailable)
		return
	}

	backend.Meta.IncrActive()
	defer backend.Meta.DecrActive()

	outURL := *backend.URL
	/*
		Strip the matched prefix before forwarding.
		Example:
		External request:  /api/v1/users/123
		Route prefix:      /api/v1
		Backend receives:  /users/123
		This allows the services to define routes without the public prefix.
		For wildcard routes, we strip everything up to the wildcard.
	*/
	outURL.Path = stripPrefix(req.URL.Path, route.Prefix)
	if outURL.Path == "" {
		outURL.Path = "/"
	}

	outURL.RawQuery = req.URL.RawQuery

	outReq, err := http.NewRequestWithContext(ctx, req.Method, outURL.String(), req.Body)
	if err != nil {
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

	fmt.Printf("[PROXY] Sending %s request to internal service %v\n", outReq.Method, fmt.Sprintf("%s://%s%s", outReq.URL.Scheme, outReq.Host, req.URL.Path))
	start := time.Now()
	resp, err := p.client.Do(outReq)
	if err != nil {
		route.Pool.RecordFailure(backend)
		http.Error(w, "bad gateway", http.StatusBadGateway)
		fmt.Printf("[PROXY] %s %s -> failed: %v\n", req.Host, req.URL.Path, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		route.Pool.RecordSuccess(backend)
	} else {
		route.Pool.RecordFailure(backend)
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

	if flusher, ok := w.(http.Flusher); ok {
		if _, err := io.Copy(flushWriter{w, flusher}, resp.Body); err != nil {
			fmt.Printf("[PROXY] Error copying response body: %v\n", err)
		}
	} else {
		if _, err := io.Copy(w, resp.Body); err != nil {
			fmt.Printf("[PROXY] Error copying response body: %v\n", err)
		}
	}
	close(done)

	fmt.Printf("[PROXY] %s %s took %v\n", req.Method, req.URL.Path, time.Since(start))
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
	// Explicitly flush the response to ensure the client receives the data immediately
	// without waiting for the buffer to be full.
	fw.flusher.Flush()
	return n, err
}
