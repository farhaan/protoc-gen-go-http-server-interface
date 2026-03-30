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

	"github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/handler"
	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/pb"
	"github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/service"
)

// Logger middleware logs requests
func Logger() pb.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
		})
	}
}

func main() {
	// ready is set to 1 once all routes are registered and the server is
	// accepting traffic. Kubernetes readiness probe checks this.
	var ready atomic.Bool

	// Create service and handler
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)

	// Create router with logging middleware
	router := pb.NewRouter(nil)
	router.Use(Logger())

	// Register all TaskService routes
	if err := pb.RegisterTaskServiceRoutes(router, taskHandler); err != nil {
		log.Fatal(err)
	}

	// Kubernetes health probes — registered directly on the underlying mux so
	// they bypass application middleware (no auth, no logging overhead).
	mux := router.Mux()
	// Liveness: process is alive and not deadlocked.
	mux.HandleFunc("GET /healthz/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	// Readiness: service is fully initialised and ready to serve traffic.
	// Replace the body with real dependency checks (DB ping, cache warmup, etc.).
	mux.HandleFunc("GET /healthz/ready", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	// Startup: used by Kubernetes to give slow-starting containers extra time
	// before liveness checks kick in. Returns 200 once the server is up.
	mux.HandleFunc("GET /healthz/startup", func(w http.ResponseWriter, r *http.Request) {
		if !ready.Load() {
			http.Error(w, "starting", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Print registered routes
	log.Println("Registered routes:")
	for _, route := range router.GetRoutes() {
		log.Printf("  %s", route)
	}

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background so we can listen for shutdown signals.
	go func() {
		log.Println("Starting server on :8080")
		ready.Store(true) // signal readiness before accepting connections
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	// Block until SIGINT or SIGTERM received.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Give in-flight requests up to 30 seconds to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown: %v", err)
	}
	log.Println("Server stopped")
}
