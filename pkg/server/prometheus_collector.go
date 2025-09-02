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
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
)

// PrometheusConfig holds configuration for Prometheus metrics integration
type PrometheusConfig struct {
	Enabled   bool              `yaml:"enabled" json:"enabled"`
	Endpoint  string            `yaml:"endpoint" json:"endpoint"`
	Namespace string            `yaml:"namespace" json:"namespace"`
	Subsystem string            `yaml:"subsystem" json:"subsystem"`
	Buckets   PrometheusBuckets `yaml:"buckets" json:"buckets"`
	Labels    map[string]string `yaml:"labels" json:"labels"`
}

// PrometheusBuckets defines bucket configurations for histograms
type PrometheusBuckets struct {
	Duration []float64 `yaml:"duration" json:"duration"`
	Size     []float64 `yaml:"size" json:"size"`
}

// DefaultPrometheusConfig returns a default Prometheus configuration
func DefaultPrometheusConfig() *PrometheusConfig {
	return &PrometheusConfig{
		Enabled:   true,
		Endpoint:  "/metrics",
		Namespace: "swit",
		Subsystem: "server",
		Buckets: PrometheusBuckets{
			Duration: []float64{0.001, 0.01, 0.1, 0.5, 1, 2.5, 5, 10},
			Size:     []float64{100, 1000, 10000, 100000, 1000000},
		},
		Labels: make(map[string]string),
	}
}

// PrometheusMetricsCollector implements MetricsCollector using Prometheus client
type PrometheusMetricsCollector struct {
	config     *PrometheusConfig
	registry   *prometheus.Registry
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
	summaries  map[string]*prometheus.SummaryVec
	mu         sync.RWMutex

	// Cardinality limiting
	maxCardinality int
	currentMetrics map[string]int
	cardinalityMu  sync.RWMutex
}

// NewPrometheusMetricsCollector creates a new Prometheus-backed metrics collector
func NewPrometheusMetricsCollector(config *PrometheusConfig) *PrometheusMetricsCollector {
	if config == nil || !config.Enabled {
		config = DefaultPrometheusConfig()
	}

	registry := prometheus.NewRegistry()

	return &PrometheusMetricsCollector{
		config:         config,
		registry:       registry,
		counters:       make(map[string]*prometheus.CounterVec),
		gauges:         make(map[string]*prometheus.GaugeVec),
		histograms:     make(map[string]*prometheus.HistogramVec),
		summaries:      make(map[string]*prometheus.SummaryVec),
		maxCardinality: 10000, // Default cardinality limit
		currentMetrics: make(map[string]int),
	}
}

// IncrementCounter increments a counter metric by 1
func (pmc *PrometheusMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	pmc.AddToCounter(name, 1, labels)
}

// AddToCounter adds a value to a counter metric
func (pmc *PrometheusMetricsCollector) AddToCounter(name string, value float64, labels map[string]string) {
	pmc.safeMetricOperation(func() error {
		counter, err := pmc.getOrCreateCounter(name, labels)
		if err != nil {
			return err
		}

		labelValues := pmc.extractLabelValues(labels, counter)
		counter.WithLabelValues(labelValues...).Add(value)
		return nil
	})
}

// SetGauge sets a gauge metric to a specific value
func (pmc *PrometheusMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	pmc.safeMetricOperation(func() error {
		gauge, err := pmc.getOrCreateGauge(name, labels)
		if err != nil {
			return err
		}

		labelValues := pmc.extractLabelValues(labels, gauge)
		gauge.WithLabelValues(labelValues...).Set(value)
		return nil
	})
}

// IncrementGauge increments a gauge metric by 1
func (pmc *PrometheusMetricsCollector) IncrementGauge(name string, labels map[string]string) {
	pmc.safeMetricOperation(func() error {
		gauge, err := pmc.getOrCreateGauge(name, labels)
		if err != nil {
			return err
		}

		labelValues := pmc.extractLabelValues(labels, gauge)
		gauge.WithLabelValues(labelValues...).Inc()
		return nil
	})
}

// DecrementGauge decrements a gauge metric by 1
func (pmc *PrometheusMetricsCollector) DecrementGauge(name string, labels map[string]string) {
	pmc.safeMetricOperation(func() error {
		gauge, err := pmc.getOrCreateGauge(name, labels)
		if err != nil {
			return err
		}

		labelValues := pmc.extractLabelValues(labels, gauge)
		gauge.WithLabelValues(labelValues...).Dec()
		return nil
	})
}

// ObserveHistogram adds an observation to a histogram metric
func (pmc *PrometheusMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	pmc.safeMetricOperation(func() error {
		histogram, err := pmc.getOrCreateHistogram(name, labels)
		if err != nil {
			return err
		}

		labelValues := pmc.extractLabelValues(labels, histogram)
		histogram.WithLabelValues(labelValues...).Observe(value)
		return nil
	})
}

// GetMetrics returns all collected metrics (for backward compatibility)
func (pmc *PrometheusMetricsCollector) GetMetrics() []Metric {
	pmc.mu.RLock()
	defer pmc.mu.RUnlock()

	var metrics []Metric

	// Gather metrics from Prometheus registry
	metricFamilies, err := pmc.registry.Gather()
	if err != nil {
		logger.Logger.Error("Failed to gather Prometheus metrics", zap.Error(err))
		return metrics
	}

	// Convert Prometheus metrics to our Metric format
	for _, mf := range metricFamilies {
		metricName := mf.GetName()

		for _, m := range mf.GetMetric() {
			labels := make(map[string]string)
			for _, lp := range m.GetLabel() {
				labels[lp.GetName()] = lp.GetValue()
			}

			var value interface{}
			var metricType MetricType

			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				value = m.GetCounter().GetValue()
				metricType = MetricTypeCounter
			case dto.MetricType_GAUGE:
				value = m.GetGauge().GetValue()
				metricType = MetricTypeGauge
			case dto.MetricType_HISTOGRAM:
				hist := m.GetHistogram()
				value = map[string]interface{}{
					"count": hist.GetSampleCount(),
					"sum":   hist.GetSampleSum(),
				}
				metricType = MetricTypeHistogram
			case dto.MetricType_SUMMARY:
				summ := m.GetSummary()
				value = map[string]interface{}{
					"count": summ.GetSampleCount(),
					"sum":   summ.GetSampleSum(),
				}
				metricType = MetricTypeSummary
			default:
				continue
			}

			metrics = append(metrics, Metric{
				Name:      metricName,
				Type:      metricType,
				Value:     value,
				Labels:    labels,
				Timestamp: time.Now(),
			})
		}
	}

	return metrics
}

// GetMetric returns a specific metric by name (for backward compatibility)
func (pmc *PrometheusMetricsCollector) GetMetric(name string) (*Metric, bool) {
	metrics := pmc.GetMetrics()
	for _, metric := range metrics {
		if metric.Name == name {
			return &metric, true
		}
	}
	return nil, false
}

// Reset clears all metrics
func (pmc *PrometheusMetricsCollector) Reset() {
	pmc.mu.Lock()
	defer pmc.mu.Unlock()

	// Create a new registry
	pmc.registry = prometheus.NewRegistry()

	// Clear all metric maps
	pmc.counters = make(map[string]*prometheus.CounterVec)
	pmc.gauges = make(map[string]*prometheus.GaugeVec)
	pmc.histograms = make(map[string]*prometheus.HistogramVec)
	pmc.summaries = make(map[string]*prometheus.SummaryVec)

	// Reset cardinality tracking
	pmc.cardinalityMu.Lock()
	pmc.currentMetrics = make(map[string]int)
	pmc.cardinalityMu.Unlock()
}

// GetHandler returns the HTTP handler for Prometheus metrics endpoint
func (pmc *PrometheusMetricsCollector) GetHandler() http.Handler {
	return promhttp.HandlerFor(pmc.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		ErrorHandling:     promhttp.ContinueOnError,
	})
}

// GetRegistry returns the Prometheus registry for advanced usage
func (pmc *PrometheusMetricsCollector) GetRegistry() *prometheus.Registry {
	return pmc.registry
}

// getOrCreateCounter gets or creates a counter metric
func (pmc *PrometheusMetricsCollector) getOrCreateCounter(name string, labels map[string]string) (*prometheus.CounterVec, error) {
	pmc.mu.RLock()
	counter, exists := pmc.counters[name]
	pmc.mu.RUnlock()

	if exists {
		return counter, nil
	}

	if !pmc.checkCardinality(name, labels) {
		return nil, fmt.Errorf("cardinality limit exceeded for metric %s", name)
	}

	pmc.mu.Lock()
	defer pmc.mu.Unlock()

	// Double-check after acquiring write lock
	if counter, exists := pmc.counters[name]; exists {
		return counter, nil
	}

	labelNames := pmc.getLabelNames(labels)
	counter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: pmc.config.Namespace,
		Subsystem: pmc.config.Subsystem,
		Name:      pmc.sanitizeMetricName(name),
		Help:      fmt.Sprintf("Counter metric %s", name),
	}, labelNames)

	if err := pmc.registry.Register(counter); err != nil {
		return nil, fmt.Errorf("failed to register counter %s: %w", name, err)
	}

	pmc.counters[name] = counter
	return counter, nil
}

// getOrCreateGauge gets or creates a gauge metric
func (pmc *PrometheusMetricsCollector) getOrCreateGauge(name string, labels map[string]string) (*prometheus.GaugeVec, error) {
	pmc.mu.RLock()
	gauge, exists := pmc.gauges[name]
	pmc.mu.RUnlock()

	if exists {
		return gauge, nil
	}

	if !pmc.checkCardinality(name, labels) {
		return nil, fmt.Errorf("cardinality limit exceeded for metric %s", name)
	}

	pmc.mu.Lock()
	defer pmc.mu.Unlock()

	// Double-check after acquiring write lock
	if gauge, exists := pmc.gauges[name]; exists {
		return gauge, nil
	}

	labelNames := pmc.getLabelNames(labels)
	gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: pmc.config.Namespace,
		Subsystem: pmc.config.Subsystem,
		Name:      pmc.sanitizeMetricName(name),
		Help:      fmt.Sprintf("Gauge metric %s", name),
	}, labelNames)

	if err := pmc.registry.Register(gauge); err != nil {
		return nil, fmt.Errorf("failed to register gauge %s: %w", name, err)
	}

	pmc.gauges[name] = gauge
	return gauge, nil
}

// getOrCreateHistogram gets or creates a histogram metric
func (pmc *PrometheusMetricsCollector) getOrCreateHistogram(name string, labels map[string]string) (*prometheus.HistogramVec, error) {
	pmc.mu.RLock()
	histogram, exists := pmc.histograms[name]
	pmc.mu.RUnlock()

	if exists {
		return histogram, nil
	}

	if !pmc.checkCardinality(name, labels) {
		return nil, fmt.Errorf("cardinality limit exceeded for metric %s", name)
	}

	pmc.mu.Lock()
	defer pmc.mu.Unlock()

	// Double-check after acquiring write lock
	if histogram, exists := pmc.histograms[name]; exists {
		return histogram, nil
	}

	labelNames := pmc.getLabelNames(labels)
	buckets := pmc.getBucketsForMetric(name)

	histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: pmc.config.Namespace,
		Subsystem: pmc.config.Subsystem,
		Name:      pmc.sanitizeMetricName(name),
		Help:      fmt.Sprintf("Histogram metric %s", name),
		Buckets:   buckets,
	}, labelNames)

	if err := pmc.registry.Register(histogram); err != nil {
		return nil, fmt.Errorf("failed to register histogram %s: %w", name, err)
	}

	pmc.histograms[name] = histogram
	return histogram, nil
}

// getLabelNames extracts and sorts label names from a label map
func (pmc *PrometheusMetricsCollector) getLabelNames(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}

	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// extractLabelValues extracts label values in the correct order for a metric
func (pmc *PrometheusMetricsCollector) extractLabelValues(labels map[string]string, metric prometheus.Collector) []string {
	// This is a simplified version - in a real implementation, you'd need to
	// extract the label names from the metric descriptor and match them
	if len(labels) == 0 {
		return nil
	}

	// Sort label names to ensure consistent ordering
	names := make([]string, 0, len(labels))
	for name := range labels {
		names = append(names, name)
	}
	sort.Strings(names)

	values := make([]string, len(names))
	for i, name := range names {
		values[i] = labels[name]
	}
	return values
}

// sanitizeMetricName ensures metric names are valid for Prometheus
func (pmc *PrometheusMetricsCollector) sanitizeMetricName(name string) string {
	// Replace invalid characters with underscores
	sanitized := strings.ReplaceAll(name, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Ensure it doesn't start with a number
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "metric_" + sanitized
	}

	return sanitized
}

// getBucketsForMetric returns appropriate buckets for a metric based on its name
func (pmc *PrometheusMetricsCollector) getBucketsForMetric(name string) []float64 {
	name = strings.ToLower(name)

	// Use duration buckets for time-related metrics
	if strings.Contains(name, "duration") || strings.Contains(name, "time") ||
		strings.Contains(name, "latency") || strings.Contains(name, "seconds") {
		return pmc.config.Buckets.Duration
	}

	// Use size buckets for size-related metrics
	if strings.Contains(name, "size") || strings.Contains(name, "bytes") ||
		strings.Contains(name, "length") {
		return pmc.config.Buckets.Size
	}

	// Default to duration buckets
	return pmc.config.Buckets.Duration
}

// checkCardinality checks if adding this metric would exceed cardinality limits
func (pmc *PrometheusMetricsCollector) checkCardinality(metricName string, labels map[string]string) bool {
	if pmc.maxCardinality <= 0 {
		return true // No limit
	}

	pmc.cardinalityMu.RLock()
	current := pmc.currentMetrics[metricName]
	pmc.cardinalityMu.RUnlock()

	// Simple cardinality check - in practice, you'd want more sophisticated tracking
	if current >= 1000 { // Per-metric limit
		logger.Logger.Warn("Metric cardinality limit approaching",
			zap.String("metric", metricName),
			zap.Int("current", current))
		return false
	}

	pmc.cardinalityMu.Lock()
	pmc.currentMetrics[metricName]++
	pmc.cardinalityMu.Unlock()

	return true
}

// safeMetricOperation wraps metric operations with error handling
func (pmc *PrometheusMetricsCollector) safeMetricOperation(operation func() error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error("Prometheus metric operation panicked",
				zap.Any("panic", r))
		}
	}()

	if err := operation(); err != nil {
		logger.Logger.Debug("Prometheus metric operation failed",
			zap.Error(err))
	}
}
