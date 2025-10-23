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

package monitoring

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// RouteManager manages API routes for the monitoring server.
type RouteManager struct {
	router *gin.Engine
	config *ServerConfig

	// Route groups for better organization
	apiGroup *gin.RouterGroup

	// API handlers
	queryAPI         *SagaQueryAPI
	controlAPI       *SagaControlAPI
	metricsAPI       *MetricsAPI
	visualizationAPI *SagaVisualizationAPI
	alertsAPI        *AlertsAPI
	realtimePush     *RealtimePusher
}

// NewRouteManager creates a new route manager.
func NewRouteManager(router *gin.Engine, config *ServerConfig) *RouteManager {
	return &RouteManager{
		router: router,
		config: config,
	}
}

// SetupRoutes initializes all routes for the monitoring server.
func (rm *RouteManager) SetupRoutes() error {
	// Create API route group
	rm.apiGroup = rm.router.Group("/api")

	// Setup health check routes
	rm.setupHealthRoutes()

	// Setup base routes
	rm.setupBaseRoutes()

	// Setup Saga query routes (if query API is configured)
	if rm.queryAPI != nil {
		rm.setupSagaQueryRoutes()
	}

	// Setup Saga control routes (if control API is configured)
	if rm.controlAPI != nil {
		rm.setupSagaControlRoutes()
	}

	// Setup metrics routes (if metrics API is configured)
	if rm.metricsAPI != nil {
		rm.setupMetricsRoutes()
	}

	// Setup visualization routes (if visualization API is configured)
	if rm.visualizationAPI != nil {
		rm.setupVisualizationRoutes()
	}

	// Setup alerts routes (if alerts API is configured)
	if rm.alertsAPI != nil {
		rm.setupAlertsRoutes()
	}

	if logger.Logger != nil {
		logger.Logger.Info("Routes configured successfully",
			zap.String("health_path", rm.config.HealthCheckPath))
	}

	return nil
}

// setupHealthRoutes configures health check related routes.
func (rm *RouteManager) setupHealthRoutes() {
	// Register health check endpoint at configured path
	rm.router.GET(rm.config.HealthCheckPath, rm.handleHealthCheck)

	// Also register under /api/health for consistency
	if rm.config.HealthCheckPath != "/api/health" {
		rm.apiGroup.GET("/health", rm.handleHealthCheck)
	}

	// Add liveness and readiness probes (Kubernetes compatible)
	rm.apiGroup.GET("/health/live", rm.handleLivenessCheck)
	rm.apiGroup.GET("/health/ready", rm.handleReadinessCheck)
}

// setupBaseRoutes configures base API routes.
func (rm *RouteManager) setupBaseRoutes() {
	// Root endpoint
	rm.router.GET("/", rm.handleRoot)

	// API info endpoint
	rm.apiGroup.GET("/info", rm.handleInfo)
}

// setupSagaQueryRoutes configures Saga query related routes.
func (rm *RouteManager) setupSagaQueryRoutes() {
	// Saga query endpoints
	sagaGroup := rm.apiGroup.Group("/sagas")
	{
		// GET /api/sagas - List Sagas with pagination and filtering
		sagaGroup.GET("", rm.queryAPI.ListSagas)

		// GET /api/sagas/:id - Get Saga details
		sagaGroup.GET("/:id", rm.queryAPI.GetSagaDetails)
	}

	if logger.Logger != nil {
		logger.Logger.Info("Saga query routes configured",
			zap.String("base_path", "/api/sagas"))
	}
}

// setupSagaControlRoutes configures Saga control related routes.
func (rm *RouteManager) setupSagaControlRoutes() {
	// Saga control endpoints
	sagaGroup := rm.apiGroup.Group("/sagas")
	{
		// POST /api/sagas/:id/cancel - Cancel a Saga
		sagaGroup.POST("/:id/cancel", rm.controlAPI.CancelSaga)

		// POST /api/sagas/:id/retry - Retry a Saga
		sagaGroup.POST("/:id/retry", rm.controlAPI.RetrySaga)
	}

	if logger.Logger != nil {
		logger.Logger.Info("Saga control routes configured",
			zap.String("base_path", "/api/sagas"))
	}
}

// setupMetricsRoutes configures metrics related routes.
func (rm *RouteManager) setupMetricsRoutes() {
	// Metrics endpoints
	metricsGroup := rm.apiGroup.Group("/metrics")
	{
		// GET /api/metrics - Get metrics data
		metricsGroup.GET("", rm.metricsAPI.GetMetrics)

		// GET /api/metrics/realtime - Get real-time metrics snapshot
		metricsGroup.GET("/realtime", rm.metricsAPI.GetRealtimeMetrics)

		// GET /api/metrics/stream - SSE stream for real-time updates
		if rm.realtimePush != nil {
			metricsGroup.GET("/stream", rm.realtimePush.HandleSSE)
		}
	}

	if logger.Logger != nil {
		logger.Logger.Info("Metrics routes configured",
			zap.String("base_path", "/api/metrics"))
	}
}

// setupVisualizationRoutes configures Saga visualization related routes.
func (rm *RouteManager) setupVisualizationRoutes() {
	// Visualization endpoints under /api/sagas/:id/visualization
	sagaGroup := rm.apiGroup.Group("/sagas")
	{
		// GET /api/sagas/:id/visualization - Get Saga flow visualization
		sagaGroup.GET("/:id/visualization", rm.visualizationAPI.GetVisualization)
	}

	if logger.Logger != nil {
		logger.Logger.Info("Saga visualization routes configured",
			zap.String("base_path", "/api/sagas/:id/visualization"))
	}
}

// setupAlertsRoutes configures alert related routes.
func (rm *RouteManager) setupAlertsRoutes() {
	// Alerts endpoints
	alertsGroup := rm.apiGroup.Group("/alerts")
	{
		// GET /api/alerts - Get alerts list with filtering
		alertsGroup.GET("", rm.alertsAPI.GetAlerts)

		// GET /api/alerts/stats - Get alert statistics
		alertsGroup.GET("/stats", rm.alertsAPI.GetAlertStats)

		// GET /api/alerts/rules - Get alert rules configuration
		alertsGroup.GET("/rules", rm.alertsAPI.GetAlertRules)

		// GET /api/alerts/:id - Get specific alert by ID
		alertsGroup.GET("/:id", rm.alertsAPI.GetAlert)

		// POST /api/alerts/:id/acknowledge - Acknowledge an alert
		alertsGroup.POST("/:id/acknowledge", rm.alertsAPI.AcknowledgeAlert)
	}

	if logger.Logger != nil {
		logger.Logger.Info("Alerts routes configured",
			zap.String("base_path", "/api/alerts"))
	}
}

// handleRoot handles the root endpoint.
// It serves the dashboard HTML page if it exists, otherwise returns JSON API info.
func (rm *RouteManager) handleRoot(c *gin.Context) {
	// Check if request accepts HTML
	acceptHeader := c.GetHeader("Accept")
	if acceptHeader != "" && (c.GetHeader("Accept") == "text/html" ||
		c.GetHeader("Accept") == "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8") {
		// Try to serve the dashboard HTML
		c.File("./web/static/index.html")
		return
	}

	// Default: return JSON API info
	c.JSON(http.StatusOK, gin.H{
		"service": "Saga Monitoring Dashboard",
		"version": "1.0.0",
		"status":  "running",
		"links": gin.H{
			"health":    rm.config.HealthCheckPath,
			"api":       "/api",
			"readiness": "/api/health/ready",
			"liveness":  "/api/health/live",
			"dashboard": "/static/index.html",
		},
	})
}

// handleInfo handles the API info endpoint.
func (rm *RouteManager) handleInfo(c *gin.Context) {
	endpoints := gin.H{
		"health":    rm.config.HealthCheckPath,
		"readiness": "/api/health/ready",
		"liveness":  "/api/health/live",
	}

	// Add Saga query endpoints if available
	if rm.queryAPI != nil {
		endpoints["sagas_list"] = "/api/sagas"
		endpoints["saga_detail"] = "/api/sagas/:id"
	}

	// Add Saga control endpoints if available
	if rm.controlAPI != nil {
		endpoints["saga_cancel"] = "/api/sagas/:id/cancel"
		endpoints["saga_retry"] = "/api/sagas/:id/retry"
	}

	c.JSON(http.StatusOK, gin.H{
		"name":        "Saga Monitoring API",
		"version":     "v1",
		"description": "Web monitoring dashboard for Saga distributed transactions",
		"endpoints":   endpoints,
	})
}

// handleHealthCheck handles the main health check endpoint.
func (rm *RouteManager) handleHealthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// If health manager is configured, use it for detailed checks
	if rm.config.HealthManager != nil {
		report, err := rm.config.HealthManager.CheckHealth(ctx)
		if err != nil {
			if logger.Logger != nil {
				logger.Logger.Error("Health check failed", zap.Error(err))
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "unhealthy",
				"error":     err.Error(),
				"timestamp": time.Now(),
			})
			return
		}

		// Determine HTTP status based on health status
		httpStatus := http.StatusOK
		if report.Status == HealthStatusUnhealthy {
			httpStatus = http.StatusServiceUnavailable
		} else if report.Status == HealthStatusDegraded {
			httpStatus = http.StatusOK // Still OK but with degraded status
		}

		c.JSON(httpStatus, gin.H{
			"status":     string(report.Status),
			"components": report.Components,
			"timestamp":  report.Timestamp,
			"check_time": report.TotalCheckDuration.String(),
		})
		return
	}

	// Basic health check without health manager
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "saga-monitoring",
		"timestamp": time.Now(),
	})
}

// handleLivenessCheck handles the Kubernetes liveness probe.
// Returns 200 if the application is alive (not deadlocked or crashed).
func (rm *RouteManager) handleLivenessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	if rm.config.HealthManager != nil {
		alive, err := rm.config.HealthManager.CheckLiveness(ctx)
		if err != nil || !alive {
			if logger.Logger != nil {
				logger.Logger.Warn("Liveness check failed",
					zap.Error(err),
					zap.Bool("alive", alive))
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not alive",
				"error":     getErrorMessage(err),
				"timestamp": time.Now(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// handleReadinessCheck handles the Kubernetes readiness probe.
// Returns 200 if the application is ready to serve traffic.
func (rm *RouteManager) handleReadinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	if rm.config.HealthManager != nil {
		ready, err := rm.config.HealthManager.CheckReadiness(ctx)
		if err != nil || !ready {
			if logger.Logger != nil {
				logger.Logger.Warn("Readiness check failed",
					zap.Error(err),
					zap.Bool("ready", ready))
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not ready",
				"error":     getErrorMessage(err),
				"timestamp": time.Now(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now(),
	})
}

// GetAPIGroup returns the API route group for registering additional routes.
func (rm *RouteManager) GetAPIGroup() *gin.RouterGroup {
	return rm.apiGroup
}

// SetQueryAPI sets the Saga query API handler and registers its routes.
func (rm *RouteManager) SetQueryAPI(queryAPI *SagaQueryAPI) {
	rm.queryAPI = queryAPI

	// Register Saga query routes if the API group has been initialized
	// (i.e., if SetupRoutes has already been called)
	if rm.apiGroup != nil && rm.queryAPI != nil {
		rm.setupSagaQueryRoutes()
	}
}

// SetControlAPI sets the Saga control API handler and registers its routes.
func (rm *RouteManager) SetControlAPI(controlAPI *SagaControlAPI) {
	rm.controlAPI = controlAPI

	// Register Saga control routes if the API group has been initialized
	// (i.e., if SetupRoutes has already been called)
	if rm.apiGroup != nil && rm.controlAPI != nil {
		rm.setupSagaControlRoutes()
	}
}

// SetMetricsAPI sets the metrics API handler and registers its routes.
func (rm *RouteManager) SetMetricsAPI(metricsAPI *MetricsAPI) {
	rm.metricsAPI = metricsAPI

	// Register metrics routes if the API group has been initialized
	// (i.e., if SetupRoutes has already been called)
	if rm.apiGroup != nil && rm.metricsAPI != nil {
		rm.setupMetricsRoutes()
	}
}

// SetVisualizationAPI sets the visualization API handler.
func (rm *RouteManager) SetVisualizationAPI(visualizationAPI *SagaVisualizationAPI) {
	rm.visualizationAPI = visualizationAPI

	// Register visualization routes if the API group has been initialized
	if rm.apiGroup != nil && rm.visualizationAPI != nil {
		rm.setupVisualizationRoutes()
	}
}

// SetAlertsAPI sets the alerts API handler and registers its routes.
func (rm *RouteManager) SetAlertsAPI(alertsAPI *AlertsAPI) {
	rm.alertsAPI = alertsAPI

	// Register alerts routes if the API group has been initialized
	if rm.apiGroup != nil && rm.alertsAPI != nil {
		rm.setupAlertsRoutes()
	}
}

// SetRealtimePusher sets the realtime pusher for SSE streaming.
func (rm *RouteManager) SetRealtimePusher(pusher *RealtimePusher) {
	rm.realtimePush = pusher

	// Register only the SSE route if metrics API and API group are already set
	// to avoid duplicate route registration
	if rm.apiGroup != nil && rm.metricsAPI != nil {
		rm.setupSSERoute()
	}
}

// setupSSERoute configures only the SSE route for real-time updates.
// This method is called separately to avoid duplicate route registration.
func (rm *RouteManager) setupSSERoute() {
	if rm.realtimePush == nil {
		return
	}

	// Register only the SSE route under /api/metrics/stream
	metricsGroup := rm.apiGroup.Group("/metrics")
	metricsGroup.GET("/stream", rm.realtimePush.HandleSSE)

	if logger.Logger != nil {
		logger.Logger.Info("SSE route configured",
			zap.String("path", "/api/metrics/stream"))
	}
}

// getErrorMessage safely extracts error message from error.
func getErrorMessage(err error) string {
	if err != nil {
		return err.Error()
	}
	return "unknown error"
}
