package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/diabeney/balto/internal/health"
)

type HTTPServer struct {
	server *http.Server
}

func New(addr string, proxyHandler http.Handler) *HTTPServer {
	mux := http.NewServeMux()

	mux.Handle("/health", http.HandlerFunc(health.CheckBaltoHealth))
	mux.Handle("/", proxyHandler)

	return &HTTPServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
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
