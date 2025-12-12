package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptrace"
	"sync/atomic"
	"time"
)

type TimingInfo struct {
	dnsStartNs     atomic.Int64
	dnsDoneNs      atomic.Int64
	connectStartNs atomic.Int64
	connectDoneNs  atomic.Int64
	tlsStartNs     atomic.Int64
	tlsDoneNs      atomic.Int64
	gotFirstByteNs atomic.Int64
	requestStartNs atomic.Int64
	requestDoneNs  atomic.Int64

	DNSLookup        time.Duration
	TCPConnection    time.Duration
	TLSHandshake     time.Duration
	ServerProcessing time.Duration
	ContentTransfer  time.Duration
	TTFB             time.Duration
	Total            time.Duration

	connReused atomic.Bool
}

func NewTimingInfo() *TimingInfo {
	t := &TimingInfo{}
	t.requestStartNs.Store(time.Now().UnixNano())
	return t
}

func (t *TimingInfo) MarkRequestDone() {
	t.requestDoneNs.Store(time.Now().UnixNano())
}

func (t *TimingInfo) Calculate() {
	dnsStart := t.dnsStartNs.Load()
	dnsDone := t.dnsDoneNs.Load()
	connectStart := t.connectStartNs.Load()
	connectDone := t.connectDoneNs.Load()
	tlsStart := t.tlsStartNs.Load()
	tlsDone := t.tlsDoneNs.Load()
	gotFirstByte := t.gotFirstByteNs.Load()
	requestStart := t.requestStartNs.Load()
	requestDone := t.requestDoneNs.Load()

	if dnsStart != 0 && dnsDone != 0 && dnsDone >= dnsStart {
		t.DNSLookup = time.Duration(dnsDone - dnsStart)
	}

	if connectStart != 0 && connectDone != 0 && connectDone >= connectStart {
		t.TCPConnection = time.Duration(connectDone - connectStart)
	}

	if tlsStart != 0 && tlsDone != 0 && tlsDone >= tlsStart {
		t.TLSHandshake = time.Duration(tlsDone - tlsStart)
	}

	if requestStart != 0 && gotFirstByte != 0 && gotFirstByte >= requestStart {
		t.TTFB = time.Duration(gotFirstByte - requestStart)
	}

	if connectDone != 0 && gotFirstByte != 0 && gotFirstByte >= connectDone {
		serverStart := connectDone
		if tlsDone != 0 && gotFirstByte >= tlsDone {
			serverStart = tlsDone
		}
		if gotFirstByte >= serverStart {
			t.ServerProcessing = time.Duration(gotFirstByte - serverStart)
		}
	}

	if requestStart != 0 && requestDone != 0 && requestDone >= requestStart {
		t.Total = time.Duration(requestDone - requestStart)
		if gotFirstByte != 0 && requestDone >= gotFirstByte {
			t.ContentTransfer = time.Duration(requestDone - gotFirstByte)
		}
	}
}

func (t *TimingInfo) SetConnReused(reused bool) {
	t.connReused.Store(reused)
}

func (t *TimingInfo) IsConnReused() bool {
	return t.connReused.Load()
}

func WithClientTrace(ctx context.Context, timing *TimingInfo) context.Context {
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			now := time.Now().UnixNano()
			timing.dnsStartNs.CompareAndSwap(0, now)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			timing.dnsDoneNs.Store(time.Now().UnixNano())
		},
		ConnectStart: func(network, addr string) {
			now := time.Now().UnixNano()
			timing.connectStartNs.CompareAndSwap(0, now)
		},
		ConnectDone: func(network, addr string, err error) {
			timing.connectDoneNs.Store(time.Now().UnixNano())
		},
		TLSHandshakeStart: func() {
			now := time.Now().UnixNano()
			timing.tlsStartNs.CompareAndSwap(0, now)
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			timing.tlsDoneNs.Store(time.Now().UnixNano())
		},
		GotFirstResponseByte: func() {
			timing.gotFirstByteNs.Store(time.Now().UnixNano())
		},
		GotConn: func(info httptrace.GotConnInfo) {
			timing.SetConnReused(info.Reused)
			if info.Reused {
				timing.connectDoneNs.Store(time.Now().UnixNano())
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
