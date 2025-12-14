package balancer

import (
	"fmt"
	"log"

	"github.com/diabeney/balto/internal/config"
	"github.com/diabeney/balto/internal/core/balancer/leastconn"
	"github.com/diabeney/balto/internal/core/balancer/roundrobin"
	"github.com/diabeney/balto/internal/core/balancer/weightedrr"
	"github.com/diabeney/balto/pkg/logger"
)

const (
	ROUND_ROBIN          string = "round-robin"
	LEAST_CONNECTIONS    string = "least-connections"
	WEIGHTED_ROUND_ROBIN string = "weighted-rr"
)

func NewLeastConnections() Balancer {
	logger.Info(logger.BALTO_BALANCER, "Initialized with least connections balancer")
	return leastconn.New()
}

func NewRoundRobin() Balancer {
	logger.Info(logger.BALTO_BALANCER, "Initialized with round robin balancer")
	return roundrobin.New()
}

func NewWeightedRR() Balancer {
	logger.Info(logger.BALTO_BALANCER, "Initialized with weighted round robin balancer")
	return weightedrr.New()
}

func InitBalancerAlgo() (Balancer, error) {
	cfg, err := config.Load()
	if err != nil {
		// !TODO: Test and see if we really need to terminate the process if the config loader throws an error. The app should work with the default configs
		log.Fatalf("Failed to load config: %v\n", err)
	}
	alg := cfg.Global.LoadBalancing.Algorithm
	switch alg {
	case ROUND_ROBIN:
		return NewRoundRobin(), nil
	case LEAST_CONNECTIONS:
		return NewLeastConnections(), nil
	case WEIGHTED_ROUND_ROBIN:
		return NewWeightedRR(), nil
	default:
		return nil, fmt.Errorf("unknown load balancing algorithm: %q (supported: %q, %q, %q)", alg, ROUND_ROBIN, LEAST_CONNECTIONS, WEIGHTED_ROUND_ROBIN)
	}
}
