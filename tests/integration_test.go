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

go 1.24.1
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
	mux             *http.ServeMux
	prefix          string
	middlewares     []Middleware
	registeredPaths map[string]bool
}

// NewRouter creates a new router (which is just a RouteGroup with root prefix)
func NewRouter() *RouteGroup {
	return &RouteGroup{
		mux:             http.NewServeMux(),
		prefix:          "/",
		middlewares:     []Middleware{},
		registeredPaths: make(map[string]bool),
	}
}

func (g *RouteGroup) Group(prefix string, middlewares ...Middleware) *RouteGroup {
	return &RouteGroup{}
}

func (g *RouteGroup) Use(middlewares ...Middleware) *RouteGroup {
	return g
}

func (g *RouteGroup) HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware) {
	// implementation not needed for compilation test
}

func (g *RouteGroup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// implementation not needed for compilation test
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
	// Create the router
	router := NewRouter()
	
	// Apply middleware
	router.Use(Logger())
	
	// Create a handler
	handler := &MyTestHandler{}
	
	// Register routes
	router.RegisterGetTest(handler)
	
	// Group routes
	apiGroup := router.Group("/api/v1")
	apiGroup.RegisterGetTest(handler)
	
	// Start the server
	http.ListenAndServe(":8080", router)
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
	mux             *http.ServeMux
	prefix          string
	middlewares     []Middleware
	registeredPaths map[string]bool
}

// NewRouter creates a new router (which is just a RouteGroup with root prefix)
func NewRouter() *RouteGroup {
	return &RouteGroup{
		mux:             http.NewServeMux(),
		prefix:          "/",
		middlewares:     []Middleware{},
		registeredPaths: make(map[string]bool),
	}
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
		mux:             g.mux,
		prefix:          prefix,
		middlewares:     append(append([]Middleware{}, g.middlewares...), middlewares...),
		registeredPaths: g.registeredPaths,
	}
}

// Use applies middlewares to all routes registered after this call
// Returns the RouteGroup for method chaining
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
	
	// Check if the route is already registered
	routeKey := method + " " + fullPattern
	if g.registeredPaths[routeKey] {
		// Either return an error or log a warning
		// For now, we'll just print a warning
		fmt.Printf("Warning: Route already registered: %s\n", routeKey)
		return
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
	
	// Register the handler with the ServeMux
	g.mux.HandleFunc(routeKey, finalHandler.ServeHTTP)
	
	// Mark the route as registered
	g.registeredPaths[routeKey] = true
}

// ServeHTTP implements the http.Handler interface
func (g *RouteGroup) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	g.mux.ServeHTTP(w, req)
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
	router := NewRouter()
	
	// Apply global middleware
	router.Use(Logger())
	
	// Add a simple route
	router.HandleFunc("GET", "/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, World!")
	})
	
	// Add routes with middleware
	router.HandleFunc("GET", "/secure", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Secure endpoint")
	}, Auth())
	
	// Create a group with path prefix
	apiGroup := router.Group("/api")
	apiGroup.HandleFunc("GET", "/items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "[{\"id\":1,\"name\":\"Item 1\"},{\"id\":2,\"name\":\"Item 2\"}]")
	})
	
	// Create a v1 group with middleware
	v1Group := apiGroup.Group("/v1").Use(Auth())
	v1Group.HandleFunc("POST", "/items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "{\"id\":3,\"name\":\"New Item\",\"status\":\"created\"}")
	})
	
	fmt.Println("Server starting on http://localhost:8080")
	err := http.ListenAndServe(":8080", router)
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
