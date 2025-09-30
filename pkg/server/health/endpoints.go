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

package health

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
	"go.uber.org/zap"
)

// EndpointHandler handles health check HTTP endpoints
type EndpointHandler struct {
	adapter     *HealthCheckAdapter
	serviceName string
	version     string
	startTime   time.Time
}

// NewEndpointHandler creates a new endpoint handler
func NewEndpointHandler(adapter *HealthCheckAdapter, serviceName, version string) *EndpointHandler {
	return &EndpointHandler{
		adapter:     adapter,
		serviceName: serviceName,
		version:     version,
		startTime:   time.Now(),
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string                        `json:"status"`
	Service      string                        `json:"service"`
	Version      string                        `json:"version"`
	Timestamp    time.Time                     `json:"timestamp"`
	Uptime       string                        `json:"uptime"`
	Dependencies map[string]*HealthCheckResult `json:"dependencies,omitempty"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Ready     bool                          `json:"ready"`
	Service   string                        `json:"service"`
	Timestamp time.Time                     `json:"timestamp"`
	Checks    map[string]*HealthCheckResult `json:"checks,omitempty"`
}

// LivenessResponse represents the liveness check response
type LivenessResponse struct {
	Alive     bool                          `json:"alive"`
	Service   string                        `json:"service"`
	Timestamp time.Time                     `json:"timestamp"`
	Checks    map[string]*HealthCheckResult `json:"checks,omitempty"`
}

// RegisterRoutes registers health check routes with a Gin router
func (h *EndpointHandler) RegisterRoutes(router *gin.Engine) {
	// Main health check endpoint
	router.GET("/health", h.HealthCheck)

	// Readiness probe endpoint (Kubernetes compatible)
	router.GET("/health/ready", h.ReadinessCheck)
	router.GET("/ready", h.ReadinessCheck) // Alternative path

	// Liveness probe endpoint (Kubernetes compatible)
	router.GET("/health/live", h.LivenessCheck)
	router.GET("/live", h.LivenessCheck) // Alternative path

	// Detailed health check endpoint
	router.GET("/health/detailed", h.DetailedHealthCheck)

	logger.Logger.Info("Registered health check endpoints",
		zap.String("service", h.serviceName))
}

// HealthCheck handles the main health check endpoint
// @Summary Health check
// @Description Returns the overall health status of the service
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Failure 503 {object} HealthResponse "Service is unhealthy"
// @Router /health [get]
func (h *EndpointHandler) HealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.adapter.CheckHealth(ctx)
	if err != nil {
		logger.Logger.Error("Health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  types.HealthStatusUnhealthy,
			"service": h.serviceName,
			"error":   err.Error(),
		})
		return
	}

	response := &HealthResponse{
		Status:    status.Status,
		Service:   h.serviceName,
		Version:   h.version,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
	}

	// Add dependency details if available
	if len(status.Dependencies) > 0 {
		response.Dependencies = make(map[string]*HealthCheckResult)
		for name, dep := range status.Dependencies {
			response.Dependencies[name] = &HealthCheckResult{
				Name:      name,
				Status:    dep.Status,
				Duration:  dep.Latency,
				Timestamp: dep.Timestamp,
				Error:     dep.Error,
			}
		}
	}

	statusCode := http.StatusOK
	if status.Status != types.HealthStatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// ReadinessCheck handles the readiness probe endpoint
// @Summary Readiness check
// @Description Returns whether the service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} ReadinessResponse "Service is ready"
// @Failure 503 {object} ReadinessResponse "Service is not ready"
// @Router /health/ready [get]
func (h *EndpointHandler) ReadinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.adapter.CheckReadiness(ctx)

	response := &ReadinessResponse{
		Service:   h.serviceName,
		Timestamp: time.Now(),
		Ready:     false,
	}

	if err != nil {
		logger.Logger.Debug("Readiness check failed", zap.Error(err))
		response.Checks = h.buildCheckResults(status)
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	if status.Status == types.HealthStatusHealthy {
		response.Ready = true
		response.Checks = h.buildCheckResults(status)
		c.JSON(http.StatusOK, response)
	} else {
		response.Checks = h.buildCheckResults(status)
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// LivenessCheck handles the liveness probe endpoint
// @Summary Liveness check
// @Description Returns whether the service is alive and should not be restarted
// @Tags health
// @Produce json
// @Success 200 {object} LivenessResponse "Service is alive"
// @Failure 503 {object} LivenessResponse "Service is not alive"
// @Router /health/live [get]
func (h *EndpointHandler) LivenessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	status, err := h.adapter.CheckLiveness(ctx)

	response := &LivenessResponse{
		Service:   h.serviceName,
		Timestamp: time.Now(),
		Alive:     false,
	}

	if err != nil {
		logger.Logger.Warn("Liveness check failed", zap.Error(err))
		response.Checks = h.buildCheckResults(status)
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	if status.Status == types.HealthStatusHealthy {
		response.Alive = true
		response.Checks = h.buildCheckResults(status)
		c.JSON(http.StatusOK, response)
	} else {
		response.Checks = h.buildCheckResults(status)
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// DetailedHealthCheck handles the detailed health check endpoint
// @Summary Detailed health check
// @Description Returns detailed health status including all checks and their results
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{} "Detailed health status"
// @Router /health/detailed [get]
func (h *EndpointHandler) DetailedHealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Get all check results
	healthStatus, _ := h.adapter.CheckHealth(ctx)
	readinessStatus, _ := h.adapter.CheckReadiness(ctx)
	livenessStatus, _ := h.adapter.CheckLiveness(ctx)

	// Get cached results
	cachedResults := h.adapter.GetAllLastResults()

	response := gin.H{
		"service":   h.serviceName,
		"version":   h.version,
		"timestamp": time.Now(),
		"uptime":    time.Since(h.startTime).String(),
		"health": gin.H{
			"status":       healthStatus.Status,
			"dependencies": h.buildCheckResults(healthStatus),
		},
		"readiness": gin.H{
			"status": readinessStatus.Status,
			"checks": h.buildCheckResults(readinessStatus),
		},
		"liveness": gin.H{
			"status": livenessStatus.Status,
			"checks": h.buildCheckResults(livenessStatus),
		},
		"cached_results": cachedResults,
	}

	statusCode := http.StatusOK
	if healthStatus.Status != types.HealthStatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// buildCheckResults converts health status dependencies to check results
func (h *EndpointHandler) buildCheckResults(status *types.HealthStatus) map[string]*HealthCheckResult {
	if status == nil || len(status.Dependencies) == 0 {
		return nil
	}

	results := make(map[string]*HealthCheckResult)
	for name, dep := range status.Dependencies {
		results[name] = &HealthCheckResult{
			Name:      name,
			Status:    dep.Status,
			Duration:  dep.Latency,
			Timestamp: dep.Timestamp,
			Error:     dep.Error,
		}
	}
	return results
}

// RegisterWithObservability registers health endpoints with the observability manager
// This integrates with the existing server framework
func RegisterWithObservability(router *gin.Engine, adapter *HealthCheckAdapter, serviceName, version string) {
	handler := NewEndpointHandler(adapter, serviceName, version)
	handler.RegisterRoutes(router)
}
