package core

import (
	"net/url"
	"sync/atomic"
	"time"

	"github.com/diabeney/balto/internal/core/circuit"
)

const (
	FlagHealthy  = 1 << 0
	FlagDraining = 1 << 1
)

type BackendMetadata struct {
	PassiveFailCount  atomic.Uint64
	ProbeFailCount    atomic.Uint64
	ProbeSuccessCount atomic.Uint64 // Consecutive successful probes
	LastFailure       atomic.Int64
	LastSuccess       atomic.Int64
	ActiveConns       atomic.Uint64
	TotalRequests     atomic.Uint64

	// Needed only for Smooth Weighted Round Robin
	TempWeight int64
}

func (m *BackendMetadata) IncrActive() {
	m.ActiveConns.Add(1)
}

func (m *BackendMetadata) DecrActive() {
	m.ActiveConns.Add(^uint64(0)) // -1
}

func (m *BackendMetadata) Active() uint64 {
	return m.ActiveConns.Load()
}

func (m *BackendMetadata) RecordSuccess() {
	m.TotalRequests.Add(1)
	m.LastSuccess.Store(time.Now().UnixNano())
}

func (m *BackendMetadata) RecordFailure() {
	m.RecordPassiveFailure()
}

func (m *BackendMetadata) RecordPassiveFailure() {
	m.TotalRequests.Add(1)
	m.PassiveFailCount.Add(1)
	m.LastFailure.Store(time.Now().UnixNano())
}

func (m *BackendMetadata) RecordProbeFailure() {
	m.ProbeFailCount.Add(1)
	m.LastFailure.Store(time.Now().UnixNano())
}

func (m *BackendMetadata) ResetFailCount() {
	m.ResetPassiveFailCount()
}

func (m *BackendMetadata) ResetPassiveFailCount() {
	m.PassiveFailCount.Store(0)
}

func (m *BackendMetadata) ResetProbeFailCount() {
	m.ProbeFailCount.Store(0)
}

func (m *BackendMetadata) ResetAllFailCounts() {
	m.ResetPassiveFailCount()
	m.ResetProbeFailCount()
	// ProbeSuccessCount is NOT reset here, it only resets on probe failures
}

func (m *BackendMetadata) IncrementProbeSuccessCount() {
	m.ProbeSuccessCount.Add(1)
}

func (m *BackendMetadata) ResetProbeSuccessCount() {
	m.ProbeSuccessCount.Store(0)
}

type Backend struct {
	ID     string
	URL    *url.URL
	Weight uint32
	State  atomic.Uint32 // bitmask flags
	Meta   *BackendMetadata

	Circuit *circuit.Breaker
}

func (b *Backend) IsHealthy() bool {
	return b.State.Load()&FlagHealthy != 0
}

func (b *Backend) IsDraining() bool {
	return b.State.Load()&FlagDraining != 0
}

func (b *Backend) SetHealthy(healthy bool) bool {
	for {
		old := b.State.Load()
		var newState uint32
		if healthy {
			newState = old | FlagHealthy
		} else {
			newState = old &^ FlagHealthy
		}
		if b.State.CompareAndSwap(old, newState) {
			return (old & FlagHealthy) != (newState & FlagHealthy)
		}
	}
}

func (b *Backend) SetDraining(draining bool) {
	for {
		old := b.State.Load()
		var newState uint32
		if draining {
			newState = old | FlagDraining
		} else {
			newState = old &^ FlagDraining
		}
		if b.State.CompareAndSwap(old, newState) {
			return
		}
	}
}
