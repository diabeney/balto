package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/diabeney/balto/internal/api"
	"github.com/diabeney/balto/internal/config"
	"github.com/diabeney/balto/internal/health"
	"github.com/diabeney/balto/internal/metrics"
)

type HTTPServer struct {
	server  *http.Server
	metrics *metrics.Collector
}

func New(addr string, proxyHandler http.Handler, cfg *config.Config) *HTTPServer {
	return NewWithMetrics(addr, proxyHandler, cfg, nil)
}

func NewWithMetrics(addr string, proxyHandler http.Handler, cfg *config.Config, m *metrics.Collector) *HTTPServer {
	mux := http.NewServeMux()

	mux.Handle("/health", http.HandlerFunc(health.CheckBaltoHealth))
	api.SetupRoutes(mux)

	// Start background goroutine to update system metrics periodically if metrics collector is provided
	if m != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				m.RecordSystemMetrics()
			}
		}()
	}

	mux.Handle("/", proxyHandler)

	readTimeout, err := cfg.Global.Timeouts.ReadDuration()
	if err != nil {
		log.Printf("Invalid read timeout config, using default: %v", err)
		readTimeout = 5 * time.Second
	}

	writeTimeout, err := cfg.Global.Timeouts.WriteDuration()
	if err != nil {
		log.Printf("Invalid write timeout config, using default: %v", err)
		writeTimeout = 5 * time.Second
	}

	idleTimeout, err := cfg.Global.Timeouts.IdleDuration()
	if err != nil {
		log.Printf("Invalid idle timeout config, using default: %v", err)
		idleTimeout = 30 * time.Second
	}

	return &HTTPServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: readTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		metrics: m,
	}
}

func (h *HTTPServer) Start() error {
	log.Printf("Starting Balto HTTP server on %s", h.server.Addr)
	err := h.server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (h *HTTPServer) Stop(ctx context.Context) error {
	log.Printf("Shutting down Balto HTTP server...")
	return h.server.Shutdown(ctx)
}
