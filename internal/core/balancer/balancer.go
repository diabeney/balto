package balancer

import "github.com/diabeney/balto/internal/core"

// Balancer defines a generic load balancing strategy.
type Balancer interface {
	// Next selects the next backend from the candidates.
	Next([]*core.Backend) *core.Backend

	// Update is called when the pool changes (add/remove).
	Update([]*core.Backend)
}

func filterCandidates(backends []*core.Backend) []*core.Backend {
	result := make([]*core.Backend, 0, len(backends))
	for _, b := range backends {
		if b.IsHealthy() && !b.IsDraining() {
			result = append(result, b)
		}
	}
	return result
}
