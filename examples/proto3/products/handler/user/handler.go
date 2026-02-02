package userhandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/pb/user" // Your generated proto package
	service "github.com/farhaan/protoc-gen-go-http-server-interface/examples/proto3/products/service/user"
)

// UserHandler implements the HTTP handlers for the User service
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(service *service.UserService) pb.UserServiceHandler {
	return &UserHandler{
		service: service,
	}
}

// HandleGetUser handles GET /users/{user_id}
func (h *UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	userId := r.PathValue("user_id")
	if userId == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// Call service
	user, err := h.service.GetUser(r.Context(), &pb.GetUserRequest{
		UserId: userId,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	writeJSONResponse(w, user)
}

// HandleListUsers handles GET /users
func (h *UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	var page, pageSize int32 = 1, 10
	var statusFilter pb.UserStatus = pb.UserStatus_USER_STATUS_UNSPECIFIED

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		pageInt, err := strconv.Atoi(pageStr)
		if err == nil && pageInt > 0 {
			page = int32(pageInt)
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		pageSizeInt, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSizeInt > 0 {
			pageSize = int32(pageSizeInt)
		}
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		statusInt, err := strconv.Atoi(statusStr)
		if err == nil && statusInt >= 0 {
			statusFilter = pb.UserStatus(statusInt)
		}
	}

	searchQuery := r.URL.Query().Get("q")

	// Call service
	response, err := h.service.ListUsers(r.Context(), &pb.ListUsersRequest{
		Page:         page,
		PageSize:     pageSize,
		SearchQuery:  searchQuery,
		StatusFilter: statusFilter,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	writeJSONResponse(w, response)
}

// HandleCreateUser handles POST /users
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}

	var createReq pb.CreateUserRequest

	// Check content type
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "application/json", "":
		// Use protojson for better handling of proto-specific features
		err = protojson.Unmarshal(body, &createReq)
	default:
		http.Error(w, fmt.Sprintf("Unsupported content type: %s", contentType), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal: %v", err), http.StatusBadRequest)
		return
	}

	// Call service
	user, err := h.service.CreateUser(r.Context(), &createReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusCreated)
	writeJSONResponse(w, user)
}

// HandleUpdateUser handles PUT /users/{user_id}
func (h *UserHandler) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	userId := r.PathValue("user_id")
	if userId == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read body: %v", err), http.StatusBadRequest)
		return
	}

	var updateReq pb.UpdateUserRequest

	// Check content type
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "application/json", "":
		// Use protojson for better handling of proto-specific features
		err = protojson.Unmarshal(body, &updateReq)
	default:
		http.Error(w, fmt.Sprintf("Unsupported content type: %s", contentType), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to unmarshal: %v", err), http.StatusBadRequest)
		return
	}

	// Set the ID from the path
	updateReq.UserId = userId

	// Call service
	user, err := h.service.UpdateUser(r.Context(), &updateReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	writeJSONResponse(w, user)
}

// HandleDeleteUser handles DELETE /users/{user_id}
func (h *UserHandler) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	userId := r.PathValue("user_id")
	if userId == "" {
		http.Error(w, "Missing user ID", http.StatusBadRequest)
		return
	}

	// Call service
	_, err := h.service.DeleteUser(r.Context(), &pb.GetUserRequest{
		UserId: userId,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusNoContent)
}

// HandleLiveness handles GET /liveness
func (h *UserHandler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	// Call service
	_, err := h.service.Liveness(r.Context(), &emptypb.Empty{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response
	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

// writeJSONResponse writes a JSON response
func writeJSONResponse(w http.ResponseWriter, msg any) {
	var data []byte
	var err error

	// Marshal to JSON
	data, err = json.Marshal(msg)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
