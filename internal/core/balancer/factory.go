package balancer

import (
	"github.com/diabeney/balto/internal/core/balancer/leastconn"
	"github.com/diabeney/balto/internal/core/balancer/roundrobin"
	"github.com/diabeney/balto/internal/core/balancer/weightedrr"
)

func NewLeastConnections() Balancer {
	return leastconn.New()
}

func NewRoundRobin() Balancer {
	return roundrobin.New()
}

func NewWeightedRR() Balancer {
	return weightedrr.New()
}
