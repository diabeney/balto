package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/diabeney/balto/internal/proxy"
	"github.com/diabeney/balto/internal/router"
	"github.com/diabeney/balto/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	cfg := []router.InitialRoutes{
		{Domain: "localhost", PathPrefix: "*", Ports: []string{"8080", "8081", "8083"}},
	}

	rt, _ := router.BuildFromConfig(cfg)

	router.SetCurrent(rt)

	px := proxy.New(router.Current())

	//TODO: Load port from config
	srv := server.New(":80", http.HandlerFunc(px.ServeHTTP))

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Balto server stopped.")
}
