package circuit

import (
	"testing"
	"time"
)

func TestBreakerTransitions(t *testing.T) {
	cfg := Config{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             100 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := New(cfg)

	if cb.State() != Closed {
		t.Errorf("expected Closed, got %v", cb.State())
	}
	if !cb.Allow() {
		t.Error("should allow in Closed state")
	}

	// Closed -> Open
	cb.RecordFailure()
	if cb.State() != Closed {
		t.Errorf("expected Closed after 1 failure, got %v", cb.State())
	}
	cb.RecordFailure()
	if cb.State() != Open {
		t.Errorf("expected Open after 2 failures, got %v", cb.State())
	}
	if cb.Allow() {
		t.Error("should not allow in Open state")
	}

	// Open -> Half-Open (Time based) with backoff (timeout doubles to 200ms)
	time.Sleep(250 * time.Millisecond)
	if !cb.Allow() {
		t.Error("should allow (transition to Half-Open) after timeout")
	}
	if cb.State() != HalfOpen {
		t.Errorf("expected Half-Open, got %v", cb.State())
	}

	// Half-Open -> Open (Failure)
	cb.RecordFailure()
	if cb.State() != Open {
		t.Errorf("expected Open after failure in Half-Open, got %v", cb.State())
	}

	// Open -> Half-Open again (timeout currently 200ms)
	time.Sleep(250 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected transition to Half-Open after timeout")
	}
	if cb.State() != HalfOpen {
		t.Errorf("expected Half-Open, got %v", cb.State())
	}

	// Half-Open -> Closed (Successes)
	cb.RecordSuccess()
	if cb.State() != HalfOpen {
		t.Errorf("expected Half-Open after 1 success (threshold 2), got %v", cb.State())
	}
	cb.RecordSuccess()
	if cb.State() != Closed {
		t.Errorf("expected Closed after 2 successes, got %v", cb.State())
	}
	if !cb.Allow() {
		t.Error("should allow in Closed state")
	}
}

func TestBreakerSuccessInOpen(t *testing.T) {
	// If a health check succeeds while Open, it should help recovery
	cfg := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          1 * time.Hour,
	}
	cb := New(cfg)

	cb.RecordFailure()
	if cb.State() != Open {
		t.Fatalf("expected Open")
	}

	cb.RecordProbeSuccess()
	if cb.State() != HalfOpen {
		t.Errorf("expected HalfOpen after probe success in Open, got %v", cb.State())
	}

	if cb.cfg.SuccessThreshold == 1 {
		cb.RecordProbeSuccess()
		if cb.State() != Closed {
			t.Errorf("expected Closed after 2nd probe success, got %v", cb.State())
		}
	}
}

func TestBreakerRecordSuccessIgnoredWhenOpen(t *testing.T) {
	cfg := Config{
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          1 * time.Second,
	}
	cb := New(cfg)

	cb.RecordFailure()
	if cb.State() != Open {
		t.Fatalf("expected Open")
	}

	cb.RecordSuccess()
	if cb.State() != Open {
		t.Fatalf("expected Open to remain until probe success, got %v", cb.State())
	}
}

func TestBreakerHalfOpenLimited(t *testing.T) {
	cfg := Config{
		FailureThreshold:    1,
		SuccessThreshold:    2,
		Timeout:             10 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := New(cfg)

	cb.RecordFailure()
	if cb.State() != Open {
		t.Fatalf("expected Open after failure")
	}

	time.Sleep(25 * time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("expected first trial request allowed")
	}

	if cb.Allow() {
		t.Fatalf("expected second concurrent trial to be denied")
	}

	cb.RecordFailure()
	if cb.State() != Open {
		t.Fatalf("expect Open after Half-Open failure")
	}
}

func TestBreakerOpenTimeoutBackoff(t *testing.T) {
	cfg := Config{
		FailureThreshold:    1,
		SuccessThreshold:    1,
		Timeout:             5 * time.Millisecond,
		MaxHalfOpenRequests: 1,
	}
	cb := New(cfg)

	if got := time.Duration(cb.openTimeout.Load()); got != cfg.Timeout {
		t.Fatalf("expected initial open timeout %v, got %v", cfg.Timeout, got)
	}

	cb.RecordFailure()
	if cb.State() != Open {
		t.Fatalf("expected Open after failure")
	}

	firstBackoff := time.Duration(cb.openTimeout.Load())
	if firstBackoff != cfg.Timeout*2 {
		t.Fatalf("expected backoff %v, got %v", cfg.Timeout*2, firstBackoff)
	}

	time.Sleep(firstBackoff + 1*time.Millisecond)
	cb.Allow()
	cb.RecordFailure()

	secondBackoff := time.Duration(cb.openTimeout.Load())
	if secondBackoff != cfg.Timeout*2 {
		t.Fatalf("expected second backoff %v, got %v", cfg.Timeout*2, secondBackoff)
	}

	time.Sleep(secondBackoff + 1*time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("expected allow after backoff")
	}
	cb.RecordSuccess()
	if cb.State() != Closed {
		t.Fatalf("expected Closed after success")
	}

	reset := time.Duration(cb.openTimeout.Load())
	if reset != cfg.Timeout {
		t.Fatalf("expected timeout reset to %v, got %v", cfg.Timeout, reset)
	}
}
