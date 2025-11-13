package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	custominsecure "github.com/paulstuart/grpc-example/insecure"
	pb "github.com/paulstuart/grpc-example/proto/pkg"
)

var (
	serverAddr   = flag.String("server", "localhost:10000", "gRPC server address")
	insecureConn = flag.Bool("insecure", false, "use insecure connection")
)

func main() {
	flag.Parse()

	log.Println("=== gRPC Client Example ===")
	log.Printf("Connecting to server: %s\n", *serverAddr)

	// Setup connection
	var opts []grpc.DialOption
	if *insecureConn {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		// Use self-signed cert with InsecureSkipVerify for development
		// This allows connecting to any hostname with the self-signed cert
		tlsConfig := &tls.Config{
			RootCAs:            custominsecure.CertPool,
			InsecureSkipVerify: true, // Skip hostname verification for self-signed certs
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	}

	conn, err := grpc.NewClient(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	//nolint:errcheck // error doesn't matter at that point
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	ctx := context.Background()

	// Demonstrate all RPC patterns
	separator := "============================================================"
	fmt.Println("\n" + separator)
	fmt.Println("1. Unary RPC: AddUser")
	fmt.Println(separator)
	demonstrateAddUser(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("2. Unary RPC: GetUser")
	fmt.Println(separator)
	demonstrateGetUser(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("3. Unary RPC: UpdateUser (with Field Mask)")
	fmt.Println(separator)
	demonstrateUpdateUser(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("4. Server Streaming RPC: ListUsers")
	fmt.Println(separator)
	demonstrateListUsers(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("5. Server Streaming RPC: ListUsersByRole")
	fmt.Println(separator)
	demonstrateListUsersByRole(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("6. Client Streaming RPC: BatchAddUsers")
	fmt.Println(separator)
	demonstrateBatchAddUsers(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("7. Bidirectional Streaming RPC: UserActivityStream")
	fmt.Println(separator)
	demonstrateUserActivityStream(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("8. Bidirectional Streaming RPC: SyncUsers")
	fmt.Println(separator)
	demonstrateSyncUsers(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("9. Unary RPC: DeleteUser")
	fmt.Println(separator)
	demonstrateDeleteUser(ctx, client)

	fmt.Println("\n" + separator)
	fmt.Println("All demonstrations completed successfully!")
	fmt.Println(separator)
}

func demonstrateAddUser(ctx context.Context, client pb.UserServiceClient) {
	user := &pb.User{
		Id:       1,
		Role:     pb.Role_ADMIN,
		Username: "admin",
		ContactInfo: &pb.User_Email{
			Email: "admin@example.com",
		},
		Profile: &pb.Profile{
			DisplayName: "Administrator",
			Bio:         "System administrator",
			Preferences: map[string]int32{
				"theme":         1,
				"notifications": 0,
			},
		},
		Tags: []string{"admin", "staff"},
		Metadata: map[string]string{
			"department": "IT",
			"location":   "HQ",
		},
		Status: pb.UserStatus_ACTIVE,
		Addresses: []*pb.Address{
			{
				Type:       pb.Address_WORK,
				Street:     "123 Main St",
				City:       "San Francisco",
				State:      "CA",
				PostalCode: "94102",
				Country:    "USA",
				IsPrimary:  true,
			},
		},
	}

	_, err := client.AddUser(ctx, user)
	if err != nil {
		log.Printf("Failed to add user: %v", err)
		return
	}

	log.Printf("✓ Successfully added user: %s (ID: %d)", user.Username, user.Id)
}

func demonstrateGetUser(ctx context.Context, client pb.UserServiceClient) {
	req := &pb.GetUserRequest{Id: 1}

	user, err := client.GetUser(ctx, req)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		return
	}

	log.Printf("✓ Retrieved user: %s (ID: %d, Role: %s)", user.Username, user.Id, user.Role)
	log.Printf("  Email: %s", user.GetEmail())
	log.Printf("  Profile: %s", user.Profile.DisplayName)
	log.Printf("  Tags: %v", user.Tags)
	log.Printf("  Metadata: %v", user.Metadata)
}

func demonstrateUpdateUser(ctx context.Context, client pb.UserServiceClient) {
	req := &pb.UpdateUserRequest{
		User: &pb.User{
			Id:     1,
			Status: pb.UserStatus_ACTIVE,
			ContactInfo: &pb.User_Email{
				Email: "admin-updated@example.com",
			},
		},
		UpdateMask: nil, // Update all fields
	}

	user, err := client.UpdateUser(ctx, req)
	if err != nil {
		log.Printf("Failed to update user: %v", err)
		return
	}

	log.Printf("✓ Updated user: %s (ID: %d)", user.Username, user.Id)
	log.Printf("  New email: %s", user.GetEmail())
}

func demonstrateListUsers(ctx context.Context, client pb.UserServiceClient) {
	// Create request with filters
	req := &pb.ListUsersRequest{
		CreatedSince: timestamppb.New(time.Now().Add(-24 * time.Hour)),
		OlderThan:    durationpb.New(time.Hour * 24 * 365), // Users older than 1 year
		Status:       pb.UserStatus_ACTIVE,
	}

	stream, err := client.ListUsers(ctx, req)
	if err != nil {
		log.Printf("Failed to list users: %v", err)
		return
	}

	count := 0
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving user: %v", err)
			break
		}

		count++
		log.Printf("✓ User %d: %s (ID: %d, Role: %s)", count, user.Username, user.Id, user.Role)
	}

	log.Printf("Total users received: %d", count)
}

func demonstrateListUsersByRole(ctx context.Context, client pb.UserServiceClient) {
	req := &pb.UserRole{Role: pb.Role_ADMIN}

	stream, err := client.ListUsersByRole(ctx, req)
	if err != nil {
		log.Printf("Failed to list users by role: %v", err)
		return
	}

	count := 0
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Error receiving user: %v", err)
			break
		}

		count++
		log.Printf("✓ Admin user %d: %s (ID: %d)", count, user.Username, user.Id)
	}

	log.Printf("Total admin users: %d", count)
}

func demonstrateBatchAddUsers(ctx context.Context, client pb.UserServiceClient) {
	stream, err := client.BatchAddUsers(ctx)
	if err != nil {
		log.Printf("Failed to start batch add: %v", err)
		return
	}

	// Send multiple users
	users := []*pb.User{
		{
			Id:          2,
			Role:        pb.Role_MEMBER,
			Username:    "user1",
			ContactInfo: &pb.User_Email{Email: "user1@example.com"},
			Status:      pb.UserStatus_ACTIVE,
		},
		{
			Id:          3,
			Role:        pb.Role_MEMBER,
			Username:    "user2",
			ContactInfo: &pb.User_Phone{Phone: "+1234567890"},
			Status:      pb.UserStatus_ACTIVE,
		},
		{
			Id:          4,
			Role:        pb.Role_MODERATOR,
			Username:    "mod1",
			ContactInfo: &pb.User_Email{Email: "mod1@example.com"},
			Status:      pb.UserStatus_ACTIVE,
		},
	}

	for _, user := range users {
		if err := stream.Send(user); err != nil {
			log.Printf("Failed to send user: %v", err)
			return
		}
		log.Printf("→ Sent user: %s (ID: %d)", user.Username, user.Id)
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Printf("Failed to receive response: %v", err)
		return
	}

	log.Printf("✓ Batch add complete:")
	log.Printf("  Total received: %d", resp.TotalReceived)
	log.Printf("  Total added: %d", resp.TotalAdded)
	log.Printf("  Total failed: %d", resp.TotalFailed)
	if len(resp.Errors) > 0 {
		log.Printf("  Errors: %v", resp.Errors)
	}
}

func demonstrateUserActivityStream(ctx context.Context, client pb.UserServiceClient) {
	stream, err := client.UserActivityStream(ctx)
	if err != nil {
		log.Printf("Failed to start activity stream: %v", err)
		return
	}

	// Goroutine to receive responses
	done := make(chan bool)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				log.Printf("Error receiving activity response: %v", err)
				done <- true
				return
			}

			log.Printf("← Response: User %d - %s (Acknowledged: %v)",
				resp.UserId, resp.Message, resp.Acknowledged)
		}
	}()

	// Send activity events
	activities := []*pb.UserActivity{
		{
			UserId:       1,
			ActivityType: pb.UserActivity_LOGIN,
			Timestamp:    timestamppb.Now(),
			Details: map[string]string{
				"ip":         "192.168.1.1",
				"user_agent": "Mozilla/5.0",
			},
		},
		{
			UserId:       1,
			ActivityType: pb.UserActivity_UPDATE_PROFILE,
			Timestamp:    timestamppb.Now(),
			Details: map[string]string{
				"field": "bio",
			},
		},
		{
			UserId:       2,
			ActivityType: pb.UserActivity_VIEW_PAGE,
			Timestamp:    timestamppb.Now(),
			Details: map[string]string{
				"page": "/dashboard",
			},
		},
	}

	for _, activity := range activities {
		if err := stream.Send(activity); err != nil {
			log.Printf("Failed to send activity: %v", err)
			break
		}
		log.Printf("→ Sent activity: User %d - %s", activity.UserId, activity.ActivityType)
		time.Sleep(100 * time.Millisecond)
	}

	_ = stream.CloseSend()
	<-done
	log.Println("✓ Activity stream completed")
}

func demonstrateSyncUsers(ctx context.Context, client pb.UserServiceClient) {
	stream, err := client.SyncUsers(ctx)
	if err != nil {
		log.Printf("Failed to start sync: %v", err)
		return
	}

	// Goroutine to receive responses
	done := make(chan bool)
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				done <- true
				return
			}
			if err != nil {
				log.Printf("Error receiving sync response: %v", err)
				done <- true
				return
			}

			status := "UNKNOWN"
			switch resp.Status {
			case pb.SyncUserResponse_SUCCESS:
				status = "SUCCESS"
			case pb.SyncUserResponse_FAILED:
				status = "FAILED"
			case pb.SyncUserResponse_PARTIAL:
				status = "PARTIAL"
			}

			if resp.Status == pb.SyncUserResponse_SUCCESS {
				log.Printf("← Sync response: User %d - %s (Fields: %v)",
					resp.UserId, status, resp.UpdatedFields)
			} else {
				log.Printf("← Sync response: User %d - %s (Error: %s)",
					resp.UserId, status, resp.ErrorMessage)
			}
		}
	}()

	// Send users to sync
	users := []*pb.User{
		{
			Id:          5,
			Role:        pb.Role_MEMBER,
			Username:    "synced_user_1",
			ContactInfo: &pb.User_Email{Email: "synced1@example.com"},
			Status:      pb.UserStatus_ACTIVE,
		},
		{
			Id:          2, // Update existing user
			Role:        pb.Role_ADMIN,
			Username:    "user1_updated",
			ContactInfo: &pb.User_Email{Email: "user1@example.com"},
			Status:      pb.UserStatus_ACTIVE,
		},
	}

	for _, user := range users {
		if err := stream.Send(user); err != nil {
			log.Printf("Failed to send user for sync: %v", err)
			break
		}
		log.Printf("→ Syncing user: %s (ID: %d)", user.Username, user.Id)
		time.Sleep(100 * time.Millisecond)
	}

	_ = stream.CloseSend()
	<-done
	log.Println("✓ User sync completed")
}

func demonstrateDeleteUser(ctx context.Context, client pb.UserServiceClient) {
	req := &pb.DeleteUserRequest{Id: 5}

	_, err := client.DeleteUser(ctx, req)
	if err != nil {
		log.Printf("Failed to delete user: %v", err)
		return
	}

	log.Printf("✓ Successfully deleted user with ID: %d", req.Id)
}
