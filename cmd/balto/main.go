package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"
	"github.com/diabeney/balto/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

    //TODO: Load from config
	srv := server.New(":8080")

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)

	defer cancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("Balto server stopped.")
}
