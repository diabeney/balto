package balancer_test

import (
	"testing"

	"github.com/diabeney/balto/internal/core/balancer"
	"github.com/diabeney/balto/internal/core/balancer/leastconn"
	"github.com/diabeney/balto/internal/core/balancer/roundrobin"
	"github.com/diabeney/balto/internal/core/balancer/weightedrr"
)

func TestBalancerInterfaceConformance(t *testing.T) {
	var _ balancer.Balancer = (*leastconn.LeastConnections)(nil)
	var _ balancer.Balancer = (*roundrobin.RoundRobin)(nil)
	var _ balancer.Balancer = (*weightedrr.WeightedRR)(nil)
}

func TestFactoryConstructors(t *testing.T) {
	tests := []struct {
		name string
		fn   func() balancer.Balancer
	}{
		{"LeastConnections", balancer.NewLeastConnections},
		{"RoundRobin", balancer.NewRoundRobin},
		{"WeightedRR", balancer.NewWeightedRR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got == nil {
				t.Fatalf("%s returned nil", tt.name)
			}
		})
	}
}
