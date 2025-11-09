package proxy_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/diabeney/balto/internal/proxy"
	"github.com/diabeney/balto/internal/router"
)

func setupTestBackend(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		w.Header().Set("X-Backend", "ok")
		w.Header().Set("X-Received-Path", r.URL.Path)
		w.Header().Set("X-Param-Id", r.Header.Get("X-Param-id"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("echo: " + string(body)))
	}))
}

func setupProxy(r *router.Router) *httptest.Server {
	router.SetCurrent(r)
	p := proxy.New(router.Current())
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		p.ServeHTTP(w, req)
	}))
}

func portFromURL(u string) string {
	parsed, _ := url.Parse(u)
	return parsed.Port()
}

func TestProxyStreamsRequestAndResponse(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()

	cfg := []router.InitialRoutes{
		{
			Domain:     "example.com",
			Ports:      []string{portFromURL(backend.URL)},
			PathPrefix: "/upload",
		},
	}
	rt, err := router.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	proxyServer := setupProxy(rt)
	defer proxyServer.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	body := strings.NewReader("Hello backend")

	req, _ := http.NewRequest("POST", proxyServer.URL+"/upload", body)
	req.Host = "example.com"

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(data), "Hello backend") {
		t.Errorf("expected streamed body, got %q", string(data))
	}

	if resp.Header.Get("X-Backend") != "ok" {
		t.Errorf("expected backend header, got %v", resp.Header.Get("X-Backend"))
	}

	if resp.Header.Get("X-Received-Path") != "/" {
		t.Errorf("expected stripped path '/', got %s", resp.Header.Get("X-Received-Path"))
	}
}

func TestProxyHandlesRouteNotFound(t *testing.T) {
	rt := router.NewRouter()
	proxyServer := setupProxy(rt)
	defer proxyServer.Close()

	resp, err := http.Get(proxyServer.URL + "/anything")
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestProxyBackendErrorReturnsBadGateway(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close() // force connection failure

	cfg := []router.InitialRoutes{
		{
			Domain:     "fail.com",
			Ports:      []string{fmt.Sprintf("%d", port)},
			PathPrefix: "/",
		},
	}
	rt, err := router.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	proxyServer := setupProxy(rt)
	defer proxyServer.Close()

	req, _ := http.NewRequest("GET", proxyServer.URL+"/", nil)
	req.Host = "fail.com"

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Network error is acceptable
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("expected 502 Bad Gateway, got %d", resp.StatusCode)
	}
}

func TestProxyCancelsBackendRequestOnClientClose(t *testing.T) {
	sawCancel := make(chan bool, 1)

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			sawCancel <- true
			return
		case <-time.After(5 * time.Second):
			_, _ = io.WriteString(w, "done")
		}
	}))
	defer backend.Close()

	port := portFromURL(backend.URL)

	cfg := []router.InitialRoutes{
		{
			Domain:     "cancel.com",
			Ports:      []string{port},
			PathPrefix: "/",
		},
	}
	rt, err := router.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	proxyServer := setupProxy(rt)
	defer proxyServer.Close()

	conn, err := net.Dial("tcp", proxyServer.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial proxy: %v", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: cancel.com\r\nConnection: close\r\n\r\n")

	time.Sleep(100 * time.Millisecond)

	conn.Close() // abruptly close client after 100ms

	select {
	case <-sawCancel:
		// Success: backend saw the abrupt client close
	case <-time.After(2 * time.Second):
		t.Errorf("expected backend context to be canceled when client disconnected")
	}
}

func TestProxyStripsPrefixCorrectly(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()

	cfg := []router.InitialRoutes{
		{
			Domain:     "strip.com",
			Ports:      []string{portFromURL(backend.URL)},
			PathPrefix: "/api/v1/*",
		},
	}
	rt, err := router.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	proxyServer := setupProxy(rt)
	defer proxyServer.Close()

	req, _ := http.NewRequest("GET", proxyServer.URL+"/api/v1/users/123", nil)
	req.Host = "strip.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Received-Path") != "/users/123" {
		t.Errorf("expected stripped path /users/123, got %s", resp.Header.Get("X-Received-Path"))
	}
}

func TestProxyPassesPathParams(t *testing.T) {
	backend := setupTestBackend(t)
	defer backend.Close()

	cfg := []router.InitialRoutes{
		{
			Domain:     "params.com",
			Ports:      []string{portFromURL(backend.URL)},
			PathPrefix: "/users/:id",
		},
	}
	rt, err := router.BuildFromConfig(cfg)
	if err != nil {
		t.Fatalf("failed to build router: %v", err)
	}

	proxyServer := setupProxy(rt)
	defer proxyServer.Close()

	req, _ := http.NewRequest("GET", proxyServer.URL+"/users/789", nil)
	req.Host = "params.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Param-Id") != "789" {
		t.Errorf("expected X-Param-Id: 789, got %s", resp.Header.Get("X-Param-Id"))
	}
}

func TestProxyHotReload(t *testing.T) {
	backend1 := setupTestBackend(t)
	backend2 := setupTestBackend(t)
	defer backend1.Close()
	defer backend2.Close()

	cfg1 := []router.InitialRoutes{
		{Domain: "hot.com", Ports: []string{portFromURL(backend1.URL)}, PathPrefix: "/v1"},
	}
	rt1, _ := router.BuildFromConfig(cfg1)

	p := proxy.New(rt1)
	proxyServer := httptest.NewServer(http.HandlerFunc(p.ServeHTTP))
	defer proxyServer.Close()

	req1, _ := http.NewRequest("GET", proxyServer.URL+"/v1", nil)
	req1.Host = "hot.com"
	resp1, err := http.DefaultClient.Do(req1)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	defer resp1.Body.Close()

	cfg2 := []router.InitialRoutes{
		{Domain: "hot.com", Ports: []string{portFromURL(backend2.URL)}, PathPrefix: "/v1"},
	}
	rt2, _ := router.BuildFromConfig(cfg2)
	p.UpdateRouter(rt2)

	time.Sleep(50 * time.Millisecond) // small delay to allow atomic update
	req2, _ := http.NewRequest("GET", proxyServer.URL+"/v1", nil)
	req2.Host = "hot.com"
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	defer resp2.Body.Close()

}
