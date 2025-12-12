package api

import (
	"net/http"

	"github.com/diabeney/balto/internal/api/core"
	"github.com/diabeney/balto/internal/api/health"
	"github.com/diabeney/balto/internal/api/services"
	"github.com/diabeney/balto/internal/metrics"
	"github.com/diabeney/balto/pkg/utils"
)

func SetupRoutes(mux *http.ServeMux) {
	core.InitServiceManager()

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
}
