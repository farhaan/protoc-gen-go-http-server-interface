# Example Project

This guide walks through setting up a complete project with the HTTP Interface Generator.

## Project Setup

First, create your project structure:

```bash
mkdir -p myproject/{cmd/server,proto,service}
cd myproject
go mod init myproject
```

## Add Dependencies

```bash
# Install required dependencies
go get google.golang.org/protobuf/cmd/protoc-gen-go
go get github.com/farhaan/protoc-gen-httpinterface

# Get Google API protos for HTTP annotations
mkdir -p third_party
git clone --depth 1 https://github.com/googleapis/googleapis.git third_party/googleapis
```

## Create Proto File

Create `proto/product.proto`:

```protobuf
syntax = "proto3";

package api;

option go_package = "myproject/proto/gen";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";

// Product represents a product in the catalog
message Product {
  string id = 1;
  string name = 2;
  string description = 3;
  double price = 4;
  int32 stock = 5;
}

// GetProductRequest is used to fetch a product by ID
message GetProductRequest {
  string product_id = 1;
}

// ListProductsRequest is used to list products with pagination
message ListProductsRequest {
  int32 page = 1;
  int32 page_size = 2;
  string search_query = 3;
}

// ListProductsResponse contains a list of products
message ListProductsResponse {
  repeated Product products = 1;
  int32 total_count = 2;
}

// ProductService provides methods to manage products
service ProductService {
  // GetProduct retrieves a product by ID
  rpc GetProduct(GetProductRequest) returns (Product) {
    option (google.api.http) = {
      get: "/products/{product_id}"
    };
  }
  
  // ListProducts lists products with filtering and pagination
  rpc ListProducts(ListProductsRequest) returns (ListProductsResponse) {
    option (google.api.http) = {
      get: "/products"
    };
  }
  
  // CreateProduct creates a new product
  rpc CreateProduct(Product) returns (Product) {
    option (google.api.http) = {
      post: "/products"
      body: "*"
    };
  }
  
  // UpdateProduct updates an existing product
  rpc UpdateProduct(Product) returns (Product) {
    option (google.api.http) = {
      put: "/products/{id}"
      body: "*"
    };
  }
  
  // DeleteProduct deletes a product
  rpc DeleteProduct(GetProductRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/products/{product_id}"
    };
  }
  
  // Health check endpoint
  rpc Liveness(google.protobuf.Empty) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      get: "/liveness"
    };
  }
}
```

## Generate Code

Create a Makefile for easier code generation:

```makefile
# Makefile

.PHONY: proto

proto:
	mkdir -p proto/gen
	protoc --proto_path=./proto \
		--proto_path=./third_party/googleapis \
		--go_out=./proto/gen \
		--go_opt=paths=source_relative \
		--httpinterface_out=./proto/gen \
		--httpinterface_opt=paths=source_relative \
		./proto/*.proto
```

Run it:

```bash
make proto
```

or, you can use `buf` (https://github.com/bufbuild/buf) instead and run the following command:
```bash
buf generate --include-imports --config buf.yaml
```

## Implement Service

Create `service/product_service.go`:

```go
package service

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "myproject/proto/gen"
)

// ProductService implements the ProductService defined in the proto
type ProductService struct {
	mu       sync.RWMutex
	products map[string]*pb.Product
}

// NewProductService creates a new product service
func NewProductService() *ProductService {
	return &ProductService{
		products: make(map[string]*pb.Product),
	}
}

// GetProduct retrieves a product by ID
func (s *ProductService) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, ok := s.products[req.ProductId]
	if !ok {
		return nil, errors.New("product not found")
	}

	// Return a copy of the product
	return &pb.Product{
		Id:          product.Id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Stock:       product.Stock,
	}, nil
}

// Implement other service methods...
```

## Implement HTTP Handlers

Create `service/product_handler.go`:

```go
package service

import (
	"encoding/json"
	"net/http"
	"strconv"

	"google.golang.org/protobuf/types/known/emptypb"

	pb "myproject/proto/gen"
)

// ProductHandler implements the HTTP handlers for the Product service
type ProductHandler struct {
	service *ProductService
}

// NewProductHandler creates a new product handler
func NewProductHandler(service *ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

// HandleGetProduct handles GET /products/{product_id}
func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from path
	productID := r.PathValue("product_id")
	if productID == "" {
		http.Error(w, "Missing product ID", http.StatusBadRequest)
		return
	}

	// Call service
	product, err := h.service.GetProduct(r.Context(), &pb.GetProductRequest{
		ProductId: productID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// Implement other handler methods...
```

## Create Server

Create `cmd/server/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "myproject/proto/gen"
	"myproject/service"
)

func main() {
	// Parse command line flags
	port := flag.Int("port", 8080, "HTTP server port")
	flag.Parse()

	// Create the router
	router := pb.NewRouter()

	// Create service and handler
	productService := service.NewProductService()
	productHandler := service.NewProductHandler(productService)

	// Register handler
	router.RegisterProductService(productHandler)

	// Start the server
	log.Printf("Starting HTTP server on port %d", *port)
	
	// Start in a goroutine to allow for graceful shutdown
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: router,
	}
	
	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
	
	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server...")
	
	// Gracefully shutdown the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}
	
	log.Println("Server stopped")
}
```

## Run the Server

```bash
go run cmd/server/main.go
```

## Test the API

```bash
# Create a product
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Test Product","description":"A test product","price":99.99,"stock":100}'

# Get a product
curl http://localhost:8080/products/123

# List products
curl http://localhost:8080/products

# Health check
curl http://localhost:8080/liveness
```

## Complete Project

Your final project structure should look like:

```
myproject/
├── main.go
├── proto/
│   └── product.proto
├── pb/
│   ├── product.pb.go
│   ├── product_grpc.pb.go
│   └── product_http.pb.go
├── service/
│   └── product
│		└── service.go
├── handler/
│   └── handler.go
├── go.mod
└── go.sum
```