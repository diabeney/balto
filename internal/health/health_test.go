package health

import (
	"context"
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

func TestStartBackendHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL + "/ok")
	pool := backendpool.New(&backendpool.PoolConfig{
		ProbePath:       "/ok",
		Timeout:         500,
		HealthThreshold: 2,
	}, &mockBalancer{})
	pool.Add("test", u, 1)
	b := pool.List()[0]

	ctx, cancel := context.WithCancel(context.Background())
	cancelFunc := StartBackendHealthCheck(ctx, pool)
	defer cancelFunc()

	time.Sleep(800 * time.Millisecond) // let ticker fire

	t.Run("Success marks healthy", func(t *testing.T) {
		if !b.IsHealthy() {
			t.Error("expected healthy after 200")
		}
		if b.Meta.TotalRequests.Load() == 0 {
			t.Error("expected requests recorded")
		}
	})

	t.Run("Failure marks unhealthy after threshold", func(t *testing.T) {
		u2, _ := url.Parse(srv.URL + "/fail")
		pool.Add("fail", u2, 1)
		b2 := pool.List()[1]
		time.Sleep(800 * time.Millisecond)
		if b2.IsHealthy() {
			t.Error("should be unhealthy after 2 failures")
		}
	})

	t.Run("Cancel stops checking", func(t *testing.T) {
		cancel()
		time.Sleep(600 * time.Millisecond)
		// No panic
	})
}
