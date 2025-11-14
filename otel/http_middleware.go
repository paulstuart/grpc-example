package otel

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// WrapHandler wraps an HTTP handler with OpenTelemetry instrumentation
// This ensures trace context propagation from HTTP/REST calls to gRPC
func WrapHandler(handler http.Handler, serviceName string) http.Handler {
	return otelhttp.NewHandler(handler, serviceName,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return r.Method + " " + r.URL.Path
		}),
	)
}

// WrapMux wraps an http.ServeMux with OpenTelemetry instrumentation
func WrapMux(mux *http.ServeMux, serviceName string) http.Handler {
	return WrapHandler(mux, serviceName)
}
