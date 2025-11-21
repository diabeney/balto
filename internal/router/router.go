package router

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/diabeney/balto/internal/core"
	"github.com/diabeney/balto/internal/core/backendpool"
	"github.com/diabeney/balto/internal/core/balancer"
	"github.com/diabeney/balto/internal/health"
)

type InitialRoutes struct {
	Domain     string   `json:"domain" yaml:"domain"`
	PathPrefix string   `json:"path_prefix" yaml:"path_prefix"`
	Ports      []string `json:"ports" yaml:"ports"`
}

type Host string

func (h Host) lower() Host { return Host(strings.ToLower(string(h))) }

func (h Host) normalize() Host {
	host := string(h.lower())
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return Host(host)
}

type Route struct {
	Prefix string
	Pool   *backendpool.Pool
}

func (r Route) NextBackend() (*core.Backend, error) {
	if r.Pool == nil {
		return nil, fmt.Errorf("no backend pool")
	}
	backend := r.Pool.Next()
	if backend == nil {
		return nil, fmt.Errorf("no healthy backend available")
	}
	return backend, nil
}

type Params map[string]string

type node struct {
	segment    string
	route      *Route
	paramName  string
	isWildcard bool
	children   map[string]*node
}

func newNode(seg string) *node {
	n := &node{
		segment:  seg,
		children: make(map[string]*node),
	}

	if len(seg) > 1 && seg[0] == ':' {
		n.paramName = seg[1:]
	} else if seg == "*" {
		n.isWildcard = true
	}
	return n
}

// insert adds a new route to the node tree while preserving immutability.
// It uses a copy-on-write approach, which performs a shallow copy of the current
// node and its children map. Unchanged subtrees are reused, while only modified
// branches are cloned. This allows multiple router versions to coexist safely,
// sharing unchanged nodes without affecting previous instances.

func (n *node) insert(segments []string, route *Route) *node {
	if len(segments) == 0 {
		copied := *n
		copied.route = route
		return &copied
	}

	seg := segments[0]
	copied := *n
	copied.children = copyMap(n.children) // Shallow copy: new map, shared node pointers

	child, exists := copied.children[seg]
	if !exists {
		child = newNode(seg)
		copied.children[seg] = child
	}

	if len(segments) == 1 {
		newChild := *child
		newChild.route = route
		copied.children[seg] = &newChild
	} else {
		copied.children[seg] = child.insert(segments[1:], route)
	}
	return &copied
}

func (n *node) lookup(segments []string, params Params) (*Route, bool) {
	if len(segments) == 0 {
		if n.route != nil {
			return n.route, true
		}
		if child, ok := n.children["*"]; ok && child.route != nil {
			return child.route, true
		}
		return nil, false
	}

	seg := segments[0]

	// Try to match the segment exactly.
	if child, ok := n.children[seg]; ok {
		if r, found := child.lookup(segments[1:], params); found {
			return r, true
		}
	}

	// Try to match the segment as a parameter.
	for paramSeg, child := range n.children {
		if len(paramSeg) > 1 && paramSeg[0] == ':' {
			params[child.paramName] = seg
			if r, found := child.lookup(segments[1:], params); found {
				return r, true
			}
			delete(params, child.paramName)
		}
	}

	// Try to match the segment as a wildcard.
	if child, ok := n.children["*"]; ok && child.isWildcard && child.route != nil {
		return child.route, true
	}

	/*
		!INFO:
		When no matching child segment is found, we intentionally avoid returning the
		current node’s route even if it exists. Allowing that fallback could cause
		unexpected behavior. For example, if a route is registered for "/api" and a
		request comes in for "/api/v1", we shouldn’t return the "/api" route unless
		a wildcard (e.g. "/*" or "/api/*") was explicitly configured to match it.
		This ensures routes are matched strictly unless a catch-all is defined.
	*/

	return nil, false
}

func copyMap(m map[string]*node) map[string]*node {
	c := make(map[string]*node, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

type Router struct {
	hosts          map[Host]*node
	healthcheckers map[string]*health.Healthchecker
}

func NewRouter() *Router {
	return &Router{
		hosts:          make(map[Host]*node),
		healthcheckers: make(map[string]*health.Healthchecker),
	}
}

func (r *Router) Add(host Host, path string, services []*url.URL) *Router {
	if host == "" || len(services) == 0 {
		return r
	}

	h := host.normalize()
	normPath := normalizePrefix(path)
	segments := pathToSegments(normPath)

	bal := balancer.NewRoundRobin()
	poolCfg := &backendpool.PoolConfig{
		HealthThreshold:            10,
		ProbeHealthThreshold:       10,
		ProbeRecoveryThreshold:     5,
		ProbePath:                  "/api/health",
		ProbeInterval:              1000,
		Timeout:                    1000,
		CircuitFailureThreshold:    10,
		CircuitSuccessThreshold:    10,
		CircuitTimeout:             10,
		CircuitMaxHalfOpenRequests: 5,
		Retry:                      10,
	}

	pool := backendpool.New(poolCfg, bal)

	for _, u := range services {
		id := fmt.Sprintf("%s-%s", h, u.String())
		pool.Add(id, u, 1)
	}

	hc := health.New(pool)

	newHosts := make(map[Host]*node, len(r.hosts)+1)
	for k, v := range r.hosts {
		newHosts[k] = v
	}
	newHealthcheckers := make(map[string]*health.Healthchecker, len(r.healthcheckers)+1)
	for k, v := range r.healthcheckers {
		newHealthcheckers[k] = v
	}

	routeKey := fmt.Sprintf("%s%s", h, normPath)
	newHealthcheckers[routeKey] = hc

	route := &Route{Prefix: path, Pool: pool}
	root := newHosts[h]
	if root == nil {
		root = &node{children: make(map[string]*node)}
	}
	newRoot := root.insert(segments, route)
	newHosts[h] = newRoot

	return &Router{
		hosts:          newHosts,
		healthcheckers: newHealthcheckers,
	}
}

func (r *Router) Lookup(host Host, path string) (Route, Params, bool) {
	root := r.hosts[host.normalize()]
	if root == nil {
		return Route{}, nil, false
	}

	segments := pathToSegments(normalizePrefix(path))
	params := make(Params)
	route, found := root.lookup(segments, params)
	if !found {
		return Route{}, nil, false
	}
	return *route, params, true
}

// Start initiates all healthcheckers associated with this router.
//
// It is called ONCE when the application starts or ONCE on a newly loaded router
// during a hot-swap.
//
// Individual changes to a pool's backend list (add/remove backends)
// are handled automatically by the healthchecker's internal reconciliation loop.
func (r *Router) Start() {
	for _, hc := range r.healthcheckers {
		// It is safe to call Start() multiple times on the healthchecker
		// because Healthchecker struct has a 'started' bool guard.
		hc.Start()
	}
}

// Stop stops all healthcheckers associated with this router.
// This should be called when the router is being replaced or the application is shutting down.
// After Stop is called, the router should not be used for routing requests.
func (r *Router) Stop() error {
	var errs []error
	for _, hc := range r.healthcheckers {
		if hc != nil {
			if err := hc.Stop(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to stop %d healthchecker(s): %v", len(errs), errs)
	}
	return nil
}

var current atomic.Pointer[Router]

// TODO: When we integrate hot reload fully, we need to make sure we stop the stop the healthchecker for the previour router
// TODO: before switch to prevent goroutine leaks
func SetCurrent(r *Router) { current.Store(r) }
func Current() *Router     { return current.Load() }

func normalizePrefix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || p == "/" {
		return "/"
	}
	p = strings.TrimSuffix(p, "/")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func pathToSegments(path string) []string {
	if path == "/" {
		return []string{}
	}
	parts := strings.Split(path, "/")
	var segs []string
	for _, p := range parts {
		if p != "" {
			segs = append(segs, p)
		}
	}
	return segs
}

func parseServices(ports []string, scheme string) ([]*url.URL, error) {
	out := make([]*url.URL, 0, len(ports))
	for _, p := range ports {
		u, err := url.Parse(scheme + "://localhost:" + p)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, nil
}

func BuildFromConfig(cfg []InitialRoutes) (*Router, error) {
	r := NewRouter()
	for _, c := range cfg {
		services, err := parseServices(c.Ports, "http")
		if err != nil {
			return nil, err
		}
		r = r.Add(Host(c.Domain), c.PathPrefix, services)
	}
	return r, nil
}
