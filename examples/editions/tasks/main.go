package main

import (
	"log"
	"net/http"
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
	// Create service and handler
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)

	// Create router with logging middleware
	router := pb.NewRouter(nil)
	router.Use(Logger())

	// Register all TaskService routes
	router.RegisterTaskServiceRoutes(taskHandler)

	// Print registered routes
	log.Println("Registered routes:")
	for _, route := range router.GetRoutes() {
		log.Printf("  %s", route)
	}

	// Start server
	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
