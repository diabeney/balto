package balancer

import "github.com/diabeney/balto/internal/core"

type LeastConnections struct {
	list []*core.Backend
}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{}
}

func (l *LeastConnections) Update(backends []*core.Backend) {
	l.list = backends
}

func (l *LeastConnections) Next(backends []*core.Backend) *core.Backend {
	candidates := filterCandidates(backends)
	if len(candidates) == 0 {
		return nil
	}

	best := candidates[0]
	bestConn := best.Meta.Active()

	for _, b := range candidates[1:] {
		if c := b.Meta.Active(); c < bestConn {
			best = b
			bestConn = c
		}
	}

	return best
}
