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
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
)

// PrometheusHTTPConfig configures the HTTP middleware for Prometheus metrics
type PrometheusHTTPConfig struct {
	// Enabled controls whether metrics collection is active
	Enabled bool `yaml:"enabled" json:"enabled"`

	// ExcludePaths contains paths to exclude from metrics (e.g., health checks)
	ExcludePaths []string `yaml:"exclude_paths" json:"exclude_paths"`

	// LabelSanitization controls whether to sanitize label values
	LabelSanitization bool `yaml:"label_sanitization" json:"label_sanitization"`

	// MaxPathCardinality limits the number of unique paths to prevent cardinality explosion
	MaxPathCardinality int `yaml:"max_path_cardinality" json:"max_path_cardinality"`

	// PathNormalization enables path normalization (e.g., /users/123 -> /users/:id)
	PathNormalization bool `yaml:"path_normalization" json:"path_normalization"`

	// CollectRequestSize enables request size metrics collection
	CollectRequestSize bool `yaml:"collect_request_size" json:"collect_request_size"`

	// CollectResponseSize enables response size metrics collection
	CollectResponseSize bool `yaml:"collect_response_size" json:"collect_response_size"`

	// ServiceName is the service name label for metrics
	ServiceName string `yaml:"service_name" json:"service_name"`
}

// DefaultPrometheusHTTPConfig returns sensible defaults for HTTP metrics middleware
func DefaultPrometheusHTTPConfig() *PrometheusHTTPConfig {
	return &PrometheusHTTPConfig{
		Enabled: true,
		ExcludePaths: []string{
			"/health",
			"/metrics",
			"/debug/health",
			"/debug/metrics",
		},
		LabelSanitization:   true,
		MaxPathCardinality:  1000,
		PathNormalization:   true,
		CollectRequestSize:  true,
		CollectResponseSize: true,
		ServiceName:         "unknown",
	}
}

// PrometheusHTTPMiddleware provides HTTP metrics collection middleware
type PrometheusHTTPMiddleware struct {
	config           *PrometheusHTTPConfig
	metricsCollector types.MetricsCollector
	pathCounter      map[string]int
}

// NewPrometheusHTTPMiddleware creates a new HTTP metrics middleware
func NewPrometheusHTTPMiddleware(collector types.MetricsCollector, config *PrometheusHTTPConfig) *PrometheusHTTPMiddleware {
	if config == nil {
		config = DefaultPrometheusHTTPConfig()
	}

	return &PrometheusHTTPMiddleware{
		config:           config,
		metricsCollector: collector,
		pathCounter:      make(map[string]int),
	}
}

// Middleware returns the Gin middleware function for HTTP metrics collection
func (m *PrometheusHTTPMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if metrics collection is disabled
		if !m.config.Enabled {
			c.Next()
			return
		}

		// Check if this path should be excluded
		if m.shouldExcludePath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Start timing the request
		start := time.Now()

		// Track active requests
		m.incrementActiveRequests()
		defer m.decrementActiveRequests()

		// Collect request size if enabled
		var requestSize int64
		if m.config.CollectRequestSize && c.Request.ContentLength > 0 {
			requestSize = c.Request.ContentLength
		}

		// Process the request
		c.Next()

		// Calculate request duration
		duration := time.Since(start)

		// Get normalized endpoint path
		endpoint := m.getNormalizedPath(c)

		// Collect metrics with error handling
		m.collectRequestMetrics(c, endpoint, duration, requestSize)
	}
}

// collectRequestMetrics collects all HTTP request metrics safely
func (m *PrometheusHTTPMiddleware) collectRequestMetrics(c *gin.Context, endpoint string, duration time.Duration, requestSize int64) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error("HTTP metrics collection panic",
				zap.String("endpoint", endpoint),
				zap.Any("panic", r))
		}
	}()

	method := c.Request.Method
	status := strconv.Itoa(c.Writer.Status())

	// Common labels for all metrics
	labels := map[string]string{
		"service":  m.config.ServiceName,
		"method":   method,
		"endpoint": endpoint,
	}

	// Collect request count with status
	countLabels := make(map[string]string)
	for k, v := range labels {
		countLabels[k] = v
	}
	countLabels["status"] = status

	m.safeIncrementCounter("http_requests_total", countLabels)

	// Collect request duration
	m.safeObserveHistogram("http_request_duration_seconds", duration.Seconds(), labels)

	// Collect request size if enabled and available
	if m.config.CollectRequestSize && requestSize > 0 {
		m.safeObserveHistogram("http_request_size_bytes", float64(requestSize), labels)
	}

	// Collect response size if enabled
	if m.config.CollectResponseSize {
		responseSize := float64(c.Writer.Size())
		if responseSize > 0 {
			m.safeObserveHistogram("http_response_size_bytes", responseSize, labels)
		}
	}
}

// incrementActiveRequests increments the active requests gauge
func (m *PrometheusHTTPMiddleware) incrementActiveRequests() {
	labels := map[string]string{
		"service": m.config.ServiceName,
	}
	m.safeIncrementGauge("http_active_requests", labels)
}

// decrementActiveRequests decrements the active requests gauge
func (m *PrometheusHTTPMiddleware) decrementActiveRequests() {
	labels := map[string]string{
		"service": m.config.ServiceName,
	}
	m.safeDecrementGauge("http_active_requests", labels)
}

// shouldExcludePath checks if a path should be excluded from metrics
func (m *PrometheusHTTPMiddleware) shouldExcludePath(path string) bool {
	for _, excludePath := range m.config.ExcludePaths {
		if strings.HasPrefix(path, excludePath) {
			return true
		}
	}
	return false
}

// getNormalizedPath returns the normalized path for metrics
func (m *PrometheusHTTPMiddleware) getNormalizedPath(c *gin.Context) string {
	// Try to get the route pattern if available
	if route := c.FullPath(); route != "" && m.config.PathNormalization {
		return m.sanitizePath(route)
	}

	// Fallback to request path
	path := c.Request.URL.Path

	// Apply path normalization if enabled
	if m.config.PathNormalization {
		path = m.normalizePath(path)
	}

	// Check cardinality limits
	if m.config.MaxPathCardinality > 0 {
		if len(m.pathCounter) >= m.config.MaxPathCardinality {
			// Use a generic "other" endpoint to prevent cardinality explosion
			return "/other"
		}
		m.pathCounter[path]++
	}

	return m.sanitizePath(path)
}

// normalizePath normalizes paths to reduce cardinality
func (m *PrometheusHTTPMiddleware) normalizePath(path string) string {
	// Split path into segments
	segments := strings.Split(strings.Trim(path, "/"), "/")

	for i, segment := range segments {
		// Replace numeric segments with :id placeholder
		if m.isNumeric(segment) {
			segments[i] = ":id"
		} else if m.isUUID(segment) {
			segments[i] = ":uuid"
		}
	}

	normalized := "/" + strings.Join(segments, "/")

	// Clean up double slashes
	normalized = strings.ReplaceAll(normalized, "//", "/")

	return normalized
}

// sanitizePath sanitizes path for use as metric label
func (m *PrometheusHTTPMiddleware) sanitizePath(path string) string {
	if !m.config.LabelSanitization {
		return path
	}

	// Remove or replace problematic characters
	sanitized := strings.ReplaceAll(path, "\"", "")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")
	sanitized = strings.ReplaceAll(sanitized, "\n", "")
	sanitized = strings.ReplaceAll(sanitized, "\r", "")

	return sanitized
}

// isNumeric checks if a string represents a number
func (m *PrometheusHTTPMiddleware) isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

// isUUID checks if a string looks like a UUID
func (m *PrometheusHTTPMiddleware) isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}

	// Simple UUID format check: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	parts := strings.Split(s, "-")
	return len(parts) == 5 &&
		len(parts[0]) == 8 &&
		len(parts[1]) == 4 &&
		len(parts[2]) == 4 &&
		len(parts[3]) == 4 &&
		len(parts[4]) == 12
}

// Safe metric collection methods with error handling

func (m *PrometheusHTTPMiddleware) safeIncrementCounter(name string, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("Counter increment panic",
				zap.String("metric", name),
				zap.Any("panic", r))
		}
	}()

	if m.metricsCollector != nil {
		m.metricsCollector.IncrementCounter(name, labels)
	}
}

func (m *PrometheusHTTPMiddleware) safeObserveHistogram(name string, value float64, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("Histogram observation panic",
				zap.String("metric", name),
				zap.Float64("value", value),
				zap.Any("panic", r))
		}
	}()

	if m.metricsCollector != nil {
		m.metricsCollector.ObserveHistogram(name, value, labels)
	}
}

func (m *PrometheusHTTPMiddleware) safeIncrementGauge(name string, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("Gauge increment panic",
				zap.String("metric", name),
				zap.Any("panic", r))
		}
	}()

	if m.metricsCollector != nil {
		m.metricsCollector.IncrementGauge(name, labels)
	}
}

func (m *PrometheusHTTPMiddleware) safeDecrementGauge(name string, labels map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Debug("Gauge decrement panic",
				zap.String("metric", name),
				zap.Any("panic", r))
		}
	}()

	if m.metricsCollector != nil {
		m.metricsCollector.DecrementGauge(name, labels)
	}
}

// GetConfig returns the middleware configuration
func (m *PrometheusHTTPMiddleware) GetConfig() *PrometheusHTTPConfig {
	return m.config
}

// UpdateConfig updates the middleware configuration (thread-safe for basic fields)
func (m *PrometheusHTTPMiddleware) UpdateConfig(config *PrometheusHTTPConfig) {
	if config == nil {
		return
	}

	// Only update fields that are safe to change at runtime
	m.config.Enabled = config.Enabled
	m.config.ExcludePaths = config.ExcludePaths
	m.config.CollectRequestSize = config.CollectRequestSize
	m.config.CollectResponseSize = config.CollectResponseSize
}

// GetPathCardinality returns the current path cardinality count
func (m *PrometheusHTTPMiddleware) GetPathCardinality() int {
	return len(m.pathCounter)
}

// ResetPathCounter resets the path counter (useful for testing)
func (m *PrometheusHTTPMiddleware) ResetPathCounter() {
	m.pathCounter = make(map[string]int)
}
