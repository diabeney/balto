package core

import (
	"net/url"
	"sync/atomic"
	"time"
)

const (
	FlagHealthy  = 1 << 0
	FlagDraining = 1 << 1
)

type BackendMetadata struct {
	FailCount     atomic.Uint64
	LastFailure   atomic.Int64
	LastSuccess   atomic.Int64
	ActiveConns   atomic.Uint64
	TotalRequests atomic.Uint64

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
	m.TotalRequests.Add(1)
	m.FailCount.Add(1)
	m.LastFailure.Store(time.Now().UnixNano())
}

func (m *BackendMetadata) ResetFailCount() {
	m.FailCount.Store(0)
}

type Backend struct {
	ID     string
	URL    *url.URL
	Weight uint32
	State  atomic.Uint32 // bitmask flags
	Meta   *BackendMetadata
}

func (b *Backend) IsHealthy() bool {
	return b.State.Load()&FlagHealthy != 0
}

func (b *Backend) IsDraining() bool {
	return b.State.Load()&FlagDraining != 0
}

func (b *Backend) SetHealthy(healthy bool) {
	for {
		old := b.State.Load()
		var newState uint32
		if healthy {
			newState = old | FlagHealthy
		} else {
			newState = old &^ FlagHealthy
		}
		if b.State.CompareAndSwap(old, newState) {
			return
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
