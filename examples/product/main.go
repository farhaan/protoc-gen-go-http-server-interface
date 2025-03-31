package main

import (
	"log"
	"net/http"
	"time"

	productHandler "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/handler/product"
	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/pb"
	productSvc "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/service/product"
)

// Logger middleware logs request details
func Logger() pb.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

// Auth middleware checks for authentication
func Auth() pb.Middleware {
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

// RateLimiter middleware limits request rates
func RateLimiter(requestsPerMinute int) pb.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Rate limiting: %d/min", requestsPerMinute)
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	// Create the router
	router := pb.NewRouter()

	// Apply global middleware
	router.Use(Logger())

	// Create service and handler
	productService := productSvc.NewProductService()
	productHandler := productHandler.NewProductHandler(productService)

	// APPROACH 1: Register individual routes with middleware
	router.RegisterGetProduct(productHandler)
	router.RegisterListProducts(productHandler, RateLimiter(30))
	router.RegisterLiveness(productHandler)

	// APPROACH 2: Group routes with middleware
	authRoutes := router.Group("/").Use(Auth())
	authRoutes.RegisterCreateProduct(productHandler)
	authRoutes.RegisterUpdateProduct(productHandler)
	authRoutes.RegisterDeleteProduct(productHandler)

	// APPROACH 3: Group routes with path prefix and middleware
	apiV1 := router.Group("/api/v1")
	apiV1.RegisterGetProduct(productHandler)
	apiV1.RegisterListProducts(productHandler, RateLimiter(30))

	// APPROACH 4: Register all routes at once
	// router.RegisterProductServiceRoutes(productHandler)

	// Start the server
	log.Println("Starting HTTP server on :8080")
	log.Println("Routes available:")
	log.Fatal(http.ListenAndServe(":8080", router))
}
