package router

import (
	"net/url"
	"reflect"
	"testing"
)

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

func newTestRouter() *Router {
	r := NewRouter()
	r = r.Add(Host("www.example.com"), "/", []*url.URL{mustParseURL("http://localhost:3000")})
	r = r.Add(Host("www.example.com"), "/api", []*url.URL{mustParseURL("http://localhost:3001")})
	r = r.Add(Host("www.example.com"), "/api/v1", []*url.URL{mustParseURL("http://localhost:3002")})
	r = r.Add(Host("www.example.com"), "/users/:id", []*url.URL{mustParseURL("http://localhost:3003")})
	r = r.Add(Host("www.example.com"), "/static/*", []*url.URL{mustParseURL("http://localhost:3004")})
	r = r.Add(Host("api.example.com"), "/v1", []*url.URL{mustParseURL("http://localhost:4000")})
	return r
}

func TestRouterAdd(t *testing.T) {
	t.Run("Add route to existing host", func(t *testing.T) {
		r := newTestRouter()
		initial := len(r.hosts[Host("www.example.com").lower()].children)

		r = r.Add(Host("www.example.com"), "/admin", []*url.URL{mustParseURL("http://localhost:5000")})

		route, _, ok := r.Lookup(Host("www.example.com"), "/admin")
		if !ok {
			t.Fatal("expected /admin to be added")
		}
		backend, err := route.NextBackend()
		if err != nil {
			t.Fatalf("failed to get backend: %v", err)
		}
		if backend.URL.String() != "http://localhost:5000" {
			t.Errorf("got %s, want http://localhost:5000", backend.URL.String())
		}
		if len(r.hosts[Host("www.example.com").lower()].children) <= initial {
			t.Error("child count did not increase")
		}
	})

	t.Run("Add route to new host", func(t *testing.T) {
		r := newTestRouter()
		r = r.Add(Host("newhost.com"), "/v2", []*url.URL{mustParseURL("http://localhost:6000")})

		route, _, ok := r.Lookup(Host("newhost.com"), "/v2")
		if !ok {
			t.Fatal("expected /v2 on newhost.com")
		}
		backend, err := route.NextBackend()
		if err != nil {
			t.Fatalf("failed to get backend: %v", err)
		}
		if backend.URL.Port() != "6000" {
			t.Errorf("got port %s, want 6000", backend.URL.Port())
		}
	})

	t.Run("Ignore empty host or empty services", func(t *testing.T) {
		r := newTestRouter()
		origPtr := reflect.ValueOf(r.hosts).Pointer()

		r = r.Add(Host(""), "/valid", []*url.URL{mustParseURL("http://localhost:9999")})
		r = r.Add(Host("www.example.com"), "/valid", []*url.URL{})

		if reflect.ValueOf(r.hosts).Pointer() != origPtr {
			t.Error("invalid Add mutated the router")
		}
	})

	// Note: No ListRoutes method, instead we verify behavior via Lookup
	t.Run("Duplicate exact route does not break lookup", func(t *testing.T) {
		r := newTestRouter()
		r = r.Add(Host("www.example.com"), "/api", []*url.URL{mustParseURL("http://localhost:9999")})

		route, _, ok := r.Lookup(Host("www.example.com"), "/api")
		if !ok {
			t.Fatal("expected /api to still match")
		}
		backend, err := route.NextBackend()
		if err != nil {
			t.Fatalf("failed to get backend: %v", err)
		}
		if backend.URL.Port() != "3001" && backend.URL.Port() != "9999" {
			t.Errorf("unexpected backend port: %s", backend.URL.Port())
		}
	})
}

func TestRouterLookup(t *testing.T) {
	r := newTestRouter()

	t.Run("Most specific match wins", func(t *testing.T) {
		cases := []struct {
			path        string
			want        string
			shouldMatch bool
		}{
			{"/", "http://localhost:3000", true},
			{"/api", "http://localhost:3001", true},
			{"/api/v1", "http://localhost:3002", true},
			{"/users/123", "http://localhost:3003", true},
			{"/static/js/app.js", "http://localhost:3004", true},
			{"/unknown", "", false},
		}
		for _, c := range cases {
			t.Run(c.path, func(t *testing.T) {
				route, _, ok := r.Lookup(Host("www.example.com"), c.path)
				if c.shouldMatch {
					if !ok {
						t.Fatalf("expected match for %q", c.path)
					}
					backend, err := route.NextBackend()
					if err != nil {
						t.Fatalf("failed to get backend: %v", err)
					}
					if backend.URL.String() != c.want {
						t.Errorf("got %s, want %s", backend.URL.String(), c.want)
					}
				} else {
					if ok {
						backend, _ := route.NextBackend()
						if backend != nil {
							t.Errorf("expected no match for %q, but got %s", c.path, backend.URL.String())
						}
					}
				}
			})
		}
	})

	t.Run("Dynamic param extraction", func(t *testing.T) {
		route, params, ok := r.Lookup(Host("www.example.com"), "/users/456")
		if !ok {
			t.Fatal("expected match")
		}
		backend, err := route.NextBackend()
		if err != nil {
			t.Fatalf("failed to get backend: %v", err)
		}
		if backend.URL.Port() != "3003" {
			t.Errorf("want backend 3003, got %s", backend.URL.Port())
		}
		if params["id"] != "456" {
			t.Errorf("want id=456, got %v", params["id"])
		}
	})

	t.Run("Wildcard matches any sub-path", func(t *testing.T) {
		route, _, ok := r.Lookup(Host("www.example.com"), "/static/images/profiles/valid-uuid")
		if !ok {
			t.Fatal("wildcard match failed")
		}
		backend, err := route.NextBackend()
		if err != nil {
			t.Fatalf("failed to get backend: %v", err)
		}
		if backend.URL.Port() != "3004" {
			t.Errorf("want backend 3004, got %s", backend.URL.Port())
		}
	})

	t.Run("Unknown host returns no match", func(t *testing.T) {
		_, _, ok := r.Lookup(Host("unknown.com"), "/")
		if ok {
			t.Error("unknown host should not match")
		}
	})

	t.Run("Trailing slashes normalized", func(t *testing.T) {
		cases := []struct {
			path string
			want string
		}{
			{"/api/", "http://localhost:3001"},
			{"//api//v1//", "http://localhost:3002"},
			{"//users//789//", "http://localhost:3003"},
		}
		for _, c := range cases {
			t.Run(c.path, func(t *testing.T) {
				route, _, ok := r.Lookup(Host("www.example.com"), c.path)
				if !ok {
					t.Fatal("no match")
				}
				backend, err := route.NextBackend()
				if err != nil {
					t.Fatalf("failed to get backend: %v", err)
				}
				if backend.URL.String() != c.want {
					t.Errorf("got %s, want %s", backend.URL.String(), c.want)
				}
			})
		}
	})

	t.Run("Host case-insensitive, path case-sensitive", func(t *testing.T) {
		_, _, ok := r.Lookup(Host("WWW.EXAMPLE.COM"), "/API")
		if ok {
			t.Error("/API (upper) should NOT match")
		}

		route, _, ok := r.Lookup(Host("WWW.EXAMPLE.COM"), "/api")
		if !ok {
			t.Error("case-insensitive host failed")
		} else {
			backend, err := route.NextBackend()
			if err != nil {
				t.Fatalf("failed to get backend: %v", err)
			}
			if backend.URL.Port() != "3001" {
				t.Errorf("case-insensitive host failed, got port %s", backend.URL.Port())
			}
		}
	})

	t.Run("Unmatched path returns no match", func(t *testing.T) {
		_, _, ok := r.Lookup(Host("www.example.com"), "/nonexistent")
		if ok {
			t.Error("unmatched path should not match without explicit wildcard")
		}
	})
}

func TestRouterImmutability(t *testing.T) {
	r1 := newTestRouter()
	r2 := r1.Add(Host("www.jedevent.com"), "/api", []*url.URL{mustParseURL("http://localhost:7000")})

	if reflect.DeepEqual(r1, r2) {
		t.Error("Add mutated original")
	}
	_, _, ok := r1.Lookup(Host("www.jedevent.com"), "/api")
	if ok {
		t.Error("original router modified")
	}
}

func TestRouterHotReload(t *testing.T) {
	r1 := newTestRouter()
	SetCurrent(r1)

	r2 := r1.Add(Host("www.example.com"), "/live", []*url.URL{
		mustParseURL("http://localhost:8000"),
	})
	SetCurrent(r2)

	route, _, ok := Current().Lookup(Host("www.example.com"), "/live")
	if !ok {
		t.Error("hot reload failed, route not found")
	} else {
		backend, err := route.NextBackend()
		if err != nil {
			t.Fatalf("failed to get backend: %v", err)
		}
		if backend.URL.Port() != "8000" {
			t.Errorf("hot reload failed, got port %s", backend.URL.Port())
		}
	}

	if _, _, ok := r1.Lookup(Host("www.example.com"), "/live"); ok {
		t.Error("original router mutated after Add()")
	}
}

func TestRouterEdgeCases(t *testing.T) {
	t.Run("Empty path matches root", func(t *testing.T) {
		r := NewRouter()
		r = r.Add(Host("example.com"), "/", []*url.URL{mustParseURL("http://localhost:9999")})
		route, _, ok := r.Lookup(Host("example.com"), "")
		if !ok {
			t.Error("empty path failed")
		} else {
			backend, err := route.NextBackend()
			if err != nil {
				t.Fatalf("failed to get backend: %v", err)
			}
			if backend.URL.Port() != "9999" {
				t.Errorf("empty path failed, got port %s", backend.URL.Port())
			}
		}
	})
	t.Run("No panic on nil router", func(t *testing.T) {
		defer func() {
			if recover() != nil {
				t.Error("Current() nil caused panic")
			}
		}()
		SetCurrent(nil)
		_ = Current()
	})
}
