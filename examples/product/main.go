package main

import (
	"log"
	"net/http"
	"time"

	productHandler "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/handler/product"
	productPb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/pb/product"
	productSvc "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/service/product"

	userHandler "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/handler/user"
	userPb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/pb/user"
	userSvc "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/service/user"
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
	productRouter.RegisterProductServiceRoutes(productHandler)

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
	productsApi.RegisterProductServiceRoutes(productHandler)

	usersApi := userRouter.Group("/api/users")
	usersApi.RegisterUserServiceRoutes(userHandler)

	// OPTION 3: Use API versioning with more specific paths
	// These won't conflict with the previous routes because they're more specific
	v1Products := productRouter.Group("/api/v1/products").Use(productPb.Middleware(Auth()))
	v1Products.RegisterGetProduct(productHandler)
	v1Products.RegisterListProducts(productHandler, productPb.Middleware(RateLimiter(60)))
	v1Products.RegisterCreateProduct(productHandler)

	v1Users := userRouter.Group("/api/v1/users").Use(userPb.Middleware(Auth()))
	v1Users.RegisterGetUser(userHandler)
	v1Users.RegisterListUsers(userHandler, userPb.Middleware(RateLimiter(30)))
	v1Users.RegisterCreateUser(userHandler)

	// Add some paths that would normally conflict, but don't because of method+path specificity
	productRouter.HandleFunc("GET", "/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Products service is healthy"))
	})

	userRouter.HandleFunc("POST", "/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Users service received health check"))
	})

	// Start the server
	log.Println("Starting HTTP server on :8080")
	log.Fatal(http.ListenAndServe(":8080", sharedMux))
}
