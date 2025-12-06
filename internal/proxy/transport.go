package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

type TimingInfo struct {
	DNSStart     time.Time
	DNSDone      time.Time
	ConnectStart time.Time
	ConnectDone  time.Time
	TLSStart     time.Time
	TLSDone      time.Time
	GotFirstByte time.Time
	RequestStart time.Time
	RequestDone  time.Time

	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
	TTFB             time.Duration
	Total            time.Duration

	mu         sync.Mutex
	connReused bool
}

func NewTimingInfo() *TimingInfo {
	return &TimingInfo{
		RequestStart: time.Now(),
	}
}

func (t *TimingInfo) Calculate() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.DNSStart.IsZero() && !t.DNSDone.IsZero() {
		t.DNSLookup = t.DNSDone.Sub(t.DNSStart)
	}

	if !t.ConnectStart.IsZero() && !t.ConnectDone.IsZero() {
		t.TCPConnection = t.ConnectDone.Sub(t.ConnectStart)
	}

	if !t.TLSStart.IsZero() && !t.TLSDone.IsZero() {
		t.TLSHandshake = t.TLSDone.Sub(t.TLSStart)
	}

	if !t.RequestStart.IsZero() && !t.GotFirstByte.IsZero() {
		t.TTFB = t.GotFirstByte.Sub(t.RequestStart)
	}

	if !t.ConnectDone.IsZero() && !t.GotFirstByte.IsZero() {
		serverStart := t.ConnectDone
		if !t.TLSDone.IsZero() {
			serverStart = t.TLSDone
		}
		t.ServerProcessing = t.GotFirstByte.Sub(serverStart)
	}

	if !t.RequestStart.IsZero() && !t.RequestDone.IsZero() {
		t.Total = t.RequestDone.Sub(t.RequestStart)
		if !t.GotFirstByte.IsZero() {
			t.ContentTransfer = t.RequestDone.Sub(t.GotFirstByte)
		}
	}
}

func (t *TimingInfo) SetConnReused(reused bool) {
	t.mu.Lock()
	t.connReused = reused
	t.mu.Unlock()
}

func (t *TimingInfo) IsConnReused() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.connReused
}

func WithClientTrace(ctx context.Context, timing *TimingInfo) context.Context {
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			timing.mu.Lock()
			timing.DNSStart = time.Now()
			timing.mu.Unlock()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			timing.mu.Lock()
			timing.DNSDone = time.Now()
			timing.mu.Unlock()
		},
		ConnectStart: func(network, addr string) {
			timing.mu.Lock()
			timing.ConnectStart = time.Now()
			timing.mu.Unlock()
		},
		ConnectDone: func(network, addr string, err error) {
			timing.mu.Lock()
			timing.ConnectDone = time.Now()
			timing.mu.Unlock()
		},
		TLSHandshakeStart: func() {
			timing.mu.Lock()
			timing.TLSStart = time.Now()
			timing.mu.Unlock()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			timing.mu.Lock()
			timing.TLSDone = time.Now()
			timing.mu.Unlock()
		},
		GotFirstResponseByte: func() {
			timing.mu.Lock()
			timing.GotFirstByte = time.Now()
			timing.mu.Unlock()
		},
		GotConn: func(info httptrace.GotConnInfo) {
			timing.SetConnReused(info.Reused)
			if info.Reused {
				timing.mu.Lock()
				timing.ConnectDone = time.Now()
				timing.mu.Unlock()
			}
		},
	}

	return httptrace.WithClientTrace(ctx, trace)
}

type TimedTransport struct {
	Base *http.Transport
}

func NewTimedTransport() *TimedTransport {
	return &TimedTransport{
		Base: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   500,
			MaxConnsPerHost:       0,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}
}

func (t *TimedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Base.RoundTrip(req)
}
