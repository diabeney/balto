package common

import "github.com/diabeney/balto/internal/core"

// FilterCandidates filters out unhealthy or draining backends.
func FilterCandidates(backends []*core.Backend) []*core.Backend {
	if len(backends) == 0 {
		return nil
	}
	result := make([]*core.Backend, 0, len(backends))
	for _, b := range backends {
		if b.IsHealthy() && !b.IsDraining() {
			result = append(result, b)
		}
	}
	return result
}
