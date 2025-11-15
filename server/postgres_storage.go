package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/paulstuart/grpc-example/proto/pkg"
)

const (
	postgresTracerName = "github.com/paulstuart/grpc-example/server/postgres"
)

// PostgresStorage implements Storage interface using PostgreSQL
type PostgresStorage struct {
	pool *pgxpool.Pool
}

// NewPostgresStorage creates a new PostgreSQL storage backend
func NewPostgresStorage(ctx context.Context, connString string) (*PostgresStorage, error) {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "NewPostgresStorage")
	span.SetAttributes(attribute.String("db.system", "postgresql"))
	defer span.End()

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create connection pool")
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to ping database")
		pool.Close()
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	storage := &PostgresStorage{pool: pool}

	// Initialize schema
	if err := storage.initSchema(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to initialize schema")
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	span.SetStatus(codes.Ok, "PostgreSQL storage initialized")
	return storage, nil
}

// initSchema creates the necessary tables
func (s *PostgresStorage) initSchema(ctx context.Context) error {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "initSchema")
	defer span.End()

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) NOT NULL UNIQUE,
		role INTEGER NOT NULL DEFAULT 0,
		email VARCHAR(255),
		phone VARCHAR(50),
		display_name VARCHAR(255),
		bio TEXT,
		avatar_url TEXT,
		date_of_birth TIMESTAMPTZ,
		preferences JSONB,
		tags TEXT[],
		metadata JSONB,
		status INTEGER NOT NULL DEFAULT 0,
		create_date TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		last_login TIMESTAMPTZ,
		addresses JSONB
	);

	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
	CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
	CREATE INDEX IF NOT EXISTS idx_users_create_date ON users(create_date);
	`

	_, err := s.pool.Exec(ctx, schema)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create schema")
		return err
	}

	span.SetStatus(codes.Ok, "Schema initialized")
	return nil
}

// Close closes the database connection pool
func (s *PostgresStorage) Close() {
	s.pool.Close()
}

// AddUser adds a new user to storage
func (s *PostgresStorage) AddUser(ctx context.Context, user *pb.User) error {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "AddUser")
	span.SetAttributes(
		attribute.String("db.operation", "INSERT"),
		attribute.String("db.table", "users"),
		attribute.String("user.username", user.Username),
	)
	defer span.End()

	// Serialize complex fields
	preferencesJSON, err := serializePreferences(user.GetProfile().GetPreferences())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize preferences")
		return fmt.Errorf("failed to serialize preferences: %w", err)
	}

	metadataJSON, err := serializeMetadata(user.GetMetadata())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize metadata")
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	addressesJSON, err := serializeAddresses(user.GetAddresses())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to serialize addresses")
		return fmt.Errorf("failed to serialize addresses: %w", err)
	}

	profile := user.GetProfile()
	var dateOfBirth *time.Time
	if profile != nil && profile.DateOfBirth != nil {
		dob := profile.DateOfBirth.AsTime()
		dateOfBirth = &dob
	}

	var lastLogin *time.Time
	if user.LastLogin != nil {
		ll := user.LastLogin.AsTime()
		lastLogin = &ll
	}

	query := `
		INSERT INTO users (
			id, username, role, email, phone,
			display_name, bio, avatar_url, date_of_birth, preferences,
			tags, metadata, status, create_date, last_login, addresses
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			role = EXCLUDED.role,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone,
			display_name = EXCLUDED.display_name,
			bio = EXCLUDED.bio,
			avatar_url = EXCLUDED.avatar_url,
			date_of_birth = EXCLUDED.date_of_birth,
			preferences = EXCLUDED.preferences,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			status = EXCLUDED.status,
			last_login = EXCLUDED.last_login,
			addresses = EXCLUDED.addresses
	`

	_, err = s.pool.Exec(ctx, query,
		user.Id,
		user.Username,
		user.Role,
		user.GetEmail(),      // Handle oneof
		user.GetPhone(),      // Handle oneof
		profile.GetDisplayName(),
		profile.GetBio(),
		profile.GetAvatarUrl(),
		dateOfBirth,
		preferencesJSON,
		user.Tags,
		metadataJSON,
		user.Status,
		user.CreateDate.AsTime(),
		lastLogin,
		addressesJSON,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to insert user")
		return fmt.Errorf("failed to add user: %w", err)
	}

	span.SetStatus(codes.Ok, "User added")
	return nil
}

// GetUser retrieves a user by ID
func (s *PostgresStorage) GetUser(ctx context.Context, id uint32) (*pb.User, error) {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "GetUser")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
		attribute.Int("user.id", int(id)),
	)
	defer span.End()

	query := `
		SELECT id, username, role, email, phone,
		       display_name, bio, avatar_url, date_of_birth, preferences,
		       tags, metadata, status, create_date, last_login, addresses
		FROM users WHERE id = $1
	`

	var user pb.User
	var email, phone, displayName, bio, avatarURL *string
	var dateOfBirth, createDate, lastLogin *time.Time
	var preferences, metadata, addresses []byte
	var tags []string

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&user.Id, &user.Username, &user.Role, &email, &phone,
		&displayName, &bio, &avatarURL, &dateOfBirth, &preferences,
		&tags, &metadata, &user.Status, &createDate, &lastLogin, &addresses,
	)

	if err == pgx.ErrNoRows {
		span.SetStatus(codes.Error, "user not found")
		return nil, fmt.Errorf("user not found: %w", err)
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Populate contact info (no longer oneof)
	if email != nil {
		user.Email = *email
	}
	if phone != nil {
		user.Phone = *phone
	}

	// Populate profile
	profile := &pb.Profile{}
	if displayName != nil {
		profile.DisplayName = *displayName
	}
	if bio != nil {
		profile.Bio = *bio
	}
	if avatarURL != nil {
		profile.AvatarUrl = *avatarURL
	}
	if dateOfBirth != nil {
		profile.DateOfBirth = timestamppb.New(*dateOfBirth)
	}
	if len(preferences) > 0 {
		if err := deserializePreferences(preferences, &profile.Preferences); err != nil {
			span.RecordError(err)
		}
	}
	user.Profile = profile

	// Populate tags
	user.Tags = tags

	// Populate metadata
	if len(metadata) > 0 {
		if err := deserializeMetadata(metadata, &user.Metadata); err != nil {
			span.RecordError(err)
		}
	}

	// Populate timestamps
	if createDate != nil {
		user.CreateDate = timestamppb.New(*createDate)
	}
	if lastLogin != nil {
		user.LastLogin = timestamppb.New(*lastLogin)
	}

	// Populate addresses
	if len(addresses) > 0 {
		if err := deserializeAddresses(addresses, &user.Addresses); err != nil {
			span.RecordError(err)
		}
	}

	span.SetStatus(codes.Ok, "User retrieved")
	return &user, nil
}

// UpdateUser updates an existing user
func (s *PostgresStorage) UpdateUser(ctx context.Context, user *pb.User) error {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "UpdateUser")
	span.SetAttributes(
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "users"),
		attribute.Int("user.id", int(user.Id)),
	)
	defer span.End()

	// Check if user exists
	exists, err := s.UserExists(ctx, user.Id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check user existence")
		return err
	}
	if !exists {
		span.SetStatus(codes.Error, "user not found")
		return fmt.Errorf("user with ID %d not found", user.Id)
	}

	// Serialize complex fields
	preferencesJSON, err := serializePreferences(user.GetProfile().GetPreferences())
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to serialize preferences: %w", err)
	}

	metadataJSON, err := serializeMetadata(user.GetMetadata())
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	addressesJSON, err := serializeAddresses(user.GetAddresses())
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to serialize addresses: %w", err)
	}

	profile := user.GetProfile()
	var dateOfBirth *time.Time
	if profile != nil && profile.DateOfBirth != nil {
		dob := profile.DateOfBirth.AsTime()
		dateOfBirth = &dob
	}

	var lastLogin *time.Time
	if user.LastLogin != nil {
		ll := user.LastLogin.AsTime()
		lastLogin = &ll
	}

	query := `
		UPDATE users SET
			username = $2, role = $3, email = $4, phone = $5,
			display_name = $6, bio = $7, avatar_url = $8, date_of_birth = $9,
			preferences = $10, tags = $11, metadata = $12, status = $13,
			last_login = $14, addresses = $15
		WHERE id = $1
	`

	_, err = s.pool.Exec(ctx, query,
		user.Id,
		user.Username,
		user.Role,
		user.GetEmail(),
		user.GetPhone(),
		profile.GetDisplayName(),
		profile.GetBio(),
		profile.GetAvatarUrl(),
		dateOfBirth,
		preferencesJSON,
		user.Tags,
		metadataJSON,
		user.Status,
		lastLogin,
		addressesJSON,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update user")
		return fmt.Errorf("failed to update user: %w", err)
	}

	span.SetStatus(codes.Ok, "User updated")
	return nil
}

// DeleteUser deletes a user by ID
func (s *PostgresStorage) DeleteUser(ctx context.Context, id uint32) error {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "DeleteUser")
	span.SetAttributes(
		attribute.String("db.operation", "DELETE"),
		attribute.String("db.table", "users"),
		attribute.Int("user.id", int(id)),
	)
	defer span.End()

	query := `DELETE FROM users WHERE id = $1`
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		span.SetStatus(codes.Error, "user not found")
		return fmt.Errorf("user with ID %d not found", id)
	}

	span.SetStatus(codes.Ok, "User deleted")
	return nil
}

// ListUsers lists all users with optional filters
func (s *PostgresStorage) ListUsers(ctx context.Context, filter *ListFilter) ([]*pb.User, error) {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "ListUsers")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
	)
	defer span.End()

	query := `
		SELECT id, username, role, email, phone,
		       display_name, bio, avatar_url, date_of_birth, preferences,
		       tags, metadata, status, create_date, last_login, addresses
		FROM users
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filter != nil {
		if filter.CreatedSince != nil {
			query += fmt.Sprintf(" AND create_date >= $%d", argIdx)
			args = append(args, time.Unix(*filter.CreatedSince, 0))
			argIdx++
		}
		if filter.OlderThan != nil {
			query += fmt.Sprintf(" AND create_date < $%d", argIdx)
			args = append(args, time.Unix(*filter.OlderThan, 0))
			argIdx++
		}
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.PageSize > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argIdx)
			args = append(args, filter.PageSize)
			argIdx++
		}
	}

	query += " ORDER BY id"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []*pb.User{}
	for rows.Next() {
		var user pb.User
		var email, phone, displayName, bio, avatarURL *string
		var dateOfBirth, createDate, lastLogin *time.Time
		var preferences, metadata, addresses []byte
		var tags []string

		err := rows.Scan(
			&user.Id, &user.Username, &user.Role, &email, &phone,
			&displayName, &bio, &avatarURL, &dateOfBirth, &preferences,
			&tags, &metadata, &user.Status, &createDate, &lastLogin, &addresses,
		)
		if err != nil {
			span.RecordError(err)
			continue
		}

		// Populate contact info (no longer oneof)
		if email != nil {
			user.Email = *email
		}
		if phone != nil {
			user.Phone = *phone
		}

		profile := &pb.Profile{}
		if displayName != nil {
			profile.DisplayName = *displayName
		}
		if bio != nil {
			profile.Bio = *bio
		}
		if avatarURL != nil {
			profile.AvatarUrl = *avatarURL
		}
		if dateOfBirth != nil {
			profile.DateOfBirth = timestamppb.New(*dateOfBirth)
		}
		if len(preferences) > 0 {
			deserializePreferences(preferences, &profile.Preferences)
		}
		user.Profile = profile

		user.Tags = tags
		if len(metadata) > 0 {
			deserializeMetadata(metadata, &user.Metadata)
		}
		if createDate != nil {
			user.CreateDate = timestamppb.New(*createDate)
		}
		if lastLogin != nil {
			user.LastLogin = timestamppb.New(*lastLogin)
		}
		if len(addresses) > 0 {
			deserializeAddresses(addresses, &user.Addresses)
		}

		users = append(users, &user)
	}

	span.SetAttributes(attribute.Int("result.count", len(users)))
	span.SetStatus(codes.Ok, "Users listed")
	return users, nil
}

// ListUsersByRole lists users filtered by role
func (s *PostgresStorage) ListUsersByRole(ctx context.Context, role pb.Role) ([]*pb.User, error) {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "ListUsersByRole")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
		attribute.Int("filter.role", int(role)),
	)
	defer span.End()

	query := `
		SELECT id, username, role, email, phone,
		       display_name, bio, avatar_url, date_of_birth, preferences,
		       tags, metadata, status, create_date, last_login, addresses
		FROM users WHERE role = $1 ORDER BY id
	`

	rows, err := s.pool.Query(ctx, query, role)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to query users by role")
		return nil, fmt.Errorf("failed to list users by role: %w", err)
	}
	defer rows.Close()

	users := []*pb.User{}
	for rows.Next() {
		var user pb.User
		var email, phone, displayName, bio, avatarURL *string
		var dateOfBirth, createDate, lastLogin *time.Time
		var preferences, metadata, addresses []byte
		var tags []string

		err := rows.Scan(
			&user.Id, &user.Username, &user.Role, &email, &phone,
			&displayName, &bio, &avatarURL, &dateOfBirth, &preferences,
			&tags, &metadata, &user.Status, &createDate, &lastLogin, &addresses,
		)
		if err != nil {
			span.RecordError(err)
			continue
		}

		// Populate contact info (no longer oneof)
		if email != nil {
			user.Email = *email
		}
		if phone != nil {
			user.Phone = *phone
		}

		profile := &pb.Profile{}
		if displayName != nil {
			profile.DisplayName = *displayName
		}
		if bio != nil {
			profile.Bio = *bio
		}
		if avatarURL != nil {
			profile.AvatarUrl = *avatarURL
		}
		if dateOfBirth != nil {
			profile.DateOfBirth = timestamppb.New(*dateOfBirth)
		}
		if len(preferences) > 0 {
			deserializePreferences(preferences, &profile.Preferences)
		}
		user.Profile = profile

		user.Tags = tags
		if len(metadata) > 0 {
			deserializeMetadata(metadata, &user.Metadata)
		}
		if createDate != nil {
			user.CreateDate = timestamppb.New(*createDate)
		}
		if lastLogin != nil {
			user.LastLogin = timestamppb.New(*lastLogin)
		}
		if len(addresses) > 0 {
			deserializeAddresses(addresses, &user.Addresses)
		}

		users = append(users, &user)
	}

	span.SetAttributes(attribute.Int("result.count", len(users)))
	span.SetStatus(codes.Ok, "Users listed by role")
	return users, nil
}

// UserExists checks if a user with the given ID exists
func (s *PostgresStorage) UserExists(ctx context.Context, id uint32) (bool, error) {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "UserExists")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
		attribute.Int("user.id", int(id)),
	)
	defer span.End()

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`
	err := s.pool.QueryRow(ctx, query, id).Scan(&exists)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to check user existence")
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	span.SetAttributes(attribute.Bool("result.exists", exists))
	span.SetStatus(codes.Ok, "User existence checked")
	return exists, nil
}

// Count returns the total number of users
func (s *PostgresStorage) Count(ctx context.Context) (int, error) {
	tracer := otel.Tracer(postgresTracerName)
	ctx, span := tracer.Start(ctx, "Count")
	span.SetAttributes(
		attribute.String("db.operation", "SELECT"),
		attribute.String("db.table", "users"),
	)
	defer span.End()

	var count int
	query := `SELECT COUNT(*) FROM users`
	err := s.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to count users")
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	span.SetAttributes(attribute.Int("result.count", count))
	span.SetStatus(codes.Ok, "Users counted")
	return count, nil
}

// Helper functions for serialization/deserialization

func serializePreferences(prefs map[string]int32) ([]byte, error) {
	if len(prefs) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(prefs)
}

func deserializePreferences(data []byte, prefs *map[string]int32) error {
	if len(data) == 0 {
		*prefs = make(map[string]int32)
		return nil
	}
	return json.Unmarshal(data, prefs)
}

func serializeMetadata(meta map[string]string) ([]byte, error) {
	if len(meta) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(meta)
}

func deserializeMetadata(data []byte, meta *map[string]string) error {
	if len(data) == 0 {
		*meta = make(map[string]string)
		return nil
	}
	return json.Unmarshal(data, meta)
}

func serializeAddresses(addrs []*pb.Address) ([]byte, error) {
	if len(addrs) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(addrs)
}

func deserializeAddresses(data []byte, addrs *[]*pb.Address) error {
	if len(data) == 0 {
		*addrs = []*pb.Address{}
		return nil
	}
	return json.Unmarshal(data, addrs)
}
