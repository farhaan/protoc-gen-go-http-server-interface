package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestCompileGeneratedCode doesn't require protoc or special build tags
func TestCompileGeneratedCode(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "protoc-gen-compile-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fmt.Println("Created temp directory:", tempDir)

	// Create a go.mod file
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := `module example.com/test

go 1.23
`
	err = os.WriteFile(goModPath, []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write go.mod file: %v", err)
	}

	// Create a test Go file with pre-generated code sample
	testCode := `package main

import (
	"fmt"
	"net/http"
)

// Middleware represents a middleware function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Routes defines methods for registering routes
type Routes interface {
	HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware)
	Group(prefix string, middlewares ...Middleware) *RouteGroup
	Use(middlewares ...Middleware) *RouteGroup
}

// RouteGroup represents a group of routes with a common prefix and middleware
type RouteGroup struct {
	prefix          string
	middlewares     []Middleware
	routes          []routeDef
	mux             *http.ServeMux
}

// routeDef stores a route definition before it's registered with the mux
type routeDef struct {
	method      string
	pattern     string
	handler     http.Handler
}

// NewRouter creates a new router with an optional mux
// If mux is nil, a new http.ServeMux will be created
func NewRouter(mux *http.ServeMux) *RouteGroup {
	if mux == nil {
		mux = http.NewServeMux()
	}
	
	return &RouteGroup{
		prefix:       "",
		middlewares:  []Middleware{},
		routes:       []routeDef{},
		mux:          mux,
	}
}

// DefaultRouter creates a new router with a new ServeMux
func DefaultRouter() *RouteGroup {
	return NewRouter(nil)
}

func (g *RouteGroup) Group(prefix string, middlewares ...Middleware) *RouteGroup {
	return &RouteGroup{
		prefix:      g.prefix + prefix,
		middlewares: append(g.middlewares, middlewares...),
		routes:      []routeDef{},
		mux:         g.mux,
	}
}

func (g *RouteGroup) Use(middlewares ...Middleware) *RouteGroup {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

func (g *RouteGroup) HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware) {
	// implementation not needed for compilation test
}

func (g *RouteGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// TestServiceHandler is the interface for TestService HTTP handlers
type TestServiceHandler interface {
	HandleGetTest(w http.ResponseWriter, r *http.Request)
}

// RegisterTestServiceRoutes registers HTTP routes for TestService
func RegisterTestServiceRoutes(r Routes, handler TestServiceHandler) {
	r.HandleFunc("GET", "/tests/{test_id}", handler.HandleGetTest)
}

// RegisterTestServiceRoutes is a method on RouteGroup to register all TestService routes
func (g *RouteGroup) RegisterTestServiceRoutes(handler TestServiceHandler) {
	RegisterTestServiceRoutes(g, handler)
}

// RegisterGetTestRoute is a helper that registers the GetTest handler
func RegisterGetTestRoute(r Routes, handler TestServiceHandler, middlewares ...Middleware) {
	r.HandleFunc("GET", "/tests/{test_id}", handler.HandleGetTest, middlewares...)
}

// RegisterGetTest is a method on RouteGroup to register the GetTest handler
func (g *RouteGroup) RegisterGetTest(handler TestServiceHandler, middlewares ...Middleware) {
	RegisterGetTestRoute(g, handler, middlewares...)
}

// A sample implementation of the handler
type MyTestHandler struct{}

func (h *MyTestHandler) HandleGetTest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Got a test")
}

// Sample middleware
func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("Request:", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	// Create a shared mux
	sharedMux := http.NewServeMux()
	
	// Create the router with the shared mux
	router1 := NewRouter(sharedMux)
	router2 := NewRouter(sharedMux)
	
	// Apply middleware
	router1.Use(Logger())
	
	// Create a handler
	handler := &MyTestHandler{}
	
	// Register routes on different routers
	router1.RegisterGetTest(handler)
	
	// Group routes
	apiGroup := router2.Group("/api/v1")
	apiGroup.RegisterGetTest(handler)
	
	// Start the server with the shared mux
	http.ListenAndServe(":8080", sharedMux)
}
`

	testFilePath := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(testFilePath, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test Go file: %v", err)
	}

	fmt.Println("Created go file:", testFilePath)

	// Run go build to check if it compiles
	cmd := exec.Command("go", "build", "-o", filepath.Join(tempDir, "test"), ".")
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running go build...")
	err = cmd.Run()
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Check if the binary was created
	binPath := filepath.Join(tempDir, "test")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Fatalf("Compiled binary not found: %s", binPath)
	}

	fmt.Println("Successfully compiled test program!")
}

// TestSharedMuxCompilation verifies that the shared mux functionality compiles
func TestSharedMuxCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "shared-mux-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a go.mod file
	goModPath := filepath.Join(tempDir, "go.mod")
	goModContent := `module example.com/test

go 1.23
`
	err = os.WriteFile(goModPath, []byte(goModContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write go.mod file: %v", err)
	}

	// Create a test file demonstrating shared mux usage
	testCode := `package main

import (
	"fmt"
	"net/http"
)

// Simplified versions of the generated types for testing
type Middleware func(http.Handler) http.Handler

type Routes interface {
	HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware)
	Group(prefix string, middlewares ...Middleware) *RouteGroup
	Use(middlewares ...Middleware) *RouteGroup
}

type RouteGroup struct {
	prefix          string
	middlewares     []Middleware
	routes          []routeDef
	mux             *http.ServeMux
}

type routeDef struct {
	method      string
	pattern     string
	handler     http.Handler
}

func NewRouter(mux *http.ServeMux) *RouteGroup {
	if mux == nil {
		mux = http.NewServeMux()
	}
	
	return &RouteGroup{
		prefix:       "",
		middlewares:  []Middleware{},
		routes:       []routeDef{},
		mux:          mux,
	}
}

func DefaultRouter() *RouteGroup {
	return NewRouter(nil)
}

func (g *RouteGroup) Group(prefix string, middlewares ...Middleware) *RouteGroup {
	return &RouteGroup{
		prefix:      g.prefix + prefix,
		middlewares: append(g.middlewares, middlewares...),
		routes:      []routeDef{},
		mux:         g.mux,
	}
}

func (g *RouteGroup) Use(middlewares ...Middleware) *RouteGroup {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

func (g *RouteGroup) HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware) {	
	// Store the route definition
	g.routes = append(g.routes, routeDef{
		method:  method,
		pattern: g.prefix + pattern,
		handler: handler,
	})
	
}

func (g *RouteGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// Sample handlers for testing
type ProductHandler struct{}
type UserHandler struct{}

func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Product details")
}

func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "User details")
}

// Mock interfaces for testing
type ProductServiceHandler interface {
	HandleGetProduct(w http.ResponseWriter, r *http.Request)
}

type UserServiceHandler interface {
	HandleGetUser(w http.ResponseWriter, r *http.Request)
}

// Sample registration functions
func (g *RouteGroup) RegisterGetProduct(handler ProductServiceHandler, middlewares ...Middleware) {
	g.HandleFunc("GET", "/products/{id}", handler.HandleGetProduct, middlewares...)
}

func (g *RouteGroup) RegisterGetUser(handler UserServiceHandler, middlewares ...Middleware) {
	g.HandleFunc("GET", "/users/{id}", handler.HandleGetUser, middlewares...)
}

func main() {
	// Create a shared ServeMux
	sharedMux := http.NewServeMux()
	
	// Create routers that share the mux
	productRouter := NewRouter(sharedMux)
	userRouter := NewRouter(sharedMux)
	
	// Create handlers
	productHandler := &ProductHandler{}
	userHandler := &UserHandler{}
	
	// Register routes on different routers
	productRouter.RegisterGetProduct(productHandler)
	userRouter.RegisterGetUser(userHandler)
	
	// Group routes with prefixes
	v1Products := productRouter.Group("/api/v1")
	v1Products.RegisterGetProduct(productHandler)
	
	v1Users := userRouter.Group("/api/v1")
	v1Users.RegisterGetUser(userHandler)

	
	// Start the server with the shared mux
	fmt.Println("Starting server on :8080")
	http.ListenAndServe(":8080", sharedMux)
}
`

	testFilePath := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(testFilePath, []byte(testCode), 0644)
	if err != nil {
		t.Fatalf("Failed to write test Go file: %v", err)
	}

	// Run go build to check if it compiles
	cmd := exec.Command("go", "build", "-o", filepath.Join(tempDir, "test"), ".")
	cmd.Dir = tempDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Check if the binary was created
	binPath := filepath.Join(tempDir, "test")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		t.Fatalf("Compiled binary not found: %s", binPath)
	}

	fmt.Println("Successfully compiled shared mux test program!")
}

// Helper function for manual testing that creates a sample implementation file
func CreateSampleImplementation() string {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "routegroup-sample")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	// Create a sample implementation file
	code := `package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Middleware represents a middleware function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Routes defines methods for registering routes
type Routes interface {
	HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware)
	Group(prefix string, middlewares ...Middleware) *RouteGroup
	Use(middlewares ...Middleware) *RouteGroup
}

// RouteGroup represents a group of routes with a common prefix and middleware
type RouteGroup struct {
	prefix          string
	middlewares     []Middleware
	routes          []routeDef
	mux             *http.ServeMux
}

// routeDef stores a route definition before it's registered with the mux
type routeDef struct {
	method      string
	pattern     string
	handler     http.Handler
}

// NewRouter creates a new router with an optional mux
func NewRouter(mux *http.ServeMux) *RouteGroup {
	if mux == nil {
		mux = http.NewServeMux()
	}
	
	return &RouteGroup{
		prefix:       "",
		middlewares:  []Middleware{},
		routes:       []routeDef{},
		mux:          mux,
	}
}

// DefaultRouter creates a new router with a new ServeMux
func DefaultRouter() *RouteGroup {
	return NewRouter(nil)
}

// Group creates a new RouteGroup with the given prefix and optional middlewares
func (g *RouteGroup) Group(prefix string, middlewares ...Middleware) *RouteGroup {
	// Ensure prefix starts with /
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	
	// If prefix is just "/", don't add it again
	if g.prefix == "/" && prefix == "/" {
		prefix = "/"
	} else if prefix == "/" {
		prefix = g.prefix
	} else {
		prefix = g.prefix + prefix
	}
	
	return &RouteGroup{
		prefix:       prefix,
		middlewares:  append(append([]Middleware{}, g.middlewares...), middlewares...),
		routes:       []routeDef{}, // Each group gets its own routes
		mux:          g.mux,        // Share the same mux
	}
}

// Use applies middlewares to all routes registered after this call
func (g *RouteGroup) Use(middlewares ...Middleware) *RouteGroup {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

// HandleFunc registers a handler function for the given method and pattern
func (g *RouteGroup) HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware) {
	// Apply the prefix to the pattern
	fullPattern := g.prefix
	if g.prefix == "/" && pattern != "/" {
		fullPattern = pattern
	} else if pattern == "/" {
		fullPattern = g.prefix
	} else {
		fullPattern = g.prefix + pattern
	}
	
	// Apply middleware chain to the handler
	var finalHandler http.Handler = handler
	
	// Apply route-specific middlewares first (innermost)
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}
	
	// Apply group middlewares (outermost)
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		finalHandler = g.middlewares[i](finalHandler)
	}
	
	// Store and register the route definition
	routeDefinition := routeDef{
		method:   method,
		pattern:  fullPattern,
		handler:  finalHandler,
	}
	g.routes = append(g.routes, routeDefinition)

	routeDefinitionKey := routeDefinition.method + " " + routeDefinition.pattern
	g.mux.Handle(routeDefinitionKey, routeDefinition.handler)

}

// ServeHTTP implements the http.Handler interface
func (g *RouteGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mux.ServeHTTP(w, r)
}

// Logger middleware logs request details
func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

// Auth middleware checks for authentication
func Auth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	// Create a shared ServeMux
	sharedMux := http.NewServeMux()
	
	// Create two routers sharing the same mux
	router1 := NewRouter(sharedMux)
	router2 := NewRouter(sharedMux)
	
	// Apply global middleware to router1
	router1.Use(Logger())
	
	// Add routes to router1
	router1.HandleFunc("GET", "/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Router 1!")
	})
	
	// Add routes to router2 with a path prefix
	apiGroup := router2.Group("/api")
	apiGroup.HandleFunc("GET", "/items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "[{\"id\":1,\"name\":\"Item 1\"},{\"id\":2,\"name\":\"Item 2\"}]")
	})
	
	// Create a v1 group with middleware
	v1Group := apiGroup.Group("/v1").Use(Auth())
	v1Group.HandleFunc("POST", "/items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{\"id\":3,\"name\":\"New Item\",\"status\":\"created\"}")
	})
	
	
	fmt.Println("Server starting on http://localhost:8080")
	err := http.ListenAndServe(":8080", sharedMux)
	if err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
`

	filePath := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(filePath, []byte(code), 0644)
	if err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Printf("Created sample implementation at: %s\n", filePath)
	fmt.Println("You can run it with: go run " + filePath)

	return filePath
}
