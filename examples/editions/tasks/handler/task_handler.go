package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/pb"
	"github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/service"
)

// TaskHandler implements pb.TaskServiceHandler
type TaskHandler struct {
	svc *service.TaskService
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(svc *service.TaskService) *TaskHandler {
	return &TaskHandler{svc: svc}
}

// HandleCreateTask handles POST /api/v1/tasks
func (h *TaskHandler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req pb.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.svc.CreateTask(req.Title, req.Description, req.ProjectId)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.CreateTaskResponse{Task: task}, http.StatusCreated)
}

// HandleGetTask handles GET /api/v1/tasks/{task_id}
func (h *TaskHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	if taskID == "" {
		writeError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	task, err := h.svc.GetTask(taskID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, "task not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.GetTaskResponse{Task: task}, http.StatusOK)
}

// HandleUpdateTask handles PUT/PATCH /api/v1/tasks/{task_id}
func (h *TaskHandler) HandleUpdateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	if taskID == "" {
		writeError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	var req pb.UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	task, err := h.svc.UpdateTask(taskID, req.Task)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, "task not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.UpdateTaskResponse{Task: task}, http.StatusOK)
}

// HandleDeleteTask handles DELETE /api/v1/tasks/{task_id}
func (h *TaskHandler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	if taskID == "" {
		writeError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	err := h.svc.DeleteTask(taskID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, "task not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.DeleteTaskResponse{Success: true}, http.StatusOK)
}

// HandleListTasks handles GET /api/v1/tasks
func (h *TaskHandler) HandleListTasks(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")

	tasks, err := h.svc.ListTasks(projectID)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.ListTasksResponse{Tasks: tasks}, http.StatusOK)
}

// HandleCompleteTask handles POST /api/v1/tasks/{task_id}/complete
func (h *TaskHandler) HandleCompleteTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("task_id")
	if taskID == "" {
		writeError(w, "task_id is required", http.StatusBadRequest)
		return
	}

	task, err := h.svc.CompleteTask(taskID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, "task not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.CompleteTaskResponse{Task: task}, http.StatusOK)
}

// HandleGetTasksByProject handles GET /api/v1/projects/{project_id}/tasks
func (h *TaskHandler) HandleGetTasksByProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	if projectID == "" {
		writeError(w, "project_id is required", http.StatusBadRequest)
		return
	}

	tasks, err := h.svc.GetTasksByProject(projectID)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.GetTasksByProjectResponse{Tasks: tasks}, http.StatusOK)
}

// HandleAssignTask handles POST /api/v1/projects/{project_id}/tasks/{task_id}/assign/{user_id}
func (h *TaskHandler) HandleAssignTask(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("project_id")
	taskID := r.PathValue("task_id")
	userID := r.PathValue("user_id")

	if projectID == "" || taskID == "" || userID == "" {
		writeError(w, "project_id, task_id, and user_id are required", http.StatusBadRequest)
		return
	}

	task, err := h.svc.AssignTask(projectID, taskID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeError(w, "task not found", http.StatusNotFound)
			return
		}
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, &pb.AssignTaskResponse{Task: task}, http.StatusOK)
}

// Helper functions
func writeJSON(w http.ResponseWriter, v interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
