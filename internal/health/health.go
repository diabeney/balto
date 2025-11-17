package health

import (
	"context"
	"net/http"
	"time"

	"github.com/diabeney/balto/internal/core/backendpool"
)

func CheckBaltoHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status": "ok"}`))
}

func StartBackendHealthCheck(ctx context.Context, pool *backendpool.Pool) context.CancelFunc {
	pCfg := pool.Config()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(time.Duration(pCfg.Timeout))
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				items := pool.List()
				for _, b := range items {
					// only probe backends that are not explicitly draining
					if b.IsDraining() {
						continue
					}
					probeURL := *b.URL
					probeURL.Path = singleJoin(probeURL.Path, pCfg.ProbePath)
					client := &http.Client{Timeout: time.Duration(pCfg.Timeout)}
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL.String(), nil)
					if err != nil {
						continue
					}
					resp, err := client.Do(req)
					if err != nil {
						pool.RecordFailure(b)
						continue
					}
					resp.Body.Close()
					pool.RecordSuccess(b)
				}
			}
		}
	}()
	return cancel
}

func singleJoin(a, b string) string {
	if a == "" {
		a = "/"
	}
	if b == "" {
		return a
	}
	if a[len(a)-1] == '/' {
		a = a[:len(a)-1]
	}
	if b[0] != '/' {
		b = "/" + b
	}
	return a + b
}
