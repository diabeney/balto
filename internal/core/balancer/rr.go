package balancer

import (
	"github.com/diabeney/balto/internal/core"
	"sync/atomic"
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
	candidates := filterCandidates(backends)
	if len(candidates) == 0 {
		return nil
	}
	idx := r.counter.Add(1)
	return candidates[idx%uint64(len(candidates))]
}
