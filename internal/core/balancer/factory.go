package balancer

import (
	"fmt"

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

func NewBalancerAlgo(algorithm string) (Balancer, error) {
	switch algorithm {
	case "round-robin":
		return NewRoundRobin(), nil
	case "least-connections":
		return NewLeastConnections(), nil
	case "weighted-rr":
		return NewWeightedRR(), nil
	default:
		return nil, fmt.Errorf("unknown load balancing algorithm: %q (supported: round-robin, least-connections, weighted-rr)", algorithm)
	}
}
