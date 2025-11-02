package server

import (
	"context"
	"log"
	"net/http"
	"time"
)

type HTTPServer struct {
	server *http.Server
}

func New(addr string) *HTTPServer {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status": "ok"}`))
	})

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
