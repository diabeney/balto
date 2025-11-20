package backendpool_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
	"github.com/diabeney/balto/internal/core/balancer"
	"github.com/diabeney/balto/internal/core/circuit"
	"github.com/diabeney/balto/internal/health"
)

// Full integration test. It includes Pool + Balancer + Healthchecker + Circuit Breaker
func TestFullResilienceFlow(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer s1.Close()

	s2State := "ok"
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			if s2State == "probe_fail" || s2State == "fail" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		if s2State == "fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer s2.Close()

	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer s3.Close()

	probeInterval := 100 * time.Millisecond
	cbTimeout := 1 * time.Second
	poolCfg := &backendpool.PoolConfig{
		ServiceName:                "integration-test",
		ProbePath:                  "/",
		Timeout:                    100,
		HealthThreshold:            2,
		ProbeHealthThreshold:       2,
		ProbeInterval:              int(probeInterval.Milliseconds()),
		ProbeRecoveryThreshold:     1, // Set to 1 for quick recovery test
		CircuitFailureThreshold:    2,
		CircuitSuccessThreshold:    1,
		CircuitTimeout:             int(cbTimeout.Seconds()),
		CircuitMaxHalfOpenRequests: 1,
	}
	bal := balancer.NewRoundRobin()
	pool := backendpool.New(poolCfg, bal)

	u1, _ := url.Parse(s1.URL)
	u2, _ := url.Parse(s2.URL)
	u3, _ := url.Parse(s3.URL)

	pool.Add("s1", u1, 1)
	pool.Add("s2", u2, 1)
	pool.Add("s3", u3, 1)

	hc := health.New(pool)
	hc.Start()
	defer func() {
		if err := hc.Stop(); err != nil {
			t.Errorf("Error stopping health checker: %v", err)
		}
	}()

	// Give probes time to run (s3 should fail)
	time.Sleep(3 * probeInterval)

	// 3. Verify s3 is excluded by Active Health Check/Circuit Breaker
	backends := pool.List()
	var b3, b2 *core.Backend
	for _, b := range backends {
		switch b.ID {
		case "s3":
			b3 = b
		case "s2":
			b2 = b
		}
	}

	if b3.IsHealthy() {
		t.Errorf("s3 should be unhealthy due to continuous probe failures (Healthy: %v, PFC: %d)",
			b3.IsHealthy(), b3.Meta.ProbeFailCount.Load())
	}
	if b3.Circuit.State() != circuit.Open {
		t.Errorf("s3 circuit breaker should be Open, got %v", b3.Circuit.State())
	}

	counts := make(map[string]int)
	for i := 0; i < 20; i++ {
		b := pool.Next()
		if b == nil {
			t.Fatal("pool.Next returned nil, expected s1 and s2 to be available")
		}
		counts[b.ID]++
	}

	if counts["s3"] > 0 {
		t.Errorf("s3 should not be selected. Counts: %v", counts)
	}
	if counts["s1"] == 0 || counts["s2"] == 0 {
		t.Errorf("Load balancing should use s1 and s2. Counts: %v", counts)
	}

	t.Run("Circuit Breaker Test", func(t *testing.T) {
		s2State = "fail"

		// Force enough failures to open the circuit breaker (Threshold: 2)
		for i := 0; i < 2; i++ {
			b := pool.Next()
			if b.ID == "s2" {
				pool.RecordFailure(b)
				if b2.Circuit.State() == circuit.Open {
					break // Breaker opened
				}
			} else {
				// Consume s1 to ensure s2 is hit on the next iteration
				i--
				pool.RecordSuccess(b)
			}
		}

		if b2.Circuit.State() != circuit.Open {
			t.Fatalf("s2 circuit should be Open after 2 passive failures, got %v", b2.Circuit.State())
		}

		counts2 := make(map[string]int)
		for i := 0; i < 20; i++ {
			b := pool.Next()
			if b != nil {
				counts2[b.ID]++
			}
		}

		if counts2["s2"] > 0 {
			t.Errorf("s2 should be excluded by the Circuit Breaker. Counts: %v", counts2)
		}
		if counts2["s1"] == 0 {
			t.Errorf("s1 should be the only one left. Counts: %v", counts2)
		}
	})
	t.Run("Automatic Recovery Test", func(t *testing.T) {
		s2State = "ok"

		// Wait for CB timeout (1s) + margin for the probe to fire and record success
		time.Sleep(cbTimeout + 3*probeInterval)

		// The time wait should have allowed the CB to move Open -> HalfOpen,
		// the probe to hit the now-healthy s2 (succeeding), and the CB to move HalfOpen -> Closed.

		if b2.Circuit.State() != circuit.Closed {
			t.Fatalf("s2 circuit should be Closed after timeout and successful probe, got %v", b2.Circuit.State())
		}
		if !b2.IsHealthy() {
			t.Fatalf("s2 should be marked Healthy by probe recovery logic.")
		}

		counts3 := make(map[string]int)
		for i := 0; i < 20; i++ {
			b := pool.Next()
			if b != nil {
				counts3[b.ID]++
			}
		}

		if counts3["s2"] == 0 {
			t.Errorf("s2 should have recovered and be selected. Counts: %v", counts3)
		}
		if counts3["s1"] == 0 {
			t.Errorf("s1 must still be available. Counts: %v", counts3)
		}
	})
}
