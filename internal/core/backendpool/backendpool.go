package backendpool

import (
	"log"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/balancer"
	"github.com/diabeney/balto/internal/core/circuit"
)

func NewBackend(id string, u *url.URL, weight uint32, cbCfg circuit.Config) *core.Backend {
	b := &core.Backend{
		ID:      id,
		URL:     u,
		Weight:  weight,
		Meta:    &core.BackendMetadata{},
		Circuit: circuit.New(cbCfg),
	}
	b.SetHealthy(true)
	return b
}

type PoolConfig struct {
	ServiceName            string
	HealthThreshold        uint64
	ProbeHealthThreshold   uint64
	ProbeRecoveryThreshold uint64 // Consecutive successful probes required to mark healthy
	ProbePath              string
	ProbeInterval          int
	Timeout                int
	Retry                  int

	// Circuit Breaker Config
	CircuitFailureThreshold    uint64
	CircuitSuccessThreshold    uint64
	CircuitTimeout             int // in seconds
	CircuitMaxHalfOpenRequests uint32
}

type BackendList struct {
	Items []*core.Backend
}

type Pool struct {
	backends atomic.Pointer[BackendList]
	balancer balancer.Balancer
	config   atomic.Pointer[PoolConfig]

	// For operations that require scanning and updates we still use a small mutex
	opMu sync.Mutex
}

func New(poolCfg *PoolConfig, bal balancer.Balancer) *Pool {
	p := &Pool{
		balancer: bal,
	}
	//TODO: Validate the required fields
	p.config.Store(poolCfg)
	p.backends.Store(&BackendList{Items: []*core.Backend{}})

	if p.balancer != nil {
		p.balancer.Update([]*core.Backend{})
	}
	return p
}

func (p *Pool) snapshot() []*core.Backend {
	bl := p.backends.Load()
	if bl == nil || bl.Items == nil {
		return nil
	}
	return bl.Items
}

// List returns a read-only reference to the current backend slice.
func (p *Pool) List() []*core.Backend {
	items := p.snapshot()
	if items == nil {
		return nil
	}
	return items
}

func (p *Pool) Add(id string, u *url.URL, weight uint32) {
	p.opMu.Lock()
	defer p.opMu.Unlock()

	old := p.backends.Load()
	oldItems := []*core.Backend{}
	if old != nil && old.Items != nil {
		oldItems = old.Items
	}

	newItems := make([]*core.Backend, len(oldItems)+1)
	copy(newItems, oldItems)

	cfg := p.Config()
	cbCfg := circuit.Config{
		FailureThreshold:    cfg.CircuitFailureThreshold,
		SuccessThreshold:    cfg.CircuitSuccessThreshold,
		Timeout:             time.Duration(cfg.CircuitTimeout) * time.Second,
		MaxHalfOpenRequests: cfg.CircuitMaxHalfOpenRequests,
	}

	newB := NewBackend(id, u, weight, cbCfg)
	newItems[len(newItems)-1] = newB

	p.backends.Store(&BackendList{Items: newItems})
	if p.balancer != nil {
		p.balancer.Update(newItems)
	}
}

func (p *Pool) Remove(id string) {
	p.opMu.Lock()
	defer p.opMu.Unlock()

	old := p.backends.Load()
	if old == nil || len(old.Items) == 0 {
		return
	}
	idx := -1
	for i, b := range old.Items {
		if b.ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}
	newItems := filterSlice(old, idx)

	p.backends.Store(&BackendList{Items: newItems})
	if p.balancer != nil {
		p.balancer.Update(newItems)
	}
}

func (p *Pool) SetConfig(cfg *PoolConfig) {
	p.config.Store(cfg)
}

func (p *Pool) Config() *PoolConfig {
	cfg := p.config.Load()
	if cfg == nil {
		return &PoolConfig{}
	}
	out := *cfg
	return &out
}

func (p *Pool) Balancer() balancer.Balancer {
	return p.balancer
}

func (p *Pool) Next() *core.Backend {
	if p.balancer == nil {
		return nil
	}
	backends := p.List()
	if len(backends) == 0 {
		return nil
	}
	candidates := make([]*core.Backend, 0, len(backends))
	for _, b := range backends {
		if !b.IsHealthy() || b.IsDraining() {
			continue
		}
		if b.Circuit != nil && !b.Circuit.Allow() {
			continue
		}
		candidates = append(candidates, b)
	}
	if len(candidates) == 0 {
		return nil
	}
	return p.balancer.Next(candidates)
}

func (p *Pool) RecordSuccess(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.RecordSuccess()
	if b.Circuit != nil {
		// RecordSuccess corresponds to request traffic, so it should not attempt
		// to reopen an Open circuit. The breaker handles that internally.
		b.Circuit.RecordSuccess()
	}
	// We dont bring back the backend if it was unhealthy due to failCount,
	// it will be brought back by the health checker. We want to make
	// the health checker the only source of truth for backend health.
}

func (p *Pool) RecordFailure(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.RecordFailure()
	if b.Circuit != nil {
		b.Circuit.RecordFailure()
	}
	cfg := p.config.Load()
	if cfg == nil {
		return
	}
	threshold := passiveThreshold(cfg)
	if threshold == 0 {
		return
	}
	if b.Meta.PassiveFailCount.Load() >= threshold {
		if b.SetHealthy(false) {
			log.Printf("BackendDown: (%s) is now unhealthy (threshold reached)", b.URL)
		}
	}
}

func (p *Pool) CheckHealth(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	cfg := p.config.Load()
	if cfg == nil {
		return
	}
	passive := b.Meta.PassiveFailCount.Load()
	probe := b.Meta.ProbeFailCount.Load()
	passiveThreshold := passiveThreshold(cfg)
	probeThreshold := probeThreshold(cfg)
	if passive >= passiveThreshold || probe >= probeThreshold {
		if b.SetHealthy(false) {
			log.Printf("BackendDown: (%s) is now unhealthy (check health)", b.URL)
		}
	}
}

func (p *Pool) ResetHealth(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.ResetAllFailCounts()
	if b.SetHealthy(true) {
		log.Printf("BackendRecovered: (%s) health reset manually", b.URL)
	}
}

func (p *Pool) MarkHealthy(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}

	b.Meta.LastSuccess.Store(time.Now().UnixNano())
	// Reset fail count on success so we start fresh
	b.Meta.ResetAllFailCounts()

	b.Meta.IncrementProbeSuccessCount()

	if b.Circuit != nil {
		b.Circuit.RecordProbeSuccess()
	}

	cfg := p.config.Load()
	if cfg == nil {
		return
	}

	// Default to 5 if not configured
	recoveryThreshold := cfg.ProbeRecoveryThreshold
	if recoveryThreshold == 0 {
		recoveryThreshold = 5
	}

	if b.Meta.ProbeSuccessCount.Load() >= recoveryThreshold {
		if !b.IsHealthy() {
			if b.SetHealthy(true) {
				log.Printf("BackendRecovered: (%s) marked healthy by Probe", b.URL)
			}
		}
	}
}

func (p *Pool) MarkUnhealthy(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.RecordProbeFailure()

	b.Meta.ResetProbeSuccessCount()

	if b.Circuit != nil {
		b.Circuit.RecordFailure()
	}

	cfg := p.config.Load()
	if cfg == nil {
		return
	}
	if b.Meta.ProbeFailCount.Load() >= probeThreshold(cfg) {
		if b.SetHealthy(false) {
			log.Printf("BackendDown: (%s) marked unhealthy by probe", b.URL)
		}
	}
}

func (p *Pool) StartDraining(id string) {
	p.opMu.Lock()
	defer p.opMu.Unlock()

	items := p.List()
	for _, b := range items {
		if b.ID == id {
			//SetDraining is internally protected by atomics,
			// so this single-field update is safe.
			b.SetDraining(true)
			return
		}
	}
}

// TODO: We can collect metrics and adjust the drain timeout based on percentile data
func (p *Pool) WaitForDrain(id string, timeout time.Duration) bool {
	// This function does NOT hold the lock inside the loop
	// because that would stall all Add/Remove operations for the entire timeout duration.
	// We only need it just for the brief moment we retrieve the list
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		p.opMu.Lock()
		items := p.List()

		found := false
		for _, b := range items {
			if b.ID == id && b.IsDraining() {
				found = true
				if b.Meta.Active() == 0 {
					// Release the lock immediately upon success
					p.opMu.Unlock()
					return true
				}
			}
		}
		p.opMu.Unlock()

		if !found {
			return false
		}
		//TODO: Make this configurable
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func filterSlice(old *BackendList, idx int) []*core.Backend {
	newItems := make([]*core.Backend, 0, len(old.Items)-1)
	newItems = append(newItems, old.Items[:idx]...)
	newItems = append(newItems, old.Items[idx+1:]...)
	return newItems
}

func passiveThreshold(cfg *PoolConfig) uint64 {
	if cfg == nil {
		return 0
	}
	return cfg.HealthThreshold
}

func probeThreshold(cfg *PoolConfig) uint64 {
	if cfg == nil {
		return 0
	}
	if cfg.ProbeHealthThreshold != 0 {
		return cfg.ProbeHealthThreshold
	}
	return passiveThreshold(cfg)
}
