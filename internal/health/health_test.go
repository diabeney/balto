package health

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
)

type mockBalancer struct{}

func (m *mockBalancer) Next([]*core.Backend) *core.Backend { return nil }
func (m *mockBalancer) Update([]*core.Backend)             {}

func TestCheckBaltoHealth(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	CheckBaltoHealth(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status": "ok"}` {
		t.Errorf("expected ok json, got %s", body)
	}
}

func TestSingleJoin(t *testing.T) {
	cases := []struct {
		a, b, want string
	}{
		{"", "", "/"},
		{"", "/health", "/health"},
		{"/api", "", "/api"},
		{"/api/", "/health", "/api/health"},
		{"/api", "health", "/api/health"},
		{"/api/", "health", "/api/health"},
	}
	for _, c := range cases {
		if got := singleJoin(c.a, c.b); got != c.want {
			t.Errorf("singleJoin(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}

func TestHealthchecker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	pool := backendpool.New(&backendpool.PoolConfig{
		ProbePath:                  "/ok",
		Timeout:                    100,
		HealthThreshold:            2,
		ProbeHealthThreshold:       2,
		ProbeInterval:              100,
		CircuitMaxHalfOpenRequests: 1,
	}, &mockBalancer{})
	pool.Add("test", u, 1)
	b := pool.List()[0]

	hc := New(pool)
	hc.Start()
	defer func() {
		if err := hc.Stop(); err != nil {
			t.Errorf("Error stopping health checker: %v", err)
		}
	}()

	// Healthchecker.manageProbes has 1s ticker.
	time.Sleep(1500 * time.Millisecond)

	t.Run("Success marks healthy", func(t *testing.T) {
		if !b.IsHealthy() {
			t.Error("expected healthy after 200")
		}
	})

	t.Run("Failure marks unhealthy after threshold", func(t *testing.T) {
		// This server path will return 500, marking it unhealthy on probe
		u2, _ := url.Parse(srv.URL + "/fail")
		pool.Add("fail", u2, 1)

		// Wait long enough for reconciliation and multiple probes to trip threshold (2)
		// 500ms is more than enough time for 2 checks at a 100ms interval.
		time.Sleep(500 * time.Millisecond)

		b2 := pool.List()[1]

		if b2.IsHealthy() {
			t.Error("should be unhealthy after failures")
		}
	})

	t.Run("Stop cleans up", func(t *testing.T) {
		if err := hc.Stop(); err != nil {
			t.Errorf("Error stopping health checker: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
		if hc.started {
			t.Error("should stop after cleanup")
		}
	})
}
