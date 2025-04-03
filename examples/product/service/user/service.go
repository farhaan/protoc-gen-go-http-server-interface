package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/farhaan/protoc-gen-go-http-server-interface/examples/product/pb/user" // Your generated proto package
)

// UserService implements the UserService defined in the proto
type UserService struct {
	mu    sync.RWMutex
	users map[string]*pb.User
}

// NewUserService creates a new user service
func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]*pb.User),
	}
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[req.UserId]
	if !ok {
		return nil, errors.New("user not found")
	}

	// Return a deep copy of the user
	return &pb.User{
		Id:        user.Id,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	if req.Username == "" {
		return nil, errors.New("username is required")
	}
	if req.Email == "" {
		return nil, errors.New("email is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate username or email
	for _, u := range s.users {
		if strings.EqualFold(u.Username, req.Username) {
			return nil, errors.New("username already taken")
		}
		if strings.EqualFold(u.Email, req.Email) {
			return nil, errors.New("email already in use")
		}
	}

	// Create a new user
	now := timestamppb.Now()
	user := &pb.User{
		Id:        uuid.New().String(),
		Username:  req.Username,
		Email:     req.Email,
		FullName:  req.FullName,
		Status:    pb.UserStatus_USER_STATUS_ACTIVE,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Store the user
	s.users[user.Id] = user

	// Return a copy
	return &pb.User{
		Id:        user.Id,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// UpdateUser updates an existing user
func (s *UserService) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	if req.UserId == "" {
		return nil, errors.New("user ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[req.UserId]
	if !ok {
		return nil, errors.New("user not found")
	}

	// Check for duplicate email (if changed)
	if req.Email != "" && req.Email != user.Email {
		for _, u := range s.users {
			if u.Id != req.UserId && strings.EqualFold(u.Email, req.Email) {
				return nil, errors.New("email already in use")
			}
		}
	}

	// Update fields if provided
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	if req.Status != pb.UserStatus_USER_STATUS_UNSPECIFIED {
		user.Status = req.Status
	}

	// Update timestamp
	user.UpdatedAt = timestamppb.Now()

	// Return a copy
	return &pb.User{
		Id:        user.Id,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, req *pb.GetUserRequest) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[req.UserId]; !ok {
		return nil, errors.New("user not found")
	}

	delete(s.users, req.UserId)
	return &emptypb.Empty{}, nil
}

// ListUsers lists all users with optional filtering
func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start and end indices for pagination
	startIndex := int((req.Page - 1) * req.PageSize)
	endIndex := int(req.Page * req.PageSize)

	// Filter and collect users
	var filteredUsers []*pb.User
	for _, user := range s.users {
		// Apply status filter if provided
		if req.StatusFilter != pb.UserStatus_USER_STATUS_UNSPECIFIED && user.Status != req.StatusFilter {
			continue
		}

		// Apply search filter if provided
		if req.SearchQuery != "" &&
			!containsIgnoreCase(user.Username, req.SearchQuery) &&
			!containsIgnoreCase(user.Email, req.SearchQuery) &&
			!containsIgnoreCase(user.FullName, req.SearchQuery) {
			continue
		}

		// Add a copy of the user
		filteredUsers = append(filteredUsers, &pb.User{
			Id:        user.Id,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			Status:    user.Status,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		})
	}

	// Prepare pagination
	totalCount := len(filteredUsers)
	if startIndex >= totalCount {
		// Return empty page if start index is out of range
		return &pb.ListUsersResponse{
			Users:      []*pb.User{},
			TotalCount: int32(totalCount),
		}, nil
	}

	if endIndex > totalCount {
		endIndex = totalCount
	}

	// Return paginated results
	return &pb.ListUsersResponse{
		Users:      filteredUsers[startIndex:endIndex],
		TotalCount: int32(totalCount),
	}, nil
}

// Liveness handles health check requests
func (s *UserService) Liveness(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Helper function
func containsIgnoreCase(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}
