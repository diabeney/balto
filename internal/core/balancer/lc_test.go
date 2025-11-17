package balancer_test

import (
	"net/url"
	"sync"
	"testing"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
	"github.com/diabeney/balto/internal/core/balancer"
)

func TestLeastConnectionsNext(t *testing.T) {
	lc := balancer.NewLeastConnections()
	u, _ := url.Parse("http://x")

	b1 := backendpool.NewBackend("1", u, 1)
	b2 := backendpool.NewBackend("2", u, 1)
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
		b2.Meta.IncrActive() // b2 has 1, b1 has 2
		lc.Update([]*core.Backend{b1, b2})
		if got := lc.Next(nil); got != b2 {
			t.Errorf("expected b2 (lower active), got %v", got)
		}
	})

	t.Run("Ignores unhealthy", func(t *testing.T) {
		b1.SetHealthy(false)
		lc.Update([]*core.Backend{b1, b2})
		if got := lc.Next(nil); got != b2 {
			t.Errorf("expected only healthy b2, got %v", got)
		}
	})

	t.Run("Ignores draining", func(t *testing.T) {
		b1.SetHealthy(true)
		b2.SetDraining(true)
		lc.Update([]*core.Backend{b1, b2})
		if got := lc.Next(nil); got != b1 {
			t.Errorf("expected only non-draining b1, got %v", got)
		}
	})
}

func TestLeastConnectionsConcurrent(t *testing.T) {
	lc := balancer.NewLeastConnections()
	u, _ := url.Parse("http://x")
	backends := make([]*core.Backend, 3)
	for i := 0; i < 3; i++ {
		backends[i] = backendpool.NewBackend(string(rune('a'+i)), u, 1)
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
