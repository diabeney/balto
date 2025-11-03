package router

import (
	"strings"
)

type Route struct {
	// Multiple URLs per service to account for load balancing
	internalUrls []string
	pathPrefix   string
}

func (r Route) isEmpty() bool {
	return len(r.internalUrls) == 0 && r.pathPrefix == ""
}

type Host string

func (h Host) toLower() Host {
	return Host(strings.ToLower(string(h)))
}

type ServiceRoutes map[Host][]Route

type Router struct {
	routes ServiceRoutes
}

func stripTrailingSlash(path string) string {
	if path == "/" {
		return path
	}
	return strings.TrimSuffix(path, "/")
}

func routesEqual(a, b Route) bool {
	if a.pathPrefix != b.pathPrefix {
		return false
	}
	if len(a.internalUrls) != len(b.internalUrls) {
		return false
	}
	for i := range a.internalUrls {
		if a.internalUrls[i] != b.internalUrls[i] {
			return false
		}
	}
	return true
}

func deepCopy(routes []Route) []Route {
	copiedRoutes := make([]Route, len(routes))
	for i, route := range routes {
			urlsCopy := make([]string, len(route.internalUrls))
			copy(urlsCopy, route.internalUrls)
			copiedRoutes[i] = Route{
				internalUrls: urlsCopy,
				pathPrefix:   route.pathPrefix,
			}
		}
		return copiedRoutes
}


func (r Router) Add(host Host, route Route) Router {
	if host == "" || route.isEmpty() {
		return r
	}

	normalizedHost := host.toLower()

	routes, ok := r.routes[normalizedHost]

	exists := false
	for _, r := range routes {
		if routesEqual(r, route) {
			exists = true
			break
		}
	}

	if !ok || !exists {
		r.routes[normalizedHost] = append(r.routes[normalizedHost], Route{
			internalUrls: route.internalUrls,
			pathPrefix:   stripTrailingSlash(route.pathPrefix),
		})
		return r
	}
	return r
}

func (r Router) Lookup(host Host, path string) (Route, bool) {
	routes, ok := r.routes[host.toLower()]
	if !ok {
		return Route{}, false
	}

	var bestMatch Route
	var longestPrefix = 0
	var normalizedPath = stripTrailingSlash(path)
	for _, route := range routes {
		if normalizedPath == route.pathPrefix {
			return route, true
		}
		if strings.HasPrefix(normalizedPath, route.pathPrefix) {
			if len(route.pathPrefix) > longestPrefix {
				bestMatch = route
				longestPrefix = len(route.pathPrefix)
			}
		}
	}

	return bestMatch, longestPrefix > 0
}

func (r Router) ListRoutesByHost(host Host) ([]Route, bool) {
	routes, ok := r.routes[host.toLower()]
	if !ok {
		return nil, false
	}
	copiedRoutes := deepCopy(routes)
	return copiedRoutes, true
}

func (r Router) ListRoutes() ServiceRoutes {
	newRoutes := make(ServiceRoutes)
	for host, routes := range r.routes {
		copiedRoutes := deepCopy(routes)
		newRoutes[host] = copiedRoutes
	}
	return newRoutes
}

