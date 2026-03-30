package benchmark_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface"
	"github.com/farhaan/protoc-gen-go-http-server-interface/httpinterface/parser"
	"tests/runtime/fixture"
)

// --- Generator benchmarks ---

func makeSingleServiceData(n int) *httpinterface.ServiceData {
	methods := make([]httpinterface.MethodInfo, n)
	for i := range n {
		methods[i] = httpinterface.MethodInfo{
			Name: fmt.Sprintf("Method%d", i),
			HTTPRules: []parser.HTTPRule{
				{Method: "GET", Pattern: fmt.Sprintf("/resource%d/{id}", i), PathParams: []string{"id"}},
			},
		}
	}
	return &httpinterface.ServiceData{
		PackageName: "benchpkg",
		Services: []httpinterface.ServiceInfo{
			{Name: "BenchService", Methods: methods},
		},
	}
}

// BenchmarkGenerate_Small benchmarks generation of a small service (5 methods).
func BenchmarkGenerate_Small(b *testing.B) {
	g := httpinterface.New()
	data := makeSingleServiceData(5)
	b.ResetTimer()
	for range b.N {
		if _, err := g.GenerateCode(data); err != nil {
			b.Fatalf("GenerateCode: %v", err)
		}
	}
}

// BenchmarkGenerate_Medium benchmarks generation of a medium service (25 methods).
func BenchmarkGenerate_Medium(b *testing.B) {
	g := httpinterface.New()
	data := makeSingleServiceData(25)
	b.ResetTimer()
	for range b.N {
		if _, err := g.GenerateCode(data); err != nil {
			b.Fatalf("GenerateCode: %v", err)
		}
	}
}

// BenchmarkGenerate_Large benchmarks generation of a large service (100 methods).
func BenchmarkGenerate_Large(b *testing.B) {
	g := httpinterface.New()
	data := makeSingleServiceData(100)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		if _, err := g.GenerateCode(data); err != nil {
			b.Fatalf("GenerateCode: %v", err)
		}
	}
}

// BenchmarkGenerate_ManyServices benchmarks generation with many small services.
func BenchmarkGenerate_ManyServices(b *testing.B) {
	g := httpinterface.New()
	services := make([]httpinterface.ServiceInfo, 20)
	for i := range services {
		services[i] = httpinterface.ServiceInfo{
			Name: fmt.Sprintf("Service%d", i),
			Methods: []httpinterface.MethodInfo{
				{Name: "Get", HTTPRules: []parser.HTTPRule{{Method: "GET", Pattern: fmt.Sprintf("/svc%d/{id}", i), PathParams: []string{"id"}}}},
				{Name: "Create", HTTPRules: []parser.HTTPRule{{Method: "POST", Pattern: fmt.Sprintf("/svc%d", i), Body: "*"}}},
			},
		}
	}
	data := &httpinterface.ServiceData{PackageName: "multisvc", Services: services}
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		if _, err := g.GenerateCode(data); err != nil {
			b.Fatalf("GenerateCode: %v", err)
		}
	}
}

// --- Route dispatch benchmarks ---

type nopHandler struct {
	fixture.UnimplementedPingServiceHandler
}

func (nopHandler) HandlePing(w http.ResponseWriter, r *http.Request)       { w.WriteHeader(200) }
func (nopHandler) HandleGetItem(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(200) }
func (nopHandler) HandleUpdateItem(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }

// BenchmarkRouteDispatch_NoMiddleware benchmarks request dispatch with no middleware.
func BenchmarkRouteDispatch_NoMiddleware(b *testing.B) {
	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, nopHandler{}); err != nil {
		b.Fatalf("register: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)
	}
}

// BenchmarkRouteDispatch_WithMiddleware benchmarks dispatch through 3 chained middlewares.
func BenchmarkRouteDispatch_WithMiddleware(b *testing.B) {
	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, nopHandler{}); err != nil {
		b.Fatalf("register: %v", err)
	}

	nop := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
		})
	}
	r.Use(nop, nop, nop)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)
	}
}

// BenchmarkRouteDispatch_PathParam benchmarks dispatch for a route with a path parameter.
func BenchmarkRouteDispatch_PathParam(b *testing.B) {
	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, nopHandler{}); err != nil {
		b.Fatalf("register: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/items/abc123", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		rw := httptest.NewRecorder()
		r.ServeHTTP(rw, req)
	}
}

// --- Middleware application benchmarks ---

// BenchmarkApplyMiddlewares_0 benchmarks applyMiddlewares-equivalent with 0 middlewares
// (measured indirectly through route dispatch).
func BenchmarkRouteDispatch_0Middlewares(b *testing.B) {
	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, nopHandler{}); err != nil {
		b.Fatalf("register: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ResetTimer()
	for range b.N {
		r.ServeHTTP(httptest.NewRecorder(), req)
	}
}

// BenchmarkRouteDispatch_10Middlewares benchmarks dispatch through 10 chained middlewares.
func BenchmarkRouteDispatch_10Middlewares(b *testing.B) {
	r := fixture.NewRouter(nil)
	if err := fixture.RegisterPingServiceRoutes(r, nopHandler{}); err != nil {
		b.Fatalf("register: %v", err)
	}

	nop := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req)
		})
	}
	for range 10 {
		r.Use(nop)
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		r.ServeHTTP(httptest.NewRecorder(), req)
	}
}

// BenchmarkGroupCreation benchmarks creating nested route groups.
func BenchmarkGroupCreation(b *testing.B) {
	r := fixture.NewRouter(nil)
	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		g := r.Group("/api")
		_ = g.Group("/v1")
	}
}
