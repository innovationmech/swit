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
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// BusinessMetricsHook defines the interface for custom business metrics hooks
type BusinessMetricsHook interface {
	// OnMetricRecorded is called when a business metric is recorded
	OnMetricRecorded(event BusinessMetricEvent)

	// GetHookName returns a unique name for this hook
	GetHookName() string
}

// BusinessMetricEvent represents a business metric event
type BusinessMetricEvent struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     interface{}       `json:"value"`
	Labels    map[string]string `json:"labels"`
	Timestamp time.Time         `json:"timestamp"`
	Source    string            `json:"source"` // Service or component that recorded the metric
}

// BusinessMetricsManager manages custom business metrics and hooks
type BusinessMetricsManager struct {
	collector   MetricsCollector
	registry    *MetricsRegistry
	hooks       []BusinessMetricsHook
	serviceName string

	// Hook management
	mu          sync.RWMutex
	hooksByName map[string]BusinessMetricsHook
	hookSem     chan struct{} // Semaphore to limit concurrent hook executions
}

// NewBusinessMetricsManager creates a new business metrics manager
func NewBusinessMetricsManager(serviceName string, collector MetricsCollector, registry *MetricsRegistry) *BusinessMetricsManager {
	if collector == nil {
		collector = NewSimpleMetricsCollector()
	}
	if registry == nil {
		registry = NewMetricsRegistry()
	}

	return &BusinessMetricsManager{
		collector:   collector,
		registry:    registry,
		serviceName: serviceName,
		hooks:       make([]BusinessMetricsHook, 0),
		hooksByName: make(map[string]BusinessMetricsHook),
		hookSem:     make(chan struct{}, 10), // Limit to 10 concurrent hook executions
	}
}

// RegisterHook registers a business metrics hook
func (bmm *BusinessMetricsManager) RegisterHook(hook BusinessMetricsHook) error {
	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	hookName := hook.GetHookName()
	if hookName == "" {
		return fmt.Errorf("hook name cannot be empty")
	}

	bmm.mu.Lock()
	defer bmm.mu.Unlock()

	// Check if hook already exists
	if _, exists := bmm.hooksByName[hookName]; exists {
		return fmt.Errorf("hook with name %s already registered", hookName)
	}

	bmm.hooks = append(bmm.hooks, hook)
	bmm.hooksByName[hookName] = hook

	logger.Logger.Debug("Business metrics hook registered",
		zap.String("hook_name", hookName))

	return nil
}

// UnregisterHook removes a business metrics hook by name
func (bmm *BusinessMetricsManager) UnregisterHook(hookName string) error {
	bmm.mu.Lock()
	defer bmm.mu.Unlock()

	hook, exists := bmm.hooksByName[hookName]
	if !exists {
		return fmt.Errorf("hook with name %s not found", hookName)
	}

	// Remove from hooks slice
	for i, h := range bmm.hooks {
		if h == hook {
			bmm.hooks = append(bmm.hooks[:i], bmm.hooks[i+1:]...)
			break
		}
	}

	// Remove from map
	delete(bmm.hooksByName, hookName)

	logger.Logger.Debug("Business metrics hook unregistered",
		zap.String("hook_name", hookName))

	return nil
}

// GetRegisteredHooks returns the names of all registered hooks
func (bmm *BusinessMetricsManager) GetRegisteredHooks() []string {
	bmm.mu.RLock()
	defer bmm.mu.RUnlock()

	names := make([]string, 0, len(bmm.hooksByName))
	for name := range bmm.hooksByName {
		names = append(names, name)
	}

	return names
}

// RecordCounter records a counter metric and triggers hooks
func (bmm *BusinessMetricsManager) RecordCounter(name string, value float64, labels map[string]string) {
	// Add service label if not present
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, exists := labels["service"]; !exists {
		labels["service"] = bmm.serviceName
	}

	// Record in collector
	if value == 1.0 {
		bmm.collector.IncrementCounter(name, labels)
	} else {
		bmm.collector.AddToCounter(name, value, labels)
	}

	// Create and trigger event
	event := BusinessMetricEvent{
		Name:      name,
		Type:      MetricTypeCounter,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		Source:    bmm.serviceName,
	}

	bmm.triggerHooks(event)
}

// RecordGauge records a gauge metric and triggers hooks
func (bmm *BusinessMetricsManager) RecordGauge(name string, value float64, labels map[string]string) {
	// Add service label if not present
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, exists := labels["service"]; !exists {
		labels["service"] = bmm.serviceName
	}

	// Record in collector
	bmm.collector.SetGauge(name, value, labels)

	// Create and trigger event
	event := BusinessMetricEvent{
		Name:      name,
		Type:      MetricTypeGauge,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		Source:    bmm.serviceName,
	}

	bmm.triggerHooks(event)
}

// RecordHistogram records a histogram metric and triggers hooks
func (bmm *BusinessMetricsManager) RecordHistogram(name string, value float64, labels map[string]string) {
	// Add service label if not present
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, exists := labels["service"]; !exists {
		labels["service"] = bmm.serviceName
	}

	// Record in collector
	bmm.collector.ObserveHistogram(name, value, labels)

	// Create and trigger event
	event := BusinessMetricEvent{
		Name:      name,
		Type:      MetricTypeHistogram,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		Source:    bmm.serviceName,
	}

	bmm.triggerHooks(event)
}

// RegisterCustomMetric registers a custom metric definition
func (bmm *BusinessMetricsManager) RegisterCustomMetric(definition MetricDefinition) error {
	return bmm.registry.RegisterMetric(definition)
}

// GetCustomMetric retrieves a custom metric definition
func (bmm *BusinessMetricsManager) GetCustomMetric(name string) (*MetricDefinition, bool) {
	return bmm.registry.GetMetricDefinition(name)
}

// GetAllMetrics returns all metric definitions (predefined + custom)
func (bmm *BusinessMetricsManager) GetAllMetrics() map[string]MetricDefinition {
	// Since MetricsRegistry doesn't have GetAllMetrics, we'll build it
	allMetrics := make(map[string]MetricDefinition)

	// Get predefined metrics
	predefinedNames := bmm.registry.GetPredefinedMetricNames()
	for _, name := range predefinedNames {
		if def, exists := bmm.registry.GetMetricDefinition(name); exists {
			allMetrics[name] = *def
		}
	}

	// Get custom metrics
	customNames := bmm.registry.GetCustomMetricNames()
	for _, name := range customNames {
		if def, exists := bmm.registry.GetMetricDefinition(name); exists {
			allMetrics[name] = *def
		}
	}

	return allMetrics
}

// GetCollector returns the underlying metrics collector
func (bmm *BusinessMetricsManager) GetCollector() MetricsCollector {
	return bmm.collector
}

// GetRegistry returns the metrics registry
func (bmm *BusinessMetricsManager) GetRegistry() *MetricsRegistry {
	return bmm.registry
}

// triggerHooks safely triggers all registered hooks
func (bmm *BusinessMetricsManager) triggerHooks(event BusinessMetricEvent) {
	bmm.mu.RLock()
	hooks := make([]BusinessMetricsHook, len(bmm.hooks))
	copy(hooks, bmm.hooks)
	bmm.mu.RUnlock()

	// Trigger hooks concurrently to avoid blocking, with concurrency control
	for _, hook := range hooks {
		go func(h BusinessMetricsHook) {
			// Acquire semaphore before executing hook
			bmm.hookSem <- struct{}{}
			defer func() {
				// Release semaphore and handle panics
				<-bmm.hookSem
				if r := recover(); r != nil {
					logger.Logger.Error("Business metrics hook panicked",
						zap.String("hook_name", h.GetHookName()),
						zap.String("metric_name", event.Name),
						zap.Any("panic", r))
				}
			}()

			h.OnMetricRecorded(event)
		}(hook)
	}
}

// DefaultBusinessMetricsHooks provides common hook implementations

// LoggingBusinessMetricsHook logs all business metric events
type LoggingBusinessMetricsHook struct {
	name string
}

// NewLoggingBusinessMetricsHook creates a new logging hook
func NewLoggingBusinessMetricsHook() *LoggingBusinessMetricsHook {
	return &LoggingBusinessMetricsHook{
		name: "logging_hook",
	}
}

// OnMetricRecorded logs the metric event
func (lbmh *LoggingBusinessMetricsHook) OnMetricRecorded(event BusinessMetricEvent) {
	logger.Logger.Debug("Business metric recorded",
		zap.String("name", event.Name),
		zap.String("type", string(event.Type)),
		zap.Any("value", event.Value),
		zap.Any("labels", event.Labels),
		zap.String("source", event.Source),
		zap.Time("timestamp", event.Timestamp))
}

// GetHookName returns the hook name
func (lbmh *LoggingBusinessMetricsHook) GetHookName() string {
	return lbmh.name
}

// AggregationBusinessMetricsHook aggregates metrics for summary reporting
type AggregationBusinessMetricsHook struct {
	name     string
	counters map[string]float64
	gauges   map[string]float64
	mu       sync.RWMutex
}

// NewAggregationBusinessMetricsHook creates a new aggregation hook
func NewAggregationBusinessMetricsHook() *AggregationBusinessMetricsHook {
	return &AggregationBusinessMetricsHook{
		name:     "aggregation_hook",
		counters: make(map[string]float64),
		gauges:   make(map[string]float64),
	}
}

// OnMetricRecorded aggregates the metric value
func (abmh *AggregationBusinessMetricsHook) OnMetricRecorded(event BusinessMetricEvent) {
	abmh.mu.Lock()
	defer abmh.mu.Unlock()

	key := fmt.Sprintf("%s_%s", event.Source, event.Name)

	switch event.Type {
	case MetricTypeCounter:
		if value, ok := event.Value.(float64); ok {
			abmh.counters[key] += value
		}
	case MetricTypeGauge:
		if value, ok := event.Value.(float64); ok {
			abmh.gauges[key] = value // Gauges are set, not added
		}
	}
}

// GetHookName returns the hook name
func (abmh *AggregationBusinessMetricsHook) GetHookName() string {
	return abmh.name
}

// GetCounterTotals returns the aggregated counter totals
func (abmh *AggregationBusinessMetricsHook) GetCounterTotals() map[string]float64 {
	abmh.mu.RLock()
	defer abmh.mu.RUnlock()

	totals := make(map[string]float64)
	for k, v := range abmh.counters {
		totals[k] = v
	}
	return totals
}

// GetGaugeValues returns the current gauge values
func (abmh *AggregationBusinessMetricsHook) GetGaugeValues() map[string]float64 {
	abmh.mu.RLock()
	defer abmh.mu.RUnlock()

	values := make(map[string]float64)
	for k, v := range abmh.gauges {
		values[k] = v
	}
	return values
}

// Reset clears all aggregated data
func (abmh *AggregationBusinessMetricsHook) Reset() {
	abmh.mu.Lock()
	defer abmh.mu.Unlock()

	abmh.counters = make(map[string]float64)
	abmh.gauges = make(map[string]float64)
}
