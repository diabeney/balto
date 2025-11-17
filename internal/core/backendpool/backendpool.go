package backendpool

import (
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/balancer"
)

func NewBackend(id string, u *url.URL, weight uint32) *core.Backend {
	b := &core.Backend{
		ID:     id,
		URL:    u,
		Weight: weight,
		Meta:   &core.BackendMetadata{},
	}
	b.SetHealthy(true)
	return b
}

type PoolConfig struct {
	ServiceName     string
	HealthThreshold uint64
	ProbePath       string
	Timeout         int
	Retry           int
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

func (p *Pool) List() []*core.Backend {
	items := p.snapshot()
	if items == nil {
		return nil
	}
	out := make([]*core.Backend, len(items))
	copy(out, items)
	return out
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

	newB := NewBackend(id, u, weight)
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
	return p.balancer.Next(backends)
}

// RecordSuccess should be called when a request to backend succeeded.
// It updates metadata and, if failure counters were causing unhealthy state, flip healthy.
func (p *Pool) RecordSuccess(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.RecordSuccess()
	// If backend was unhealthy due to failCount, bring it back based on thresholds
	cfg := p.config.Load()
	if cfg == nil {
		return
	}
	if !b.IsHealthy() {
		b.SetHealthy(true)
	}
}

func (p *Pool) RecordFailure(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.RecordFailure()
	cfg := p.config.Load()
	if cfg == nil {
		return
	}
	if b.Meta.FailCount.Load() >= cfg.HealthThreshold {
		b.SetHealthy(false)
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
	if b.Meta.FailCount.Load() >= cfg.HealthThreshold {
		b.SetHealthy(false)
	}
}

func (p *Pool) ResetHealth(b *core.Backend) {
	if b == nil || b.Meta == nil {
		return
	}
	b.Meta.ResetFailCount()
	b.SetHealthy(true)
}

func (p *Pool) StartDraining(id string) {
	items := p.snapshot()
	for _, b := range items {
		if b.ID == id {
			b.SetDraining(true)
			return
		}
	}
}

// TODO: We can collect metrics and adjust the drain timeout based on percentile data
func (p *Pool) WaitForDrain(id string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		items := p.snapshot()
		found := false
		for _, b := range items {
			if b.ID == id && b.IsDraining() {
				found = true
				if b.Meta.Active() == 0 {
					return true
				}
			}
		}
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
