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
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/saga/state"
	"go.uber.org/zap"
)

// maxTrackedAcknowledgments bounds the in-memory acknowledgment store to
// match the bounded alert history kept by the alert manager.
const maxTrackedAcknowledgments = 1000

// AlertsAPI provides API endpoints for accessing and managing alerts.
// It integrates with the recovery alerting system to expose alerts via REST API.
//
// Alert acknowledgments are tracked in-memory by this API layer, matching the
// in-memory alert history of the underlying alert manager. They are not
// persisted across restarts.
type AlertsAPI struct {
	alertManager AlertManager

	ackMu           sync.RWMutex
	acknowledgments map[string]*AlertAcknowledgment
	ackOrder        []string
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
		alertManager:    alertManager,
		acknowledgments: make(map[string]*AlertAcknowledgment),
	}
}

// AlertAcknowledgment records the acknowledgment of an alert by an operator.
type AlertAcknowledgment struct {
	// AlertID is the ID of the acknowledged alert.
	AlertID string `json:"alertId"`

	// AcknowledgedBy identifies the operator who acknowledged the alert.
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`

	// Comment is an optional note attached to the acknowledgment.
	Comment string `json:"comment,omitempty"`

	// AcknowledgedAt is the time the alert was acknowledged.
	AcknowledgedAt time.Time `json:"acknowledgedAt"`
}

// AcknowledgeAlertRequest represents the optional request body for alert acknowledgment.
type AcknowledgeAlertRequest struct {
	// AcknowledgedBy identifies the operator acknowledging the alert.
	AcknowledgedBy string `json:"acknowledgedBy" binding:"omitempty,max=256"`

	// Comment is an optional note attached to the acknowledgment.
	Comment string `json:"comment" binding:"omitempty,max=1024"`
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

	// Acknowledged indicates whether the alert has been acknowledged by an operator.
	Acknowledged bool `json:"acknowledged"`

	// Acknowledgment contains acknowledgment details if the alert was acknowledged.
	Acknowledgment *AlertAcknowledgment `json:"acknowledgment,omitempty"`
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
		dto := fromRecoveryAlert(alert)
		api.attachAcknowledgment(dto)
		alertDTOs = append(alertDTOs, *dto)
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
	api.attachAcknowledgment(alertDTO)

	if logger.Logger != nil {
		logger.Logger.Info("Alert retrieved successfully",
			zap.String("alert_id", sanitizeForLog(alertID)))
	}

	c.JSON(http.StatusOK, alertDTO)
}

// AcknowledgeAlert handles POST /api/alerts/:id/acknowledge - acknowledges an alert.
//
// The acknowledgment is recorded in-memory by the API layer and reflected in
// subsequent alert queries via the "acknowledged" and "acknowledgment" fields.
// Acknowledging an already-acknowledged alert is idempotent and returns the
// existing acknowledgment.
//
// Request body (optional):
//
//	{
//	  "acknowledgedBy": "operator@example.com",
//	  "comment": "Investigating the failure spike"
//	}
//
// Response:
//
//	{
//	  "success": true,
//	  "message": "Alert acknowledged successfully",
//	  "alertId": "HighRecoveryFailureRate-1234567890",
//	  "acknowledgment": {
//	    "alertId": "HighRecoveryFailureRate-1234567890",
//	    "acknowledgedBy": "operator@example.com",
//	    "acknowledgedAt": "2025-10-23T10:35:00Z"
//	  }
//	}
func (api *AlertsAPI) AcknowledgeAlert(c *gin.Context) {
	alertID := c.Param("id")

	if logger.Logger != nil {
		logger.Logger.Info("Acknowledge alert request received",
			zap.String("alert_id", sanitizeForLog(alertID)))
	}

	// Parse the optional request body
	var req AcknowledgeAlertRequest
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_request",
				Message: "Invalid request body: " + err.Error(),
			})
			return
		}
	}

	// Verify the alert exists before acknowledging it
	if !api.alertExists(alertID) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Alert not found",
		})
		return
	}

	ack, created := api.recordAcknowledgment(alertID, req.AcknowledgedBy, req.Comment)

	message := "Alert acknowledged successfully"
	if !created {
		message = "Alert was already acknowledged"
	}

	if logger.Logger != nil {
		logger.Logger.Info("Alert acknowledged",
			zap.String("alert_id", sanitizeForLog(alertID)),
			zap.Bool("newly_acknowledged", created))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"message":        message,
		"alertId":        alertID,
		"acknowledgment": ack,
	})
}

// alertExists checks whether an alert with the given ID is present in the alert history.
func (api *AlertsAPI) alertExists(alertID string) bool {
	alerts := api.alertManager.GetAlertHistory(maxTrackedAcknowledgments)
	for _, alert := range alerts {
		if alert.ID == alertID {
			return true
		}
	}
	return false
}

// recordAcknowledgment stores an acknowledgment for the given alert ID.
// It is idempotent: if the alert is already acknowledged, the existing
// acknowledgment is returned and created is false.
func (api *AlertsAPI) recordAcknowledgment(alertID, acknowledgedBy, comment string) (ack *AlertAcknowledgment, created bool) {
	api.ackMu.Lock()
	defer api.ackMu.Unlock()

	if existing, ok := api.acknowledgments[alertID]; ok {
		return existing, false
	}

	ack = &AlertAcknowledgment{
		AlertID:        alertID,
		AcknowledgedBy: acknowledgedBy,
		Comment:        comment,
		AcknowledgedAt: time.Now(),
	}
	api.acknowledgments[alertID] = ack
	api.ackOrder = append(api.ackOrder, alertID)

	// Evict the oldest acknowledgments to keep the store bounded.
	for len(api.ackOrder) > maxTrackedAcknowledgments {
		oldest := api.ackOrder[0]
		api.ackOrder = api.ackOrder[1:]
		delete(api.acknowledgments, oldest)
	}

	return ack, true
}

// getAcknowledgment returns the acknowledgment for the given alert ID, if any.
func (api *AlertsAPI) getAcknowledgment(alertID string) *AlertAcknowledgment {
	api.ackMu.RLock()
	defer api.ackMu.RUnlock()
	return api.acknowledgments[alertID]
}

// attachAcknowledgment enriches an alert DTO with acknowledgment state.
func (api *AlertsAPI) attachAcknowledgment(dto *AlertDTO) {
	if ack := api.getAcknowledgment(dto.ID); ack != nil {
		dto.Acknowledged = true
		dto.Acknowledgment = ack
	}
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
