package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	productHandler "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/handler/product"
	productPb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/pb/product"
	productSvc "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/service/product"

	userHandler "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/handler/user"
	userPb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/pb/user"
	userSvc "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/service/user"
)

// Common middleware types
type Middleware func(http.Handler) http.Handler

// Logger middleware logs request details
func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
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

// RateLimiter middleware limits request rates
func RateLimiter(requestsPerMinute int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Rate limiting: %d/min", requestsPerMinute)
			next.ServeHTTP(w, r)
		})
	}
}

func main() {
	var ready atomic.Bool

	// Create a shared ServeMux
	sharedMux := http.NewServeMux()

	// Create separate routers that share the same mux
	productRouter := productPb.NewRouter(sharedMux)
	userRouter := userPb.NewRouter(sharedMux)

	// Apply global middleware to each router
	productRouter.Use(productPb.Middleware(Logger()))
	userRouter.Use(userPb.Middleware(Logger()))

	// Create service and handler instances
	productService := productSvc.NewProductService()
	productHandler := productHandler.NewProductHandler(productService)
	_ = productPb.RegisterProductServiceRoutes(productRouter, productHandler)

	userService := userSvc.NewUserService()
	userHandler := userHandler.NewUserHandler(userService)

	// OPTION 1: Register routes at the root level
	// This might cause conflicts since both services have a "/liveness" endpoint
	// Uncomment these lines to see what happens (likely a panic due to route conflicts)
	/*
		productRouter.RegisterProductServiceRoutes(productHandler)
		userRouter.RegisterUserServiceRoutes(userHandler)
	*/

	// OPTION 2: Add path prefixes to completely avoid conflicts
	// Create service-specific groups with different prefixes
	productsApi := productRouter.Group("/api/products")
	_ = productPb.RegisterProductServiceRoutes(productsApi, productHandler)

	usersApi := userRouter.Group("/api/users")
	_ = userPb.RegisterUserServiceRoutes(usersApi, userHandler)

	// OPTION 3: Use API versioning with more specific paths
	// These won't conflict with the previous routes because they're more specific
	v1Products := productRouter.Group("/api/v1/products").Use(productPb.Middleware(Auth()))
	_ = productPb.RegisterGetProductRoute(v1Products, productHandler)
	_ = productPb.RegisterListProductsRoute(v1Products, productHandler, productPb.Middleware(RateLimiter(60)))
	_ = productPb.RegisterCreateProductRoute(v1Products, productHandler)

	v1Users := userRouter.Group("/api/v1/users").Use(userPb.Middleware(Auth()))
	_ = userPb.RegisterGetUserRoute(v1Users, userHandler)
	_ = userPb.RegisterListUsersRoute(v1Users, userHandler, userPb.Middleware(RateLimiter(30)))
	_ = userPb.RegisterCreateUserRoute(v1Users, userHandler)

	// Kubernetes health probes registered directly on the shared mux so they
	// are reachable regardless of which service router handles a given prefix.
	sharedMux.HandleFunc("GET /healthz/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	sharedMux.HandleFunc("GET /healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	sharedMux.HandleFunc("GET /healthz/startup", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      sharedMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Starting HTTP server on :8080")
		ready.Store(true)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown: %v", err)
	}
	log.Println("Server stopped")
}
