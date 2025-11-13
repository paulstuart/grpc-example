package interceptors

import (
	"context"
	"log"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// MetricsCollector collects simple metrics for demonstration
type MetricsCollector struct {
	mu               sync.RWMutex
	totalRequests    int64
	totalErrors      int64
	requestDurations map[string][]time.Duration
	methodCounts     map[string]int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		requestDurations: make(map[string][]time.Duration),
		methodCounts:     make(map[string]int64),
	}
}

// Global metrics collector instance
var globalMetrics = NewMetricsCollector()

// GetMetrics returns the global metrics collector
func GetMetrics() *MetricsCollector {
	return globalMetrics
}

// MetricsUnaryInterceptor collects metrics for unary RPCs
func MetricsUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call the handler
		resp, err := handler(ctx, req)

		duration := time.Since(start)
		globalMetrics.recordRequest(info.FullMethod, duration, err != nil)

		return resp, err
	}
}

// MetricsStreamInterceptor collects metrics for streaming RPCs
func MetricsStreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call the handler
		err := handler(srv, ss)

		duration := time.Since(start)
		globalMetrics.recordRequest(info.FullMethod, duration, err != nil)

		return err
	}
}

// recordRequest records a request's metrics
func (m *MetricsCollector) recordRequest(method string, duration time.Duration, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	m.methodCounts[method]++
	m.requestDurations[method] = append(m.requestDurations[method], duration)

	if isError {
		m.totalErrors++
	}
}

// GetStats returns current statistics
func (m *MetricsCollector) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_requests"] = m.totalRequests
	stats["total_errors"] = m.totalErrors
	stats["error_rate"] = float64(0)

	if m.totalRequests > 0 {
		stats["error_rate"] = float64(m.totalErrors) / float64(m.totalRequests) * 100
	}

	methodStats := make(map[string]interface{})
	for method, count := range m.methodCounts {
		durations := m.requestDurations[method]
		if len(durations) > 0 {
			var total time.Duration
			for _, d := range durations {
				total += d
			}
			avg := total / time.Duration(len(durations))

			methodStats[method] = map[string]interface{}{
				"count":    count,
				"avg_duration": avg.String(),
			}
		}
	}
	stats["methods"] = methodStats

	return stats
}

// PrintStats logs current statistics
func (m *MetricsCollector) PrintStats() {
	stats := m.GetStats()
	log.Printf("[Metrics] Total Requests: %d", stats["total_requests"])
	log.Printf("[Metrics] Total Errors: %d", stats["total_errors"])
	log.Printf("[Metrics] Error Rate: %.2f%%", stats["error_rate"])

	if methods, ok := stats["methods"].(map[string]interface{}); ok {
		for method, methodStats := range methods {
			log.Printf("[Metrics] %s: %v", method, methodStats)
		}
	}
}

// Reset clears all metrics
func (m *MetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests = 0
	m.totalErrors = 0
	m.requestDurations = make(map[string][]time.Duration)
	m.methodCounts = make(map[string]int64)
}
