package gorilla

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/pb"
	"github.com/gorilla/mux"
)

// MockTaskHandler implements pb.TaskServiceHandler for testing
// Note: Uses mux.Vars instead of r.PathValue for path parameters
type MockTaskHandler struct{}

func (h *MockTaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (h *MockTaskHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	// Gorilla uses mux.Vars instead of r.PathValue
	vars := mux.Vars(r)
	taskID := vars["task_id"]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "title": "Test Task"})
}

func (h *MockTaskHandler) HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["task_id"]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "status": "updated"})
}

func (h *MockTaskHandler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (h *MockTaskHandler) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]string{"tasks": {"task1", "task2"}})
}

func (h *MockTaskHandler) HandleCompleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["task_id"]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "status": "completed"})
}

func (h *MockTaskHandler) HandleGetTasksByProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectID := vars["project_id"]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"project_id": projectID})
}

func (h *MockTaskHandler) HandleAssignTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["task_id"]
	userID := vars["user_id"]
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"task_id": taskID, "user_id": userID})
}

func TestRegisterTaskServiceRoutes_Gorilla(t *testing.T) {
	routes := New(nil)
	handler := &MockTaskHandler{}

	// Register all routes using generated function
	err := pb.RegisterTaskServiceRoutes(routes, handler)
	if err != nil {
		t.Fatalf("RegisterTaskServiceRoutes failed: %v", err)
	}

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET task by ID",
			method:         "GET",
			path:           "/api/v1/tasks/123",
			expectedStatus: http.StatusOK,
			expectedBody:   `"task_id":"123"`,
		},
		{
			name:           "POST create task",
			method:         "POST",
			path:           "/api/v1/tasks",
			expectedStatus: http.StatusCreated,
			expectedBody:   `"status":"created"`,
		},
		{
			name:           "GET list tasks",
			method:         "GET",
			path:           "/api/v1/tasks",
			expectedStatus: http.StatusOK,
			expectedBody:   `"tasks"`,
		},
		{
			name:           "PUT update task",
			method:         "PUT",
			path:           "/api/v1/tasks/456",
			expectedStatus: http.StatusOK,
			expectedBody:   `"task_id":"456"`,
		},
		{
			name:           "DELETE task",
			method:         "DELETE",
			path:           "/api/v1/tasks/789",
			expectedStatus: http.StatusOK,
			expectedBody:   `"success":true`,
		},
		{
			name:           "POST complete task",
			method:         "POST",
			path:           "/api/v1/tasks/123/complete",
			expectedStatus: http.StatusOK,
			expectedBody:   `"status":"completed"`,
		},
		{
			name:           "GET tasks by project",
			method:         "GET",
			path:           "/api/v1/projects/proj1/tasks",
			expectedStatus: http.StatusOK,
			expectedBody:   `"project_id":"proj1"`,
		},
		{
			name:           "POST assign task",
			method:         "POST",
			path:           "/api/v1/projects/proj1/tasks/task1/assign/user1",
			expectedStatus: http.StatusOK,
			expectedBody:   `"user_id":"user1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			routes.Router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			if tt.expectedBody != "" && !strings.Contains(rec.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestRegisterSingleRoute_WithMiddleware_Gorilla(t *testing.T) {
	routes := New(nil)
	handler := &MockTaskHandler{}

	middlewareCalled := false
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			w.Header().Set("X-Auth", "verified")
			next.ServeHTTP(w, r)
		})
	}

	// Register single route with middleware using generated function
	err := pb.RegisterGetTaskRoute(routes, handler, authMiddleware)
	if err != nil {
		t.Fatalf("RegisterGetTaskRoute failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/tasks/123", nil)
	rec := httptest.NewRecorder()
	routes.Router.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("middleware was not called")
	}
	if rec.Header().Get("X-Auth") != "verified" {
		t.Error("middleware header not set")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestGorilla_RouterMiddleware(t *testing.T) {
	r := mux.NewRouter()

	// Add middleware at router level (gorilla's native middleware)
	middlewareCalled := false
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	})

	routes := New(r)
	handler := &MockTaskHandler{}

	err := pb.RegisterTaskServiceRoutes(routes, handler)
	if err != nil {
		t.Fatalf("RegisterTaskServiceRoutes failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/tasks", nil)
	rec := httptest.NewRecorder()
	routes.Router.ServeHTTP(rec, req)

	if !middlewareCalled {
		t.Error("gorilla router middleware was not called")
	}
}
