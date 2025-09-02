// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package server

import (
	"fmt"
	"sort"
	"sync"
)

// MetricDefinition represents a metric definition with its metadata
type MetricDefinition struct {
	Name        string              `json:"name"`
	Type        MetricType          `json:"type"`
	Description string              `json:"description"`
	Labels      []string            `json:"labels"`
	Buckets     []float64           `json:"buckets,omitempty"`    // For histograms
	Objectives  map[float64]float64 `json:"objectives,omitempty"` // For summaries
}

// MetricsRegistry manages predefined and custom metric definitions
type MetricsRegistry struct {
	predefinedMetrics map[string]MetricDefinition
	customMetrics     map[string]MetricDefinition
	mu                sync.RWMutex
}

// NewMetricsRegistry creates a new metrics registry with predefined framework metrics
func NewMetricsRegistry() *MetricsRegistry {
	mr := &MetricsRegistry{
		predefinedMetrics: make(map[string]MetricDefinition),
		customMetrics:     make(map[string]MetricDefinition),
	}

	mr.registerPredefinedMetrics()
	return mr
}

// RegisterMetric registers a custom metric definition
func (mr *MetricsRegistry) RegisterMetric(definition MetricDefinition) error {
	if definition.Name == "" {
		return fmt.Errorf("metric name cannot be empty")
	}

	if definition.Type == "" {
		return fmt.Errorf("metric type cannot be empty")
	}

	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Check if already exists in predefined metrics
	if _, exists := mr.predefinedMetrics[definition.Name]; exists {
		return fmt.Errorf("metric %s already exists as predefined metric", definition.Name)
	}

	// Check if already exists in custom metrics
	if _, exists := mr.customMetrics[definition.Name]; exists {
		return fmt.Errorf("metric %s already exists as custom metric", definition.Name)
	}

	mr.customMetrics[definition.Name] = definition
	return nil
}

// GetMetricDefinition retrieves a metric definition by name
func (mr *MetricsRegistry) GetMetricDefinition(name string) (*MetricDefinition, bool) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	// Check predefined metrics first
	if def, exists := mr.predefinedMetrics[name]; exists {
		return &def, true
	}

	// Check custom metrics
	if def, exists := mr.customMetrics[name]; exists {
		return &def, true
	}

	return nil, false
}

// ListMetrics returns all registered metrics (predefined and custom)
func (mr *MetricsRegistry) ListMetrics() []MetricDefinition {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	metrics := make([]MetricDefinition, 0, len(mr.predefinedMetrics)+len(mr.customMetrics))

	// Add predefined metrics
	for _, def := range mr.predefinedMetrics {
		metrics = append(metrics, def)
	}

	// Add custom metrics
	for _, def := range mr.customMetrics {
		metrics = append(metrics, def)
	}

	// Sort by name for consistent output
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})

	return metrics
}

// ListPredefinedMetrics returns only predefined framework metrics
func (mr *MetricsRegistry) ListPredefinedMetrics() []MetricDefinition {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	metrics := make([]MetricDefinition, 0, len(mr.predefinedMetrics))
	for _, def := range mr.predefinedMetrics {
		metrics = append(metrics, def)
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})

	return metrics
}

// ListCustomMetrics returns only custom registered metrics
func (mr *MetricsRegistry) ListCustomMetrics() []MetricDefinition {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	metrics := make([]MetricDefinition, 0, len(mr.customMetrics))
	for _, def := range mr.customMetrics {
		metrics = append(metrics, def)
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})

	return metrics
}

// UnregisterCustomMetric removes a custom metric definition
func (mr *MetricsRegistry) UnregisterCustomMetric(name string) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	if _, exists := mr.customMetrics[name]; !exists {
		return fmt.Errorf("custom metric %s not found", name)
	}

	delete(mr.customMetrics, name)
	return nil
}

// IsRegistered checks if a metric is registered (predefined or custom)
func (mr *MetricsRegistry) IsRegistered(name string) bool {
	_, exists := mr.GetMetricDefinition(name)
	return exists
}

// ValidateMetric validates a metric definition
func (mr *MetricsRegistry) ValidateMetric(definition MetricDefinition) error {
	if definition.Name == "" {
		return fmt.Errorf("metric name is required")
	}

	if definition.Type == "" {
		return fmt.Errorf("metric type is required")
	}

	// Validate metric type
	switch definition.Type {
	case MetricTypeCounter, MetricTypeGauge, MetricTypeHistogram, MetricTypeSummary:
		// Valid types
	default:
		return fmt.Errorf("invalid metric type: %s", definition.Type)
	}

	// Validate histogram-specific fields
	if definition.Type == MetricTypeHistogram && len(definition.Buckets) == 0 {
		return fmt.Errorf("histogram metric must have buckets defined")
	}

	// Validate summary-specific fields
	if definition.Type == MetricTypeSummary && len(definition.Objectives) == 0 {
		return fmt.Errorf("summary metric must have objectives defined")
	}

	// Validate label names
	for _, label := range definition.Labels {
		if label == "" {
			return fmt.Errorf("label name cannot be empty")
		}
	}

	return nil
}

// registerPredefinedMetrics registers all predefined framework metrics
func (mr *MetricsRegistry) registerPredefinedMetrics() {
	// HTTP Metrics
	mr.predefinedMetrics["http_requests_total"] = MetricDefinition{
		Name:        "http_requests_total",
		Type:        MetricTypeCounter,
		Description: "Total number of HTTP requests",
		Labels:      []string{"method", "endpoint", "status"},
	}

	mr.predefinedMetrics["http_request_duration_seconds"] = MetricDefinition{
		Name:        "http_request_duration_seconds",
		Type:        MetricTypeHistogram,
		Description: "HTTP request duration in seconds",
		Labels:      []string{"method", "endpoint"},
		Buckets:     []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
	}

	mr.predefinedMetrics["http_request_size_bytes"] = MetricDefinition{
		Name:        "http_request_size_bytes",
		Type:        MetricTypeHistogram,
		Description: "HTTP request size in bytes",
		Labels:      []string{"method", "endpoint"},
		Buckets:     []float64{100, 1000, 10000, 100000, 1000000},
	}

	mr.predefinedMetrics["http_response_size_bytes"] = MetricDefinition{
		Name:        "http_response_size_bytes",
		Type:        MetricTypeHistogram,
		Description: "HTTP response size in bytes",
		Labels:      []string{"method", "endpoint"},
		Buckets:     []float64{100, 1000, 10000, 100000, 1000000},
	}

	mr.predefinedMetrics["http_active_requests"] = MetricDefinition{
		Name:        "http_active_requests",
		Type:        MetricTypeGauge,
		Description: "Number of active HTTP requests",
		Labels:      []string{},
	}

	// gRPC Metrics
	mr.predefinedMetrics["grpc_server_started_total"] = MetricDefinition{
		Name:        "grpc_server_started_total",
		Type:        MetricTypeCounter,
		Description: "Total number of gRPC calls started",
		Labels:      []string{"method"},
	}

	mr.predefinedMetrics["grpc_server_handled_total"] = MetricDefinition{
		Name:        "grpc_server_handled_total",
		Type:        MetricTypeCounter,
		Description: "Total number of gRPC calls handled",
		Labels:      []string{"method", "code"},
	}

	mr.predefinedMetrics["grpc_server_handling_seconds"] = MetricDefinition{
		Name:        "grpc_server_handling_seconds",
		Type:        MetricTypeHistogram,
		Description: "gRPC call handling duration in seconds",
		Labels:      []string{"method"},
		Buckets:     []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
	}

	mr.predefinedMetrics["grpc_server_msg_received_total"] = MetricDefinition{
		Name:        "grpc_server_msg_received_total",
		Type:        MetricTypeCounter,
		Description: "Total number of gRPC messages received",
		Labels:      []string{"method"},
	}

	mr.predefinedMetrics["grpc_server_msg_sent_total"] = MetricDefinition{
		Name:        "grpc_server_msg_sent_total",
		Type:        MetricTypeCounter,
		Description: "Total number of gRPC messages sent",
		Labels:      []string{"method"},
	}

	// Server Metrics
	mr.predefinedMetrics["server_uptime_seconds"] = MetricDefinition{
		Name:        "server_uptime_seconds",
		Type:        MetricTypeGauge,
		Description: "Server uptime in seconds",
		Labels:      []string{"service"},
	}

	mr.predefinedMetrics["server_startup_duration_seconds"] = MetricDefinition{
		Name:        "server_startup_duration_seconds",
		Type:        MetricTypeHistogram,
		Description: "Server startup duration in seconds",
		Labels:      []string{"service"},
		Buckets:     []float64{0.1, 0.5, 1, 2, 5, 10, 30},
	}

	mr.predefinedMetrics["server_shutdown_duration_seconds"] = MetricDefinition{
		Name:        "server_shutdown_duration_seconds",
		Type:        MetricTypeHistogram,
		Description: "Server shutdown duration in seconds",
		Labels:      []string{"service"},
		Buckets:     []float64{0.1, 0.5, 1, 2, 5, 10, 30},
	}

	mr.predefinedMetrics["server_goroutines"] = MetricDefinition{
		Name:        "server_goroutines",
		Type:        MetricTypeGauge,
		Description: "Number of goroutines",
		Labels:      []string{"service"},
	}

	mr.predefinedMetrics["server_memory_bytes"] = MetricDefinition{
		Name:        "server_memory_bytes",
		Type:        MetricTypeGauge,
		Description: "Server memory usage in bytes",
		Labels:      []string{"service", "type"},
	}

	mr.predefinedMetrics["server_gc_duration_seconds"] = MetricDefinition{
		Name:        "server_gc_duration_seconds",
		Type:        MetricTypeGauge,
		Description: "Time spent in garbage collection",
		Labels:      []string{"service"},
	}

	mr.predefinedMetrics["server_start_time"] = MetricDefinition{
		Name:        "server_start_time",
		Type:        MetricTypeGauge,
		Description: "Server start time as unix timestamp",
		Labels:      []string{"service"},
	}

	// Transport Metrics
	mr.predefinedMetrics["transport_status"] = MetricDefinition{
		Name:        "transport_status",
		Type:        MetricTypeGauge,
		Description: "Transport status (1 = up, 0 = down)",
		Labels:      []string{"transport", "status"},
	}

	mr.predefinedMetrics["transport_connections_active"] = MetricDefinition{
		Name:        "transport_connections_active",
		Type:        MetricTypeGauge,
		Description: "Number of active connections",
		Labels:      []string{"transport"},
	}

	mr.predefinedMetrics["transport_connections_total"] = MetricDefinition{
		Name:        "transport_connections_total",
		Type:        MetricTypeCounter,
		Description: "Total number of connections",
		Labels:      []string{"transport"},
	}

	mr.predefinedMetrics["transport_starts_total"] = MetricDefinition{
		Name:        "transport_starts_total",
		Type:        MetricTypeCounter,
		Description: "Total number of transport starts",
		Labels:      []string{"service", "transport"},
	}

	mr.predefinedMetrics["transport_stops_total"] = MetricDefinition{
		Name:        "transport_stops_total",
		Type:        MetricTypeCounter,
		Description: "Total number of transport stops",
		Labels:      []string{"service", "transport"},
	}

	mr.predefinedMetrics["active_transports"] = MetricDefinition{
		Name:        "active_transports",
		Type:        MetricTypeGauge,
		Description: "Number of active transports",
		Labels:      []string{"service", "transport"},
	}

	// Service Registration Metrics
	mr.predefinedMetrics["service_registrations_total"] = MetricDefinition{
		Name:        "service_registrations_total",
		Type:        MetricTypeCounter,
		Description: "Total number of service registrations",
		Labels:      []string{"service", "type"},
	}

	mr.predefinedMetrics["registered_services"] = MetricDefinition{
		Name:        "registered_services",
		Type:        MetricTypeGauge,
		Description: "Number of registered services",
		Labels:      []string{"service", "type"},
	}

	// Error Metrics
	mr.predefinedMetrics["errors_total"] = MetricDefinition{
		Name:        "errors_total",
		Type:        MetricTypeCounter,
		Description: "Total number of errors",
		Labels:      []string{"service", "error_type", "operation"},
	}

	// Operation Metrics
	mr.predefinedMetrics["operation_duration_seconds"] = MetricDefinition{
		Name:        "operation_duration_seconds",
		Type:        MetricTypeHistogram,
		Description: "Operation duration in seconds",
		Labels:      []string{"service", "operation"},
		Buckets:     []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
	}
}

// GetPredefinedMetricNames returns a list of all predefined metric names
func (mr *MetricsRegistry) GetPredefinedMetricNames() []string {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	names := make([]string, 0, len(mr.predefinedMetrics))
	for name := range mr.predefinedMetrics {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetCustomMetricNames returns a list of all custom metric names
func (mr *MetricsRegistry) GetCustomMetricNames() []string {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	names := make([]string, 0, len(mr.customMetrics))
	for name := range mr.customMetrics {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetMetricsCount returns the total number of registered metrics
func (mr *MetricsRegistry) GetMetricsCount() (predefined int, custom int, total int) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	predefined = len(mr.predefinedMetrics)
	custom = len(mr.customMetrics)
	total = predefined + custom
	return
}
