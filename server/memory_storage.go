package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/paulstuart/grpc-example/proto/pkg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MemoryStorage implements the Storage interface using in-memory storage
type MemoryStorage struct {
	mu    sync.RWMutex
	users map[uint32]*pb.User
}

// NewMemoryStorage creates a new in-memory storage backend
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		users: make(map[uint32]*pb.User),
	}
}

// Verify that MemoryStorage implements Storage interface
var _ Storage = (*MemoryStorage)(nil)

// AddUser adds a new user to memory storage
func (m *MemoryStorage) AddUser(ctx context.Context, user *pb.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user already exists
	if _, exists := m.users[user.Id]; exists {
		return status.Error(codes.AlreadyExists, "user already exists")
	}

	// Set create date if not provided
	if user.CreateDate == nil {
		user.CreateDate = timestamppb.New(time.Now())
	}

	// Set default status if not provided
	if user.Status == pb.UserStatus_INACTIVE {
		user.Status = pb.UserStatus_ACTIVE
	}

	// Clone the user to avoid external modifications
	m.users[user.Id] = cloneUser(user)

	return nil
}

// GetUser retrieves a user by ID
func (m *MemoryStorage) GetUser(ctx context.Context, id uint32) (*pb.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return cloneUser(user), nil
}

// UpdateUser updates an existing user
func (m *MemoryStorage) UpdateUser(ctx context.Context, user *pb.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[user.Id]; !exists {
		return status.Error(codes.NotFound, "user not found")
	}

	m.users[user.Id] = cloneUser(user)
	return nil
}

// DeleteUser deletes a user by ID
func (m *MemoryStorage) DeleteUser(ctx context.Context, id uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[id]; !exists {
		return status.Error(codes.NotFound, "user not found")
	}

	delete(m.users, id)
	return nil
}

// ListUsers lists all users with optional filters
func (m *MemoryStorage) ListUsers(ctx context.Context, filter *ListFilter) ([]*pb.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*pb.User

	for _, user := range m.users {
		// Apply filters
		if filter != nil {
			if filter.CreatedSince != nil {
				createdSince := time.Unix(*filter.CreatedSince, 0)
				if user.CreateDate.AsTime().Before(createdSince) {
					continue
				}
			}

			if filter.OlderThan != nil {
				olderThan := time.Unix(*filter.OlderThan, 0)
				if time.Since(user.CreateDate.AsTime()) <= time.Since(olderThan) {
					continue
				}
			}

			if filter.Status != nil && user.Status != *filter.Status {
				continue
			}
		}

		result = append(result, cloneUser(user))
	}

	return result, nil
}

// ListUsersByRole lists users filtered by role
func (m *MemoryStorage) ListUsersByRole(ctx context.Context, role pb.Role) ([]*pb.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*pb.User

	for _, user := range m.users {
		if user.Role == role {
			result = append(result, cloneUser(user))
		}
	}

	return result, nil
}

// UserExists checks if a user with the given ID exists
func (m *MemoryStorage) UserExists(ctx context.Context, id uint32) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.users[id]
	return exists, nil
}

// Count returns the total number of users
func (m *MemoryStorage) Count(ctx context.Context) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.users), nil
}

// cloneUser creates a deep copy of a user
func cloneUser(user *pb.User) *pb.User {
	if user == nil {
		return nil
	}

	clone := &pb.User{
		Id:         user.Id,
		Role:       user.Role,
		CreateDate: user.CreateDate,
		Username:   user.Username,
		Profile:    cloneProfile(user.Profile),
		Tags:       append([]string{}, user.Tags...),
		Metadata:   cloneMap(user.Metadata),
		Status:     user.Status,
		LastLogin:  user.LastLogin,
		Addresses:  cloneAddresses(user.Addresses),
	}

	// Clone the oneof contact_info
	switch v := user.ContactInfo.(type) {
	case *pb.User_Email:
		clone.ContactInfo = &pb.User_Email{Email: v.Email}
	case *pb.User_Phone:
		clone.ContactInfo = &pb.User_Phone{Phone: v.Phone}
	}

	return clone
}

func cloneProfile(profile *pb.Profile) *pb.Profile {
	if profile == nil {
		return nil
	}

	return &pb.Profile{
		DisplayName:  profile.DisplayName,
		Bio:          profile.Bio,
		AvatarUrl:    profile.AvatarUrl,
		DateOfBirth:  profile.DateOfBirth,
		Preferences:  cloneIntMap(profile.Preferences),
	}
}

func cloneAddresses(addresses []*pb.Address) []*pb.Address {
	if addresses == nil {
		return nil
	}

	result := make([]*pb.Address, len(addresses))
	for i, addr := range addresses {
		if addr != nil {
			result[i] = &pb.Address{
				Type:       addr.Type,
				Street:     addr.Street,
				City:       addr.City,
				State:      addr.State,
				PostalCode: addr.PostalCode,
				Country:    addr.Country,
				IsPrimary:  addr.IsPrimary,
			}
		}
	}
	return result
}

func cloneMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}

	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func cloneIntMap(m map[string]int32) map[string]int32 {
	if m == nil {
		return nil
	}

	result := make(map[string]int32, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// String provides a string representation of the storage state
func (m *MemoryStorage) String() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return fmt.Sprintf("MemoryStorage{users: %d}", len(m.users))
}
