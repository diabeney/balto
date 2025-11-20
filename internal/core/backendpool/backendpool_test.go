package backendpool

import (
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/circuit"
)

type mockBalancer struct {
	mu       sync.Mutex
	calls    int
	lastList []*core.Backend
}

func (m *mockBalancer) Next(backends []*core.Backend) *core.Backend {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	if len(backends) > 0 {
		return backends[0]
	}
	return nil
}

func (m *mockBalancer) Update(backends []*core.Backend) {
	m.mu.Lock()
	m.lastList = backends
	m.mu.Unlock()
}

func TestPoolNew(t *testing.T) {
	cfg := &PoolConfig{ServiceName: "test", HealthThreshold: 3, ProbeHealthThreshold: 2, CircuitMaxHalfOpenRequests: 2}
	bal := &mockBalancer{}
	p := New(cfg, bal)

	if p.Config().ServiceName != "test" {
		t.Errorf("expected service name 'test', got %s", p.Config().ServiceName)
	}
	if p.Config().HealthThreshold != 3 {
		t.Errorf("expected threshold 3, got %d", p.Config().HealthThreshold)
	}
	if p.Config().ProbeHealthThreshold != 2 {
		t.Errorf("expected probe threshold 2, got %d", p.Config().ProbeHealthThreshold)
	}
	if p.Config().CircuitMaxHalfOpenRequests != 2 {
		t.Errorf("expected circuit half-open limit 2, got %d", p.Config().CircuitMaxHalfOpenRequests)
	}
}

func TestPoolAddRemove(t *testing.T) {
	bal := &mockBalancer{}
	p := New(&PoolConfig{}, bal)
	u1, _ := url.Parse("http://a")
	u2, _ := url.Parse("http://b")

	t.Run("Add increases count", func(t *testing.T) {
		p.Add("1", u1, 1)
		p.Add("2", u2, 1)
		if len(p.List()) != 2 {
			t.Errorf("expected 2 backends, got %d", len(p.List()))
		}
	})

	t.Run("Remove decreases count", func(t *testing.T) {
		p.Remove("1")
		list := p.List()
		if len(list) != 1 || list[0].ID != "2" {
			t.Errorf("expected only backend 2, got %v", list)
		}
	})

	t.Run("Remove non-existent does nothing", func(t *testing.T) {
		p.Remove("ghost")
		if len(p.List()) != 1 {
			t.Errorf("expected 1 backend after removing non-existent, got %d", len(p.List()))
		}
	})
}

func TestPoolConfigHotReload(t *testing.T) {
	p := New(&PoolConfig{ServiceName: "old"}, nil)
	p.SetConfig(&PoolConfig{ServiceName: "new", HealthThreshold: 5, ProbeHealthThreshold: 7, CircuitMaxHalfOpenRequests: 3})
	cfg := p.Config()
	if cfg.ServiceName != "new" || cfg.HealthThreshold != 5 || cfg.ProbeHealthThreshold != 7 || cfg.CircuitMaxHalfOpenRequests != 3 {
		t.Errorf("config not updated: %+v", cfg)
	}
}

func TestPoolCircuitIntegration(t *testing.T) {
	u, _ := url.Parse("http://circuit")
	p := New(&PoolConfig{
		HealthThreshold:            5, // High health threshold
		ProbeHealthThreshold:       2,
		CircuitFailureThreshold:    2, // Low circuit threshold
		CircuitSuccessThreshold:    1,
		CircuitTimeout:             1,
		CircuitMaxHalfOpenRequests: 1,
	}, &mockBalancer{})

	p.Add("cb", u, 1)
	b := p.List()[0]

	b.Circuit = circuit.New(circuit.Config{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	})

	if b.Circuit == nil {
		t.Fatal("Circuit breaker not initialized")
	}

	p.RecordFailure(b)
	p.RecordFailure(b)

	if b.Circuit.State() != circuit.Open {
		t.Errorf("expected Open after 2 failures, got %v", b.Circuit.State())
	}

	// Balancer should not select it
	// mockBalancer returns backends[0] if list is not empty
	if initial := p.Next(); initial != nil {
		t.Errorf("expected no backend selected (circuit open), got %v", initial)
	}

	time.Sleep(1100 * time.Millisecond)

	// Balancer logic calls Allow() inside Next(). Poll until the backend becomes eligible.
	var selected *core.Backend
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		selected = p.Next()
		if selected != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if selected == nil {
		t.Fatal("expected backend selected in Half-Open")
	}
	if b.Circuit.State() != circuit.HalfOpen {
		t.Errorf("expected circuit Half-Open after selection, got %v", b.Circuit.State())
	}

	p.RecordSuccess(b)
	if b.Circuit.State() != circuit.Closed {
		t.Errorf("expected Closed after success, got %v", b.Circuit.State())
	}
}

func TestPassiveAndProbeFailureTracking(t *testing.T) {
	u, _ := url.Parse("http://dual")
	p := New(&PoolConfig{
		HealthThreshold:      2,
		ProbeHealthThreshold: 2,
	}, &mockBalancer{})
	p.Add("dual", u, 1)
	b := p.List()[0]

	t.Run("Passive failures mark unhealthy after threshold", func(t *testing.T) {
		p.RecordFailure(b)
		if !b.IsHealthy() {
			t.Fatal("backend should remain healthy after first passive failure")
		}
		p.RecordFailure(b)
		if b.IsHealthy() {
			t.Fatal("backend should be unhealthy after reaching passive threshold")
		}
		if b.Meta.PassiveFailCount.Load() != 2 {
			t.Fatalf("expected passive fail count 2, got %d", b.Meta.PassiveFailCount.Load())
		}
	})

	p.ResetHealth(b)

	t.Run("Probe failures use probe counter only", func(t *testing.T) {
		if !b.IsHealthy() {
			t.Fatal("backend should have been restored by MarkHealthy")
		}
		p.MarkUnhealthy(b)
		if b.Meta.ProbeFailCount.Load() != 1 {
			t.Fatalf("expected probe fail count 1, got %d", b.Meta.ProbeFailCount.Load())
		}
		if !b.IsHealthy() {
			t.Fatal("probe threshold not reached yet, should stay healthy")
		}
		p.MarkUnhealthy(b)
		if b.Meta.ProbeFailCount.Load() != 2 {
			t.Fatalf("expected probe fail count 2, got %d", b.Meta.ProbeFailCount.Load())
		}
		if b.IsHealthy() {
			t.Fatal("backend should be unhealthy after reaching probe threshold")
		}
		if b.Meta.PassiveFailCount.Load() != 0 {
			t.Fatalf("passive fail count should remain 0, got %d", b.Meta.PassiveFailCount.Load())
		}
	})
}

func TestProbeRecoveryThreshold(t *testing.T) {
	u, _ := url.Parse("http://recovery")
	p := New(&PoolConfig{
		HealthThreshold:        2,
		ProbeHealthThreshold:   2,
		ProbeRecoveryThreshold: 3,
	}, &mockBalancer{})
	p.Add("recovery", u, 1)
	b := p.List()[0]

	b.SetHealthy(false)
	if b.IsHealthy() {
		t.Fatal("backend should start unhealthy for this test")
	}

	p.MarkHealthy(b)
	if b.IsHealthy() {
		t.Fatal("backend should not be healthy after 1 success (threshold is 3)")
	}
	if b.Meta.ProbeSuccessCount.Load() != 1 {
		t.Fatalf("expected probe success count 1, got %d", b.Meta.ProbeSuccessCount.Load())
	}

	p.MarkHealthy(b)
	if b.IsHealthy() {
		t.Fatal("backend should not be healthy after 2 successes (threshold is 3)")
	}
	if b.Meta.ProbeSuccessCount.Load() != 2 {
		t.Fatalf("expected probe success count 2, got %d", b.Meta.ProbeSuccessCount.Load())
	}

	p.MarkHealthy(b)
	if !b.IsHealthy() {
		t.Fatal("backend should be healthy after 3 consecutive successes")
	}
	if b.Meta.ProbeSuccessCount.Load() != 3 {
		t.Fatalf("expected probe success count 3, got %d", b.Meta.ProbeSuccessCount.Load())
	}

	// Test that a failure resets the counter
	p.MarkUnhealthy(b)
	if b.Meta.ProbeSuccessCount.Load() != 0 {
		t.Fatalf("probe success count should reset to 0 after failure, got %d", b.Meta.ProbeSuccessCount.Load())
	}
}
