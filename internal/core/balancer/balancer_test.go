package balancer

import (
	"net/url"
	"testing"

	"github.com/diabeney/balto/internal/core"
)

func TestFilterCandidates(t *testing.T) {
	u, _ := url.Parse("http://x")
	b1 := &core.Backend{ID: "1", URL: u, Meta: &core.BackendMetadata{}}
	b2 := &core.Backend{ID: "2", URL: u, Meta: &core.BackendMetadata{}}
	b3 := &core.Backend{ID: "3", URL: u, Meta: &core.BackendMetadata{}}

	t.Run("Empty input returns empty", func(t *testing.T) {
		if got := filterCandidates(nil); len(got) != 0 {
			t.Errorf("expected empty, got %d", len(got))
		}
	})

	t.Run("All healthy and not draining", func(t *testing.T) {
		b1.SetHealthy(true)
		b2.SetHealthy(true)
		b1.SetDraining(false)
		b2.SetDraining(false)
		list := []*core.Backend{b1, b2}
		if got := filterCandidates(list); len(got) != 2 {
			t.Errorf("expected 2, got %d", len(got))
		}
	})

	t.Run("Filters unhealthy", func(t *testing.T) {
		b1.SetHealthy(false)
		b2.SetHealthy(true)
		if got := filterCandidates([]*core.Backend{b1, b2}); len(got) != 1 || got[0] != b2 {
			t.Errorf("expected only b2, got %v", got)
		}
	})

	t.Run("Filters draining", func(t *testing.T) {
		b1.SetHealthy(true)
		b1.SetDraining(true)
		b2.SetHealthy(true)
		b2.SetDraining(false)
		if got := filterCandidates([]*core.Backend{b1, b2}); len(got) != 1 || got[0] != b2 {
			t.Errorf("expected only b2, got %v", got)
		}
	})

	t.Run("Preserves order", func(t *testing.T) {
		b1.SetHealthy(true)
		b1.SetDraining(false)
		b2.SetHealthy(true)
		b2.SetDraining(false)
		b3.SetHealthy(true)
		b3.SetDraining(false)
		input := []*core.Backend{b1, b2, b3}
		got := filterCandidates(input)
		if len(got) != 3 || got[0] != b1 || got[1] != b2 || got[2] != b3 {
			t.Errorf("order not preserved: %v", got)
		}
	})
}

func TestBalancerInterface_Conformance(t *testing.T) {
	// Compile-time check to ensure implementations satisfy interface
	var _ Balancer = (*LeastConnections)(nil)
	var _ Balancer = (*RoundRobin)(nil)
	var _ Balancer = (*WeightedRR)(nil)
}
