# protoc-gen-go-http-server-interface

A Protocol Buffer code generator plugin that produces HTTP server interfaces and middleware support from Protocol Buffer service definitions.

## System Requirements

- **Go**: Go 1.16 or higher
- **Protocol Buffers**: protoc 3.14.0 or higher
- **Google API Proto Files**: Required for HTTP annotations
  - Install with: `go get -u google.golang.org/genproto/googleapis/api/annotations`
- **Go Protobuf Plugin**: Required for Go code generation
  - Install with: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`



## Overview

This plugin generates Go HTTP handler interfaces, route registration functions, and middleware support from Protocol Buffer service definitions with Google HTTP annotations. It provides a clean, expressive API for registering HTTP routes with middleware.

### Key Features

- **Strongly Typed Handler Interfaces**: Generated interfaces match your Protocol Buffer service definitions
- **Middleware Support**: Global, group-level, and route-specific middleware
- **Path Prefixing**: Easy API versioning with route groups
- **Method Chaining**: Fluent API for intuitive route registration
- **Duplicate Route Protection**: Prevents accidental registration of duplicate routes
- **Minimal Abstractions**: Simple design with just one core type (RouteGroup)

## Design Philosophy

The design follows a domain-driven approach with these principles:

1. **Simplicity over complexity**: One core type (`RouteGroup`) handles both root routing and sub-groups
2. **Type safety**: Generated interfaces ensure compile-time checks for handler implementations 
3. **Familiar patterns**: API inspired by popular Go HTTP frameworks (Chi, Echo, Gin)
4. **Middleware chain clarity**: Clear execution order (global → group → route-specific)
5. **No external dependencies**: Uses only the Go standard library

## Installation

```bash
go install github.com/farhaan/protoc-gen-go-http-server-interface@latest
```

Ensure that the `protoc` compiler is installed and that `$GOPATH/bin` is in your `$PATH`.

## Usage

### 1. Define your Protocol Buffer services with HTTP annotations

```protobuf
syntax = "proto3";

package example.service.v1;
option go_package = "example.com/api/service/v1;servicev1";

import "google/api/annotations.proto";

service ProductService {
  rpc GetProduct(GetProductRequest) returns (GetProductResponse) {
    option (google.api.http) = {
      get: "/products/{product_id}"
    };
  }
  
  rpc CreateProduct(CreateProductRequest) returns (CreateProductResponse) {
    option (google.api.http) = {
      post: "/products"
      body: "*"
    };
  }
  
  // Additional methods...
}

// Message definitions...
```

### 2. Generate code using the plugin

```bash
protoc --go_out=. --go-http-server-interface_out=. your_proto_file.proto
```

### 3. Implement the generated handler interface

```go
package producthandler

import (
	"encoding/json"
	"net/http"
	
	pb "example.com/api/service/v1"
)

type ProductHandler struct {
	service ProductService
}

func NewProductHandler(service ProductService) *ProductHandler {
	return &ProductHandler{service: service}
}

func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	// Extract product_id from URL
	productID := extractPathParam(r, "product_id")
	
	// Call your service
	product, err := h.service.GetProduct(r.Context(), &pb.GetProductRequest{
		ProductId: productID,
	})
	
	// Handle response
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// Implement other methods...
```

### 4. Register routes with your HTTP server

```go
package main

import (
	"log"
	"net/http"
	
	pb "example.com/api/service/v1"
	"example.com/internal/handler"
	"example.com/internal/service"
)

func main() {
	// Create the router
	router := pb.NewRouter()
	
	// Apply global middleware
	router.Use(Logger())
	
	// Create service and handler
	productService := service.NewProductService()
	productHandler := handler.NewProductHandler(productService)
	
	// Register individual routes with middleware
	router.RegisterGetProduct(productHandler)
	router.RegisterListProducts(productHandler, RateLimiter(30))
	
	// Group routes with middleware
	authRoutes := router.Group("/admin").Use(Authentication())
	authRoutes.RegisterCreateProduct(productHandler)
	authRoutes.RegisterUpdateProduct(productHandler)
	authRoutes.RegisterDeleteProduct(productHandler)
	
	// Group routes with path prefix for API versioning
	apiV1 := router.Group("/api/v1")
	apiV1.RegisterGetProduct(productHandler)
	
	// Start the server
	log.Println("Server starting on :8080")
	http.ListenAndServe(":8080", router)
}
```

## Middleware Examples

### Logging Middleware

```go
func Logger() pb.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		})
	}
}
```

### Authentication Middleware

```go
func Authentication() pb.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Validate token...
			next.ServeHTTP(w, r)
		})
	}
}
```

### Rate Limiting Middleware

```go
func RateLimiter(requestsPerMinute int) pb.Middleware {
	limiter := rate.NewLimiter(rate.Limit(requestsPerMinute/60.0), 1)
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

## Core Components

### The Routes Interface

```go
// Routes defines methods for registering routes
type Routes interface {
	// HandleFunc registers a handler function for the given method and pattern
	HandleFunc(method, pattern string, handler http.HandlerFunc, middlewares ...Middleware)
	
	// Group creates a new RouteGroup with the given prefix and optional middlewares
	Group(prefix string, middlewares ...Middleware) *RouteGroup
	
	// Use applies middlewares to all routes registered after this call
	Use(middlewares ...Middleware) *RouteGroup
}
```

### The RouteGroup Type

```go
// RouteGroup represents a group of routes with a common prefix and middleware
type RouteGroup struct {
	mux             *http.ServeMux
	prefix          string
	middlewares     []Middleware
	registeredPaths map[string]bool
}
```

### Middleware Type

```go
// Middleware represents a middleware function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler
```

## Generated Code

For each Protocol Buffer service, the plugin generates:

1. A handler interface matching the service methods
2. Global registration functions
3. RouteGroup methods for registration
4. Individual route registration helper functions

Example generated code for a `ProductService`:

```go
// ProductServiceHandler is the interface for ProductService HTTP handlers
type ProductServiceHandler interface {
	HandleGetProduct(w http.ResponseWriter, r *http.Request)
	HandleListProducts(w http.ResponseWriter, r *http.Request)
	HandleCreateProduct(w http.ResponseWriter, r *http.Request)
	HandleUpdateProduct(w http.ResponseWriter, r *http.Request)
	HandleDeleteProduct(w http.ResponseWriter, r *http.Request)
}

// RegisterProductServiceRoutes registers HTTP routes for ProductService
func RegisterProductServiceRoutes(r Routes, handler ProductServiceHandler) {
	r.HandleFunc("GET", "/products/{product_id}", handler.HandleGetProduct)
	r.HandleFunc("GET", "/products", handler.HandleListProducts)
	r.HandleFunc("POST", "/products", handler.HandleCreateProduct)
	r.HandleFunc("PUT", "/products/{product_id}", handler.HandleUpdateProduct)
	r.HandleFunc("DELETE", "/products/{product_id}", handler.HandleDeleteProduct)
}

// RegisterProductServiceRoutes is a method on RouteGroup to register all ProductService routes
func (g *RouteGroup) RegisterProductServiceRoutes(handler ProductServiceHandler) {
	RegisterProductServiceRoutes(g, handler)
}

// Individual route registration functions...
```

## Middleware Execution Order

Middlewares are executed in the following order:

1. **Global middlewares** (applied to the router)
2. **Group middlewares** (applied to route groups)
3. **Route-specific middlewares** (applied to individual routes)

This order ensures that route-specific concerns are handled first (innermost), followed by group concerns, and finally global concerns (outermost).

## Advanced Usage

### Nested Groups

```go
router := pb.NewRouter()

// Create a base API group
api := router.Group("/api")

// Create versioned API groups
v1 := api.Group("/v1")
v2 := api.Group("/v2")

// Add routes to each version
v1.RegisterGetProduct(productHandlerV1)
v2.RegisterGetProduct(productHandlerV2)
```

### Method Chaining

```go
router := pb.NewRouter()

// Chain method calls
router.Use(Logger()).
       Group("/api").
       Use(RateLimiter(60)).
       RegisterGetProduct(productHandler)
```

### Multiple Middlewares

```go
router := pb.NewRouter()

// Apply multiple middlewares to a route
router.RegisterCreateProduct(
	productHandler,
	Authentication(),
	RateLimiter(30),
	Validation(),
)
```

## Testing

Run tests with:

```bash
go test ./...
```

The package includes comprehensive tests for:
- Basic routing
- Route groups
- Middleware application
- Middleware ordering
- Method chaining
- Duplicate route protection

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.