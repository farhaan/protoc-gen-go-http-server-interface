package behavior_test

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"tests/runtime/fixture"
)

// T14: Dispatch-time middleware — middleware added via Use() AFTER RegisterXxxRoutes still runs.
func TestMiddleware_DispatchTime(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)

	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(r, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Add middleware AFTER registration — must still run at request time.
	mwRan := false
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			mwRan = true
			next.ServeHTTP(w, req)
		})
	})

	rw := httptest.NewRecorder()
	r.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/ping", nil))

	if !h.pingCalled {
		t.Error("handler not called")
	}
	if !mwRan {
		t.Error("dispatch-time middleware did not run (registered after route)")
	}
}

// T15: Middleware order — middlewares execute FIFO (first Use() call is outermost wrapper).
func TestMiddleware_Order(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)

	var order []string
	makeMiddleware := func(name string) fixture.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, req)
				order = append(order, name+":after")
			})
		}
	}

	r.Use(makeMiddleware("mw1"))
	r.Use(makeMiddleware("mw2"))

	r.HandleFunc(http.MethodGet, "/ping", func(w http.ResponseWriter, req *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(200)
	})

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	// Expected: mw1 wraps mw2 wraps handler (FIFO = outermost first).
	want := []string{"mw1:before", "mw2:before", "handler", "mw2:after", "mw1:after"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i, s := range order {
		if s != want[i] {
			t.Errorf("order[%d]: expected %q, got %q", i, want[i], s)
		}
	}
}

// T16: Per-route middleware via RegisterXxxRoute.
func TestMiddleware_PerRoute(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)

	pingMwRan := false
	pingMw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			pingMwRan = true
			next.ServeHTTP(w, req)
		})
	}

	h := &pingHandler{}

	// Register only the Ping route with a per-route middleware.
	if err := fixture.RegisterPingRoute(r, h, pingMw); err != nil {
		t.Fatalf("RegisterPingRoute: %v", err)
	}
	// Register GetItem without any middleware.
	if err := fixture.RegisterGetItemRoute(r, h); err != nil {
		t.Fatalf("RegisterGetItemRoute: %v", err)
	}

	// Hit /ping — per-route mw should run.
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	if !pingMwRan {
		t.Error("per-route middleware did not run for /ping")
	}

	// Hit /items/{item_id} — per-route mw should NOT run.
	pingMwRan = false
	req := httptest.NewRequest(http.MethodGet, "/items/abc", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
	if pingMwRan {
		t.Error("per-route middleware ran for /items/{item_id} — should not")
	}
}

// T16b: UpdateItem route registers both PUT and PATCH bindings.
func TestRegisterUpdateItemRoute_BothBindings(t *testing.T) {
	t.Parallel()
	r := fixture.NewRouter(nil)
	h := &pingHandler{}
	if err := fixture.RegisterUpdateItemRoute(r, h); err != nil {
		t.Fatalf("RegisterUpdateItemRoute: %v", err)
	}

	for _, method := range []string{http.MethodPut, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			h.updateItemCalled = false
			req := httptest.NewRequest(method, "/items/123", nil)
			r.ServeHTTP(httptest.NewRecorder(), req)
			if !h.updateItemCalled {
				t.Errorf("%s /items/123 did not reach HandleUpdateItem", method)
			}
		})
	}
}

// T17: Goroutine leak — serving requests must not leak goroutines.
func TestMiddleware_NoGoroutineLeak(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}

	r := fixture.NewRouter(nil)
	h := &pingHandler{}
	if err := fixture.RegisterPingServiceRoutes(r, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Warmup.
	for range 10 {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	}
	runtime.GC()
	before := runtime.NumGoroutine()

	const requests = 500
	for range requests {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	}

	runtime.GC()
	after := runtime.NumGoroutine()

	// Allow a small buffer for background goroutines (test runtime, GC, etc.).
	const maxLeak = 5
	if after-before > maxLeak {
		t.Errorf("possible goroutine leak: before=%d after=%d (delta=%d, max allowed=%d)",
			before, after, after-before, maxLeak)
	}
}
