package router

import (
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
)

type InitialRoutes struct {
	Domain     string   `json:"domain" yaml:"domain"`
	PathPrefix string   `json:"path_prefix" yaml:"path_prefix"`
	Ports      []string `json:"ports" yaml:"ports"`
}

type Host string

func (h Host) lower() Host { return Host(strings.ToLower(string(h))) }

type Route struct {
	Prefix string
	URLs   []*url.URL // This is the pre-parsed backend URLs for the route.
}

func (r Route) FirstURL() (*url.URL, error) {
	if len(r.URLs) == 0 {
		return nil, fmt.Errorf("no backend")
	}
	return r.URLs[0], nil
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
	hosts map[Host]*node
}

func NewRouter() *Router {
	return &Router{hosts: make(map[Host]*node)}
}

func (r *Router) Add(host Host, path string, services []*url.URL) *Router {
	if host == "" || len(services) == 0 {
		return r
	}
	h := host.lower()
	segments := pathToSegments(normalizePrefix(path))
	route := &Route{Prefix: path, URLs: services}

	newMap := make(map[Host]*node, len(r.hosts)+1)
	for k, v := range r.hosts {
		newMap[k] = v
	}

	root := r.hosts[h]
	if root == nil {
		root = &node{children: make(map[string]*node)}
	}
	newRoot := root.insert(segments, route)
	newMap[h] = newRoot

	return &Router{hosts: newMap}
}

func (r *Router) Lookup(host Host, path string) (Route, Params, bool) {
	root := r.hosts[host.lower()]
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

var current atomic.Pointer[Router]

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
