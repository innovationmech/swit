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
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/innovationmech/swit/pkg/types"
)

// mockMetricsCollector implements types.MetricsCollector for testing
type mockMetricsCollector struct {
	mu         sync.RWMutex
	counters   map[string]float64
	gauges     map[string]float64
	histograms map[string][]float64
	labels     map[string]map[string]string
	panics     bool
}

func newMockMetricsCollector() *mockMetricsCollector {
	return &mockMetricsCollector{
		counters:   make(map[string]float64),
		gauges:     make(map[string]float64),
		histograms: make(map[string][]float64),
		labels:     make(map[string]map[string]string),
	}
}

func (m *mockMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.counters[key]++
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockMetricsCollector) AddToCounter(name string, value float64, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.counters[key] += value
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.gauges[key] = value
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockMetricsCollector) IncrementGauge(name string, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.gauges[key]++
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockMetricsCollector) DecrementGauge(name string, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.gauges[key]--
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	if m.panics {
		panic("test panic")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	key := m.getKey(name, labels)
	m.histograms[key] = append(m.histograms[key], value)
	m.labels[key] = m.copyLabels(labels)
}

func (m *mockMetricsCollector) GetMetrics() []types.Metric {
	return nil
}

func (m *mockMetricsCollector) GetMetric(name string) (*types.Metric, bool) {
	return nil, false
}

func (m *mockMetricsCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.counters = make(map[string]float64)
	m.gauges = make(map[string]float64)
	m.histograms = make(map[string][]float64)
	m.labels = make(map[string]map[string]string)
}

func (m *mockMetricsCollector) getKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Sort label keys for deterministic key generation
	var keys []string
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	parts = append(parts, name)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
	}
	return strings.Join(parts, ",")
}

func (m *mockMetricsCollector) copyLabels(labels map[string]string) map[string]string {
	copied := make(map[string]string)
	for k, v := range labels {
		copied[k] = v
	}
	return copied
}

// Helper methods for testing
func (m *mockMetricsCollector) getCounterValue(name string, labels map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getKey(name, labels)
	return m.counters[key]
}

func (m *mockMetricsCollector) getGaugeValue(name string, labels map[string]string) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getKey(name, labels)
	return m.gauges[key]
}

func (m *mockMetricsCollector) getHistogramValues(name string, labels map[string]string) []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := m.getKey(name, labels)
	values := make([]float64, len(m.histograms[key]))
	copy(values, m.histograms[key])
	return values
}

func (m *mockMetricsCollector) setPanics(panics bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.panics = panics
}

func TestDefaultPrometheusHTTPConfig(t *testing.T) {
	config := DefaultPrometheusHTTPConfig()

	assert.True(t, config.Enabled)
	assert.True(t, config.LabelSanitization)
	assert.True(t, config.PathNormalization)
	assert.True(t, config.CollectRequestSize)
	assert.True(t, config.CollectResponseSize)
	assert.Equal(t, 1000, config.MaxPathCardinality)
	assert.Equal(t, "unknown", config.ServiceName)

	expectedExcludePaths := []string{
		"/health",
		"/metrics",
		"/debug/health",
		"/debug/metrics",
	}
	assert.Equal(t, expectedExcludePaths, config.ExcludePaths)
}

func TestNewPrometheusHTTPMiddleware(t *testing.T) {
	collector := newMockMetricsCollector()

	t.Run("WithConfig", func(t *testing.T) {
		config := &PrometheusHTTPConfig{
			Enabled:     true,
			ServiceName: "test-service",
		}

		middleware := NewPrometheusHTTPMiddleware(collector, config)
		assert.NotNil(t, middleware)
		assert.Equal(t, config, middleware.config)
		assert.Equal(t, collector, middleware.metricsCollector)
		// Registry is no longer exposed in the middleware
		assert.NotNil(t, middleware.pathCounter)
	})

	t.Run("WithNilConfig", func(t *testing.T) {
		middleware := NewPrometheusHTTPMiddleware(collector, nil)
		assert.NotNil(t, middleware)
		assert.NotNil(t, middleware.config)
		assert.Equal(t, DefaultPrometheusHTTPConfig(), middleware.config)
	})
}

func TestPrometheusHTTPMiddleware_BasicMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:             true,
		ServiceName:         "test-service",
		CollectRequestSize:  true,
		CollectResponseSize: true,
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make a request
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify metrics were collected
	expectedLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users",
	}

	// Check request count with status
	countLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users",
		"status":   "200",
	}
	assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", countLabels))

	// Check request duration
	durations := collector.getHistogramValues("http_request_duration_seconds", expectedLabels)
	assert.Len(t, durations, 1)
	assert.Greater(t, durations[0], float64(0))

	// Check response size
	responseSizes := collector.getHistogramValues("http_response_size_bytes", expectedLabels)
	assert.Len(t, responseSizes, 1)
	assert.Greater(t, responseSizes[0], float64(0))

	// Check active requests (should be back to 0)
	activeLabels := map[string]string{"service": "test-service"}
	assert.Equal(t, float64(0), collector.getGaugeValue("http_active_requests", activeLabels))
}

func TestPrometheusHTTPMiddleware_DifferentHTTPMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:           true,
		ServiceName:       "test-service",
		PathNormalization: true,
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/users", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	router.POST("/api/users", func(c *gin.Context) { c.JSON(201, gin.H{}) })
	router.PUT("/api/users/:id", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	router.DELETE("/api/users/:id", func(c *gin.Context) { c.JSON(204, gin.H{}) })

	tests := []struct {
		method       string
		path         string
		expectedPath string
		status       int
	}{
		{"GET", "/api/users", "/api/users", 200},
		{"POST", "/api/users", "/api/users", 201},
		{"PUT", "/api/users/123", "/api/users/:id", 200},
		{"DELETE", "/api/users/456", "/api/users/:id", 204},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			collector.Reset() // Reset for each test

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			countLabels := map[string]string{
				"service":  "test-service",
				"method":   tt.method,
				"endpoint": tt.expectedPath,
				"status":   fmt.Sprintf("%d", tt.status),
			}

			assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", countLabels))
		})
	}
}

func TestPrometheusHTTPMiddleware_StatusCodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	middleware := NewPrometheusHTTPMiddleware(collector, &PrometheusHTTPConfig{
		Enabled:     true,
		ServiceName: "test-service",
	})

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/success", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	router.GET("/created", func(c *gin.Context) { c.JSON(201, gin.H{}) })
	router.GET("/badrequest", func(c *gin.Context) { c.JSON(400, gin.H{}) })
	router.GET("/notfound", func(c *gin.Context) { c.JSON(404, gin.H{}) })
	router.GET("/error", func(c *gin.Context) { c.JSON(500, gin.H{}) })

	statuses := []int{200, 201, 400, 404, 500}

	for _, status := range statuses {
		path := ""
		switch status {
		case 200:
			path = "/success"
		case 201:
			path = "/created"
		case 400:
			path = "/badrequest"
		case 404:
			path = "/notfound"
		case 500:
			path = "/error"
		}

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		countLabels := map[string]string{
			"service":  "test-service",
			"method":   "GET",
			"endpoint": path,
			"status":   fmt.Sprintf("%d", status),
		}

		assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", countLabels))
	}
}

func TestPrometheusHTTPMiddleware_ExcludePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:      true,
		ServiceName:  "test-service",
		ExcludePaths: []string{"/health", "/metrics"},
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	router.GET("/metrics", func(c *gin.Context) { c.JSON(200, gin.H{}) })
	router.GET("/api/users", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Request excluded paths
	req1 := httptest.NewRequest("GET", "/health", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/metrics", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Request non-excluded path
	req3 := httptest.NewRequest("GET", "/api/users", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	// Verify excluded paths don't have metrics
	excludedLabels1 := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/health",
		"status":   "200",
	}
	excludedLabels2 := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/metrics",
		"status":   "200",
	}

	assert.Equal(t, float64(0), collector.getCounterValue("http_requests_total", excludedLabels1))
	assert.Equal(t, float64(0), collector.getCounterValue("http_requests_total", excludedLabels2))

	// Verify non-excluded path has metrics
	includedLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users",
		"status":   "200",
	}
	assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", includedLabels))
}

func TestPrometheusHTTPMiddleware_PathNormalization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:           true,
		ServiceName:       "test-service",
		PathNormalization: true,
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	// Use a catch-all route to test manual path normalization
	// This simulates cases where c.FullPath() doesn't return a useful pattern
	router := gin.New()
	router.Use(middleware.Middleware())
	router.NoRoute(func(c *gin.Context) { c.JSON(200, gin.H{}) })

	tests := []struct {
		path         string
		expectedPath string
	}{
		{"/api/users/123", "/api/users/:id"},
		{"/api/users/456", "/api/users/:id"},
		{"/api/users/abc-def-ghi", "/api/users/abc-def-ghi"}, // Not normalized (not numeric or UUID)
		{"/api/users/f47ac10b-58cc-4372-a567-0e02b2c3d479", "/api/users/:uuid"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})
	}

	// Count requests by normalized endpoint
	countLabelsId := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users/:id",
		"status":   "200",
	}

	countLabelsUuid := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users/:uuid",
		"status":   "200",
	}

	// Should have 2 requests with :id normalization (123, 456)
	assert.Equal(t, float64(2), collector.getCounterValue("http_requests_total", countLabelsId))

	// Should have 1 request with :uuid normalization
	assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", countLabelsUuid))

	// Should have 1 request with no normalization (abc-def-ghi)
	countLabelsLiteral := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users/abc-def-ghi",
		"status":   "200",
	}
	assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", countLabelsLiteral))
}

func TestPrometheusHTTPMiddleware_CardinalityLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:            true,
		ServiceName:        "test-service",
		MaxPathCardinality: 2, // Low limit for testing
		PathNormalization:  false,
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/*path", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Make requests that would exceed cardinality
	paths := []string{"/path1", "/path2", "/path3", "/path4"}
	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// First two paths should be tracked normally
	for i := 0; i < 2; i++ {
		labels := map[string]string{
			"service":  "test-service",
			"method":   "GET",
			"endpoint": paths[i],
			"status":   "200",
		}
		assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", labels))
	}

	// Remaining paths should be collapsed to "/other"
	otherLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/other",
		"status":   "200",
	}
	assert.Equal(t, float64(2), collector.getCounterValue("http_requests_total", otherLabels))

	// Verify cardinality
	assert.Equal(t, 2, middleware.GetPathCardinality())
}

func TestPrometheusHTTPMiddleware_RequestAndResponseSize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:             true,
		ServiceName:         "test-service",
		CollectRequestSize:  true,
		CollectResponseSize: true,
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.POST("/api/data", func(c *gin.Context) {
		c.JSON(200, gin.H{"result": "processed", "data": "response body content"})
	})

	// Create request with body
	body := `{"test": "data", "content": "request body content"}`
	req := httptest.NewRequest("POST", "/api/data", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	expectedLabels := map[string]string{
		"service":  "test-service",
		"method":   "POST",
		"endpoint": "/api/data",
	}

	// Check request size was recorded
	requestSizes := collector.getHistogramValues("http_request_size_bytes", expectedLabels)
	assert.Len(t, requestSizes, 1)
	assert.Equal(t, float64(len(body)), requestSizes[0])

	// Check response size was recorded
	responseSizes := collector.getHistogramValues("http_response_size_bytes", expectedLabels)
	assert.Len(t, responseSizes, 1)
	assert.Greater(t, responseSizes[0], float64(0))
}

func TestPrometheusHTTPMiddleware_LabelSanitization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:           true,
		ServiceName:       "test-service",
		LabelSanitization: true,
		PathNormalization: false,
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/*path", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Test path sanitization directly in middleware by making a valid request
	// and checking that our sanitize function works
	validPath := "/api/test-path"
	req := httptest.NewRequest("GET", validPath, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check that the path is recorded correctly
	countLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": validPath,
		"status":   "200",
	}

	assert.Equal(t, float64(1), collector.getCounterValue("http_requests_total", countLabels))

	// Test the sanitization function directly
	unsanitizedPath := "/api/test\"with\\quotes\nand\rreturns"
	sanitized := middleware.sanitizePath(unsanitizedPath)
	expected := "/api/testwithquotesandreturns"
	assert.Equal(t, expected, sanitized)
}

func TestPrometheusHTTPMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:     false,
		ServiceName: "test-service",
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/users", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// No metrics should be collected when disabled
	countLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users",
		"status":   "200",
	}

	assert.Equal(t, float64(0), collector.getCounterValue("http_requests_total", countLabels))
}

func TestPrometheusHTTPMiddleware_PanicRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/users", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Enable panics in the collector
	collector.setPanics(true)

	// Request should still complete despite panic in metrics collection
	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	assert.Equal(t, 200, w.Code)
}

func TestPrometheusHTTPMiddleware_UpdateConfig(t *testing.T) {
	collector := newMockMetricsCollector()
	middleware := NewPrometheusHTTPMiddleware(collector, &PrometheusHTTPConfig{
		Enabled:             false,
		ExcludePaths:        []string{"/health"},
		CollectRequestSize:  false,
		CollectResponseSize: false,
	})

	newConfig := &PrometheusHTTPConfig{
		Enabled:             true,
		ExcludePaths:        []string{"/health", "/metrics"},
		CollectRequestSize:  true,
		CollectResponseSize: true,
	}

	middleware.UpdateConfig(newConfig)

	config := middleware.GetConfig()
	assert.True(t, config.Enabled)
	assert.Equal(t, []string{"/health", "/metrics"}, config.ExcludePaths)
	assert.True(t, config.CollectRequestSize)
	assert.True(t, config.CollectResponseSize)
}

func TestPrometheusHTTPMiddleware_UpdateConfigNil(t *testing.T) {
	collector := newMockMetricsCollector()
	originalConfig := &PrometheusHTTPConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	middleware := NewPrometheusHTTPMiddleware(collector, originalConfig)
	middleware.UpdateConfig(nil)

	// Config should remain unchanged
	assert.Equal(t, originalConfig, middleware.GetConfig())
}

func TestPrometheusHTTPMiddleware_ResetPathCounter(t *testing.T) {
	collector := newMockMetricsCollector()
	middleware := NewPrometheusHTTPMiddleware(collector, &PrometheusHTTPConfig{
		Enabled:            true,
		ServiceName:        "test-service",
		MaxPathCardinality: 10,
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/*path", func(c *gin.Context) { c.JSON(200, gin.H{}) })

	// Make some requests to populate path counter
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", fmt.Sprintf("/path%d", i), nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	assert.Equal(t, 5, middleware.GetPathCardinality())

	// Reset the counter
	middleware.ResetPathCounter()
	assert.Equal(t, 0, middleware.GetPathCardinality())
}

func TestPrometheusHTTPMiddleware_UtilityFunctions(t *testing.T) {
	middleware := NewPrometheusHTTPMiddleware(nil, nil)

	t.Run("isNumeric", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bool
		}{
			{"123", true},
			{"0", true},
			{"-123", true},
			{"abc", false},
			{"12a", false},
			{"", false},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expected, middleware.isNumeric(tt.input), "input: %s", tt.input)
		}
	})

	t.Run("isUUID", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bool
		}{
			{"f47ac10b-58cc-4372-a567-0e02b2c3d479", true},
			{"00000000-0000-0000-0000-000000000000", true},
			{"not-a-uuid", false},
			{"f47ac10b-58cc-4372-a567-0e02b2c3d47", false},   // too short
			{"f47ac10b-58cc-4372-a567-0e02b2c3d4799", false}, // too long
			{"", false},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expected, middleware.isUUID(tt.input), "input: %s", tt.input)
		}
	})

	t.Run("shouldExcludePath", func(t *testing.T) {
		config := &PrometheusHTTPConfig{
			ExcludePaths: []string{"/health", "/metrics", "/debug"},
		}
		middleware := NewPrometheusHTTPMiddleware(nil, config)

		tests := []struct {
			path     string
			excluded bool
		}{
			{"/health", true},
			{"/health/check", true},
			{"/metrics", true},
			{"/debug/info", true},
			{"/api/users", false},
			{"/healthcheck", true}, // Should be excluded due to /health prefix
		}

		for _, tt := range tests {
			assert.Equal(t, tt.excluded, middleware.shouldExcludePath(tt.path), "path: %s", tt.path)
		}
	})
}

func TestPrometheusHTTPMiddleware_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	config := &PrometheusHTTPConfig{
		Enabled:     true,
		ServiceName: "test-service",
	}

	middleware := NewPrometheusHTTPMiddleware(collector, config)

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/users", func(c *gin.Context) {
		time.Sleep(10 * time.Millisecond) // Simulate some processing time
		c.JSON(200, gin.H{"message": "success"})
	})

	// Make concurrent requests
	const numRequests = 10
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/api/users", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code)
		}()
	}

	wg.Wait()

	// Verify all requests were counted
	countLabels := map[string]string{
		"service":  "test-service",
		"method":   "GET",
		"endpoint": "/api/users",
		"status":   "200",
	}

	assert.Equal(t, float64(numRequests), collector.getCounterValue("http_requests_total", countLabels))

	// Verify active requests gauge is back to 0
	activeLabels := map[string]string{"service": "test-service"}
	assert.Equal(t, float64(0), collector.getGaugeValue("http_active_requests", activeLabels))
}

// Benchmark tests to verify <5% overhead requirement
func BenchmarkPrometheusHTTPMiddleware_Enabled(b *testing.B) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	middleware := NewPrometheusHTTPMiddleware(collector, &PrometheusHTTPConfig{
		Enabled:     true,
		ServiceName: "test-service",
	})

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/benchmark", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "benchmark"})
	})

	req := httptest.NewRequest("GET", "/api/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkPrometheusHTTPMiddleware_Disabled(b *testing.B) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	middleware := NewPrometheusHTTPMiddleware(collector, &PrometheusHTTPConfig{
		Enabled:     false,
		ServiceName: "test-service",
	})

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/benchmark", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "benchmark"})
	})

	req := httptest.NewRequest("GET", "/api/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkPrometheusHTTPMiddleware_WithoutMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/benchmark", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "benchmark"})
	})

	req := httptest.NewRequest("GET", "/api/benchmark", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkPrometheusHTTPMiddleware_PathNormalization(b *testing.B) {
	gin.SetMode(gin.TestMode)

	collector := newMockMetricsCollector()
	middleware := NewPrometheusHTTPMiddleware(collector, &PrometheusHTTPConfig{
		Enabled:           true,
		ServiceName:       "test-service",
		PathNormalization: true,
	})

	router := gin.New()
	router.Use(middleware.Middleware())
	router.GET("/api/users/*id", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "benchmark"})
	})

	requests := make([]*http.Request, 1000)
	for i := range requests {
		requests[i] = httptest.NewRequest("GET", fmt.Sprintf("/api/users/%d", i), nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := requests[i%len(requests)]
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
