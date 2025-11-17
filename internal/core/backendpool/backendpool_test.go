package backendpool

import (
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/diabeney/balto/internal/core"
)

type mockBalancer struct {
	mu       sync.Mutex
	calls    int
	lastList []*core.Backend
}

func (m *mockBalancer) Next([]*core.Backend) *core.Backend {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	return nil
}

func (m *mockBalancer) Update(backends []*core.Backend) {
	m.mu.Lock()
	m.lastList = backends
	m.mu.Unlock()
}

func TestPoolNew(t *testing.T) {
	cfg := &PoolConfig{ServiceName: "test", HealthThreshold: 3}
	bal := &mockBalancer{}
	p := New(cfg, bal)

	if p.Config().ServiceName != "test" {
		t.Errorf("expected service name 'test', got %s", p.Config().ServiceName)
	}
	if p.Config().HealthThreshold != 3 {
		t.Errorf("expected threshold 3, got %d", p.Config().HealthThreshold)
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
	p.SetConfig(&PoolConfig{ServiceName: "new", HealthThreshold: 5})
	cfg := p.Config()
	if cfg.ServiceName != "new" || cfg.HealthThreshold != 5 {
		t.Errorf("config not updated: %+v", cfg)
	}
}

func TestPoolHealthTracking(t *testing.T) {
	p := New(&PoolConfig{HealthThreshold: 2}, nil)
	u, _ := url.Parse("http://x")
	p.Add("x", u, 1)
	b := p.List()[0]

	t.Run("RecordFailure marks unhealthy after threshold", func(t *testing.T) {
		p.RecordFailure(b)
		p.RecordFailure(b)
		if b.IsHealthy() {
			t.Error("should be unhealthy after 2 failures")
		}
	})

	t.Run("RecordSuccess brings back healthy", func(t *testing.T) {
		p.RecordSuccess(b)
		if !b.IsHealthy() {
			t.Error("should be healthy after success")
		}
	})

	t.Run("CheckHealth enforces threshold", func(t *testing.T) {
		b.Meta.ResetFailCount()
		b.Meta.RecordFailure()
		b.Meta.RecordFailure()
		p.CheckHealth(b)
		if b.IsHealthy() {
			t.Error("CheckHealth should mark unhealthy")
		}
	})

	t.Run("ResetHealth clears and heals", func(t *testing.T) {
		p.ResetHealth(b)
		if !b.IsHealthy() || b.Meta.FailCount.Load() != 0 {
			t.Error("ResetHealth failed")
		}
	})
}

func TestPoolDraining(t *testing.T) {
	p := New(&PoolConfig{}, nil)
	u, _ := url.Parse("http://d")
	p.Add("d", u, 1)
	b := p.List()[0]

	t.Run("StartDraining sets flag", func(t *testing.T) {
		p.StartDraining("d")
		if !b.IsDraining() {
			t.Error("backend should be draining")
		}
	})

	t.Run("WaitForDrain waits for active conns", func(t *testing.T) {
		b.Meta.IncrActive()
		done := make(chan bool)
		go func() {
			done <- p.WaitForDrain("d", 2*time.Second)
		}()
		time.Sleep(100 * time.Millisecond)
		b.Meta.DecrActive()
		if !<-done {
			t.Error("WaitForDrain returned false")
		}
	})

	t.Run("WaitForDrain timeout", func(t *testing.T) {
		b.Meta.IncrActive()
		if p.WaitForDrain("d", 100*time.Millisecond) {
			t.Error("expected timeout")
		}
	})
}

func TestPoolConcurrentAddRemove(t *testing.T) {
	p := New(&PoolConfig{}, nil)
	u, _ := url.Parse("http://c")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			p.Add(string(rune(id)), u, 1)
		}(i)
	}
	wg.Wait()
	if len(p.List()) < 1 {
		t.Errorf("expected at least 1 backend, got %d", len(p.List()))
	}
}
