package api

import (
	"net/http"

	"github.com/diabeney/balto/internal/api/core"
	"github.com/diabeney/balto/internal/api/health"
	metricsAPI "github.com/diabeney/balto/internal/api/metrics"
	"github.com/diabeney/balto/internal/api/services"
	"github.com/diabeney/balto/internal/api/websocket"
	"github.com/diabeney/balto/internal/metrics"
	"github.com/diabeney/balto/pkg/broadcast"
	"github.com/diabeney/balto/pkg/utils"
)

func SetupRoutes(mux *http.ServeMux, broadcaster *broadcast.Broadcaster) {
	core.InitServiceManager(broadcaster)
	if broadcaster != nil {
		websocket.SetupBroadcastSubscription(broadcaster)
	}

	mux.HandleFunc("/balto/health/stats", health.Handler)

	mux.HandleFunc("/balto/services", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			services.ListHandler(w, r)
		case http.MethodPost:
			services.AddHandler(w, r)
		default:
			utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/balto/services/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			services.GetHandler(w, r)
		case http.MethodPut, http.MethodPatch:
			services.UpdateHandler(w, r)
		default:
			utils.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})

	mux.HandleFunc("/balto/metrics", func(w http.ResponseWriter, r *http.Request) {
		collector := metrics.GetGlobal()
		if collector == nil {
			utils.WriteError(w, http.StatusServiceUnavailable, "metrics collector not initialized")
			return
		}
		collector.Handler().ServeHTTP(w, r)
	})

	mux.HandleFunc("/balto/metrics/aggregated", metricsAPI.AggregatedHandler)
	mux.HandleFunc("/balto/metrics/timeseries", metricsAPI.TimeSeriesHandler)
	mux.HandleFunc("/balto/metrics/timeseries/all", metricsAPI.AllTimeSeriesHandler)
	mux.HandleFunc("/balto/metrics/snapshot", metricsAPI.SnapshotHandler)
	mux.HandleFunc("/balto/metrics/backends", metricsAPI.BackendsHandler)
	mux.HandleFunc("/balto/metrics/latency", metricsAPI.LatencyBreakdownHandler)
	mux.HandleFunc("/balto/metrics/system", metricsAPI.SystemHandler)
	mux.HandleFunc("/balto/metrics/stream", metricsAPI.StreamHandler)
	mux.HandleFunc("/balto/metrics/requests", metricsAPI.RecentRequestsHandler)

	mux.HandleFunc("/balto/ws", websocket.Handler)
}
