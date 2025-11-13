package server

import (
	"context"

	pb "github.com/paulstuart/grpc-example/proto"
)

// Storage defines the interface for user data persistence
// This abstraction allows for multiple backend implementations
// (e.g., in-memory, SQL database, NoSQL database, etc.)
type Storage interface {
	// AddUser adds a new user to storage
	AddUser(ctx context.Context, user *pb.User) error

	// GetUser retrieves a user by ID
	GetUser(ctx context.Context, id uint32) (*pb.User, error)

	// UpdateUser updates an existing user
	UpdateUser(ctx context.Context, user *pb.User) error

	// DeleteUser deletes a user by ID
	DeleteUser(ctx context.Context, id uint32) error

	// ListUsers lists all users with optional filters
	ListUsers(ctx context.Context, filter *ListFilter) ([]*pb.User, error)

	// ListUsersByRole lists users filtered by role
	ListUsersByRole(ctx context.Context, role pb.Role) ([]*pb.User, error)

	// UserExists checks if a user with the given ID exists
	UserExists(ctx context.Context, id uint32) (bool, error)

	// Count returns the total number of users
	Count(ctx context.Context) (int, error)
}

// ListFilter defines filters for listing users
type ListFilter struct {
	CreatedSince *int64
	OlderThan    *int64
	Status       *pb.UserStatus
	PageSize     int32
	PageToken    string
}
