package behavior_test

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"tests/runtime/fixture"
)

// pingHandler is a minimal PingServiceHandler for testing.
type pingHandler struct {
	fixture.UnimplementedPingServiceHandler
	pingCalled       bool
	getItemCalled    bool
	updateItemCalled bool
}

func (h *pingHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	h.pingCalled = true
	w.WriteHeader(http.StatusOK)
}

func (h *pingHandler) HandleGetItem(w http.ResponseWriter, r *http.Request) {
	h.getItemCalled = true
	w.WriteHeader(http.StatusOK)
}

func (h *pingHandler) HandleUpdateItem(w http.ResponseWriter, r *http.Request) {
	h.updateItemCalled = true
	w.WriteHeader(http.StatusOK)
}

// T1: Group prefix — routes registered inside a group are served under the group's prefix.
func TestRouteGroup_PrefixRouting(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	g := r.Group("/api")
	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(g, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rw.Code)
	}
	if !h.pingCalled {
		t.Error("HandlePing was not called")
	}
}

// T2: Group isolation — routes from different groups do not collide.
func TestRouteGroup_Isolation(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	r := fixture.NewRouter(mux)

	v1 := r.Group("/v1")
	v2 := r.Group("/v2")

	called := ""
	v1h := &stubRoutes{fn: func(w http.ResponseWriter, r *http.Request) { called = "v1"; w.WriteHeader(200) }}
	v2h := &stubRoutes{fn: func(w http.ResponseWriter, r *http.Request) { called = "v2"; w.WriteHeader(200) }}

	v1.HandleFunc(http.MethodGet, "/ping", v1h.fn)
	v2.HandleFunc(http.MethodGet, "/ping", v2h.fn)

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/v1/ping", nil))
	if called != "v1" {
		t.Errorf("expected v1, got %q", called)
	}

	called = ""
	rw = httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/v2/ping", nil))
	if called != "v2" {
		t.Errorf("expected v2, got %q", called)
	}
}

// T3: Parent inheritance — nested Group inherits parent prefix.
func TestRouteGroup_NestedPrefix(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	inner := r.Group("/api").Group("/v1")
	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(inner, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rw.Code)
	}
	if !h.pingCalled {
		t.Error("HandlePing was not called for nested prefix")
	}
}

// T4: GetRoutes — returns all registered routes for the group.
func TestRouteGroup_GetRoutes(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	g := r.Group("")
	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(g, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Cast to *fixture.RouteGroup to call GetRoutes.
	rg, ok := g.(*fixture.RouteGroup)
	if !ok {
		// Group() returns Router interface; use the root router's GetRoutes instead.
		t.Skip("cannot access GetRoutes on Router interface directly")
	}

	routes := rg.GetRoutes()
	want := []string{
		"GET /ping",
		"GET /items/{item_id}",
		"PUT /items/{item_id}",
		"PATCH /items/{item_id}",
	}
	if len(routes) != len(want) {
		t.Errorf("expected %d routes, got %d: %v", len(want), len(routes), routes)
		return
	}
	sort.Strings(routes)
	sort.Strings(want)
	for i, r := range routes {
		if r != want[i] {
			t.Errorf("route[%d]: got %q, want %q", i, r, want[i])
		}
	}
}

// Root router GetRoutes test (non-grouped).
func TestNewRouter_GetRoutes(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(r, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	routes := r.GetRoutes()
	if len(routes) != 4 {
		t.Errorf("expected 4 routes, got %d: %v", len(routes), routes)
	}
}

// T5: ErrNilRouter — RegisterPingServiceRoutes returns ErrNilRouter for nil router.
func TestRegisterPingServiceRoutes_NilRouter(t *testing.T) {
	t.Parallel()
	h := &pingHandler{}
	err := fixture.RegisterPingServiceRoutes(nil, h)
	if err != fixture.ErrNilRouter {
		t.Errorf("expected ErrNilRouter, got %v", err)
	}
}

// T6: ErrNilHandler — RegisterPingServiceRoutes returns ErrNilHandler for nil handler.
func TestRegisterPingServiceRoutes_NilHandler(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	err := fixture.RegisterPingServiceRoutes(r, nil)
	if err != fixture.ErrNilHandler {
		t.Errorf("expected ErrNilHandler, got %v", err)
	}
}

// T5b/T6b: Individual route registration also returns correct sentinel errors.
func TestRegisterIndividualRoute_NilErrors(t *testing.T) {
	t.Parallel()
	h := &pingHandler{}

	if err := fixture.RegisterPingRoute(nil, h); err != fixture.ErrNilRouter {
		t.Errorf("RegisterPingRoute nil router: expected ErrNilRouter, got %v", err)
	}
	if err := fixture.RegisterPingRoute(fixture.NewRouter(nil), nil); err != fixture.ErrNilHandler {
		t.Errorf("RegisterPingRoute nil handler: expected ErrNilHandler, got %v", err)
	}

	if err := fixture.RegisterGetItemRoute(nil, h); err != fixture.ErrNilRouter {
		t.Errorf("RegisterGetItemRoute nil router: expected ErrNilRouter, got %v", err)
	}
	if err := fixture.RegisterUpdateItemRoute(nil, h); err != fixture.ErrNilRouter {
		t.Errorf("RegisterUpdateItemRoute nil router: expected ErrNilRouter, got %v", err)
	}
}

// T7: MustRegisterPingServiceRoutes panics on nil router.
func TestMustRegisterPingServiceRoutes_PanicsOnNilRouter(t *testing.T) {
	t.Parallel()
	h := &pingHandler{}
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()
	fixture.MustRegisterPingServiceRoutes(nil, h)
}

// T7b: MustRegisterPingServiceRoutes panics on nil handler.
func TestMustRegisterPingServiceRoutes_PanicsOnNilHandler(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()
	fixture.MustRegisterPingServiceRoutes(r, nil)
}

// T8: UnimplementedPingServiceHandler returns 501.
func TestUnimplementedHandler_Returns501(t *testing.T) {
	t.Parallel()
	var h fixture.UnimplementedPingServiceHandler

	tests := []struct {
		name string
		fn   func(http.ResponseWriter, *http.Request)
		msg  string
	}{
		{"Ping", h.HandlePing, "not implemented: Ping"},
		{"GetItem", h.HandleGetItem, "not implemented: GetItem"},
		{"UpdateItem", h.HandleUpdateItem, "not implemented: UpdateItem"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rw := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.fn(rw, req)
			if rw.Code != http.StatusNotImplemented {
				t.Errorf("expected 501, got %d", rw.Code)
			}
		})
	}
}

// T9: ResponseWriterWrapper captures status code.
func TestResponseWriterWrapper_CapturesStatusCode(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	rw := fixture.NewResponseWriterWrapper(rec)

	if rw.StatusCode != http.StatusOK {
		t.Errorf("default StatusCode: expected 200, got %d", rw.StatusCode)
	}

	rw.WriteHeader(http.StatusCreated)
	if rw.StatusCode != http.StatusCreated {
		t.Errorf("after WriteHeader(201): expected 201, got %d", rw.StatusCode)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("underlying recorder: expected 201, got %d", rec.Code)
	}
}

// T9b: ResponseWriterWrapper.Flush does not panic even when underlying writer is not http.Flusher.
func TestResponseWriterWrapper_FlushNoPanic(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	rw := fixture.NewResponseWriterWrapper(rec)
	// httptest.ResponseRecorder does implement http.Flusher, so this exercises the Flusher path.
	rw.Flush()
	// Ensure no panic = pass.
}

// T11: joinPath via Group — Group prefix is joined correctly.
func TestRouteGroup_JoinPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix  string
		pattern string
		want    string
	}{
		{"/api", "/ping", "/api/ping"},
		{"/api/", "/ping", "/api/ping"},
		{"api", "/ping", "/api/ping"}, // auto-prefixed slash
		{"", "/ping", "/ping"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"+"+tt.pattern, func(t *testing.T) {
			t.Parallel()
			// Fresh router per subtest to avoid duplicate route panics.
			r := fixture.NewRouter(nil)
			called := false
			g := r.Group(tt.prefix)
			g.HandleFunc(http.MethodGet, tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				called = true
				w.WriteHeader(200)
			})

			rw := httptest.NewRecorder()
			r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, tt.want, nil))
			if !called {
				t.Errorf("handler not reached at %q (prefix=%q, pattern=%q)", tt.want, tt.prefix, tt.pattern)
			}
		})
	}
}

// T12: nil middleware filtering — Use(nil, mw) should not panic and mw should run.
func TestRouteGroup_NilMiddlewareFiltered(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	ran := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ran = true
			next.ServeHTTP(w, req)
		})
	}
	// nil middleware should be silently ignored
	r.Use(nil, mw)

	r.HandleFunc(http.MethodGet, "/ping", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
	})

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/ping", nil))
	if !ran {
		t.Error("non-nil middleware did not run")
	}
}

// T13: Shared mux — two RouteGroups sharing the same *http.ServeMux see all routes.
func TestRouteGroup_SharedMux(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	r1 := fixture.NewRouter(mux)
	r2 := fixture.NewRouter(mux)

	r1Called, r2Called := false, false
	r1.HandleFunc(http.MethodGet, "/service1", func(w http.ResponseWriter, req *http.Request) {
		r1Called = true
		w.WriteHeader(200)
	})
	r2.HandleFunc(http.MethodGet, "/service2", func(w http.ResponseWriter, req *http.Request) {
		r2Called = true
		w.WriteHeader(200)
	})

	// Both routers share the mux, so either can serve the other's routes.
	r1.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/service2", nil))
	if !r2Called {
		t.Error("r2's route not reachable via r1.ServeHTTP (shared mux)")
	}

	r2.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/service1", nil))
	if !r1Called {
		t.Error("r1's route not reachable via r2.ServeHTTP (shared mux)")
	}
}

// T14b: Parent Use() after Group() — child routes pick up parent's middleware.
func TestRouteGroup_ParentMiddlewareAfterGroup(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	child := r.Group("/api") // create child BEFORE adding parent middleware

	ran := false
	r.Use(func(next http.Handler) http.Handler { // added to parent AFTER Group()
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ran = true
			next.ServeHTTP(w, req)
		})
	})

	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(child, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/api/ping", nil))

	if rw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rw.Code)
	}
	if !ran {
		t.Error("parent middleware (added after Group()) did not run on child route")
	}
}

// T14c: Nested group inherits all ancestors' middlewares.
func TestRouteGroup_NestedGroupInheritsAncestorMiddleware(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)

	var order []string
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "root")
			next.ServeHTTP(w, req)
		})
	})

	mid := r.Group("/api")
	mid.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "mid")
			next.ServeHTTP(w, req)
		})
	})

	leaf := mid.(*fixture.RouteGroup).Group("/v1") // type assert to access Group on *RouteGroup
	leaf.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			order = append(order, "leaf")
			next.ServeHTTP(w, req)
		})
	})

	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(leaf, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	if rw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rw.Code)
	}
	if len(order) != 3 || order[0] != "root" || order[1] != "mid" || order[2] != "leaf" {
		t.Errorf("expected [root mid leaf], got %v", order)
	}
}

// stubRoutes is a helper for routing tests.
type stubRoutes struct {
	fn http.HandlerFunc
}
