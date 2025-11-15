package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	pb "github.com/paulstuart/grpc-example/proto/pkg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the UserService gRPC server
type Server struct {
	pb.UnimplementedUserServiceServer
	storage Storage
}

// New creates a new gRPC server with the given storage backend
func New(storage Storage) *Server {
	return &Server{
		storage: storage,
	}
}

// NewWithDefaultStorage creates a new gRPC server with in-memory storage
func NewWithDefaultStorage() *Server {
	return New(NewMemoryStorage())
}

// AddUser implements the Unary RPC for adding a single user
func (s *Server) AddUser(ctx context.Context, user *pb.User) (*emptypb.Empty, error) {
	// Validate first user must be admin
	count, err := s.storage.Count(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check user count")
	}

	if count == 0 && user.Role != pb.Role_ADMIN {
		return nil, status.Error(codes.InvalidArgument, "first user created must be an admin")
	}

	// Validate required fields
	if user.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "user ID must be greater than 0")
	}

	if user.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	err = s.storage.AddUser(ctx, user)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetUser implements the Unary RPC for retrieving a user by ID
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "user ID must be greater than 0")
	}

	user, err := s.storage.GetUser(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser implements the Unary RPC for updating a user with field mask
func (s *Server) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}

	if req.User.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "user ID must be greater than 0")
	}

	// Get existing user
	existingUser, err := s.storage.GetUser(ctx, req.User.Id)
	if err != nil {
		return nil, err
	}

	// Apply field mask if provided
	if req.UpdateMask != nil && len(req.UpdateMask.Paths) > 0 {
		for _, path := range req.UpdateMask.Paths {
			if path == "id" {
				return nil, status.Error(codes.InvalidArgument, "cannot update id field")
			}

			switch path {
			case "role":
				existingUser.Role = req.User.Role
			case "username":
				existingUser.Username = req.User.Username
			case "email":
				existingUser.Email = req.User.Email
			case "phone":
				existingUser.Phone = req.User.Phone
			case "profile":
				existingUser.Profile = req.User.Profile
			case "tags":
				existingUser.Tags = req.User.Tags
			case "metadata":
				existingUser.Metadata = req.User.Metadata
			case "status":
				existingUser.Status = req.User.Status
			case "last_login":
				existingUser.LastLogin = req.User.LastLogin
			case "addresses":
				existingUser.Addresses = req.User.Addresses
			default:
				return nil, status.Errorf(codes.InvalidArgument, "invalid field path: %s", path)
			}
		}
	} else {
		// If no mask provided, update all fields except ID and create_date
		existingUser.Role = req.User.Role
		existingUser.Username = req.User.Username
		existingUser.Email = req.User.Email
		existingUser.Phone = req.User.Phone
		existingUser.Profile = req.User.Profile
		existingUser.Tags = req.User.Tags
		existingUser.Metadata = req.User.Metadata
		existingUser.Status = req.User.Status
		existingUser.LastLogin = req.User.LastLogin
		existingUser.Addresses = req.User.Addresses
	}

	err = s.storage.UpdateUser(ctx, existingUser)
	if err != nil {
		return nil, err
	}

	return existingUser, nil
}

// DeleteUser implements the Unary RPC for deleting a user
func (s *Server) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	if req.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "user ID must be greater than 0")
	}

	err := s.storage.DeleteUser(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ListUsers implements the Server Streaming RPC for listing users with filters
func (s *Server) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
	filter := &ListFilter{}

	if req.CreatedSince != nil {
		ts := req.CreatedSince.AsTime().Unix()
		filter.CreatedSince = &ts
	}

	if req.OlderThan != nil {
		// Convert duration to timestamp
		ts := time.Now().Add(-req.OlderThan.AsDuration()).Unix()
		filter.OlderThan = &ts
	}

	if req.Status != pb.UserStatus_INACTIVE || req.GetStatus() != 0 {
		status := req.Status
		filter.Status = &status
	}

	filter.PageSize = req.PageSize
	filter.PageToken = req.PageToken

	users, err := s.storage.ListUsers(stream.Context(), filter)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return status.Error(codes.NotFound, "no users found")
	}

	// Stream users to client
	for _, user := range users {
		if err := stream.Send(user); err != nil {
			return err
		}
	}

	return nil
}

// ListUsersByRole implements the Server Streaming RPC for listing users by role
func (s *Server) ListUsersByRole(req *pb.UserRole, stream pb.UserService_ListUsersByRoleServer) error {
	users, err := s.storage.ListUsersByRole(stream.Context(), req.Role)
	if err != nil {
		return err
	}

	if len(users) == 0 {
		return status.Error(codes.NotFound, fmt.Sprintf("no users found with role %s", req.Role))
	}

	// Stream users to client
	for _, user := range users {
		if err := stream.Send(user); err != nil {
			return err
		}
	}

	return nil
}

// BatchAddUsers implements the Client Streaming RPC for batch adding users
func (s *Server) BatchAddUsers(stream pb.UserService_BatchAddUsersServer) error {
	var totalReceived, totalAdded, totalFailed int32
	var errors []string

	for {
		user, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			response := &pb.BatchAddUsersResponse{
				TotalReceived: totalReceived,
				TotalAdded:    totalAdded,
				TotalFailed:   totalFailed,
				Errors:        errors,
				ProcessedAt:   timestamppb.New(time.Now()),
			}
			return stream.SendAndClose(response)
		}
		if err != nil {
			return err
		}

		totalReceived++

		// Validate and add user
		if user.Id == 0 {
			totalFailed++
			errors = append(errors, fmt.Sprintf("user %d: ID must be greater than 0", totalReceived))
			continue
		}

		if user.Username == "" {
			totalFailed++
			errors = append(errors, fmt.Sprintf("user %d: username is required", totalReceived))
			continue
		}

		err = s.storage.AddUser(stream.Context(), user)
		if err != nil {
			totalFailed++
			errors = append(errors, fmt.Sprintf("user %d: %v", totalReceived, err))
			continue
		}

		totalAdded++
	}
}

// UserActivityStream implements the Bidirectional Streaming RPC for user activity
func (s *Server) UserActivityStream(stream pb.UserService_UserActivityStreamServer) error {
	for {
		activity, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Validate user exists
		exists, err := s.storage.UserExists(stream.Context(), activity.UserId)
		if err != nil {
			return err
		}

		response := &pb.UserActivityResponse{
			UserId:       activity.UserId,
			Acknowledged: exists,
			ProcessedAt:  timestamppb.New(time.Now()),
		}

		if !exists {
			response.Message = fmt.Sprintf("User %d not found", activity.UserId)
		} else {
			response.Message = fmt.Sprintf("Activity %s recorded for user %d", activity.ActivityType, activity.UserId)

			// Update last login for LOGIN activities
			if activity.ActivityType == pb.UserActivity_LOGIN {
				user, err := s.storage.GetUser(stream.Context(), activity.UserId)
				if err == nil {
					user.LastLogin = activity.Timestamp
					if err := s.storage.UpdateUser(stream.Context(), user); err != nil {
						slog.Error("failed to update last login", "user_id", activity.UserId, "error", err)
					}
				}
			}
		}

		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

// SyncUsers implements the Bidirectional Streaming RPC for syncing users
func (s *Server) SyncUsers(stream pb.UserService_SyncUsersServer) error {
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		response := &pb.SyncUserResponse{
			UserId: user.Id,
		}

		// Validate user
		if user.Id == 0 {
			response.Status = pb.SyncUserResponse_FAILED
			response.ErrorMessage = "user ID must be greater than 0"
			if err := stream.Send(response); err != nil {
				return err
			}
			continue
		}

		// Check if user exists
		exists, err := s.storage.UserExists(stream.Context(), user.Id)
		if err != nil {
			response.Status = pb.SyncUserResponse_FAILED
			response.ErrorMessage = err.Error()
			if err := stream.Send(response); err != nil {
				return err
			}
			continue
		}

		if exists {
			// Update existing user
			err = s.storage.UpdateUser(stream.Context(), user)
			if err != nil {
				response.Status = pb.SyncUserResponse_FAILED
				response.ErrorMessage = err.Error()
			} else {
				response.Status = pb.SyncUserResponse_SUCCESS
				response.UpdatedFields = []string{"role", "username", "profile", "status"}
			}
		} else {
			// Add new user
			err = s.storage.AddUser(stream.Context(), user)
			if err != nil {
				response.Status = pb.SyncUserResponse_FAILED
				response.ErrorMessage = err.Error()
			} else {
				response.Status = pb.SyncUserResponse_SUCCESS
				response.UpdatedFields = []string{"created"}
			}
		}

		if err := stream.Send(response); err != nil {
			return err
		}
	}
}
