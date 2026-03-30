package performance_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	"tests/runtime/fixture"
)

// T32: Heap stability — repeated code generation must not cause unbounded heap growth.
func TestMemory_GeneratorHeapStability(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping heap stability test in short mode")
	}

	g := httpinterface.New()
	serviceData := &httpinterface.ServiceData{
		PackageName: "mempkg",
		Services: []httpinterface.ServiceInfo{
			{
				Name: "MemService",
				Methods: []httpinterface.MethodInfo{
					{Name: "GetA", HTTPRules: []parser.HTTPRule{{Method: "GET", Pattern: "/a/{id}", PathParams: []string{"id"}}}},
					{Name: "GetB", HTTPRules: []parser.HTTPRule{{Method: "GET", Pattern: "/b/{id}", PathParams: []string{"id"}}}},
					{Name: "Create", HTTPRules: []parser.HTTPRule{{Method: "POST", Pattern: "/c", Body: "*"}}},
					{Name: "Update", HTTPRules: []parser.HTTPRule{
						{Method: "PUT", Pattern: "/c/{id}", PathParams: []string{"id"}},
						{Method: "PATCH", Pattern: "/c/{id}", PathParams: []string{"id"}},
					}},
					{Name: "Delete", HTTPRules: []parser.HTTPRule{{Method: "DELETE", Pattern: "/c/{id}", PathParams: []string{"id"}}}},
				},
			},
		},
	}

	// Warmup to allow any one-time allocations to settle.
	for range 100 {
		if _, err := g.GenerateCode(serviceData); err != nil {
			t.Fatalf("GenerateCode: %v", err)
		}
	}
	runtime.GC()
	runtime.GC()

	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	const iterations = 10_000
	for range iterations {
		if _, err := g.GenerateCode(serviceData); err != nil {
			t.Fatalf("GenerateCode: %v", err)
		}
	}

	runtime.GC()
	runtime.GC()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	// HeapInuse growth should be under 10 MB for 10k generations.
	const maxHeapGrowthBytes = 10 * 1024 * 1024
	heapGrowth := int64(after.HeapInuse) - int64(before.HeapInuse)
	t.Logf("HeapInuse before=%d after=%d growth=%d bytes over %d iterations",
		before.HeapInuse, after.HeapInuse, heapGrowth, iterations)

	if heapGrowth > maxHeapGrowthBytes {
		t.Errorf("heap grew by %d bytes (%.1f MB) over %d iterations — possible memory leak (max allowed: %d MB)",
			heapGrowth, float64(heapGrowth)/(1024*1024), iterations, maxHeapGrowthBytes/(1024*1024))
	}
}

// T33: Goroutine leak — serving many requests via the generated router must not leak goroutines.
func TestMemory_RouterGoroutineLeak(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping goroutine leak test in short mode")
	}

	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, &pingNopHandler{}); err != nil {
		t.Fatalf("register: %v", err)
	}

	// Add some middleware to exercise the applyMiddlewares path.
	nop := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
		})
	}
	r.Use(nop, nop)

	// Warmup.
	for range 50 {
		serveRequest(r, http.MethodGet, "/ping")
	}
	runtime.GC()
	before := runtime.NumGoroutine()

	const requests = 1000
	for i := range requests {
		switch i % 4 {
		case 0:
			serveRequest(r, http.MethodGet, "/ping")
		case 1:
			serveRequest(r, http.MethodGet, "/items/test-id")
		case 2:
			serveRequest(r, http.MethodPut, "/items/test-id")
		case 3:
			serveRequest(r, http.MethodPatch, "/items/test-id")
		}
	}

	runtime.GC()
	after := runtime.NumGoroutine()

	const maxLeak = 5
	delta := after - before
	t.Logf("goroutines before=%d after=%d delta=%d (after %d requests)", before, after, delta, requests)
	if delta > maxLeak {
		t.Errorf("goroutine leak: delta=%d exceeds max allowed %d", delta, maxLeak)
	}
}

// T34: Memory allocation per request — verify each dispatch doesn't allocate excessively.
// Uses a minimal no-op ResponseWriter to avoid httptest.NewRecorder() overhead.
func TestMemory_PerRequestAllocations(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, &pingNopHandler{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	nopW := &nopResponseWriter{}

	// Warmup.
	for range 50 {
		r.ServeHTTP(nopW, req)
	}

	// Measure allocations per request using a no-op ResponseWriter.
	const iterations = 1000
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for range iterations {
		r.ServeHTTP(nopW, req)
	}
	runtime.ReadMemStats(&after)
	totalAllocs := after.Mallocs - before.Mallocs

	allocsPerRequest := float64(totalAllocs) / float64(iterations)
	t.Logf("allocations per request: %.1f (total=%d over %d requests)", allocsPerRequest, totalAllocs, iterations)

	// maxAllocsPerRequest is set by build-tag gated constants:
	//   race_disabled_test.go  → 10.0 (baseline)
	//   race_enabled_test.go   → 25.0 (race detector adds shadow-memory overhead)
	if allocsPerRequest > maxAllocsPerRequest {
		t.Errorf("too many allocations per request: %.1f (max allowed: %.1f)", allocsPerRequest, maxAllocsPerRequest)
	}
}

// T35: Multiple generators don't share state — verify no cross-contamination.
func TestMemory_GeneratorStateIsolation(t *testing.T) {
	t.Parallel()

	g1 := httpinterface.New()
	g2 := httpinterface.New()

	makeData := func(pkgName string) *httpinterface.ServiceData {
		return &httpinterface.ServiceData{
			PackageName: pkgName,
			Services: []httpinterface.ServiceInfo{
				{
					Name: "IsolatedService",
					Methods: []httpinterface.MethodInfo{
						{Name: "Get", HTTPRules: []parser.HTTPRule{{Method: "GET", Pattern: "/items"}}},
					},
				},
			},
		}
	}

	// Generate from g1 and g2 concurrently to expose any shared state.
	const goroutines = 10
	errs := make(chan error, goroutines*2)

	for i := range goroutines {
		go func(id int) {
			d1 := makeData(fmt.Sprintf("pkg1_%d", id))
			d2 := makeData(fmt.Sprintf("pkg2_%d", id))

			out1, err := g1.GenerateCode(d1)
			if err != nil {
				errs <- fmt.Errorf("g1: %w", err)
				return
			}
			out2, err := g2.GenerateCode(d2)
			if err != nil {
				errs <- fmt.Errorf("g2: %w", err)
				return
			}

			if !containsStr(out1, fmt.Sprintf("package pkg1_%d", id)) {
				errs <- fmt.Errorf("g1 output missing correct package name pkg1_%d", id)
				return
			}
			if !containsStr(out2, fmt.Sprintf("package pkg2_%d", id)) {
				errs <- fmt.Errorf("g2 output missing correct package name pkg2_%d", id)
				return
			}
			errs <- nil
		}(i)
	}

	for range goroutines {
		if err := <-errs; err != nil {
			t.Error(err)
		}
	}
}

// pingNopHandler is a minimal PingServiceHandler that does nothing.
type pingNopHandler struct {
	fixture.UnimplementedPingServiceHandler
}

func (pingNopHandler) HandlePing(w http.ResponseWriter, r *http.Request)       { w.WriteHeader(200) }
func (pingNopHandler) HandleGetItem(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(200) }
func (pingNopHandler) HandleUpdateItem(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }

// nopResponseWriter is a minimal http.ResponseWriter that discards all output.
type nopResponseWriter struct{ header http.Header }

func (w *nopResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}
func (w *nopResponseWriter) Write(b []byte) (int, error)  { return len(b), nil }
func (w *nopResponseWriter) WriteHeader(statusCode int)   {}

func serveRequest(r *fixture.RouteGroup, method, path string) {
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}
