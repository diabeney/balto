package balancer_test

import (
	"net/url"
	"testing"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
	"github.com/diabeney/balto/internal/core/balancer"
	"github.com/diabeney/balto/internal/core/circuit"
)

func TestWeightedRRNext(t *testing.T) {
	wrr := balancer.NewWeightedRR()
	u, _ := url.Parse("http://x")
	cbCfg := circuit.Config{}

	b1 := backendpool.NewBackend("1", u, 1, cbCfg)
	b2 := backendpool.NewBackend("2", u, 3, cbCfg)

	t.Run("Empty list returns nil", func(t *testing.T) {
		wrr.Update(nil)
		if got := wrr.Next(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("Single backend always selected", func(t *testing.T) {
		wrr.Update([]*core.Backend{b1})
		for i := 0; i < 5; i++ {
			if got := wrr.Next(nil); got != b1 {
				t.Errorf("expected b1, got %v", got)
			}
		}
	})

	t.Run("Weight ratio 1:3", func(t *testing.T) {
		wrr.Update([]*core.Backend{b1, b2})
		counts := make(map[string]int)
		for i := 0; i < 400; i++ {
			b := wrr.Next(nil)
			counts[b.ID]++
		}
		c1, c2 := counts["1"], counts["2"]
		ratio := float64(c2) / float64(c1)
		if ratio < 2.8 || ratio > 3.2 {
			t.Errorf("expected ~3:1 ratio, got %d:%d (%.2f)", c1, c2, ratio)
		}
	})

	t.Run("Resets TempWeight correctly", func(t *testing.T) {
		wrr.Update([]*core.Backend{b1, b2})
		b1.Meta.TempWeight = 0
		b2.Meta.TempWeight = 0
		for i := 0; i < 10; i++ {
			wrr.Next(nil)
		}
		sum := b1.Meta.TempWeight + b2.Meta.TempWeight
		if sum != 0 {
			t.Errorf("TempWeight sum should be 0 after cycle: b1=%d, b2=%d, sum=%d", b1.Meta.TempWeight, b2.Meta.TempWeight, sum)
		}
	})

	t.Run("Pool filtering handles unhealthy/draining", func(t *testing.T) {
		b1.SetHealthy(false)
		b2.SetDraining(true)
		wrr.Update([]*core.Backend{b2})
		if got := wrr.Next(nil); got != b2 {
			t.Errorf("expected pool to pre-filter unhealthy/draining backends, got %v", got)
		}
	})
}
