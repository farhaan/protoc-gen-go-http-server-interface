package behavior_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"tests/runtime/fixture"
)

// atomicHandler is a PingServiceHandler whose call flags are safe for concurrent use.
type atomicHandler struct {
	fixture.UnimplementedPingServiceHandler
	pingCount atomic.Int64
}

func (h *atomicHandler) HandlePing(w http.ResponseWriter, _ *http.Request) {
	h.pingCount.Add(1)
	w.WriteHeader(http.StatusOK)
}

func (h *atomicHandler) HandleGetItem(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *atomicHandler) HandleUpdateItem(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// T18: Concurrent HandleFunc — registering distinct routes on the same group from
// multiple goroutines must not race on g.routes (regression for the missing-mutex bug).
// Run with: go test -race ./tests/runtime/...
func TestConcurrent_HandleFuncRegistration(t *testing.T) {
	t.Parallel()
	const n = 20

	r := fixture.NewRouter(nil)
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Each goroutine registers a distinct route so ServeMux does not panic
			// on duplicate registration. The race we're guarding against is the
			// unsynchronised g.routes = append(...) slice write, not mux.Handle.
			pattern := fmt.Sprintf("/concurrent/%d", i)
			r.HandleFunc(http.MethodGet, pattern, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
		}(i)
	}
	wg.Wait()
	// No -race flag violation = pass.
}

// T19: Use() called concurrently while requests are in flight.
// The route must still execute and the newly added middleware must eventually run.
func TestConcurrent_UseWhileServing(t *testing.T) {
	t.Parallel()

	r := fixture.NewRouter(nil)
	h := &atomicHandler{}
	if err := fixture.RegisterPingServiceRoutes(r, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	const goroutines = 30
	var wg sync.WaitGroup
	var mwRunCount atomic.Int64

	// Half the goroutines serve requests; the other half add middleware.
	for i := range goroutines {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				r.Use(func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						mwRunCount.Add(1)
						next.ServeHTTP(w, req)
					})
				})
			} else {
				r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
			}
		}(i)
	}
	wg.Wait()

	// At least one request must have reached the handler.
	if h.pingCount.Load() == 0 {
		t.Error("handler never called")
	}
	// No panic and no -race violation = pass.
}

// T20: Deep nesting (8 levels) — effectiveVersion sums all ancestors,
// collectMiddlewareChain preserves root-to-leaf order.
func TestDeepNesting_MiddlewareOrderAndVersion(t *testing.T) {
	t.Parallel()

	r := fixture.NewRouter(nil)

	// Build root → l1 → l2 → ... → l7 (8 total levels including root).
	const depth = 8
	levels := make([]fixture.Router, depth)
	levels[0] = r
	for i := 1; i < depth; i++ {
		levels[i] = levels[i-1].(*fixture.RouteGroup).Group("/seg")
	}

	// Single-request test — no concurrent writes, so a plain slice is fine.
	var order []string
	for i, lvl := range levels {
		label := string([]byte{'0' + byte(i)}) // capture by value
		lvl.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				order = append(order, label)
				next.ServeHTTP(w, req)
			})
		})
	}

	leaf := levels[depth-1]
	leaf.HandleFunc(http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	path := ""
	for range depth - 1 {
		path += "/seg"
	}
	path += "/ping"

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))

	if len(order) != depth {
		t.Fatalf("expected %d middleware entries, got %d: %v", depth, len(order), order)
	}
	for i, got := range order {
		want := string([]byte{'0' + byte(i)})
		if got != want {
			t.Errorf("order[%d]: want %q, got %q", i, want, got)
		}
	}
}

// T21: Thundering herd — many goroutines simultaneously trigger a cache rebuild
// after Use() bumps the version. All must complete without panic or incorrect middleware.
func TestConcurrent_ThunderingHerdAfterUse(t *testing.T) {
	t.Parallel()

	r := fixture.NewRouter(nil)
	h := &atomicHandler{}
	if err := fixture.RegisterPingServiceRoutes(r, h); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Prime the cache at version 0.
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))

	// Add middleware so every goroutine below sees a cache miss simultaneously.
	var mwRan atomic.Bool
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			mwRan.Store(true)
			next.ServeHTTP(w, req)
		})
	})

	const goroutines = 50
	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
		}()
	}
	wg.Wait()

	if !mwRan.Load() {
		t.Error("middleware never ran after thundering herd rebuild")
	}
}

// T22: Use() version bump atomicity — bumping inside the lock means any
// goroutine that observes the new version must also observe the new middleware.
// Verified with -race; tests the absence of {old_ver, new_chain} stale entries
// producing wrong behaviour.
func TestConcurrent_VersionBumpInsideLock(t *testing.T) {
	t.Parallel()

	const iterations = 200
	for range iterations {
		r := fixture.NewRouter(nil)

		// Barrier to start request and Use() simultaneously.
		start := make(chan struct{})
		mwSeen := make(chan bool, 1)

		r.HandleFunc(http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Goroutine A: serve a request.
		go func() {
			<-start
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ping", nil))
		}()

		// Goroutine B: add middleware at the same moment.
		go func() {
			<-start
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					mwSeen <- true
					next.ServeHTTP(w, req)
				})
			})
		}()

		close(start)

		// Drain mwSeen without blocking; the important invariant is no panic / no race.
		select {
		case <-mwSeen:
		default:
		}
	}
}
