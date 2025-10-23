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
	"github.com/innovationmech/swit/pkg/saga"
	"go.uber.org/zap"
)

// MetricsAPI provides API endpoints for accessing Saga metrics and real-time status.
// It supports queries for current metrics, historical data, and aggregations.
type MetricsAPI struct {
	collector   MetricsCollector
	coordinator saga.SagaCoordinator
}

// NewMetricsAPI creates a new metrics API handler.
//
// Parameters:
//   - collector: The metrics collector for retrieving metrics data.
//   - coordinator: The Saga coordinator for accessing Saga state.
//
// Returns:
//   - A configured MetricsAPI ready to handle metrics requests.
func NewMetricsAPI(collector MetricsCollector, coordinator saga.SagaCoordinator) *MetricsAPI {
	return &MetricsAPI{
		collector:   collector,
		coordinator: coordinator,
	}
}

// MetricsRequest represents query parameters for metrics requests.
type MetricsRequest struct {
	// TimeRange specifies the time window for metrics (e.g., "1h", "24h", "7d")
	// Defaults to "1h"
	TimeRange string `form:"timeRange" json:"timeRange" binding:"omitempty"`

	// GroupBy specifies how to group metrics: "state", "definition", "hour", "day"
	GroupBy string `form:"groupBy" json:"groupBy" binding:"omitempty,oneof=state definition hour day"`

	// MetricTypes specifies which metrics to include (comma-separated)
	// Available: "count", "duration", "success_rate", "failure_rate"
	MetricTypes []string `form:"metricTypes" json:"metricTypes" binding:"omitempty"`
}

// MetricsResponse represents the response for metrics queries.
type MetricsResponse struct {
	// Summary contains overall metrics
	Summary MetricsSummary `json:"summary"`

	// Timeseries contains metrics over time (if requested)
	Timeseries []TimeseriesPoint `json:"timeseries,omitempty"`

	// Aggregations contains grouped metrics (if groupBy is specified)
	Aggregations map[string]MetricsSummary `json:"aggregations,omitempty"`

	// Timestamp when metrics were collected
	Timestamp time.Time `json:"timestamp"`
}

// MetricsSummary contains aggregated metrics for a specific scope.
type MetricsSummary struct {
	// Total number of Sagas
	Total int64 `json:"total"`

	// Number of active Sagas
	Active int64 `json:"active"`

	// Number of completed Sagas
	Completed int64 `json:"completed"`

	// Number of failed Sagas
	Failed int64 `json:"failed"`

	// Success rate (completed / total) as percentage
	SuccessRate float64 `json:"successRate"`

	// Failure rate (failed / total) as percentage
	FailureRate float64 `json:"failureRate"`

	// Average duration in seconds
	AvgDuration float64 `json:"avgDuration"`

	// Total duration in seconds
	TotalDuration float64 `json:"totalDuration"`

	// Failure reasons breakdown
	FailureReasons map[string]int64 `json:"failureReasons,omitempty"`
}

// TimeseriesPoint represents a single point in a time series.
type TimeseriesPoint struct {
	Timestamp time.Time      `json:"timestamp"`
	Metrics   MetricsSummary `json:"metrics"`
}

// GetMetrics handles GET /api/metrics - retrieves current metrics data.
//
// Query parameters:
//   - timeRange: Time window for metrics (e.g., "1h", "24h")
//   - groupBy: How to group metrics ("state", "definition")
//   - metricTypes: Which metrics to include (comma-separated)
//
// Response:
//
//	{
//	  "summary": {
//	    "total": 1000,
//	    "active": 10,
//	    "completed": 950,
//	    "failed": 40,
//	    "successRate": 95.0,
//	    "failureRate": 4.0,
//	    "avgDuration": 2.5
//	  },
//	  "timestamp": "2025-10-23T10:30:00Z"
//	}
func (api *MetricsAPI) GetMetrics(c *gin.Context) {
	startTime := time.Now()

	if logger.Logger != nil {
		logger.Logger.Info("Get metrics request received")
	}

	// Parse query parameters
	var req MetricsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Invalid metrics request",
				zap.Error(err))
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Get current metrics from collector
	metrics := api.collector.GetMetrics()

	// Calculate summary
	summary := calculateMetricsSummary(metrics)

	response := MetricsResponse{
		Summary:   summary,
		Timestamp: time.Now(),
	}

	// Add aggregations if groupBy is specified
	if req.GroupBy != "" {
		aggregations, err := api.getAggregatedMetrics(req.GroupBy)
		if err != nil {
			if logger.Logger != nil {
				logger.Logger.Warn("Failed to get aggregated metrics",
					zap.String("group_by", sanitizeForLog(req.GroupBy)),
					zap.Error(err))
			}
			// Continue without aggregations rather than failing the whole request
		} else {
			response.Aggregations = aggregations
		}
	}

	duration := time.Since(startTime)
	if logger.Logger != nil {
		logger.Logger.Info("Metrics retrieved successfully",
			zap.Duration("duration", duration),
			zap.Int64("total_sagas", summary.Total),
			zap.Int64("active_sagas", summary.Active))
	}

	c.JSON(http.StatusOK, response)
}

// GetRealtimeMetrics handles GET /api/metrics/realtime - retrieves real-time metrics.
//
// This endpoint provides a snapshot of current metrics without historical data,
// optimized for frequent polling or dashboard displays.
//
// Response:
//
//	{
//	  "active": 10,
//	  "successRate": 95.0,
//	  "avgDuration": 2.5,
//	  "throughput": 50.0,
//	  "timestamp": "2025-10-23T10:30:00Z"
//	}
func (api *MetricsAPI) GetRealtimeMetrics(c *gin.Context) {
	metrics := api.collector.GetMetrics()
	summary := calculateMetricsSummary(metrics)

	// Calculate throughput (sagas per second) based on recent activity
	// This is a simplified calculation - in production you'd track this more accurately
	throughput := 0.0
	if summary.TotalDuration > 0 {
		// Approximate throughput based on completed sagas and total duration
		throughput = float64(summary.Completed) / summary.TotalDuration
	}

	response := gin.H{
		"active":      summary.Active,
		"successRate": summary.SuccessRate,
		"failureRate": summary.FailureRate,
		"avgDuration": summary.AvgDuration,
		"throughput":  throughput,
		"total":       summary.Total,
		"completed":   summary.Completed,
		"failed":      summary.Failed,
		"timestamp":   time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// calculateMetricsSummary computes summary metrics from raw metrics data.
func calculateMetricsSummary(metrics *Metrics) MetricsSummary {
	total := metrics.SagasStarted
	completed := metrics.SagasCompleted
	failed := metrics.SagasFailed
	active := metrics.ActiveSagas

	summary := MetricsSummary{
		Total:          total,
		Active:         active,
		Completed:      completed,
		Failed:         failed,
		AvgDuration:    metrics.AvgDuration,
		TotalDuration:  metrics.TotalDuration,
		FailureReasons: metrics.FailureReasons,
	}

	// Calculate success and failure rates
	if total > 0 {
		summary.SuccessRate = float64(completed) / float64(total) * 100
		summary.FailureRate = float64(failed) / float64(total) * 100
	}

	return summary
}

// getAggregatedMetrics retrieves metrics grouped by the specified dimension.
func (api *MetricsAPI) getAggregatedMetrics(groupBy string) (map[string]MetricsSummary, error) {
	aggregations := make(map[string]MetricsSummary)

	switch groupBy {
	case "state":
		// Group metrics by Saga state
		// This requires accessing the coordinator to get all Sagas and group them
		// For now, we return a simplified version
		states := []string{"Running", "Completed", "Failed", "Compensated"}
		for _, state := range states {
			// In a real implementation, we'd query the coordinator for Sagas in each state
			// and calculate metrics for each group
			aggregations[state] = MetricsSummary{
				Total: 0, // Placeholder
			}
		}

	case "definition":
		// Group metrics by Saga definition
		// This would require tracking per-definition metrics in the collector
		aggregations["placeholder"] = MetricsSummary{
			Total: 0,
		}

	default:
		// No aggregation
	}

	return aggregations, nil
}
