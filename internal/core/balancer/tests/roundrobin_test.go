package balancer_test

import (
	"net/url"
	"sync"
	"testing"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
	"github.com/diabeney/balto/internal/core/balancer"
)

func TestRoundRobinNext(t *testing.T) {
	rr := balancer.NewRoundRobin()
	u, _ := url.Parse("http://x")
	b1 := backendpool.NewBackend("1", u, 1)
	b2 := backendpool.NewBackend("2", u, 1)
	b3 := backendpool.NewBackend("3", u, 1)

	t.Run("Empty list returns nil", func(t *testing.T) {
		rr.Update(nil)
		if got := rr.Next(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("Cycles through backends", func(t *testing.T) {
		rr.Update([]*core.Backend{b1, b2, b3})
		seen := make(map[string]bool)
		for i := 0; i < 6; i++ {
			b := rr.Next(nil)
			seen[b.ID] = true
		}
		if len(seen) != 3 {
			t.Errorf("expected all 3 backends in cycle, got %d", len(seen))
		}
	})

	t.Run("Respects filterCandidates", func(t *testing.T) {
		b1.SetHealthy(false)
		b2.SetDraining(true)
		rr.Update([]*core.Backend{b1, b2, b3})
		for i := 0; i < 3; i++ {
			if got := rr.Next(nil); got != b3 {
				t.Errorf("expected only b3, got %v", got)
			}
		}
	})
}

func TestRoundRobinConcurrent(t *testing.T) {
	rr := balancer.NewRoundRobin()
	u, _ := url.Parse("http://x")
	backends := []*core.Backend{backendpool.NewBackend("a", u, 1), backendpool.NewBackend("b", u, 1)}
	rr.Update(backends)

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr.Next(nil)
		}()
	}
	wg.Wait()
}
