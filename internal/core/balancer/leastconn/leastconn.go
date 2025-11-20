package leastconn

import (
	"github.com/diabeney/balto/internal/core"
)

type LeastConnections struct {
	list []*core.Backend
}

func New() *LeastConnections {
	return &LeastConnections{}
}

func (l *LeastConnections) Update(backends []*core.Backend) {
	l.list = backends
}

func (l *LeastConnections) Next(backends []*core.Backend) *core.Backend {
	if backends == nil {
		backends = l.list
	}
	if len(backends) == 0 {
		return nil
	}

	best := backends[0]
	bestConn := best.Meta.Active()

	for _, b := range backends[1:] {
		if c := b.Meta.Active(); c < bestConn {
			best = b
			bestConn = c
		}
	}

	return best
}
