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
	"github.com/innovationmech/swit/pkg/saga/state"
	"go.uber.org/zap"
)

// AlertsAPI provides API endpoints for accessing and managing alerts.
// It integrates with the recovery alerting system to expose alerts via REST API.
type AlertsAPI struct {
	alertManager AlertManager
}

// AlertManager is an interface for accessing alert data and managing alert rules.
type AlertManager interface {
	// GetAlertHistory returns recent alerts with optional limit.
	GetAlertHistory(limit int) []*state.RecoveryAlert
	// GetAlertRules returns all configured alert rules.
	GetAlertRules() []*state.AlertRule
	// GetAlertStats returns statistics about alerts.
	GetAlertStats() AlertStats
}

// AlertStats contains statistics about the alerting system.
type AlertStats struct {
	TotalAlerts      int64            `json:"totalAlerts"`
	ActiveAlerts     int64            `json:"activeAlerts"`
	CriticalAlerts   int64            `json:"criticalAlerts"`
	WarningAlerts    int64            `json:"warningAlerts"`
	InfoAlerts       int64            `json:"infoAlerts"`
	AlertsByRule     map[string]int64 `json:"alertsByRule,omitempty"`
	LastAlertTime    *time.Time       `json:"lastAlertTime,omitempty"`
	AlertRatePerHour float64          `json:"alertRatePerHour"`
}

// NewAlertsAPI creates a new alerts API handler.
//
// Parameters:
//   - alertManager: The alert manager for accessing alert data.
//
// Returns:
//   - A configured AlertsAPI ready to handle alert requests.
func NewAlertsAPI(alertManager AlertManager) *AlertsAPI {
	return &AlertsAPI{
		alertManager: alertManager,
	}
}

// AlertListRequest represents query parameters for alert list requests.
type AlertListRequest struct {
	// Limit specifies the maximum number of alerts to return (default: 50, max: 500)
	Limit int `form:"limit" json:"limit" binding:"omitempty,min=1,max=500"`

	// Severity filters alerts by severity level: "info", "warning", "critical"
	Severity string `form:"severity" json:"severity" binding:"omitempty,oneof=info warning critical"`

	// Since filters alerts that occurred after this timestamp
	Since *time.Time `form:"since" json:"since" time_format:"2006-01-02T15:04:05Z07:00" binding:"omitempty"`

	// Rule filters alerts by rule name
	Rule string `form:"rule" json:"rule" binding:"omitempty"`
}

// ApplyDefaults applies default values to the request.
func (r *AlertListRequest) ApplyDefaults() {
	if r.Limit <= 0 {
		r.Limit = 50
	}
	if r.Limit > 500 {
		r.Limit = 500
	}
}

// AlertDTO represents the data transfer object for an alert.
type AlertDTO struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Timestamp   time.Time         `json:"timestamp"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Value       interface{}       `json:"value,omitempty"`
	Threshold   interface{}       `json:"threshold,omitempty"`
}

// AlertListResponse represents the response for alert list queries.
type AlertListResponse struct {
	// List of alerts
	Alerts []AlertDTO `json:"alerts"`

	// Total number of alerts
	Total int `json:"total"`

	// Statistics about alerts
	Stats AlertStats `json:"stats"`

	// Timestamp when the response was generated
	Timestamp time.Time `json:"timestamp"`
}

// AlertRuleDTO represents the data transfer object for an alert rule.
type AlertRuleDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Enabled     bool   `json:"enabled"`
}

// AlertRulesResponse represents the response for alert rules query.
type AlertRulesResponse struct {
	Rules     []AlertRuleDTO `json:"rules"`
	Total     int            `json:"total"`
	Timestamp time.Time      `json:"timestamp"`
}

// AlertStatsResponse represents the response for alert statistics query.
type AlertStatsResponse struct {
	Stats     AlertStats `json:"stats"`
	Timestamp time.Time  `json:"timestamp"`
}

// GetAlerts handles GET /api/alerts - retrieves alert list with filtering.
//
// Query parameters:
//   - limit: Maximum number of alerts to return (default: 50, max: 500)
//   - severity: Filter by severity ("info", "warning", "critical")
//   - since: Filter alerts after this timestamp (RFC3339 format)
//   - rule: Filter by rule name
//
// Response:
//
//	{
//	  "alerts": [
//	    {
//	      "id": "HighRecoveryFailureRate-1234567890",
//	      "name": "HighRecoveryFailureRate",
//	      "description": "Recovery failure rate exceeds threshold",
//	      "severity": "warning",
//	      "timestamp": "2025-10-23T10:30:00Z",
//	      "value": 0.15,
//	      "threshold": 0.10
//	    }
//	  ],
//	  "total": 25,
//	  "stats": {...},
//	  "timestamp": "2025-10-23T10:35:00Z"
//	}
func (api *AlertsAPI) GetAlerts(c *gin.Context) {
	startTime := time.Now()

	if logger.Logger != nil {
		logger.Logger.Info("Get alerts request received")
	}

	// Parse query parameters
	var req AlertListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Invalid alerts request",
				zap.Error(err))
		}
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid query parameters: " + err.Error(),
		})
		return
	}

	// Apply defaults
	req.ApplyDefaults()

	// Get alert history from manager
	alerts := api.alertManager.GetAlertHistory(req.Limit)

	// Filter alerts based on request parameters
	filteredAlerts := filterAlerts(alerts, &req)

	// Convert to DTOs
	alertDTOs := make([]AlertDTO, 0, len(filteredAlerts))
	for _, alert := range filteredAlerts {
		alertDTOs = append(alertDTOs, *fromRecoveryAlert(alert))
	}

	// Get alert statistics
	stats := api.alertManager.GetAlertStats()

	response := AlertListResponse{
		Alerts:    alertDTOs,
		Total:     len(alertDTOs),
		Stats:     stats,
		Timestamp: time.Now(),
	}

	duration := time.Since(startTime)
	if logger.Logger != nil {
		logger.Logger.Info("Alerts retrieved successfully",
			zap.Duration("duration", duration),
			zap.Int("total_alerts", len(alertDTOs)))
	}

	c.JSON(http.StatusOK, response)
}

// GetAlertRules handles GET /api/alerts/rules - retrieves alert rule configuration.
//
// Response:
//
//	{
//	  "rules": [
//	    {
//	      "name": "HighRecoveryFailureRate",
//	      "description": "Recovery failure rate exceeds threshold",
//	      "severity": "warning",
//	      "enabled": true
//	    }
//	  ],
//	  "total": 4,
//	  "timestamp": "2025-10-23T10:35:00Z"
//	}
func (api *AlertsAPI) GetAlertRules(c *gin.Context) {
	if logger.Logger != nil {
		logger.Logger.Info("Get alert rules request received")
	}

	// Get alert rules from manager
	rules := api.alertManager.GetAlertRules()

	// Convert to DTOs
	ruleDTOs := make([]AlertRuleDTO, 0, len(rules))
	for _, rule := range rules {
		ruleDTOs = append(ruleDTOs, AlertRuleDTO{
			Name:        rule.Name,
			Description: rule.Description,
			Severity:    string(rule.Severity),
			Enabled:     rule.Enabled,
		})
	}

	response := AlertRulesResponse{
		Rules:     ruleDTOs,
		Total:     len(ruleDTOs),
		Timestamp: time.Now(),
	}

	if logger.Logger != nil {
		logger.Logger.Info("Alert rules retrieved successfully",
			zap.Int("total_rules", len(ruleDTOs)))
	}

	c.JSON(http.StatusOK, response)
}

// GetAlertStats handles GET /api/alerts/stats - retrieves alert statistics.
//
// Response:
//
//	{
//	  "stats": {
//	    "totalAlerts": 100,
//	    "activeAlerts": 5,
//	    "criticalAlerts": 2,
//	    "warningAlerts": 3,
//	    "infoAlerts": 0,
//	    "alertRatePerHour": 4.5,
//	    "lastAlertTime": "2025-10-23T10:30:00Z"
//	  },
//	  "timestamp": "2025-10-23T10:35:00Z"
//	}
func (api *AlertsAPI) GetAlertStats(c *gin.Context) {
	if logger.Logger != nil {
		logger.Logger.Info("Get alert stats request received")
	}

	// Get alert statistics from manager
	stats := api.alertManager.GetAlertStats()

	response := AlertStatsResponse{
		Stats:     stats,
		Timestamp: time.Now(),
	}

	if logger.Logger != nil {
		logger.Logger.Info("Alert stats retrieved successfully",
			zap.Int64("total_alerts", stats.TotalAlerts),
			zap.Int64("active_alerts", stats.ActiveAlerts))
	}

	c.JSON(http.StatusOK, response)
}

// GetAlert handles GET /api/alerts/:id - retrieves a specific alert by ID.
//
// Response:
//
//	{
//	  "id": "HighRecoveryFailureRate-1234567890",
//	  "name": "HighRecoveryFailureRate",
//	  "description": "Recovery failure rate exceeds threshold",
//	  "severity": "warning",
//	  "timestamp": "2025-10-23T10:30:00Z",
//	  "value": 0.15,
//	  "threshold": 0.10
//	}
func (api *AlertsAPI) GetAlert(c *gin.Context) {
	alertID := c.Param("id")

	if logger.Logger != nil {
		logger.Logger.Info("Get alert request received",
			zap.String("alert_id", sanitizeForLog(alertID)))
	}

	// Get all alerts and find the one with matching ID
	alerts := api.alertManager.GetAlertHistory(1000) // Get a large enough set

	var foundAlert *state.RecoveryAlert
	for _, alert := range alerts {
		if alert.ID == alertID {
			foundAlert = alert
			break
		}
	}

	if foundAlert == nil {
		if logger.Logger != nil {
			logger.Logger.Warn("Alert not found",
				zap.String("alert_id", sanitizeForLog(alertID)))
		}
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Alert not found",
		})
		return
	}

	alertDTO := fromRecoveryAlert(foundAlert)

	if logger.Logger != nil {
		logger.Logger.Info("Alert retrieved successfully",
			zap.String("alert_id", sanitizeForLog(alertID)))
	}

	c.JSON(http.StatusOK, alertDTO)
}

// AcknowledgeAlert handles POST /api/alerts/:id/acknowledge - acknowledges an alert.
//
// This endpoint is a placeholder for future functionality where alerts can be
// acknowledged or silenced by operators.
//
// Response:
//
//	{
//	  "success": true,
//	  "message": "Alert acknowledged successfully"
//	}
func (api *AlertsAPI) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")

	if logger.Logger != nil {
		logger.Logger.Info("Acknowledge alert request received",
			zap.String("alert_id", sanitizeForLog(alertID)))
	}

	// TODO: Implement alert acknowledgment functionality
	// This would require extending the RecoveryAlertingManager to support
	// alert acknowledgment and silence periods

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Alert acknowledgment functionality not yet implemented",
		"alertId": alertID,
	})
}

// filterAlerts filters alerts based on the request parameters.
func filterAlerts(alerts []*state.RecoveryAlert, req *AlertListRequest) []*state.RecoveryAlert {
	filtered := make([]*state.RecoveryAlert, 0, len(alerts))

	for _, alert := range alerts {
		// Filter by severity
		if req.Severity != "" && string(alert.Severity) != req.Severity {
			continue
		}

		// Filter by timestamp
		if req.Since != nil && alert.Timestamp.Before(*req.Since) {
			continue
		}

		// Filter by rule name
		if req.Rule != "" && alert.Name != req.Rule {
			continue
		}

		filtered = append(filtered, alert)
	}

	return filtered
}

// fromRecoveryAlert converts a state.RecoveryAlert to AlertDTO.
func fromRecoveryAlert(alert *state.RecoveryAlert) *AlertDTO {
	return &AlertDTO{
		ID:          alert.ID,
		Name:        alert.Name,
		Description: alert.Description,
		Severity:    string(alert.Severity),
		Timestamp:   alert.Timestamp,
		Labels:      alert.Labels,
		Annotations: alert.Annotations,
		Value:       alert.Value,
		Threshold:   alert.Threshold,
	}
}
