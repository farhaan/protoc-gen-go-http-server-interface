// Package fixture provides a static copy of generated HTTP server interface code
// for a PingService, used by runtime behavioral tests.
//
// PingService methods:
//   - Ping:       GET /ping          (no path params)
//   - GetItem:    GET /items/{item_id}  (one path param)
//   - UpdateItem: PUT /items/{item_id}, PATCH /items/{item_id}  (two bindings)
//
// This file mirrors what protoc-gen-go-http-server-interface would generate.
// Update by running: make regenerate (or copy from a freshly generated file).
package fixture

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

// Middleware represents a middleware function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Routes defines the minimal interface for route registration.
// This interface is intentionally minimal to maximize compatibility with
// standard library and third-party routers.
//
// Stability: this interface is stable. New methods will not be added without a
// major version bump.
type Routes interface {
	HandleFunc(method, pattern string, handler http.HandlerFunc)
}

// Router extends Routes with grouping and middleware support.
type Router interface {
	Routes
	Group(prefix string, middlewares ...Middleware) Router
	Use(middlewares ...Middleware) Router
}

// compiledRoute is a cached middleware chain for a single route.
type compiledRoute struct {
	ver   uint64
	chain http.Handler
}

// RouteGroup implements Router using http.ServeMux.
type RouteGroup struct {
	mux         *http.ServeMux
	prefix      string
	parent      *RouteGroup
	mu          sync.RWMutex
	middlewares []Middleware
	version     atomic.Uint64
	routes      []string
}

// NewRouter creates a new router with an optional mux.
// If mux is nil, a new http.ServeMux will be created.
func NewRouter(mux *http.ServeMux) *RouteGroup {
	if mux == nil {
		mux = http.NewServeMux()
	}
	return &RouteGroup{
		mux:    mux,
		routes: []string{},
	}
}

// Mux returns the underlying http.ServeMux.
func (g *RouteGroup) Mux() *http.ServeMux {
	return g.mux
}

// joinPath safely joins URL path segments.
func joinPath(base, path string) string {
	if path == "" || path == "/" {
		return base
	}
	if base == "" || base == "/" {
		return path
	}
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(path, "/")
}

// effectiveVersion returns the sum of this group's version and all ancestor versions.
func (g *RouteGroup) effectiveVersion() uint64 {
	ver := g.version.Load()
	if g.parent != nil {
		ver += g.parent.effectiveVersion()
	}
	return ver
}

// collectMiddlewareChain returns all middlewares from the root down to g.
func collectMiddlewareChain(g *RouteGroup) []Middleware {
	var chain []Middleware
	if g.parent != nil {
		chain = collectMiddlewareChain(g.parent)
	}
	g.mu.RLock()
	chain = append(chain, g.middlewares...)
	g.mu.RUnlock()
	return chain
}

// Group creates a child RouteGroup with the given prefix.
// The child inherits the parent's middleware chain at dispatch time.
func (g *RouteGroup) Group(prefix string, middlewares ...Middleware) Router {
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	child := &RouteGroup{
		mux:    g.mux,
		prefix: joinPath(g.prefix, prefix),
		parent: g,
		routes: []string{},
	}
	if len(middlewares) > 0 {
		child.middlewares = appendMiddlewares(nil, middlewares)
	}
	return child
}

// Use appends middlewares to all routes in this group.
// Middlewares added after HandleFunc are applied at request dispatch time.
func (g *RouteGroup) Use(middlewares ...Middleware) Router {
	g.mu.Lock()
	g.middlewares = appendMiddlewares(g.middlewares, middlewares)
	g.version.Add(1) // bumped inside the lock so the version increment is visible
	g.mu.Unlock()    // atomically with the middleware append
	return g
}

// HandleFunc registers a handler function for the given method and pattern.
// All ancestor and group middlewares are applied at request dispatch time,
// compiled once per effective version and cached atomically.
func (g *RouteGroup) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	fullPattern := joinPath(g.prefix, pattern)
	routeKey := method + " " + fullPattern
	group := g
	var cache atomic.Pointer[compiledRoute]
	g.mux.Handle(routeKey, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ver := group.effectiveVersion()
		if c := cache.Load(); c != nil && c.ver == ver {
			c.chain.ServeHTTP(w, r)
			return
		}
		chain := applyMiddlewares(handler, collectMiddlewareChain(group))
		cache.Store(&compiledRoute{ver: ver, chain: chain})
		chain.ServeHTTP(w, r)
	}))
	g.mu.Lock()
	g.routes = append(g.routes, routeKey)
	g.mu.Unlock()
}

// GetRoutes returns all registered routes for this group.
func (g *RouteGroup) GetRoutes() []string {
	return g.routes
}

// ServeHTTP implements the http.Handler interface.
func (g *RouteGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// appendMiddlewares combines parent and new middlewares, filtering out nils.
func appendMiddlewares(parent, additional []Middleware) []Middleware {
	result := make([]Middleware, 0, len(parent)+len(additional))
	for _, mw := range parent {
		if mw != nil {
			result = append(result, mw)
		}
	}
	for _, mw := range additional {
		if mw != nil {
			result = append(result, mw)
		}
	}
	return result
}

// applyMiddlewares wraps handler with the given middlewares (outermost first).
func applyMiddlewares(handler http.Handler, middlewares []Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		if middlewares[i] != nil {
			handler = middlewares[i](handler)
		}
	}
	return handler
}

// ResponseWriterWrapper wraps http.ResponseWriter to capture the HTTP status code
// and safely forward http.Flusher calls to the underlying writer.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	// StatusCode is the HTTP status code written by the handler.
	// Defaults to 200 if WriteHeader was never called.
	StatusCode int
}

// NewResponseWriterWrapper returns a ResponseWriterWrapper around w.
func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{ResponseWriter: w, StatusCode: http.StatusOK}
}

// WriteHeader captures the status code and delegates to the underlying writer.
func (rw *ResponseWriterWrapper) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying http.Flusher if supported.
func (rw *ResponseWriterWrapper) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// ErrNilRouter is returned when a nil router is passed to a register function.
var ErrNilRouter = errors.New("protogen: router is nil")

// ErrNilHandler is returned when a nil handler is passed to a register function.
var ErrNilHandler = errors.New("protogen: handler is nil")

// Compile-time assertions: RouteGroup must satisfy both interfaces.
var (
	_ Routes = (*RouteGroup)(nil)
	_ Router = (*RouteGroup)(nil)
)

// PingServiceHandler is the interface for PingService HTTP handlers.
// Embed UnimplementedPingServiceHandler to forward-compatibly implement this interface.
type PingServiceHandler interface {
	// HandlePing handles GET /ping.
	HandlePing(w http.ResponseWriter, r *http.Request)
	// HandleGetItem handles GET /items/{item_id}.
	// Path params (stdlib): r.PathValue("item_id")
	// In tests: use req.SetPathValue("param_name", value) for each path param (Go 1.22+).
	// Router-specific: chi.URLParam(r, "param") | mux.Vars(r)["param"]
	HandleGetItem(w http.ResponseWriter, r *http.Request)
	// HandleUpdateItem handles PUT /items/{item_id}, PATCH /items/{item_id}.
	// Path params (stdlib): r.PathValue("item_id")
	// In tests: use req.SetPathValue("param_name", value) for each path param (Go 1.22+).
	// Router-specific: chi.URLParam(r, "param") | mux.Vars(r)["param"]
	HandleUpdateItem(w http.ResponseWriter, r *http.Request)
}

// UnimplementedPingServiceHandler provides default 501 Not Implemented stubs.
// Embed this struct in your handler to forward-compatibly implement PingServiceHandler.
type UnimplementedPingServiceHandler struct{}

// HandlePing returns 501 Not Implemented for Ping.
func (UnimplementedPingServiceHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented: Ping", http.StatusNotImplemented)
}

// HandleGetItem returns 501 Not Implemented for GetItem.
func (UnimplementedPingServiceHandler) HandleGetItem(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented: GetItem", http.StatusNotImplemented)
}

// HandleUpdateItem returns 501 Not Implemented for UpdateItem.
func (UnimplementedPingServiceHandler) HandleUpdateItem(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented: UpdateItem", http.StatusNotImplemented)
}

// RegisterPingServiceRoutes registers HTTP routes for PingService.
// Returns an error if router or handler is nil.
func RegisterPingServiceRoutes(r Routes, handler PingServiceHandler) error {
	if r == nil {
		return ErrNilRouter
	}
	if handler == nil {
		return ErrNilHandler
	}
	r.HandleFunc(http.MethodGet, "/ping", handler.HandlePing)
	r.HandleFunc(http.MethodGet, "/items/{item_id}", handler.HandleGetItem)
	r.HandleFunc(http.MethodPut, "/items/{item_id}", handler.HandleUpdateItem)
	r.HandleFunc(http.MethodPatch, "/items/{item_id}", handler.HandleUpdateItem)
	return nil
}

// MustRegisterPingServiceRoutes registers HTTP routes for PingService.
// Panics if router or handler is nil.
func MustRegisterPingServiceRoutes(r Routes, handler PingServiceHandler) {
	if err := RegisterPingServiceRoutes(r, handler); err != nil {
		panic(err)
	}
}

// RegisterPingRoute registers the Ping handler (1 binding).
func RegisterPingRoute(r Routes, handler PingServiceHandler, middlewares ...Middleware) error {
	if r == nil {
		return ErrNilRouter
	}
	if handler == nil {
		return ErrNilHandler
	}
	h := applyMiddlewares(http.HandlerFunc(handler.HandlePing), middlewares)
	r.HandleFunc(http.MethodGet, "/ping", h.ServeHTTP)
	return nil
}

// RegisterGetItemRoute registers the GetItem handler (1 binding).
func RegisterGetItemRoute(r Routes, handler PingServiceHandler, middlewares ...Middleware) error {
	if r == nil {
		return ErrNilRouter
	}
	if handler == nil {
		return ErrNilHandler
	}
	h := applyMiddlewares(http.HandlerFunc(handler.HandleGetItem), middlewares)
	r.HandleFunc(http.MethodGet, "/items/{item_id}", h.ServeHTTP)
	return nil
}

// RegisterUpdateItemRoute registers the UpdateItem handler (2 bindings: PUT + PATCH).
func RegisterUpdateItemRoute(r Routes, handler PingServiceHandler, middlewares ...Middleware) error {
	if r == nil {
		return ErrNilRouter
	}
	if handler == nil {
		return ErrNilHandler
	}
	h := applyMiddlewares(http.HandlerFunc(handler.HandleUpdateItem), middlewares)
	r.HandleFunc(http.MethodPut, "/items/{item_id}", h.ServeHTTP)
	r.HandleFunc(http.MethodPatch, "/items/{item_id}", h.ServeHTTP)
	return nil
}

