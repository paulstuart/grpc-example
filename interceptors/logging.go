package interceptors

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
)

// LoggingUnaryInterceptor logs unary RPC calls
func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		log.Printf("[Unary] Started %s", info.FullMethod)

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		if err != nil {
			log.Printf("[Unary] Completed %s with error: %v (duration: %v)", info.FullMethod, err, duration)
		} else {
			log.Printf("[Unary] Completed %s successfully (duration: %v)", info.FullMethod, duration)
		}

		return resp, err
	}
}

// LoggingStreamInterceptor logs streaming RPC calls
func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		streamType := "Unknown"
		if info.IsClientStream && info.IsServerStream {
			streamType = "Bidirectional"
		} else if info.IsClientStream {
			streamType = "ClientStream"
		} else if info.IsServerStream {
			streamType = "ServerStream"
		}

		log.Printf("[%s] Started %s", streamType, info.FullMethod)

		// Call the handler
		err := handler(srv, ss)

		duration := time.Since(start)
		if err != nil {
			log.Printf("[%s] Completed %s with error: %v (duration: %v)", streamType, info.FullMethod, err, duration)
		} else {
			log.Printf("[%s] Completed %s successfully (duration: %v)", streamType, info.FullMethod, duration)
		}

		return err
	}
}
