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

package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/messaging"
)

// MetricsCollector defines the interface for metrics collection backends.
type MetricsCollector interface {
	// IncrementCounter increments a counter metric with the given labels and value.
	IncrementCounter(name string, labels map[string]string, value float64)

	// ObserveHistogram records an observation for a histogram metric.
	ObserveHistogram(name string, labels map[string]string, value float64)

	// SetGauge sets a gauge metric to the given value.
	SetGauge(name string, labels map[string]string, value float64)

	// AddToCounter adds a value to a counter metric.
	AddToCounter(name string, labels map[string]string, value float64)
}

// InMemoryMetricsCollector provides an in-memory implementation for testing and development.
type InMemoryMetricsCollector struct {
	counters   map[string]*Counter
	histograms map[string]*Histogram
	gauges     map[string]*Gauge
	mutex      sync.RWMutex
}

// Counter represents an in-memory counter metric.
type Counter struct {
	value float64
	mutex sync.RWMutex
}

// Add adds a value to the counter.
func (c *Counter) Add(value float64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value += value
}

// Get returns the current counter value.
func (c *Counter) Get() float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.value
}

// Histogram represents an in-memory histogram metric.
type Histogram struct {
	observations []float64
	sum          float64
	count        int64
	mutex        sync.RWMutex
}

// Observe adds an observation to the histogram.
func (h *Histogram) Observe(value float64) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.observations = append(h.observations, value)
	h.sum += value
	h.count++
}

// GetStats returns histogram statistics.
func (h *Histogram) GetStats() (sum float64, count int64, observations []float64) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	obsSlice := make([]float64, len(h.observations))
	copy(obsSlice, h.observations)
	return h.sum, h.count, obsSlice
}

// Gauge represents an in-memory gauge metric.
type Gauge struct {
	value float64
	mutex sync.RWMutex
}

// Set sets the gauge to the given value.
func (g *Gauge) Set(value float64) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.value = value
}

// Get returns the current gauge value.
func (g *Gauge) Get() float64 {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.value
}

// NewInMemoryMetricsCollector creates a new in-memory metrics collector.
func NewInMemoryMetricsCollector() *InMemoryMetricsCollector {
	return &InMemoryMetricsCollector{
		counters:   make(map[string]*Counter),
		histograms: make(map[string]*Histogram),
		gauges:     make(map[string]*Gauge),
	}
}

// IncrementCounter increments a counter metric.
func (im *InMemoryMetricsCollector) IncrementCounter(name string, labels map[string]string, value float64) {
	key := im.buildMetricKey(name, labels)
	im.mutex.Lock()
	defer im.mutex.Unlock()

	counter, exists := im.counters[key]
	if !exists {
		counter = &Counter{}
		im.counters[key] = counter
	}
	counter.Add(value)
}

// ObserveHistogram records an observation for a histogram.
func (im *InMemoryMetricsCollector) ObserveHistogram(name string, labels map[string]string, value float64) {
	key := im.buildMetricKey(name, labels)
	im.mutex.Lock()
	defer im.mutex.Unlock()

	histogram, exists := im.histograms[key]
	if !exists {
		histogram = &Histogram{}
		im.histograms[key] = histogram
	}
	histogram.Observe(value)
}

// SetGauge sets a gauge metric value.
func (im *InMemoryMetricsCollector) SetGauge(name string, labels map[string]string, value float64) {
	key := im.buildMetricKey(name, labels)
	im.mutex.Lock()
	defer im.mutex.Unlock()

	gauge, exists := im.gauges[key]
	if !exists {
		gauge = &Gauge{}
		im.gauges[key] = gauge
	}
	gauge.Set(value)
}

// AddToCounter adds a value to a counter metric.
func (im *InMemoryMetricsCollector) AddToCounter(name string, labels map[string]string, value float64) {
	im.IncrementCounter(name, labels, value)
}

// buildMetricKey builds a unique key for a metric with its labels.
func (im *InMemoryMetricsCollector) buildMetricKey(name string, labels map[string]string) string {
	key := name
	for k, v := range labels {
		key += fmt.Sprintf("_%s_%s", k, v)
	}
	return key
}

// GetCounterValue returns the current value of a counter metric.
func (im *InMemoryMetricsCollector) GetCounterValue(name string, labels map[string]string) float64 {
	key := im.buildMetricKey(name, labels)
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	counter, exists := im.counters[key]
	if !exists {
		return 0
	}
	return counter.Get()
}

// GetHistogramStats returns histogram statistics.
func (im *InMemoryMetricsCollector) GetHistogramStats(name string, labels map[string]string) (sum float64, count int64, observations []float64) {
	key := im.buildMetricKey(name, labels)
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	histogram, exists := im.histograms[key]
	if !exists {
		return 0, 0, nil
	}
	return histogram.GetStats()
}

// GetGaugeValue returns the current value of a gauge metric.
func (im *InMemoryMetricsCollector) GetGaugeValue(name string, labels map[string]string) float64 {
	key := im.buildMetricKey(name, labels)
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	gauge, exists := im.gauges[key]
	if !exists {
		return 0
	}
	return gauge.Get()
}

// MetricsConfig holds configuration for the metrics middleware.
type MetricsConfig struct {
	Collector             MetricsCollector
	EnabledMetrics        map[string]bool
	CustomLabels          map[string]string
	HistogramBuckets      []float64
	CollectionInterval    time.Duration
	EnableMessageSizeInfo bool
	EnableRetryMetrics    bool
}

// DefaultMetricsConfig returns a default metrics configuration.
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Collector: NewInMemoryMetricsCollector(),
		EnabledMetrics: map[string]bool{
			"messages_received_total":             true,
			"messages_processed_total":            true,
			"messages_failed_total":               true,
			"message_processing_duration_seconds": true,
			"message_size_bytes":                  true,
			"message_errors_by_type_total":        true,
		},
		CustomLabels:          make(map[string]string),
		HistogramBuckets:      []float64{0.001, 0.01, 0.1, 1, 5, 10, 30, 60},
		CollectionInterval:    time.Minute,
		EnableMessageSizeInfo: true,
		EnableRetryMetrics:    true,
	}
}

// StandardMetricsMiddleware provides comprehensive metrics collection for message processing.
type StandardMetricsMiddleware struct {
	config             *MetricsConfig
	activeMessages     *Gauge
	messageQueue       *Gauge
	lastCollectionTime time.Time
	mutex              sync.RWMutex
}

// NewStandardMetricsMiddleware creates a new standard metrics middleware with the given configuration.
func NewStandardMetricsMiddleware(config *MetricsConfig) *StandardMetricsMiddleware {
	if config == nil {
		config = DefaultMetricsConfig()
	}
	if config.Collector == nil {
		config.Collector = NewInMemoryMetricsCollector()
	}

	middleware := &StandardMetricsMiddleware{
		config:             config,
		activeMessages:     &Gauge{},
		messageQueue:       &Gauge{},
		lastCollectionTime: time.Now(),
	}

	// Initialize gauge metrics
	config.Collector.SetGauge("messages_active", config.CustomLabels, 0)
	config.Collector.SetGauge("message_queue_depth", config.CustomLabels, 0)

	return middleware
}

// Name returns the middleware name.
func (smm *StandardMetricsMiddleware) Name() string {
	return "standard-metrics"
}

// Wrap wraps a handler with metrics collection functionality.
func (smm *StandardMetricsMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		start := time.Now()

		// Build labels for metrics
		labels := smm.buildLabels(message)

		// Increment active messages
		smm.activeMessages.Set(smm.activeMessages.Get() + 1)
		smm.config.Collector.SetGauge("messages_active", labels, smm.activeMessages.Get())

		// Track message received
		if smm.isMetricEnabled("messages_received_total") {
			smm.config.Collector.IncrementCounter("messages_received_total", labels, 1)
		}

		// Track message size if enabled
		if smm.config.EnableMessageSizeInfo && smm.isMetricEnabled("message_size_bytes") {
			messageSize := float64(len(message.Payload))
			smm.config.Collector.ObserveHistogram("message_size_bytes", labels, messageSize)
		}

		// Process the message
		err := next.Handle(ctx, message)

		// Record processing time
		processingTime := time.Since(start).Seconds()

		// Decrement active messages
		smm.activeMessages.Set(smm.activeMessages.Get() - 1)
		smm.config.Collector.SetGauge("messages_active", labels, smm.activeMessages.Get())

		if err != nil {
			// Track failed message
			if smm.isMetricEnabled("messages_failed_total") {
				smm.config.Collector.IncrementCounter("messages_failed_total", labels, 1)
			}

			// Track error types
			if smm.isMetricEnabled("message_errors_by_type_total") {
				errorLabels := smm.buildErrorLabels(labels, err)
				smm.config.Collector.IncrementCounter("message_errors_by_type_total", errorLabels, 1)
			}

			// Track retry metrics if enabled
			if smm.config.EnableRetryMetrics {
				smm.trackRetryMetrics(message, labels)
			}
		} else {
			// Track successfully processed message
			if smm.isMetricEnabled("messages_processed_total") {
				smm.config.Collector.IncrementCounter("messages_processed_total", labels, 1)
			}
		}

		// Record processing duration
		if smm.isMetricEnabled("message_processing_duration_seconds") {
			smm.config.Collector.ObserveHistogram("message_processing_duration_seconds", labels, processingTime)
		}

		return err
	})
}

// buildLabels constructs metric labels for a message.
func (smm *StandardMetricsMiddleware) buildLabels(message *messaging.Message) map[string]string {
	labels := make(map[string]string)

	// Add custom labels
	for k, v := range smm.config.CustomLabels {
		labels[k] = v
	}

	// Add message-specific labels
	labels["topic"] = message.Topic
	if len(message.Key) > 0 {
		labels["partition_key"] = string(message.Key)
	}

	return labels
}

// buildErrorLabels constructs error-specific metric labels.
func (smm *StandardMetricsMiddleware) buildErrorLabels(baseLabels map[string]string, err error) map[string]string {
	errorLabels := make(map[string]string)
	for k, v := range baseLabels {
		errorLabels[k] = v
	}

	errorLabels["error_type"] = smm.getErrorType(err)

	return errorLabels
}

// getErrorType extracts error type from error for metrics labeling.
func (smm *StandardMetricsMiddleware) getErrorType(err error) string {
	if msgErr, ok := err.(*messaging.MessagingError); ok {
		return string(msgErr.Code)
	}
	return "unknown"
}

// trackRetryMetrics tracks retry-related metrics.
func (smm *StandardMetricsMiddleware) trackRetryMetrics(message *messaging.Message, labels map[string]string) {
	if message.DeliveryAttempt > 0 {
		retryLabels := make(map[string]string)
		for k, v := range labels {
			retryLabels[k] = v
		}
		retryLabels["retry_count"] = fmt.Sprintf("%d", message.DeliveryAttempt)
		smm.config.Collector.IncrementCounter("message_retries_total", retryLabels, 1)
	}
}

// isMetricEnabled checks if a specific metric is enabled in the configuration.
func (smm *StandardMetricsMiddleware) isMetricEnabled(metricName string) bool {
	enabled, exists := smm.config.EnabledMetrics[metricName]
	return !exists || enabled // Default to enabled if not explicitly disabled
}

// GetMetrics returns a snapshot of current metrics.
func (smm *StandardMetricsMiddleware) GetMetrics() map[string]interface{} {
	smm.mutex.RLock()
	defer smm.mutex.RUnlock()

	metrics := make(map[string]interface{})

	// Add gauge metrics
	metrics["messages_active"] = smm.activeMessages.Get()
	metrics["message_queue_depth"] = smm.messageQueue.Get()

	// If using in-memory collector, add its metrics
	if inMemoryCollector, ok := smm.config.Collector.(*InMemoryMetricsCollector); ok {
		// Add counter values
		for metricName := range smm.config.EnabledMetrics {
			if contains([]string{"messages_received_total", "messages_processed_total", "messages_failed_total"}, metricName) {
				labels := smm.config.CustomLabels
				metrics[metricName] = inMemoryCollector.GetCounterValue(metricName, labels)
			}
		}
	}

	return metrics
}

// contains checks if a string is in a slice.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrometheusMetricsCollector provides a Prometheus-compatible metrics collector interface.
type PrometheusMetricsCollector struct {
	// This would be implemented using prometheus/client_golang in a real implementation
	// For now, we'll use the in-memory collector as a placeholder
	*InMemoryMetricsCollector
}

// NewPrometheusMetricsCollector creates a new Prometheus metrics collector.
// In a real implementation, this would use the prometheus/client_golang library.
func NewPrometheusMetricsCollector() *PrometheusMetricsCollector {
	return &PrometheusMetricsCollector{
		InMemoryMetricsCollector: NewInMemoryMetricsCollector(),
	}
}

// CustomMetricsMiddleware provides customizable metrics collection with user-defined metrics.
type CustomMetricsMiddleware struct {
	collector    MetricsCollector
	metricsDefs  map[string]MetricDefinition
	customLabels map[string]string
	mutex        sync.RWMutex
}

// MetricDefinition defines a custom metric.
type MetricDefinition struct {
	Name        string
	Type        MetricType
	Description string
	Labels      []string
	Buckets     []float64 // For histograms
}

// MetricType represents the type of metric.
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// NewCustomMetricsMiddleware creates a new custom metrics middleware.
func NewCustomMetricsMiddleware(collector MetricsCollector, definitions []MetricDefinition, customLabels map[string]string) *CustomMetricsMiddleware {
	metricsDefs := make(map[string]MetricDefinition)
	for _, def := range definitions {
		metricsDefs[def.Name] = def
	}

	if customLabels == nil {
		customLabels = make(map[string]string)
	}

	return &CustomMetricsMiddleware{
		collector:    collector,
		metricsDefs:  metricsDefs,
		customLabels: customLabels,
	}
}

// Name returns the middleware name.
func (cmm *CustomMetricsMiddleware) Name() string {
	return "custom-metrics"
}

// Wrap wraps a handler with custom metrics collection.
func (cmm *CustomMetricsMiddleware) Wrap(next messaging.MessageHandler) messaging.MessageHandler {
	return messaging.MessageHandlerFunc(func(ctx context.Context, message *messaging.Message) error {
		start := time.Now()

		// Build base labels
		labels := cmm.buildLabels(message)

		// Record custom metrics before processing
		cmm.recordPreProcessingMetrics(message, labels)

		err := next.Handle(ctx, message)

		processingTime := time.Since(start)

		// Record custom metrics after processing
		cmm.recordPostProcessingMetrics(message, labels, processingTime, err)

		return err
	})
}

// buildLabels constructs metric labels with custom labels and message information.
func (cmm *CustomMetricsMiddleware) buildLabels(message *messaging.Message) map[string]string {
	labels := make(map[string]string)

	// Add custom labels
	for k, v := range cmm.customLabels {
		labels[k] = v
	}

	// Add standard message labels
	labels["topic"] = message.Topic
	if len(message.Key) > 0 {
		labels["partition"] = string(message.Key)
	}

	return labels
}

// recordPreProcessingMetrics records metrics before message processing.
func (cmm *CustomMetricsMiddleware) recordPreProcessingMetrics(message *messaging.Message, labels map[string]string) {
	for name, def := range cmm.metricsDefs {
		switch def.Type {
		case MetricTypeCounter:
			if name == "messages_received" {
				cmm.collector.IncrementCounter(name, labels, 1)
			}
		case MetricTypeGauge:
			if name == "message_size" {
				cmm.collector.SetGauge(name, labels, float64(len(message.Payload)))
			}
		}
	}
}

// recordPostProcessingMetrics records metrics after message processing.
func (cmm *CustomMetricsMiddleware) recordPostProcessingMetrics(message *messaging.Message, labels map[string]string, processingTime time.Duration, err error) {
	for name, def := range cmm.metricsDefs {
		switch def.Type {
		case MetricTypeCounter:
			if name == "messages_processed" && err == nil {
				cmm.collector.IncrementCounter(name, labels, 1)
			} else if name == "messages_failed" && err != nil {
				cmm.collector.IncrementCounter(name, labels, 1)
			}
		case MetricTypeHistogram:
			if name == "processing_duration" {
				cmm.collector.ObserveHistogram(name, labels, processingTime.Seconds())
			}
		}
	}
}

// CreateMetricsMiddleware is a factory function to create metrics middleware from configuration.
func CreateMetricsMiddleware(config map[string]interface{}) (messaging.Middleware, error) {
	metricsConfig := DefaultMetricsConfig()

	// Configure collector type
	if collectorType, ok := config["collector_type"]; ok {
		if collectorTypeStr, ok := collectorType.(string); ok {
			switch collectorTypeStr {
			case "prometheus":
				metricsConfig.Collector = NewPrometheusMetricsCollector()
			case "in_memory":
				metricsConfig.Collector = NewInMemoryMetricsCollector()
			default:
				metricsConfig.Collector = NewInMemoryMetricsCollector()
			}
		}
	}

	// Configure enabled metrics
	if enabledMetrics, ok := config["enabled_metrics"]; ok {
		if enabledMetricsMap, ok := enabledMetrics.(map[string]interface{}); ok {
			metricsConfig.EnabledMetrics = make(map[string]bool)
			for k, v := range enabledMetricsMap {
				if enabled, ok := v.(bool); ok {
					metricsConfig.EnabledMetrics[k] = enabled
				}
			}
		}
	}

	// Configure custom labels
	if customLabels, ok := config["custom_labels"]; ok {
		if customLabelsMap, ok := customLabels.(map[string]interface{}); ok {
			metricsConfig.CustomLabels = make(map[string]string)
			for k, v := range customLabelsMap {
				if labelValue, ok := v.(string); ok {
					metricsConfig.CustomLabels[k] = labelValue
				}
			}
		}
	}

	return NewStandardMetricsMiddleware(metricsConfig), nil
}
