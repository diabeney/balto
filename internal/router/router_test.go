package router

import (
	"reflect"
	"testing"
)

func newTestRouter() Router {
	return Router{
		routes: ServiceRoutes{
			"www.jedevent.com": []Route{
				{internalUrls: []string{"http://localhost:3000"}, pathPrefix: "/"},
				{internalUrls: []string{"http://localhost:3001"}, pathPrefix: "/api"},
				{internalUrls: []string{"http://localhost:3002"}, pathPrefix: "/api/v1"},
			},
		},
	}
}

func TestRouterAdd(t *testing.T) {
	t.Run("Add route to existing host", func(t *testing.T) {
		router := newTestRouter()
		route := Route{internalUrls: []string{"http://localhost:3003"}, pathPrefix: "/ussd"}

		expected := Router{
			routes: ServiceRoutes{
				"www.jedevent.com": []Route{
					{internalUrls: []string{"http://localhost:3000"}, pathPrefix: "/"},
					{internalUrls: []string{"http://localhost:3001"}, pathPrefix: "/api"},
					{internalUrls: []string{"http://localhost:3002"}, pathPrefix: "/api/v1"},
					{internalUrls: []string{"http://localhost:3003"}, pathPrefix: "/ussd"},
				},
			},
		}

		router = router.Add("www.jedevent.com", route)
		if !reflect.DeepEqual(router, expected) {
			t.Errorf("expected %+v, got %+v", expected, router)
		}
	})

	t.Run("Add route to new host", func(t *testing.T) {
		router := newTestRouter()
		newHost := "api.jedevent.com"
		newRoute := Route{internalUrls: []string{"http://localhost:4000"}, pathPrefix: "/v1"}

		router = router.Add(Host(newHost), newRoute)

		routes, exists := router.routes[Host(newHost)]
		if !exists {
			t.Fatalf("expected new host '%s' to exist", newHost)
		}
		if len(routes) != 1 || !reflect.DeepEqual(routes[0].internalUrls, []string{"http://localhost:4000"}) {
			t.Errorf("expected 1 route, got %+v", routes)
		}
	})

	t.Run("Ignore empty host or invalid route", func(t *testing.T) {
		router := newTestRouter()
		initial := len(router.routes["www.jedevent.com"])
		router = router.Add("", Route{internalUrls: []string{"http://localhost:9999"}, pathPrefix: "/"})
		router = router.Add("www.jedevent.com", Route{})

		if len(router.routes["www.jedevent.com"]) != initial {
			t.Errorf("expected invalid routes to be ignored")
		}
	})

	t.Run("Prevent duplicate route insertions", func(t *testing.T) {
		router := newTestRouter()
		dup := Route{internalUrls: []string{"http://localhost:3000"}, pathPrefix: "/"}
		router = router.Add("www.jedevent.com", dup)

		count := 0
		for _, r := range router.routes["www.jedevent.com"] {
			if reflect.DeepEqual(r.internalUrls, dup.internalUrls) && r.pathPrefix == dup.pathPrefix {
				count++
			}
		}
		if count > 1 {
			t.Errorf("duplicate route detected for same host/path combination")
		}
	})
}

func TestRouterLookup(t *testing.T) {
	router := newTestRouter()

	t.Run("Most specific match is returned", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"/", "http://localhost:3000"},
			{"/api", "http://localhost:3001"},
			{"/api/v1", "http://localhost:3002"},
			{"/api/v1/users", "http://localhost:3002"},
			{"/unknown", "http://localhost:3000"},
		}

		for _, tt := range tests {
			r, ok := router.Lookup("www.jedevent.com", tt.path)
			if !ok {
				t.Errorf("expected match for %s", tt.path)
				continue
			}
			if len(r.internalUrls) == 0 || r.internalUrls[0] != tt.expected {
				t.Errorf("for path %s expected %s but got %+v", tt.path, tt.expected, r.internalUrls)
			}
		}
	})

	t.Run("Unknown host returns no match", func(t *testing.T) {
		_, ok := router.Lookup("unknown.com", "/")
		if ok {
			t.Errorf("expected no match for unknown host")
		}
	})

	t.Run("Empty routes map returns no match", func(t *testing.T) {
		emptyRouter := Router{routes: make(ServiceRoutes)}
		_, ok := emptyRouter.Lookup("www.jedevent.com", "/")
		if ok {
			t.Errorf("expected no match for empty route set")
		}
	})

	t.Run("Trailing slashes normalized", func(t *testing.T) {
		tests := []struct {
			path     string
			expected string
		}{
			{"/api/", "http://localhost:3001"},
			{"/api/v1/", "http://localhost:3002"},
			{"/api/v1/users/", "http://localhost:3002"},
			{"/", "http://localhost:3000"},
		}

		for _, tt := range tests {
			r, ok := router.Lookup("www.jedevent.com", tt.path)
			if !ok {
				t.Errorf("expected match for %s", tt.path)
				continue
			}
			if len(r.internalUrls) == 0 || r.internalUrls[0] != tt.expected {
				t.Errorf("for path %s expected %s but got %+v", tt.path, tt.expected, r.internalUrls)
			}
		}
	})

	t.Run("Case sensitivity behavior", func(t *testing.T) {
		r, ok := router.Lookup("WWW.JEDEVENT.COM", "/api")
		if !ok || r.internalUrls[0] != "http://localhost:3001" {
			t.Errorf("expected host lookup to be case-insensitive")
		}

		r, ok = router.Lookup("www.jedevent.com", "/API")
		if ok && r.internalUrls[0] == "http://localhost:3001" {
			t.Errorf("expected path matching to be case-sensitive")
		}
	})

	t.Run("Root route acts as fallback", func(t *testing.T) {
		r, ok := router.Lookup("www.jedevent.com", "/nonexistent/path")
		if !ok {
			t.Errorf("expected fallback to root route")
		}
		if r.internalUrls[0] != "http://localhost:3000" {
			t.Errorf("expected fallback to root, got %+v", r.internalUrls)
		}
	})
}

func TestListRoutes(t *testing.T) {
	router := Router{
		routes: ServiceRoutes{
			"api.example.com": {
				{internalUrls: []string{"http://localhost:3001"}, pathPrefix: "/users"},
				{internalUrls: []string{"http://localhost:3002"}, pathPrefix: "/orders"},
			},
			"app.example.com": {
				{internalUrls: []string{"http://localhost:3000"}, pathPrefix: "/"},
			},
		},
	}

	expected := ServiceRoutes{
		"api.example.com": {
			{internalUrls: []string{"http://localhost:3001"}, pathPrefix: "/users"},
			{internalUrls: []string{"http://localhost:3002"}, pathPrefix: "/orders"},
		},
		"app.example.com": {
			{internalUrls: []string{"http://localhost:3000"}, pathPrefix: "/"},
		},
	}

	got := router.ListRoutes()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("ListRoutes() = %+v, want %+v", got, expected)
	}
}

func TestListRoutesImmutability(t *testing.T) {
	r := Router{
		routes: ServiceRoutes{
			"app.example.com": {
				{internalUrls: []string{"http://localhost:8080"}, pathPrefix: "/"},
			},
		},
	}

	got := r.ListRoutes()
	got["app.example.com"][0].internalUrls[0] = "http://localhost:9090"

	if r.routes["app.example.com"][0].internalUrls[0] != "http://localhost:8080" {
		t.Errorf("ListRoutes() returned a reference that modified Router.routes")
	}
}

func TestListRoutesByHost(t *testing.T) {
	router := Router{
		routes: ServiceRoutes{
			"api.example.com": {
				{internalUrls: []string{"http://localhost:3001"}, pathPrefix: "/api"},
				{internalUrls: []string{"http://localhost:3002"}, pathPrefix: "/v2"},
			},
		},
	}

	t.Run("existing host returns routes", func(t *testing.T) {
		routes, ok := router.ListRoutesByHost("api.example.com")
		if !ok {
			t.Fatal("expected ok=true for existing host")
		}
		if len(routes) != 2 {
			t.Errorf("expected 2 routes, got %d", len(routes))
		}
	})

	t.Run("nonexistent host returns empty slice and false", func(t *testing.T) {
		routes, ok := router.ListRoutesByHost("missing.com")
		if ok {
			t.Error("expected ok=false for missing host")
		}
		if len(routes) != 0 {
			t.Error("expected empty slice for missing host")
		}
	})

	t.Run("case-insensitive host lookup", func(t *testing.T) {
		routes, ok := router.ListRoutesByHost("API.EXAMPLE.COM")
		if !ok || len(routes) != 2 {
			t.Errorf("expected 2 routes for case-insensitive match, got %d", len(routes))
		}
	})
}