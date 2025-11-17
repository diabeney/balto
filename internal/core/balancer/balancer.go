package balancer

import "github.com/diabeney/balto/internal/core"

// Balancer defines a generic load balancing strategy.
type Balancer interface {
	// Next selects the next backend from the candidates.
	Next([]*core.Backend) *core.Backend

	// Update is called when the pool changes (add/remove).
	Update([]*core.Backend)
}
