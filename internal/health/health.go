package health

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
)

func CheckBaltoHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}

type Healthchecker struct {
	pool     *backendpool.Pool
	interval time.Duration
	timeout  time.Duration
	client   *http.Client
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	probes   map[string]context.CancelFunc
	wg       sync.WaitGroup
	started  bool
}

func New(pool *backendpool.Pool) *Healthchecker {
	cfg := pool.Config()

	//TODO: Maybe a proper way to set defaults?
	timeout := time.Duration(cfg.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 500 * time.Millisecond
	} else if timeout < 50*time.Millisecond {
		timeout = 50 * time.Millisecond
	}

	interval := time.Duration(cfg.ProbeInterval) * time.Millisecond
	if interval <= 0 {
		interval = 1 * time.Second
	} else if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: timeout,
		}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: timeout / 2,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
	}

	return &Healthchecker{
		pool:     pool,
		interval: interval,
		timeout:  timeout,
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (h *Healthchecker) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.started {
		return
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())
	h.probes = make(map[string]context.CancelFunc)
	h.started = true
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.manageProbes()
	}()
}

func (h *Healthchecker) Stop() error {
	h.mu.Lock()
	if !h.started {
		h.mu.Unlock()
		return nil
	}

	h.started = false
	cancel := h.cancel
	h.cancel = nil

	for _, probeCancel := range h.probes {
		probeCancel()
	}
	// Don't delete from map, let probe goroutines clean up themselves
	h.mu.Unlock()

	// Cancel main context to stop manageProbes
	if cancel != nil {
		cancel()
	}

	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("healthchecker failed to stop within 10s")
	}
}

func (h *Healthchecker) manageProbes() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	h.reconcile()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.reconcile()
		}
	}
}

func (h *Healthchecker) reconcile() {
	backends := h.pool.List()
	currentIDs := make(map[string]*core.Backend, len(backends))
	for _, b := range backends {
		currentIDs[b.ID] = b
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for id, cancel := range h.probes {
		if _, exists := currentIDs[id]; !exists {
			cancel()
			delete(h.probes, id)
		}
	}

	for id, b := range currentIDs {
		if _, exists := h.probes[id]; !exists {
			h.startProbeForBackendLocked(b)
		}
	}
}

func (h *Healthchecker) startProbeForBackendLocked(b *core.Backend) {
	if !h.started || h.ctx == nil {
		return
	}

	ctx, cancel := context.WithCancel(h.ctx)
	h.probes[b.ID] = cancel

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.probeLoop(ctx, b)

		// Clean up on exit
		h.mu.Lock()
		delete(h.probes, b.ID)
		h.mu.Unlock()
	}()
}

func (h *Healthchecker) probeLoop(ctx context.Context, b *core.Backend) {
	delay := h.interval + h.jitter()
	timer := time.NewTimer(delay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if !b.IsDraining() {
				h.runProbe(ctx, b)
			}
			timer.Reset(h.interval + h.jitter())
		}
	}
}

func (h *Healthchecker) runProbe(ctx context.Context, b *core.Backend) {
	cfg := h.pool.Config()
	probePath := cfg.ProbePath

	scheme := b.URL.Scheme
	if scheme == "http" || scheme == "https" {
		h.probeHTTP(ctx, b, probePath)
	} else {
		h.probeTCP(ctx, b)
	}
}

func (h *Healthchecker) probeHTTP(ctx context.Context, b *core.Backend, path string) {
	probeURL := *b.URL
	probeURL.Path = singleJoin(probeURL.Path, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL.String(), nil)
	if err != nil {
		log.Printf("Error creating request for %s: %v", probeURL.String(), err)
		return
	}

	resp, err := h.client.Do(req)
	if err != nil {
		h.pool.MarkUnhealthy(b)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		h.pool.MarkHealthy(b)
	} else {
		h.pool.MarkUnhealthy(b)
	}
}

func (h *Healthchecker) probeTCP(ctx context.Context, b *core.Backend) {
	d := net.Dialer{Timeout: h.timeout}
	conn, err := d.DialContext(ctx, "tcp", b.URL.Host)
	if err != nil {
		h.pool.MarkUnhealthy(b)
		return
	}
	conn.Close()
	h.pool.MarkHealthy(b)
}

func singleJoin(a, b string) string {
	if a == "" {
		a = "/"
	}
	if b == "" {
		return a
	}
	if a[len(a)-1] == '/' {
		a = a[:len(a)-1]
	}
	if b[0] != '/' {
		b = "/" + b
	}
	return a + b
}

func (h *Healthchecker) jitter() time.Duration {
	if h.interval <= 0 {
		return 0
	}
	max := h.interval / 5
	if max <= 0 {
		max = 10 * time.Millisecond
	}
	return time.Duration(rand.Int63n(int64(max)))
}
