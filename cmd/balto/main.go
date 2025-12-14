package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/diabeney/balto/internal/api/core"
	"github.com/diabeney/balto/internal/config"
	"github.com/diabeney/balto/internal/metrics"
	"github.com/diabeney/balto/internal/proxy"
	"github.com/diabeney/balto/internal/router"
	"github.com/diabeney/balto/internal/server"
	"github.com/diabeney/balto/pkg/logger"
	"github.com/diabeney/balto/pkg/utils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()
	cfg, err := config.Load()
	if err != nil {
		// !TODO: Test and see if we really need to terminate the process if the config loader throws an error. The app should work with the default configs
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(&logger.Config{
		Level: cfg.Global.Logging.Level,
		Path:  cfg.Global.Logging.Path,
	})

	routes := make([]router.InitialRoutes, len(cfg.Services))
	for i, svc := range cfg.Services {
		routes[i] = router.InitialRoutes{
			Domain:     svc.Domain,
			PathPrefix: svc.PathPrefix,
			Ports:      svc.Ports,
		}
	}

	rt, err := router.BuildFromConfig(routes)
	if err != nil {
		log.Fatalf("Failed to build router: %v", err)
	}

	rt.StartHealthCheckers()

	router.SetCurrent(rt)

	var metricsCollector *metrics.Collector
	if cfg.Global.Metrics.Enabled {
		metricsCollector = metrics.NewCollector()
		metricsCollector.Register()
	}

	px := proxy.NewWithMetrics(router.Current(), metricsCollector)

	srv := server.NewWithMetrics(cfg.Server.Address(), http.HandlerFunc(px.ServeHTTP), cfg, metricsCollector)

	sm := core.GetServiceManager()
	if sm != nil {
		services := make([]core.ServiceInfo, len(cfg.Services))
		for i, svc := range cfg.Services {
			services[i] = core.ServiceInfo{
				ID:         utils.HashServiceID(svc.Domain, svc.PathPrefix),
				Domain:     svc.Domain,
				PathPrefix: svc.PathPrefix,
				Ports:      svc.Ports,
			}
		}

		err := sm.InitializeFromConfig(services)

		if err != nil {
			log.Fatalf("Failed to initialize routes from config: %v", err)
		}
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if rt := router.Current(); rt != nil {
		logger.Info(logger.BALTO_SERVER, "Stopping all health checkers...")
		if err := rt.Stop(); err != nil {
			logger.Error(logger.BALTO_SERVER, "Error stopping router health checkers", "error", err)
		}
	}

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error(logger.BALTO_SERVER, "Shutdown error", "error", err)
	}

	logger.Info(logger.BALTO_SERVER, "Balto server stopped.")
}
