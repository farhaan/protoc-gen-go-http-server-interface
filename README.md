# protoc-gen-go-http-server-interface

A Protocol Buffer code generator plugin that produces HTTP server interfaces and middleware support from Protocol Buffer service definitions.

## System Requirements

- **Go**: Go 1.23 or higher
- **Protocol Buffers**: protoc 3.14.0 or higher
- **Google API Proto Files**: Required for HTTP annotations
  - Install with: `go get -u google.golang.org/genproto/googleapis/api/annotations`
- **Go Protobuf Plugin**: Required for Go code generation
  - Install with: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`



## Overview

A protocol buffer compiler plugin that generates HTTP server interfaces from proto files. This plugin allows you to define your API using Protocol Buffers and automatically generate Go HTTP server code with middleware support, route grouping, and flexible path handling.

### Key Features

- **Strongly Typed Handler Interfaces**: Generated interfaces match your Protocol Buffer service definitions
- **Middleware Support**: Global, group-level, and route-specific middleware
- **Path Prefixing**: Easy API versioning with route groups
- **Method Chaining**: Fluent API for intuitive route registration

## Design Philosophy

The design follows a domain-driven approach with these principles:

2. **Type safety**: Generated interfaces ensure compile-time checks for handler implementations 
3. **Familiar patterns**: API inspired by popular Go HTTP frameworks (Chi, Echo, Gin)
4. **Middleware chain clarity**: Clear execution order (global → group → route-specific)
5. **(Almost) No external dependencies**: Uses only the Go standard library

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
	router.RegisterListProducts(productHandler, pb.Middleware(RateLimiter(30)))
	
	// Group routes with middleware
	authRoutes := router.Group("/admin").Use(pb.Middleware(Authentication()))
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
	routes          []routeDef
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
router.Use(pb.Middleware(Logger())).
       Group("/api").
       Use(pb.Middleware(RateLimiter(60))).
       RegisterGetProduct(productHandler)
```

### Shared ServeMux Support
A key feature of this plugin is the ability to use multiple services with a single HTTP server. This allows you to:

- Register routes from multiple proto-generated services onto a single HTTP server
- Apply common middleware across all services
- Organize routes with prefixes to avoid conflicts
- Leverage Go 1.22's advanced routing features (method+path specificity)

```go
// Create a shared ServeMux
sharedMux := http.NewServeMux()

// Create routers sharing the mux
productRouter := productPb.NewRouter(sharedMux)
userRouter := userPb.NewRouter(sharedMux)

// Apply global middleware to each router
productRouter.Use(productPb.Middleware(Logger()))
userRouter.Use(userPb.Middleware(Logger()))
// Create service and handler instances
productService := productSvc.NewProductService()
productHandler := productHandler.NewProductHandler(productService)

userService := userSvc.NewUserService()
userHandler := userHandler.NewUserHandler(userService)

// Add path prefixes to avoid conflicts
productsApi := productRouter.Group("/api/products")
productsApi.RegisterProductServiceRoutes(productHandler)

usersApi := userRouter.Group("/api/users")
usersApi.RegisterUserServiceRoutes(userHandler)

// Start the server
http.ListenAndServe(":8080", sharedMux)

```

### Avoiding Route Conflicts
When using a shared ServeMux with multiple services, you may need to handle route conflicts. There are several approaches:

#### Use distinct path prefixes:
```go
productsApi := productRouter.Group("/api/products")
usersApi := userRouter.Group("/api/users")
```

#### Use API versioning:
```go
v1Products := productRouter.Group("/api/v1/products")
v2Products := productRouter.Group("/api/v2/products")
```

#### Selectively register routes:
```go
// Register only specific methods instead of all routes
router.RegisterGetUser(userHandler)
router.RegisterListUsers(userHandler)
```

Go 1.22's ServeMux will detect conflicts at registration time, so you'll get an immediate panic if any route conflicts exist.

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


## protoc-gen-go-http-server-interface Options

The protoc-gen-go-http-server-interface plugin supports several command-line options to customize its behavior.

### Option Format

Options are passed to the plugin using the `opt` parameter in your `protoc` command or configuration files like `buf.gen.yaml`.

For example:
```yaml
# In buf.gen.yaml
plugins:
  - local: protoc-gen-go-http-server-interface
    out: pb
    opt: paths=source_relative,output_prefix=api
```

Or with protoc directly:
```bash
protoc --go-http-server-interface_out=paths=source_relative,output_prefix=api:./out proto/service.proto
```

### Available Options

| Option | Description | Default |
|--------|-------------|---------|
| `paths` | File path resolution mode. Set to `source_relative` to match the directory structure of the input .proto files, or `import` to use go import paths. | `import` |
| `output_prefix` | Customize the prefix of the generated files. For example, if set to `api`, a file named `service.proto` will generate `api_service.pb.go` instead of `service_http.pb.go`. | (none) |

### Example Usage

#### Using source-relative paths

This is particularly useful when you want to keep the generated code next to your proto files, maintaining the same directory structure.

```yaml
plugins:
  - local: protoc-gen-go-http-server-interface
    out: .
    opt: paths=source_relative
```

With this option, if you have a proto file at `api/v1/service.proto`, the generated code will be at `api/v1/service_http.pb.go`.

#### Custom output prefix

If you want to give your generated files a consistent prefix:

```yaml
plugins:
  - local: protoc-gen-go-http-server-interface
    out: pb
    opt: output_prefix=api
```

This will generate files like `api_service.pb.go` instead of the default `service_http.pb.go`.

#### Combining options

Options can be combined by separating them with commas:

```yaml
plugins:
  - local: protoc-gen-go-http-server-interface
    out: .
    opt: paths=source_relative,output_prefix=api
```

This will:
1. Place generated files in the same directory structure as source files
2. Use the prefix `api_` for all generated files


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
