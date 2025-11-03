package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpointOK(t *testing.T) {
    s := New(":0")

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


