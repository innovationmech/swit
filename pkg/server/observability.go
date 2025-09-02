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
	"context"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/types"
)

// Use MetricsCollector and Metric from types package to avoid duplication
type MetricsCollector = types.MetricsCollector
type Metric = types.Metric
type MetricType = types.MetricType

// SimpleMetricsCollector provides a basic in-memory metrics collector
type SimpleMetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*metricData
}

type metricData struct {
	metric    Metric
	counter   int64
	gauge     float64
	histogram []float64
}

// NewSimpleMetricsCollector creates a new simple metrics collector
func NewSimpleMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		metrics: make(map[string]*metricData),
	}
}

// IncrementCounter increments a counter metric by 1
func (smc *SimpleMetricsCollector) IncrementCounter(name string, labels map[string]string) {
	smc.AddToCounter(name, 1, labels)
}

// AddToCounter adds a value to a counter metric
func (smc *SimpleMetricsCollector) AddToCounter(name string, value float64, labels map[string]string) {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	key := smc.getMetricKey(name, labels)
	data, exists := smc.metrics[key]
	if !exists {
		data = &metricData{
			metric: Metric{
				Name:      name,
				Type:      types.CounterType,
				Labels:    labels,
				Timestamp: time.Now(),
			},
		}
		smc.metrics[key] = data
	}

	atomic.AddInt64(&data.counter, int64(value))
	data.metric.Value = atomic.LoadInt64(&data.counter)
	data.metric.Timestamp = time.Now()
}

// SetGauge sets a gauge metric to a specific value
func (smc *SimpleMetricsCollector) SetGauge(name string, value float64, labels map[string]string) {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	key := smc.getMetricKey(name, labels)
	data, exists := smc.metrics[key]
	if !exists {
		data = &metricData{
			metric: Metric{
				Name:      name,
				Type:      types.GaugeType,
				Labels:    labels,
				Timestamp: time.Now(),
			},
		}
		smc.metrics[key] = data
	}

	data.gauge = value
	data.metric.Value = value
	data.metric.Timestamp = time.Now()
}

// IncrementGauge increments a gauge metric by 1
func (smc *SimpleMetricsCollector) IncrementGauge(name string, labels map[string]string) {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	key := smc.getMetricKey(name, labels)
	data, exists := smc.metrics[key]
	if !exists {
		data = &metricData{
			metric: Metric{
				Name:      name,
				Type:      types.GaugeType,
				Labels:    labels,
				Timestamp: time.Now(),
			},
		}
		smc.metrics[key] = data
	}

	data.gauge++
	data.metric.Value = data.gauge
	data.metric.Timestamp = time.Now()
}

// DecrementGauge decrements a gauge metric by 1
func (smc *SimpleMetricsCollector) DecrementGauge(name string, labels map[string]string) {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	key := smc.getMetricKey(name, labels)
	data, exists := smc.metrics[key]
	if !exists {
		// If gauge doesn't exist, don't create it - this prevents negative values
		// DecrementGauge should only be called on existing gauges
		return
	}

	data.gauge--
	data.metric.Value = data.gauge
	data.metric.Timestamp = time.Now()
}

// ObserveHistogram adds an observation to a histogram metric
func (smc *SimpleMetricsCollector) ObserveHistogram(name string, value float64, labels map[string]string) {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	key := smc.getMetricKey(name, labels)
	data, exists := smc.metrics[key]
	if !exists {
		data = &metricData{
			metric: Metric{
				Name:      name,
				Type:      types.HistogramType,
				Labels:    labels,
				Timestamp: time.Now(),
			},
			histogram: make([]float64, 0),
		}
		smc.metrics[key] = data
	}

	data.histogram = append(data.histogram, value)

	// Calculate basic statistics for the histogram
	sum := 0.0
	for _, v := range data.histogram {
		sum += v
	}

	data.metric.Value = map[string]interface{}{
		"count": len(data.histogram),
		"sum":   sum,
		"avg":   sum / float64(len(data.histogram)),
	}
	data.metric.Timestamp = time.Now()
}

// GetMetrics returns all collected metrics
func (smc *SimpleMetricsCollector) GetMetrics() []Metric {
	smc.mu.RLock()
	defer smc.mu.RUnlock()

	metrics := make([]Metric, 0, len(smc.metrics))
	for _, data := range smc.metrics {
		metrics = append(metrics, data.metric)
	}

	return metrics
}

// GetMetric returns a specific metric by name
func (smc *SimpleMetricsCollector) GetMetric(name string) (*Metric, bool) {
	smc.mu.RLock()
	defer smc.mu.RUnlock()

	for _, data := range smc.metrics {
		if data.metric.Name == name {
			metric := data.metric
			return &metric, true
		}
	}

	return nil, false
}

// Reset clears all metrics
func (smc *SimpleMetricsCollector) Reset() {
	smc.mu.Lock()
	defer smc.mu.Unlock()

	smc.metrics = make(map[string]*metricData)
}

// getMetricKey generates a unique key for a metric with labels
func (smc *SimpleMetricsCollector) getMetricKey(name string, labels map[string]string) string {
	if len(labels) == 0 {
		return name
	}

	// Sort label keys to ensure consistent key generation
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	// Use a simple sort to ensure deterministic ordering
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	key := name
	for _, k := range keys {
		key += fmt.Sprintf("_%s_%s", k, labels[k])
	}
	return key
}

// ServerMetrics provides server-specific metrics collection
type ServerMetrics struct {
	collector   MetricsCollector
	serviceName string
	startTime   time.Time
}

// NewServerMetrics creates a new server metrics collector
func NewServerMetrics(serviceName string, collector MetricsCollector) *ServerMetrics {
	if collector == nil {
		collector = NewSimpleMetricsCollector()
	}

	return &ServerMetrics{
		collector:   collector,
		serviceName: serviceName,
		startTime:   time.Now(),
	}
}

// RecordServerStart records server startup metrics
func (sm *ServerMetrics) RecordServerStart() {
	labels := map[string]string{"service": sm.serviceName}
	sm.collector.IncrementCounter("server_starts_total", labels)
	sm.collector.SetGauge("server_start_time", float64(sm.startTime.Unix()), labels)
}

// RecordServerStop records server shutdown metrics
func (sm *ServerMetrics) RecordServerStop() {
	labels := map[string]string{"service": sm.serviceName}
	sm.collector.IncrementCounter("server_stops_total", labels)
	uptime := time.Since(sm.startTime).Seconds()
	sm.collector.ObserveHistogram("server_uptime_seconds", uptime, labels)
}

// RecordTransportStart records transport startup metrics
func (sm *ServerMetrics) RecordTransportStart(transportType string) {
	labels := map[string]string{
		"service":   sm.serviceName,
		"transport": transportType,
	}
	sm.collector.IncrementCounter("transport_starts_total", labels)
	sm.collector.IncrementGauge("active_transports", labels)
}

// RecordTransportStop records transport shutdown metrics
func (sm *ServerMetrics) RecordTransportStop(transportType string) {
	labels := map[string]string{
		"service":   sm.serviceName,
		"transport": transportType,
	}
	sm.collector.IncrementCounter("transport_stops_total", labels)
	sm.collector.DecrementGauge("active_transports", labels)
}

// RecordServiceRegistration records service registration metrics
func (sm *ServerMetrics) RecordServiceRegistration(serviceName, serviceType string) {
	labels := map[string]string{
		"service":            sm.serviceName,
		"registered_service": serviceName,
		"service_type":       serviceType,
	}
	sm.collector.IncrementCounter("service_registrations_total", labels)
	sm.collector.IncrementGauge("registered_services", labels)
}

// RecordError records error metrics
func (sm *ServerMetrics) RecordError(errorType, operation string) {
	labels := map[string]string{
		"service":    sm.serviceName,
		"error_type": errorType,
		"operation":  operation,
	}
	sm.collector.IncrementCounter("errors_total", labels)
}

// RecordOperationDuration records operation duration metrics
func (sm *ServerMetrics) RecordOperationDuration(operation string, duration time.Duration) {
	labels := map[string]string{
		"service":   sm.serviceName,
		"operation": operation,
	}
	sm.collector.ObserveHistogram("operation_duration_seconds", duration.Seconds(), labels)
}

// GetCollector returns the underlying metrics collector
func (sm *ServerMetrics) GetCollector() MetricsCollector {
	return sm.collector
}

// ServerStatus represents the overall server status
type ServerStatus struct {
	ServiceName   string                         `json:"service_name"`
	Status        string                         `json:"status"`
	StartTime     time.Time                      `json:"start_time"`
	Uptime        string                         `json:"uptime"`
	Version       string                         `json:"version,omitempty"`
	BuildInfo     map[string]string              `json:"build_info,omitempty"`
	Configuration map[string]interface{}         `json:"configuration,omitempty"`
	Transports    []TransportStatus              `json:"transports"`
	Dependencies  []DependencyStatus             `json:"dependencies,omitempty"`
	HealthChecks  map[string]*types.HealthStatus `json:"health_checks,omitempty"`
	Metrics       []Metric                       `json:"metrics,omitempty"`
	SystemInfo    SystemInfo                     `json:"system_info"`
}

// DependencyStatus represents the status of a dependency
type DependencyStatus struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Error       string    `json:"error,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	NumGoroutine int    `json:"num_goroutine"`
	MemStats     struct {
		Alloc      uint64 `json:"alloc"`
		TotalAlloc uint64 `json:"total_alloc"`
		Sys        uint64 `json:"sys"`
		NumGC      uint32 `json:"num_gc"`
	} `json:"mem_stats"`
}

// PrometheusHandler defines the interface for Prometheus HTTP handler
type PrometheusHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}

// ObservabilityManager manages observability features for the server
type ObservabilityManager struct {
	serviceName       string
	metrics           *ServerMetrics
	startTime         time.Time
	version           string
	buildInfo         map[string]string
	prometheusHandler PrometheusHandler

	// Prometheus integration
	prometheusCollector *types.PrometheusMetricsCollector
	metricsRegistry     *MetricsRegistry

	// Business metrics
	businessMetricsManager *BusinessMetricsManager

	// System metrics tracking
	mu            sync.RWMutex
	lastMemStats  runtime.MemStats
	lastMemUpdate time.Time
	memStatsCache time.Duration // Cache duration for memory stats
}

// NewObservabilityManager creates a new observability manager
func NewObservabilityManager(serviceName string, prometheusConfig *PrometheusConfig, collector MetricsCollector) *ObservabilityManager {
	// Initialize Prometheus collector and registry
	var prometheusCollector *types.PrometheusMetricsCollector
	if collector != nil {
		// Try to cast to Prometheus collector
		if pmc, ok := collector.(*types.PrometheusMetricsCollector); ok {
			prometheusCollector = pmc
		}
	}

	// If no Prometheus collector provided, create one with the given configuration
	if prometheusCollector == nil {
		if prometheusConfig == nil || !prometheusConfig.Enabled {
			prometheusConfig = types.DefaultPrometheusConfig()
		}
		prometheusCollector = types.NewPrometheusMetricsCollector(prometheusConfig)
	}

	metricsRegistry := NewMetricsRegistry()

	om := &ObservabilityManager{
		serviceName:            serviceName,
		metrics:                NewServerMetrics(serviceName, prometheusCollector),
		startTime:              time.Now(),
		buildInfo:              make(map[string]string),
		prometheusCollector:    prometheusCollector,
		metricsRegistry:        metricsRegistry,
		businessMetricsManager: NewBusinessMetricsManager(serviceName, prometheusCollector, metricsRegistry),
		memStatsCache:          5 * time.Second, // Cache memory stats for 5 seconds
	}

	// Set the Prometheus handler
	om.prometheusHandler = prometheusCollector.GetHandler()

	return om
}

// SetVersion sets the service version
func (om *ObservabilityManager) SetVersion(version string) {
	om.version = version
}

// SetBuildInfo sets build information
func (om *ObservabilityManager) SetBuildInfo(info map[string]string) {
	om.buildInfo = info
}

// SetPrometheusHandler sets the Prometheus HTTP handler
func (om *ObservabilityManager) SetPrometheusHandler(handler PrometheusHandler) {
	om.prometheusHandler = handler
}

// GetMetrics returns the metrics collector
func (om *ObservabilityManager) GetMetrics() *ServerMetrics {
	return om.metrics
}

// RecordServerStartup records server startup metrics including startup duration
func (om *ObservabilityManager) RecordServerStartup(duration time.Duration) {
	om.mu.Lock()
	defer om.mu.Unlock()

	labels := map[string]string{"service": om.serviceName}

	// Record startup duration
	om.prometheusCollector.ObserveHistogram("server_startup_duration_seconds", duration.Seconds(), labels)

	// Set server start time for uptime calculation
	om.prometheusCollector.SetGauge("server_start_time", float64(om.startTime.Unix()), labels)

	// Record server startup event
	om.metrics.RecordServerStart()
}

// RecordServerShutdown records server shutdown metrics
func (om *ObservabilityManager) RecordServerShutdown(duration time.Duration) {
	om.mu.Lock()
	defer om.mu.Unlock()

	labels := map[string]string{"service": om.serviceName}

	// Record shutdown duration
	om.prometheusCollector.ObserveHistogram("server_shutdown_duration_seconds", duration.Seconds(), labels)

	// Record server shutdown event
	om.metrics.RecordServerStop()
}

// RecordTransportLifecycle records transport start/stop events
func (om *ObservabilityManager) RecordTransportStart(transportType string) {
	labels := map[string]string{
		"service":   om.serviceName,
		"transport": transportType,
	}

	// Record transport start
	om.prometheusCollector.IncrementCounter("transport_starts_total", labels)
	om.prometheusCollector.IncrementGauge("active_transports", labels)
	om.metrics.RecordTransportStart(transportType)
}

func (om *ObservabilityManager) RecordTransportStop(transportType string) {
	labels := map[string]string{
		"service":   om.serviceName,
		"transport": transportType,
	}

	// Record transport stop
	om.prometheusCollector.IncrementCounter("transport_stops_total", labels)
	om.prometheusCollector.DecrementGauge("active_transports", labels)
	om.metrics.RecordTransportStop(transportType)
}

// RecordServiceRegistration records service registration events
func (om *ObservabilityManager) RecordServiceRegistration(serviceName, serviceType string) {
	labels := map[string]string{
		"service":            om.serviceName,
		"registered_service": serviceName,
		"service_type":       serviceType,
	}

	// Record service registration
	om.prometheusCollector.IncrementCounter("service_registrations_total", labels)
	om.prometheusCollector.IncrementGauge("registered_services", labels)
	om.metrics.RecordServiceRegistration(serviceName, serviceType)
}

// getCachedMemStats returns cached memory stats or fetches new ones if cache is expired
func (om *ObservabilityManager) getCachedMemStats() runtime.MemStats {
	now := time.Now()

	// Check if cache is still valid
	if !om.lastMemUpdate.IsZero() && now.Sub(om.lastMemUpdate) < om.memStatsCache {
		return om.lastMemStats
	}

	// Cache is expired, fetch new stats
	runtime.ReadMemStats(&om.lastMemStats)
	om.lastMemUpdate = now

	return om.lastMemStats
}

// createLabelsWithType creates a copy of base labels with an added type label
func createLabelsWithType(baseLabels map[string]string, labelType string) map[string]string {
	labels := make(map[string]string, len(baseLabels)+1)
	for k, v := range baseLabels {
		labels[k] = v
	}
	labels["type"] = labelType
	return labels
}

// UpdateSystemMetrics updates system performance metrics (memory, goroutines, GC, uptime)
func (om *ObservabilityManager) UpdateSystemMetrics() {
	om.mu.Lock()
	defer om.mu.Unlock()

	baseLabels := map[string]string{"service": om.serviceName}

	// Get cached memory stats to avoid expensive runtime.ReadMemStats calls
	memStats := om.getCachedMemStats()

	// Update memory metrics with different types using helper function
	om.prometheusCollector.SetGauge("memory_bytes", float64(memStats.Alloc),
		createLabelsWithType(baseLabels, "alloc"))

	om.prometheusCollector.SetGauge("memory_bytes", float64(memStats.Sys),
		createLabelsWithType(baseLabels, "sys"))

	om.prometheusCollector.SetGauge("memory_bytes", float64(memStats.TotalAlloc),
		createLabelsWithType(baseLabels, "total_alloc"))

	// Update goroutine count
	om.prometheusCollector.SetGauge("goroutines", float64(runtime.NumGoroutine()), baseLabels)

	// Update GC metrics
	gcDuration := time.Duration(memStats.PauseTotalNs).Seconds()
	om.prometheusCollector.SetGauge("gc_duration_seconds", gcDuration, baseLabels)

	// Update uptime
	uptime := time.Since(om.startTime).Seconds()
	om.prometheusCollector.SetGauge("uptime_seconds", uptime, baseLabels)
}

// StartSystemMetricsCollection starts periodic collection of system metrics
func (om *ObservabilityManager) StartSystemMetricsCollection(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second // Default collection interval
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				om.UpdateSystemMetrics()
			}
		}
	}()
}

// GetPrometheusCollector returns the Prometheus metrics collector for advanced usage
func (om *ObservabilityManager) GetPrometheusCollector() *types.PrometheusMetricsCollector {
	return om.prometheusCollector
}

// GetMetricsRegistry returns the metrics registry
func (om *ObservabilityManager) GetMetricsRegistry() *MetricsRegistry {
	return om.metricsRegistry
}

// GetBusinessMetricsManager returns the business metrics manager
func (om *ObservabilityManager) GetBusinessMetricsManager() *BusinessMetricsManager {
	return om.businessMetricsManager
}

// GetServerStatus returns comprehensive server status
func (om *ObservabilityManager) GetServerStatus(server BusinessServerCore) *ServerStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := &ServerStatus{
		ServiceName: om.serviceName,
		Status:      "running",
		StartTime:   om.startTime,
		Uptime:      time.Since(om.startTime).String(),
		Version:     om.version,
		BuildInfo:   om.buildInfo,
		Transports:  make([]TransportStatus, 0),
		SystemInfo: SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemStats: struct {
				Alloc      uint64 `json:"alloc"`
				TotalAlloc uint64 `json:"total_alloc"`
				Sys        uint64 `json:"sys"`
				NumGC      uint32 `json:"num_gc"`
			}{
				Alloc:      memStats.Alloc,
				TotalAlloc: memStats.TotalAlloc,
				Sys:        memStats.Sys,
				NumGC:      memStats.NumGC,
			},
		},
	}

	// Get transport status
	if server != nil {
		transportStatus := server.GetTransportStatus()
		for _, ts := range transportStatus {
			status.Transports = append(status.Transports, ts)
		}

		// Get health checks
		ctx := context.Background()
		healthChecks := server.GetTransportHealth(ctx)
		if len(healthChecks) > 0 {
			status.HealthChecks = make(map[string]*types.HealthStatus)
			for transport, services := range healthChecks {
				for service, health := range services {
					key := fmt.Sprintf("%s_%s", transport, service)
					status.HealthChecks[key] = health
				}
			}
		}
	}

	// Get metrics
	status.Metrics = om.metrics.collector.GetMetrics()

	return status
}

// RegisterDebugEndpoints registers debugging endpoints with the HTTP router
func (om *ObservabilityManager) RegisterDebugEndpoints(router *gin.Engine, server BusinessServerCore) {
	debugGroup := router.Group("/debug")

	// Status endpoint
	debugGroup.GET("/status", func(c *gin.Context) {
		status := om.GetServerStatus(server)
		c.JSON(http.StatusOK, status)
	})

	// Metrics endpoint (JSON format for debug)
	debugGroup.GET("/metrics", func(c *gin.Context) {
		metrics := om.metrics.collector.GetMetrics()
		c.JSON(http.StatusOK, map[string]interface{}{
			"metrics":   metrics,
			"timestamp": time.Now(),
		})
	})

	// Health endpoint with detailed information
	debugGroup.GET("/health", func(c *gin.Context) {
		ctx := c.Request.Context()

		healthStatus := &types.HealthStatus{
			Status:    types.HealthStatusHealthy,
			Timestamp: time.Now(),
			Version:   om.version,
		}

		// Check transport health if server is available
		if server != nil {
			transportHealth := server.GetTransportHealth(ctx)
			healthDetails := make(map[string]interface{})

			allHealthy := true
			for transport, services := range transportHealth {
				for service, health := range services {
					key := fmt.Sprintf("%s_%s", transport, service)
					healthDetails[key] = health
					if health.Status != types.HealthStatusHealthy {
						allHealthy = false
					}
				}
			}

			if !allHealthy {
				healthStatus.Status = types.HealthStatusUnhealthy
			}

			// Convert healthDetails to Dependencies
			for key, health := range healthDetails {
				if h, ok := health.(*types.HealthStatus); ok {
					depStatus := types.DependencyStatus{
						Status:    h.Status,
						Timestamp: h.Timestamp,
					}
					healthStatus.AddDependency(key, depStatus)
				}
			}
		}

		statusCode := http.StatusOK
		if healthStatus.Status != types.HealthStatusHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, healthStatus)
	})

	// Configuration endpoint (sanitized)
	debugGroup.GET("/config", func(c *gin.Context) {
		// Return sanitized configuration (remove sensitive data)
		config := map[string]interface{}{
			"service_name": om.serviceName,
			"version":      om.version,
			"build_info":   om.buildInfo,
			"start_time":   om.startTime,
		}

		c.JSON(http.StatusOK, config)
	})

	// System info endpoint
	debugGroup.GET("/system", func(c *gin.Context) {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		systemInfo := SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemStats: struct {
				Alloc      uint64 `json:"alloc"`
				TotalAlloc uint64 `json:"total_alloc"`
				Sys        uint64 `json:"sys"`
				NumGC      uint32 `json:"num_gc"`
			}{
				Alloc:      memStats.Alloc,
				TotalAlloc: memStats.TotalAlloc,
				Sys:        memStats.Sys,
				NumGC:      memStats.NumGC,
			},
		}

		c.JSON(http.StatusOK, systemInfo)
	})
}

// RegisterHealthEndpoint registers a comprehensive health check endpoint
func (om *ObservabilityManager) RegisterHealthEndpoint(router *gin.Engine, server BusinessServerCore) {
	router.GET("/health", func(c *gin.Context) {
		ctx := c.Request.Context()

		healthStatus := &types.HealthStatus{
			Status:    types.HealthStatusHealthy,
			Timestamp: time.Now(),
			Version:   om.version,
		}

		// Basic health check - server is running
		if server == nil {
			healthStatus.Status = types.HealthStatusUnhealthy
			healthStatus.AddDependency("server", types.DependencyStatus{
				Status:    types.DependencyStatusDown,
				Error:     "server not available",
				Timestamp: time.Now(),
			})
			c.JSON(http.StatusServiceUnavailable, healthStatus)
			return
		}

		// Check transport health
		transportHealth := server.GetTransportHealth(ctx)

		allHealthy := true
		for transport, services := range transportHealth {
			for service, health := range services {
				key := fmt.Sprintf("%s_%s", transport, service)
				depStatus := types.DependencyStatus{
					Status:    health.Status,
					Timestamp: health.Timestamp,
				}
				healthStatus.AddDependency(key, depStatus)

				if health.Status != types.HealthStatusHealthy {
					allHealthy = false
				}
			}
		}

		if !allHealthy {
			healthStatus.Status = types.HealthStatusUnhealthy
		}

		statusCode := http.StatusOK
		if healthStatus.Status != types.HealthStatusHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, healthStatus)
	})
}

// RegisterPrometheusEndpoint registers the Prometheus metrics endpoint
func (om *ObservabilityManager) RegisterPrometheusEndpoint(router *gin.Engine) {
	// Register the main /metrics endpoint for Prometheus format
	router.GET("/metrics", func(c *gin.Context) {
		if om.prometheusHandler != nil {
			// Use Prometheus handler if available
			om.prometheusHandler.ServeHTTP(c.Writer, c.Request)
		} else {
			// Fallback to JSON format if Prometheus handler is not set
			metrics := om.metrics.collector.GetMetrics()
			c.JSON(http.StatusOK, map[string]interface{}{
				"metrics":   metrics,
				"timestamp": time.Now(),
				"format":    "json", // Indicate this is the fallback format
			})
		}
	})
}

// RegisterObservabilityEndpoints registers all observability endpoints (convenience method)
func (om *ObservabilityManager) RegisterObservabilityEndpoints(router *gin.Engine, server BusinessServerCore) {
	// Register the main Prometheus endpoint only if Prometheus middleware is not enabled
	// to avoid duplicate route registration
	if serverImpl, ok := server.(*BusinessServerImpl); ok {
		if !serverImpl.config.Prometheus.Enabled {
			om.RegisterPrometheusEndpoint(router)
		}
	} else {
		// Fallback: register if we can't determine the configuration
		om.RegisterPrometheusEndpoint(router)
	}

	// Register health endpoint
	om.RegisterHealthEndpoint(router, server)

	// Register debug endpoints
	om.RegisterDebugEndpoints(router, server)
}
