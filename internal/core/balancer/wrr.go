package balancer

import "github.com/diabeney/balto/internal/core"

type WeightedRR struct {
	list []*core.Backend
}

func NewWeightedRR() *WeightedRR {
	return &WeightedRR{}
}

func (w *WeightedRR) Update(backends []*core.Backend) {
	w.list = backends
}

func (w *WeightedRR) Next(backends []*core.Backend) *core.Backend {
	candidates := filterCandidates(backends)
	if len(candidates) == 0 {
		return nil
	}

	var best *core.Backend
	var bestScore int64 = -1

	// smooth WRR weight selection
	for _, b := range candidates {
		b.Meta.TempWeight += int64(b.Weight)
		if b.Meta.TempWeight > bestScore {
			bestScore = b.Meta.TempWeight
			best = b
		}
	}

	best.Meta.TempWeight -= int64(totalWeight(candidates))
	return best
}

func totalWeight(list []*core.Backend) uint32 {
	var sum uint32
	for _, b := range list {
		sum += b.Weight
	}
	return sum
}
