package otel

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	Enabled        bool
}

// Shutdown is a function that shuts down the OpenTelemetry providers
type Shutdown func(context.Context) error

// Setup initializes OpenTelemetry with tracing and metrics
func Setup(ctx context.Context, config Config) (Shutdown, error) {
	if !config.Enabled {
		log.Println("OpenTelemetry is disabled")
		return func(context.Context) error { return nil }, nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup trace provider
	traceShutdown, err := setupTraceProvider(ctx, res, config.OTLPEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to setup trace provider: %w", err)
	}

	// Setup metric provider
	metricShutdown, err := setupMetricProvider(ctx, res, config.OTLPEndpoint)
	if err != nil {
		// Try to shutdown trace provider if metric setup fails
		_ = traceShutdown(ctx)
		return nil, fmt.Errorf("failed to setup metric provider: %w", err)
	}

	log.Printf("OpenTelemetry initialized: service=%s, version=%s, endpoint=%s",
		config.ServiceName, config.ServiceVersion, config.OTLPEndpoint)

	// Return combined shutdown function
	shutdown := func(ctx context.Context) error {
		var err error
		if mErr := metricShutdown(ctx); mErr != nil {
			err = fmt.Errorf("metric shutdown error: %w", mErr)
		}
		if tErr := traceShutdown(ctx); tErr != nil {
			if err != nil {
				err = fmt.Errorf("%v; trace shutdown error: %w", err, tErr)
			} else {
				err = fmt.Errorf("trace shutdown error: %w", tErr)
			}
		}
		return err
	}

	return shutdown, nil
}

// setupTraceProvider creates and registers a trace provider
func setupTraceProvider(ctx context.Context, res *resource.Resource, endpoint string) (Shutdown, error) {
	// Create OTLP trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create trace provider with batch span processor
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		// Sample all traces in development, adjust for production
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set global trace provider
	otel.SetTracerProvider(provider)

	// Set global propagator to propagate trace context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Trace provider initialized")

	return provider.Shutdown, nil
}

// setupMetricProvider creates and registers a metric provider
func setupMetricProvider(ctx context.Context, res *resource.Resource, endpoint string) (Shutdown, error) {
	// Create OTLP metric exporter
	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(), // Use TLS in production
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create metric provider with periodic reader
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(10*time.Second),
			),
		),
	)

	// Set global meter provider
	otel.SetMeterProvider(provider)

	log.Println("Metric provider initialized")

	return provider.Shutdown, nil
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// GetMeter returns a meter for the given name
func GetMeter(name string) metric.Meter {
	return otel.Meter(name)
}
