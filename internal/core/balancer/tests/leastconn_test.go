package balancer_test

import (
	"net/url"
	"sync"
	"testing"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
	"github.com/diabeney/balto/internal/core/balancer"
	"github.com/diabeney/balto/internal/core/circuit"
)

func TestLeastConnectionsNext(t *testing.T) {
	lc := balancer.NewLeastConnections()
	u, _ := url.Parse("http://x")
	cbCfg := circuit.Config{}

	b1 := backendpool.NewBackend("1", u, 1, cbCfg)
	b2 := backendpool.NewBackend("2", u, 1, cbCfg)
	b1.Meta.IncrActive()
	b1.Meta.IncrActive()

	t.Run("Empty list returns nil", func(t *testing.T) {
		lc.Update(nil)
		if got := lc.Next(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("Single backend returns it", func(t *testing.T) {
		lc.Update([]*core.Backend{b1})
		if got := lc.Next(nil); got != b1 {
			t.Errorf("expected b1, got %v", got)
		}
	})

	t.Run("Chooses lowest active", func(t *testing.T) {
		b2.Meta.IncrActive()
		lc.Update([]*core.Backend{b1, b2})
		if got := lc.Next(nil); got != b2 {
			t.Errorf("expected b2 (lower active), got %v", got)
		}
	})

	t.Run("Pool filtering handles unhealthy/draining", func(t *testing.T) {
		b1.SetHealthy(false)
		b2.SetDraining(true)
		lc.Update([]*core.Backend{b1, b2})
		if got := lc.Next([]*core.Backend{b2}); got != b2 {
			t.Errorf("pool should only pass healthy/draining-free backends, got %v", got)
		}
	})
}

func TestLeastConnectionsConcurrent(t *testing.T) {
	lc := balancer.NewLeastConnections()
	u, _ := url.Parse("http://x")
	cbCfg := circuit.Config{}
	backends := make([]*core.Backend, 3)
	for i := 0; i < 3; i++ {
		backends[i] = backendpool.NewBackend(string(rune('a'+i)), u, 1, cbCfg)
	}
	lc.Update(backends)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lc.Next(nil)
		}()
	}
	wg.Wait()
}
