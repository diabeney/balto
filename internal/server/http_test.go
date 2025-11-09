package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/diabeney/balto/internal/proxy"
	"github.com/diabeney/balto/internal/router"
)

func TestHealthEndpointOK(t *testing.T) {
	healthRoutes := []router.InitialRoutes{
		{
			Domain:     "www.jedevent.com",
			Ports:      []string{"80"},
			PathPrefix: "/",
		},
	}
	rt, err := router.BuildFromConfig(healthRoutes)

	if err != nil {
		t.Errorf("error building routes from config")
	}

	router.SetCurrent(rt)

	proxySrv := proxy.New(router.Current())
	s := New(":0", http.HandlerFunc(proxySrv.ServeHTTP))

	testSrv := httptest.NewServer(s.server.Handler)
	defer testSrv.Close()

	resp, err := http.Get(testSrv.URL + "/health")
	if err != nil {
		t.Fatalf("failed to GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	expected := "{\"status\": \"ok\"}"
	if string(body) != expected {
		t.Fatalf("expected body %s, got %s", expected, string(body))
	}
}
