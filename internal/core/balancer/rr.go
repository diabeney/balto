package balancer

import (
	"sync/atomic"

	"github.com/diabeney/balto/internal/core"
)

type RoundRobin struct {
	counter atomic.Uint64
	list    []*core.Backend
}

func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

func (r *RoundRobin) Update(backends []*core.Backend) {
	r.list = backends
}

func (r *RoundRobin) Next(backends []*core.Backend) *core.Backend {
	if backends == nil {
		backends = r.list
	}
	candidates := filterCandidates(backends)
	if len(candidates) == 0 {
		return nil
	}
	idx := r.counter.Add(1)
	return candidates[idx%uint64(len(candidates))]
}
