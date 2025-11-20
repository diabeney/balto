package circuit

import (
	"sync"
	"sync/atomic"
	"time"
)

type State uint32

const (
	Closed   State = 0
	Open     State = 1
	HalfOpen State = 2
)

func (s State) String() string {
	switch s {
	case Closed:
		return "Closed"
	case Open:
		return "Open"
	case HalfOpen:
		return "Half-Open"
	default:
		return "Unknown"
	}
}

type Config struct {
	FailureThreshold    uint64        // consecutive failures before opening the breaker
	SuccessThreshold    uint64        // consecutive successes in Half-Open required to close
	Timeout             time.Duration // base Open timeout before allowing Half-Open
	MaxHalfOpenRequests uint32        // bounds the number of concurrent trial requests while Half-Open.
}

type Breaker struct {
	// Hot path fields
	state            atomic.Uint32
	openTime         atomic.Int64 // Timestamp (UnixNano) when the breaker transitioned to Open.
	halfOpenInFlight atomic.Uint32
	openTimeout      atomic.Int64

	// Cold path fields
	mu        sync.Mutex
	failures  uint64 // Consecutive failures in Closed state.
	successes uint64 // Consecutive successes in Half-Open state.

	cfg Config
}

func New(cfg Config) *Breaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold == 0 {
		cfg.SuccessThreshold = 5
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.MaxHalfOpenRequests == 0 {
		cfg.MaxHalfOpenRequests = 3
	}

	b := &Breaker{
		cfg: cfg,
	}
	b.state.Store(uint32(Closed))
	b.openTimeout.Store(cfg.Timeout.Nanoseconds())
	return b
}

func (b *Breaker) State() State {
	return State(b.state.Load())
}

// Allow checks if a request should be allowed. It is lock-free for the Closed state.
func (b *Breaker) Allow() bool {
	s := State(b.state.Load())

	if s == Closed {
		return true
	}

	if s == Open {
		return b.checkAndTransitionOpen()
	}

	if s == HalfOpen {
		return b.tryAcquireHalfOpenSlot()
	}

	return false
}

// RecordSuccess processes a successful request from regular traffic.
func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	s := State(b.state.Load())

	if s == Closed {
		b.failures = 0
		b.successes = 0
		return
	}

	if s == HalfOpen {
		b.releaseHalfOpenSlotLocked()
		b.successes++
		if b.successes >= b.cfg.SuccessThreshold {
			b.transitionToClosedLocked()
		}
		return
	}
	// Successes while Open are ignored, only probes can trigger recovery.
}

// RecordProbeSuccess processes a successful result from health checker.
func (b *Breaker) RecordProbeSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()

	s := State(b.state.Load())

	if s == Closed {
		b.failures = 0
		b.successes = 0
		return
	}

	if s == HalfOpen {
		b.releaseHalfOpenSlotLocked()
		b.successes++
		if b.successes >= b.cfg.SuccessThreshold {
			b.transitionToClosedLocked()
		}
		return
	}

	if s == Open {
		// A successful probe transitions the breaker to Half-Open
		// and clears counters to begin the recovery phase.
		b.transitionToHalfOpenLocked()
	}
}

// RecordFailure processes a failed request result (either traffic or probe).
func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	s := State(b.state.Load())

	if s == Closed {
		b.failures++
		if b.failures >= b.cfg.FailureThreshold {
			b.transitionToOpenLocked()
		}
		return
	}

	if s == HalfOpen {
		// Failure immediately transitions back to Open.
		b.releaseHalfOpenSlotLocked()
		b.transitionToOpenLocked()
		return
	}
}

func (b *Breaker) transitionToOpenLocked() {
	b.state.Store(uint32(Open))
	b.openTime.Store(time.Now().UnixNano())
	b.successes = 0
	b.failures = 0
	b.halfOpenInFlight.Store(0)
	b.adjustOpenTimeoutLocked(increaseTimeout)
}

func (b *Breaker) transitionToHalfOpenLocked() {
	b.state.Store(uint32(HalfOpen))
	b.openTime.Store(0)
	b.failures = 0
	b.successes = 0
	b.halfOpenInFlight.Store(0)
	b.adjustOpenTimeoutLocked(resetTimeout)
}

func (b *Breaker) transitionToClosedLocked() {
	b.state.Store(uint32(Closed))
	b.failures = 0
	b.successes = 0
	b.halfOpenInFlight.Store(0)
	b.adjustOpenTimeoutLocked(resetTimeout)
}

// tryAcquireHalfOpenSlot attempts to atomically increment the in-flight counter.
func (b *Breaker) tryAcquireHalfOpenSlot() bool {
	limit := b.cfg.MaxHalfOpenRequests
	if limit == 0 {
		limit = 1
	}
	for {
		cur := b.halfOpenInFlight.Load()
		if cur >= limit {
			return false
		}
		if b.halfOpenInFlight.CompareAndSwap(cur, cur+1) {
			return true
		}
	}
}

func (b *Breaker) releaseHalfOpenSlotLocked() {
	b.halfOpenInFlight.Add(^uint32(0))
}

func (b *Breaker) checkAndTransitionOpen() bool {
	now := time.Now().UnixNano()
	openedAt := b.openTime.Load()

	timeout := time.Duration(b.openTimeout.Load())
	if timeout <= 0 {
		timeout = b.cfg.Timeout
	}

	if time.Duration(now-openedAt) <= timeout {
		return false
	}

	// Winner attempts to transition state using CAS.
	if b.state.CompareAndSwap(uint32(Open), uint32(HalfOpen)) {

		// Goroutine that won the CAS is responsible for cold path cleanup.
		b.mu.Lock()
		b.transitionToHalfOpenLocked()
		b.mu.Unlock()

		// The winning goroutine must then acquire a slot to ensure its subsequent
		// completion (RecordSuccess/Failure) correctly releases the slot.
		return b.tryAcquireHalfOpenSlot()

	} else {
		// Another goroutine won the CAS or the state changed.
		// If the state is now HalfOpen, try to acquire a slot.
		return State(b.state.Load()) == HalfOpen && b.tryAcquireHalfOpenSlot()
	}
}

const (
	increaseTimeout = iota
	resetTimeout
)

func (b *Breaker) adjustOpenTimeoutLocked(mode int) {
	base := b.cfg.Timeout
	if base <= 0 {
		base = 10 * time.Second
	}

	current := time.Duration(b.openTimeout.Load())
	if current <= 0 {
		current = base
	}

	switch mode {
	case increaseTimeout:
		next := current * 2
		max := base * 10
		// Cap exponential backoff at 10× the base timeout so the breaker still
		// periodically tests the backend instead of backing off forever.
		if next > max {
			next = max
		}
		b.openTimeout.Store(next.Nanoseconds())
	default:
		b.openTimeout.Store(base.Nanoseconds())
	}
}
