package core

import (
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestBackendMetadataActiveConnections(t *testing.T) {
	t.Run("IncrActive and DecrActive", func(t *testing.T) {
		m := &BackendMetadata{}
		m.IncrActive()
		m.IncrActive()
		if m.Active() != 2 {
			t.Errorf("expected 2 active, got %d", m.Active())
		}
		m.DecrActive()
		if m.Active() != 1 {
			t.Errorf("expected 1 active after decr, got %d", m.Active())
		}
	})

	t.Run("DecrActive below zero", func(t *testing.T) {
		m := &BackendMetadata{}
		m.DecrActive()
		if m.Active() != ^uint64(0) {
			t.Errorf("expected max uint64 on underflow, got %d", m.Active())
		}
	})

	t.Run("Concurrent Incr/Decr", func(t *testing.T) {
		m := &BackendMetadata{}
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				m.IncrActive()
				time.Sleep(1 * time.Millisecond)
				m.DecrActive()
			}()
		}
		wg.Wait()
		if m.Active() != 0 {
			t.Errorf("expected 0 active after concurrent ops, got %d", m.Active())
		}
	})
}

func TestBackendMetadataRecordSuccess(t *testing.T) {
	t.Run("Increments TotalRequests and updates LastSuccess", func(t *testing.T) {
		m := &BackendMetadata{}
		before := time.Now().UnixNano()
		m.RecordSuccess()
		after := time.Now().UnixNano()

		if m.TotalRequests.Load() != 1 {
			t.Errorf("expected 1 total request, got %d", m.TotalRequests.Load())
		}
		last := m.LastSuccess.Load()
		if last < before || last > after {
			t.Errorf("LastSuccess timestamp out of range: %d (before: %d, after: %d)", last, before, after)
		}
	})

	t.Run("Concurrent RecordSuccess", func(t *testing.T) {
		m := &BackendMetadata{}
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				m.RecordSuccess()
			}()
		}
		wg.Wait()
		if m.TotalRequests.Load() != 100 {
			t.Errorf("expected 100 total requests, got %d", m.TotalRequests.Load())
		}
	})
}

func TestBackendMetadataRecordFailure(t *testing.T) {
	t.Run("Increments PassiveFailCount and TotalRequests", func(t *testing.T) {
		m := &BackendMetadata{}
		m.RecordFailure()
		if m.PassiveFailCount.Load() != 1 {
			t.Errorf("expected 1 passive fail, got %d", m.PassiveFailCount.Load())
		}
		if m.TotalRequests.Load() != 1 {
			t.Errorf("expected 1 total, got %d", m.TotalRequests.Load())
		}
	})

	t.Run("RecordProbeFailure increments ProbeFailCount only", func(t *testing.T) {
		m := &BackendMetadata{}
		m.RecordProbeFailure()
		if m.ProbeFailCount.Load() != 1 {
			t.Errorf("expected 1 probe fail, got %d", m.ProbeFailCount.Load())
		}
		if m.TotalRequests.Load() != 0 {
			t.Errorf("expected probe failures not to affect total, got %d", m.TotalRequests.Load())
		}
	})

	t.Run("ResetAllFailCounts", func(t *testing.T) {
		m := &BackendMetadata{}
		m.RecordFailure()
		m.RecordProbeFailure()
		m.ResetAllFailCounts()
		if m.PassiveFailCount.Load() != 0 {
			t.Errorf("expected 0 passive fails after reset, got %d", m.PassiveFailCount.Load())
		}
		if m.ProbeFailCount.Load() != 0 {
			t.Errorf("expected 0 probe fails after reset, got %d", m.ProbeFailCount.Load())
		}
	})
}

func TestBackendStateFlags(t *testing.T) {
	u, _ := url.Parse("http://localhost")
	b := &Backend{
		ID:     "test",
		URL:    u,
		Weight: 1,
		Meta:   &BackendMetadata{},
	}
	b.SetHealthy(true)

	t.Run("Default state is healthy", func(t *testing.T) {
		if !b.IsHealthy() {
			t.Error("new backend should be healthy by default")
		}
	})

	t.Run("SetHealthy toggles correctly", func(t *testing.T) {
		b.SetHealthy(false)
		if b.IsHealthy() {
			t.Error("expected unhealthy after SetHealthy(false)")
		}
		b.SetHealthy(true)
		if !b.IsHealthy() {
			t.Error("expected healthy after SetHealthy(true)")
		}
	})

	t.Run("SetDraining toggles correctly", func(t *testing.T) {
		b.SetDraining(true)
		if !b.IsDraining() {
			t.Error("expected draining after SetDraining(true)")
		}
		b.SetDraining(false)
		if b.IsDraining() {
			t.Error("expected not draining after SetDraining(false)")
		}
	})

	t.Run("Concurrent state changes", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(healthy bool) {
				defer wg.Done()
				b.SetHealthy(healthy)
			}(i%2 == 0)
		}
		wg.Wait()
	})
}
