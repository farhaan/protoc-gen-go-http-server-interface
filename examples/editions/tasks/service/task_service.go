package service

import (
	"errors"
	"sync"
	"time"

	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/editions/tasks/pb"
)

var (
	ErrNotFound = errors.New("task not found")
)

// TaskService provides business logic for tasks
type TaskService struct {
	mu    sync.RWMutex
	tasks map[string]*pb.Task
	idSeq int
}

// NewTaskService creates a new TaskService
func NewTaskService() *TaskService {
	return &TaskService{
		tasks: make(map[string]*pb.Task),
		idSeq: 0,
	}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(title, description, projectID string) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.idSeq++
	now := time.Now().Unix()

	task := &pb.Task{
		Id:          intToID(s.idSeq),
		Title:       title,
		Description: description,
		ProjectId:   projectID,
		Status:      pb.TaskStatus_TASK_STATUS_PENDING,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tasks[task.Id] = task
	return task, nil
}

// GetTask retrieves a task by ID
func (s *TaskService) GetTask(taskID string) (*pb.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrNotFound
	}
	return task, nil
}

// UpdateTask updates an existing task
func (s *TaskService) UpdateTask(taskID string, update *pb.Task) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrNotFound
	}

	if update != nil {
		if update.Title != "" {
			task.Title = update.Title
		}
		if update.Description != "" {
			task.Description = update.Description
		}
		if update.Status != pb.TaskStatus_TASK_STATUS_UNSPECIFIED {
			task.Status = update.Status
		}
	}
	task.UpdatedAt = time.Now().Unix()

	return task, nil
}

// DeleteTask deletes a task
func (s *TaskService) DeleteTask(taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[taskID]; !ok {
		return ErrNotFound
	}
	delete(s.tasks, taskID)
	return nil
}

// ListTasks lists all tasks, optionally filtered by project
func (s *TaskService) ListTasks(projectID string) ([]*pb.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*pb.Task
	for _, task := range s.tasks {
		if projectID == "" || task.ProjectId == projectID {
			result = append(result, task)
		}
	}
	return result, nil
}

// CompleteTask marks a task as complete
func (s *TaskService) CompleteTask(taskID string) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrNotFound
	}

	task.Status = pb.TaskStatus_TASK_STATUS_COMPLETED
	task.UpdatedAt = time.Now().Unix()
	return task, nil
}

// GetTasksByProject returns all tasks for a project
func (s *TaskService) GetTasksByProject(projectID string) ([]*pb.Task, error) {
	return s.ListTasks(projectID)
}

// AssignTask assigns a task to a user
func (s *TaskService) AssignTask(projectID, taskID, userID string) (*pb.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, ErrNotFound
	}

	if task.ProjectId != projectID {
		return nil, errors.New("task does not belong to project")
	}

	task.AssigneeId = userID
	task.UpdatedAt = time.Now().Unix()
	return task, nil
}

func intToID(n int) string {
	return "task-" + intToStr(n)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
