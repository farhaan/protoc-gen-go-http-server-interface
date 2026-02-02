package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/handler"
	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/pb"
	"github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/service"
)

// setupTestServer creates a test server with all routes registered
func setupTestServer() *httptest.Server {
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)

	router := pb.NewRouter(nil)
	router.RegisterTaskServiceRoutes(taskHandler)

	return httptest.NewServer(router)
}

// TestEditionsE2E_CreateTask tests task creation
func TestEditionsE2E_CreateTask(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a task
	reqBody := `{"title": "Test Task", "description": "Test Description", "project_id": "proj-1"}`
	resp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
	}

	var result pb.CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Task == nil {
		t.Fatal("Expected task in response")
	}
	if result.Task.GetTitle() != "Test Task" {
		t.Errorf("Expected title 'Test Task', got %q", result.Task.GetTitle())
	}
	if result.Task.GetStatus() != pb.TaskStatus_TASK_STATUS_PENDING {
		t.Errorf("Expected status PENDING, got %v", result.Task.GetStatus())
	}
}

// TestEditionsE2E_GetTask tests task retrieval
func TestEditionsE2E_GetTask(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// First create a task
	createReq := `{"title": "Get Test", "description": "Desc", "project_id": "proj-1"}`
	createResp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer createResp.Body.Close()

	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)

	// Get the task
	resp, err := http.Get(server.URL + "/api/v1/tasks/" + created.Task.GetId())
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result pb.GetTaskResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Task.GetId() != created.Task.GetId() {
		t.Errorf("Expected task ID %s, got %s", created.Task.GetId(), result.Task.GetId())
	}
}

// TestEditionsE2E_GetTaskNotFound tests 404 for non-existent task
func TestEditionsE2E_GetTaskNotFound(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks/non-existent-id")
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

// TestEditionsE2E_UpdateTaskPUT tests PUT update
func TestEditionsE2E_UpdateTaskPUT(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a task
	createReq := `{"title": "Original", "description": "Desc", "project_id": "proj-1"}`
	createResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Update via PUT
	updateReq := `{"task": {"title": "Updated Title"}}`
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/v1/tasks/"+created.Task.GetId(), bytes.NewBufferString(updateReq))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result pb.UpdateTaskResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Task.GetTitle() != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got %q", result.Task.GetTitle())
	}
}

// TestEditionsE2E_UpdateTaskPATCH tests PATCH update (additional binding)
func TestEditionsE2E_UpdateTaskPATCH(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a task
	createReq := `{"title": "Original", "description": "Desc", "project_id": "proj-1"}`
	createResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Update via PATCH
	updateReq := `{"task": {"description": "Patched Description"}}`
	req, _ := http.NewRequest(http.MethodPatch, server.URL+"/api/v1/tasks/"+created.Task.GetId(), bytes.NewBufferString(updateReq))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to patch task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result pb.UpdateTaskResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Task.GetDescription() != "Patched Description" {
		t.Errorf("Expected description 'Patched Description', got %q", result.Task.GetDescription())
	}
}

// TestEditionsE2E_DeleteTask tests task deletion
func TestEditionsE2E_DeleteTask(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a task
	createReq := `{"title": "To Delete", "description": "Desc", "project_id": "proj-1"}`
	createResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Delete the task
	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/v1/tasks/"+created.Task.GetId(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify it's deleted
	getResp, _ := http.Get(server.URL + "/api/v1/tasks/" + created.Task.GetId())
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected 404 after deletion, got %d", getResp.StatusCode)
	}
}

// TestEditionsE2E_ListTasks tests listing tasks
func TestEditionsE2E_ListTasks(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create multiple tasks
	for i := 0; i < 3; i++ {
		createReq := `{"title": "Task", "description": "Desc", "project_id": "proj-1"}`
		resp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
		resp.Body.Close()
	}

	// List all tasks
	resp, err := http.Get(server.URL + "/api/v1/tasks")
	if err != nil {
		t.Fatalf("Failed to list tasks: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result pb.ListTasksResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(result.Tasks))
	}
}

// TestEditionsE2E_CompleteTask tests the custom complete action
func TestEditionsE2E_CompleteTask(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a task
	createReq := `{"title": "To Complete", "description": "Desc", "project_id": "proj-1"}`
	createResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Complete the task
	resp, err := http.Post(server.URL+"/api/v1/tasks/"+created.Task.GetId()+"/complete", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to complete task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result pb.CompleteTaskResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Task.GetStatus() != pb.TaskStatus_TASK_STATUS_COMPLETED {
		t.Errorf("Expected status COMPLETED, got %v", result.Task.GetStatus())
	}
}

// TestEditionsE2E_GetTasksByProject tests nested path parameter
func TestEditionsE2E_GetTasksByProject(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create tasks in different projects
	for _, projID := range []string{"proj-1", "proj-1", "proj-2"} {
		createReq := `{"title": "Task", "description": "Desc", "project_id": "` + projID + `"}`
		resp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
		resp.Body.Close()
	}

	// Get tasks for project 1
	resp, err := http.Get(server.URL + "/api/v1/projects/proj-1/tasks")
	if err != nil {
		t.Fatalf("Failed to get tasks by project: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", resp.StatusCode)
	}

	var result pb.GetTasksByProjectResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if len(result.Tasks) != 2 {
		t.Errorf("Expected 2 tasks for proj-1, got %d", len(result.Tasks))
	}
}

// TestEditionsE2E_AssignTask tests multiple path parameters
func TestEditionsE2E_AssignTask(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create a task
	createReq := `{"title": "To Assign", "description": "Desc", "project_id": "proj-1"}`
	createResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Assign the task
	url := server.URL + "/api/v1/projects/proj-1/tasks/" + created.Task.GetId() + "/assign/user-123"
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to assign task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 200, got %d: %s", resp.StatusCode, string(body))
	}

	var result pb.AssignTaskResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Task.GetAssigneeId() != "user-123" {
		t.Errorf("Expected assignee_id 'user-123', got %q", result.Task.GetAssigneeId())
	}
}

// TestEditionsE2E_MiddlewareChain tests middleware is applied
func TestEditionsE2E_MiddlewareChain(t *testing.T) {
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)

	middlewareCalled := false
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareCalled = true
			next.ServeHTTP(w, r)
		})
	}

	router := pb.NewRouter(nil)
	router.Use(testMiddleware)
	router.RegisterTaskServiceRoutes(taskHandler)

	server := httptest.NewServer(router)
	defer server.Close()

	// Make a request
	resp, _ := http.Get(server.URL + "/api/v1/tasks")
	resp.Body.Close()

	if !middlewareCalled {
		t.Error("Middleware was not called")
	}
}

// TestEditionsE2E_RouteSpecificMiddleware tests route-specific middleware
func TestEditionsE2E_RouteSpecificMiddleware(t *testing.T) {
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)

	createCalled := false
	createMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			createCalled = true
			next.ServeHTTP(w, r)
		})
	}

	router := pb.NewRouter(nil)
	router.RegisterCreateTask(taskHandler, createMiddleware)
	router.RegisterListTasks(taskHandler) // No middleware

	server := httptest.NewServer(router)
	defer server.Close()

	// Call create - middleware should trigger
	reqBody := `{"title": "Test"}`
	resp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(reqBody))
	resp.Body.Close()

	if !createCalled {
		t.Error("Create middleware was not called")
	}
}

// TestEditionsE2E_RouteGroup tests route grouping with prefix
func TestEditionsE2E_RouteGroup(t *testing.T) {
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)

	router := pb.NewRouter(nil)
	v2 := router.Group("/v2")
	v2.RegisterTaskServiceRoutes(taskHandler)

	server := httptest.NewServer(router)
	defer server.Close()

	// Create task via v2 prefix
	reqBody := `{"title": "V2 Task"}`
	resp, err := http.Post(server.URL+"/v2/api/v1/tasks", "application/json", bytes.NewBufferString(reqBody))
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 201, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestEditionsE2E_SharedServeMux tests multiple handlers on shared mux
func TestEditionsE2E_SharedServeMux(t *testing.T) {
	sharedMux := http.NewServeMux()

	// Task service
	taskService := service.NewTaskService()
	taskHandler := handler.NewTaskHandler(taskService)
	taskRouter := pb.NewRouter(sharedMux)
	taskRouter.RegisterTaskServiceRoutes(taskHandler)

	// Add a custom health endpoint
	sharedMux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	server := httptest.NewServer(sharedMux)
	defer server.Close()

	// Test task endpoint works
	reqBody := `{"title": "Test"}`
	taskResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(reqBody))
	if taskResp.StatusCode != http.StatusCreated {
		t.Errorf("Task creation failed: %d", taskResp.StatusCode)
	}
	taskResp.Body.Close()

	// Test health endpoint works
	healthResp, _ := http.Get(server.URL + "/health")
	if healthResp.StatusCode != http.StatusOK {
		t.Errorf("Health check failed: %d", healthResp.StatusCode)
	}
	healthResp.Body.Close()
}

// TestEditionsE2E_FullCRUDWorkflow tests complete CRUD workflow
func TestEditionsE2E_FullCRUDWorkflow(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// 1. CREATE
	createReq := `{"title": "CRUD Test", "description": "Full workflow", "project_id": "proj-crud"}`
	createResp, _ := http.Post(server.URL+"/api/v1/tasks", "application/json", bytes.NewBufferString(createReq))
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("CREATE failed: %d", createResp.StatusCode)
	}
	var created pb.CreateTaskResponse
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()
	taskID := created.Task.GetId()
	t.Logf("Created task: %s", taskID)

	// 2. READ
	getResp, _ := http.Get(server.URL + "/api/v1/tasks/" + taskID)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("READ failed: %d", getResp.StatusCode)
	}
	getResp.Body.Close()
	t.Log("Read task successfully")

	// 3. UPDATE (PUT)
	updateReq := `{"task": {"title": "Updated CRUD"}}`
	putReq, _ := http.NewRequest(http.MethodPut, server.URL+"/api/v1/tasks/"+taskID, bytes.NewBufferString(updateReq))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, _ := http.DefaultClient.Do(putReq)
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("UPDATE (PUT) failed: %d", putResp.StatusCode)
	}
	putResp.Body.Close()
	t.Log("Updated task via PUT")

	// 4. UPDATE (PATCH)
	patchReq := `{"task": {"description": "Patched"}}`
	patchHttpReq, _ := http.NewRequest(http.MethodPatch, server.URL+"/api/v1/tasks/"+taskID, bytes.NewBufferString(patchReq))
	patchHttpReq.Header.Set("Content-Type", "application/json")
	patchResp, _ := http.DefaultClient.Do(patchHttpReq)
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("UPDATE (PATCH) failed: %d", patchResp.StatusCode)
	}
	patchResp.Body.Close()
	t.Log("Updated task via PATCH")

	// 5. CUSTOM ACTION (Complete)
	completeResp, _ := http.Post(server.URL+"/api/v1/tasks/"+taskID+"/complete", "application/json", nil)
	if completeResp.StatusCode != http.StatusOK {
		t.Fatalf("COMPLETE failed: %d", completeResp.StatusCode)
	}
	var completed pb.CompleteTaskResponse
	json.NewDecoder(completeResp.Body).Decode(&completed)
	completeResp.Body.Close()
	if completed.Task.GetStatus() != pb.TaskStatus_TASK_STATUS_COMPLETED {
		t.Fatalf("Task not completed: %v", completed.Task.GetStatus())
	}
	t.Log("Completed task")

	// 6. LIST
	listResp, _ := http.Get(server.URL + "/api/v1/tasks")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("LIST failed: %d", listResp.StatusCode)
	}
	var listed pb.ListTasksResponse
	json.NewDecoder(listResp.Body).Decode(&listed)
	listResp.Body.Close()
	if len(listed.Tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(listed.Tasks))
	}
	t.Log("Listed tasks")

	// 7. DELETE
	delReq, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/v1/tasks/"+taskID, nil)
	delResp, _ := http.DefaultClient.Do(delReq)
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE failed: %d", delResp.StatusCode)
	}
	delResp.Body.Close()
	t.Log("Deleted task")

	// 8. Verify deletion
	verifyResp, _ := http.Get(server.URL + "/api/v1/tasks/" + taskID)
	if verifyResp.StatusCode != http.StatusNotFound {
		t.Fatalf("Expected 404 after delete, got %d", verifyResp.StatusCode)
	}
	verifyResp.Body.Close()
	t.Log("Verified deletion")

	t.Log("Full CRUD workflow completed successfully!")
}
