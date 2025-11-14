package interceptors

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

const (
	instrumentationName = "github.com/paulstuart/grpc-example/interceptors"
)

// OtelMetrics holds OpenTelemetry metric instruments
type OtelMetrics struct {
	requestCounter   metric.Int64Counter
	requestDuration  metric.Float64Histogram
	errorCounter     metric.Int64Counter
	activeRequests   metric.Int64UpDownCounter
}

var globalOtelMetrics *OtelMetrics

// InitializeOtelMetrics creates and registers OpenTelemetry metrics
func InitializeOtelMetrics() error {
	meter := otel.Meter(instrumentationName)

	requestCounter, err := meter.Int64Counter(
		"grpc.server.request.count",
		metric.WithDescription("Total number of gRPC requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	requestDuration, err := meter.Float64Histogram(
		"grpc.server.request.duration",
		metric.WithDescription("Duration of gRPC requests"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	errorCounter, err := meter.Int64Counter(
		"grpc.server.request.errors",
		metric.WithDescription("Total number of gRPC errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return err
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"grpc.server.active_requests",
		metric.WithDescription("Number of active gRPC requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return err
	}

	globalOtelMetrics = &OtelMetrics{
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		errorCounter:    errorCounter,
		activeRequests:  activeRequests,
	}

	log.Println("OpenTelemetry metrics initialized")
	return nil
}

// OtelLoggingUnaryInterceptor combines logging with OpenTelemetry tracing
func OtelLoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(instrumentationName)

	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", extractService(info.FullMethod)),
				attribute.String("rpc.method", extractMethod(info.FullMethod)),
				attribute.String("rpc.grpc.kind", "unary"),
			),
		)
		defer span.End()

		start := time.Now()
		log.Printf("[Unary] Started %s", info.FullMethod)

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Record span status and attributes
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)

			st, _ := status.FromError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", st.Code().String()),
				attribute.String("error.message", st.Message()),
			)

			log.Printf("[Unary] Completed %s with error: %v (duration: %v)",
				info.FullMethod, err, duration)
		} else {
			span.SetStatus(codes.Ok, "Success")
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
			log.Printf("[Unary] Completed %s successfully (duration: %v)",
				info.FullMethod, duration)
		}

		span.SetAttributes(attribute.Int64("rpc.duration_ms", duration.Milliseconds()))

		// Add user info from JWT claims if available
		if claims := GetClaimsFromContext(ctx); claims != nil {
			span.SetAttributes(
				attribute.String("user.id", claims.Username),
				attribute.String("user.email", claims.Email),
				attribute.StringSlice("user.roles", claims.Roles),
			)
		}

		return resp, err
	}
}

// OtelLoggingStreamInterceptor combines logging with OpenTelemetry tracing for streams
func OtelLoggingStreamInterceptor() grpc.StreamServerInterceptor {
	tracer := otel.Tracer(instrumentationName)

	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx, span := tracer.Start(ss.Context(), info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.service", extractService(info.FullMethod)),
				attribute.String("rpc.method", extractMethod(info.FullMethod)),
				attribute.Bool("rpc.grpc.is_client_stream", info.IsClientStream),
				attribute.Bool("rpc.grpc.is_server_stream", info.IsServerStream),
			),
		)
		defer span.End()

		start := time.Now()

		streamType := determineStreamType(info)
		span.SetAttributes(attribute.String("rpc.grpc.kind", streamType))

		log.Printf("[%s] Started %s", streamType, info.FullMethod)

		// Wrap the stream to use the traced context
		wrappedStream := &tracedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Call the handler
		err := handler(srv, wrappedStream)

		duration := time.Since(start)

		// Record span status
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)

			st, _ := status.FromError(err)
			span.SetAttributes(
				attribute.String("rpc.grpc.status_code", st.Code().String()),
				attribute.String("error.message", st.Message()),
			)

			log.Printf("[%s] Completed %s with error: %v (duration: %v)",
				streamType, info.FullMethod, err, duration)
		} else {
			span.SetStatus(codes.Ok, "Success")
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
			log.Printf("[%s] Completed %s successfully (duration: %v)",
				streamType, info.FullMethod, duration)
		}

		span.SetAttributes(attribute.Int64("rpc.duration_ms", duration.Milliseconds()))

		// Add user info from JWT claims if available
		if claims := GetClaimsFromContext(ctx); claims != nil {
			span.SetAttributes(
				attribute.String("user.id", claims.Username),
				attribute.String("user.email", claims.Email),
				attribute.StringSlice("user.roles", claims.Roles),
			)
		}

		return err
	}
}

// OtelMetricsUnaryInterceptor collects OpenTelemetry metrics for unary RPCs
func OtelMetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if globalOtelMetrics == nil {
			// Fallback if metrics not initialized
			return handler(ctx, req)
		}

		attrs := []attribute.KeyValue{
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.service", extractService(info.FullMethod)),
			attribute.String("rpc.grpc.kind", "unary"),
		}

		// Increment active requests
		globalOtelMetrics.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
		defer globalOtelMetrics.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))

		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start).Milliseconds()

		// Add status code to attributes
		statusAttrs := append(attrs, attribute.String("rpc.grpc.status_code", getStatusCode(err)))

		// Record metrics
		globalOtelMetrics.requestCounter.Add(ctx, 1, metric.WithAttributes(statusAttrs...))
		globalOtelMetrics.requestDuration.Record(ctx, float64(duration), metric.WithAttributes(statusAttrs...))

		if err != nil {
			globalOtelMetrics.errorCounter.Add(ctx, 1, metric.WithAttributes(statusAttrs...))
		}

		return resp, err
	}
}

// OtelMetricsStreamInterceptor collects OpenTelemetry metrics for streaming RPCs
func OtelMetricsStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		if globalOtelMetrics == nil {
			// Fallback if metrics not initialized
			return handler(srv, ss)
		}

		ctx := ss.Context()
		streamType := determineStreamType(info)

		attrs := []attribute.KeyValue{
			attribute.String("rpc.method", info.FullMethod),
			attribute.String("rpc.service", extractService(info.FullMethod)),
			attribute.String("rpc.grpc.kind", streamType),
		}

		// Increment active requests
		globalOtelMetrics.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
		defer globalOtelMetrics.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))

		start := time.Now()

		// Call the handler
		err := handler(srv, ss)

		duration := time.Since(start).Milliseconds()

		// Add status code to attributes
		statusAttrs := append(attrs, attribute.String("rpc.grpc.status_code", getStatusCode(err)))

		// Record metrics
		globalOtelMetrics.requestCounter.Add(ctx, 1, metric.WithAttributes(statusAttrs...))
		globalOtelMetrics.requestDuration.Record(ctx, float64(duration), metric.WithAttributes(statusAttrs...))

		if err != nil {
			globalOtelMetrics.errorCounter.Add(ctx, 1, metric.WithAttributes(statusAttrs...))
		}

		return err
	}
}

// Helper functions

func extractService(fullMethod string) string {
	// fullMethod format: /package.Service/Method
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	for i, c := range fullMethod {
		if c == '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}

func extractMethod(fullMethod string) string {
	// fullMethod format: /package.Service/Method
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}

func determineStreamType(info *grpc.StreamServerInfo) string {
	if info.IsClientStream && info.IsServerStream {
		return "bidi_stream"
	} else if info.IsClientStream {
		return "client_stream"
	} else if info.IsServerStream {
		return "server_stream"
	}
	return "unknown"
}

func getStatusCode(err error) string {
	if err == nil {
		return "OK"
	}
	st, _ := status.FromError(err)
	return st.Code().String()
}

// tracedServerStream wraps grpc.ServerStream with a traced context
type tracedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *tracedServerStream) Context() context.Context {
	return s.ctx
}
