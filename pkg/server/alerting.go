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
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/innovationmech/swit/pkg/logger"
	"github.com/innovationmech/swit/pkg/types"
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	// AlertSeverityInfo indicates informational alerts
	AlertSeverityInfo AlertSeverity = "info"
	// AlertSeverityWarning indicates warning-level alerts
	AlertSeverityWarning AlertSeverity = "warning"
	// AlertSeverityCritical indicates critical alerts
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a performance or SLO alert
type Alert struct {
	// ID is a unique identifier for this alert
	ID string `json:"id"`

	// Name is the alert name/title
	Name string `json:"name"`

	// Description provides details about the alert
	Description string `json:"description"`

	// Severity indicates the alert level
	Severity AlertSeverity `json:"severity"`

	// Timestamp when the alert was triggered
	Timestamp time.Time `json:"timestamp"`

	// Labels for categorization and routing
	Labels map[string]string `json:"labels"`

	// Annotations provide additional context
	Annotations map[string]string `json:"annotations"`

	// Value that triggered the alert
	Value interface{} `json:"value"`

	// Threshold that was exceeded
	Threshold interface{} `json:"threshold"`

	// Source indicates where the alert originated (performance, slo, custom)
	Source string `json:"source"`
}

// AlertHandler is a function that processes alerts
type AlertHandler func(alert *Alert) error

// AlertingConfig holds configuration for the alerting system
type AlertingConfig struct {
	// Enabled determines if alerting is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// DeduplicationWindow is the time window to deduplicate alerts
	DeduplicationWindow time.Duration `json:"deduplication_window" yaml:"deduplication_window"`

	// MaxAlertsPerMinute limits alert rate
	MaxAlertsPerMinute int `json:"max_alerts_per_minute" yaml:"max_alerts_per_minute"`

	// EnableSLOAlerts determines if SLO-based alerts are enabled
	EnableSLOAlerts bool `json:"enable_slo_alerts" yaml:"enable_slo_alerts"`

	// EnablePerformanceAlerts determines if performance threshold alerts are enabled
	EnablePerformanceAlerts bool `json:"enable_performance_alerts" yaml:"enable_performance_alerts"`
}

// DefaultAlertingConfig returns default alerting configuration
func DefaultAlertingConfig() *AlertingConfig {
	return &AlertingConfig{
		Enabled:                 false,
		DeduplicationWindow:     5 * time.Minute,
		MaxAlertsPerMinute:      10,
		EnableSLOAlerts:         true,
		EnablePerformanceAlerts: true,
	}
}

// AlertingManager manages alert generation and routing
type AlertingManager struct {
	config           *AlertingConfig
	handlers         []AlertHandler
	recentAlerts     map[string]time.Time
	alertCounts      map[string]int
	sloMonitor       *types.SLOMonitor
	metricsCollector types.MetricsCollector
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewAlertingManager creates a new alerting manager
func NewAlertingManager(config *AlertingConfig, sloMonitor *types.SLOMonitor, metricsCollector types.MetricsCollector) *AlertingManager {
	if config == nil {
		config = DefaultAlertingConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &AlertingManager{
		config:           config,
		handlers:         make([]AlertHandler, 0),
		recentAlerts:     make(map[string]time.Time),
		alertCounts:      make(map[string]int),
		sloMonitor:       sloMonitor,
		metricsCollector: metricsCollector,
		ctx:              ctx,
		cancel:           cancel,
	}
}

// AddHandler registers an alert handler
func (am *AlertingManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.handlers = append(am.handlers, handler)
}

// TriggerAlert triggers an alert and routes it to all handlers
func (am *AlertingManager) TriggerAlert(alert *Alert) error {
	if !am.config.Enabled {
		return nil
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	// Check deduplication
	alertKey := fmt.Sprintf("%s:%s", alert.Source, alert.Name)
	if lastTime, exists := am.recentAlerts[alertKey]; exists {
		if time.Since(lastTime) < am.config.DeduplicationWindow {
			// Skip duplicate alert
			return nil
		}
	}

	// Check rate limiting
	minute := time.Now().Truncate(time.Minute)
	minuteKey := fmt.Sprintf("%v", minute.Unix())
	if am.alertCounts[minuteKey] >= am.config.MaxAlertsPerMinute {
		logger.Logger.Warn("Alert rate limit exceeded, dropping alert",
			zap.String("alert", alert.Name),
			zap.Int("limit", am.config.MaxAlertsPerMinute))
		return fmt.Errorf("alert rate limit exceeded")
	}

	// Record alert
	am.recentAlerts[alertKey] = time.Now()
	am.alertCounts[minuteKey]++

	// Emit metrics
	if am.metricsCollector != nil {
		am.metricsCollector.IncrementCounter("alerts_triggered_total", map[string]string{
			"severity": string(alert.Severity),
			"source":   alert.Source,
			"name":     alert.Name,
		})
	}

	// Route to handlers
	for _, handler := range am.handlers {
		go func(h AlertHandler) {
			if err := h(alert); err != nil {
				logger.Logger.Error("Alert handler failed",
					zap.String("alert", alert.Name),
					zap.Error(err))
			}
		}(handler)
	}

	return nil
}

// Start begins monitoring and periodic cleanup
func (am *AlertingManager) Start() {
	if !am.config.Enabled {
		return
	}

	go am.cleanupLoop()
}

// Stop stops the alerting manager
func (am *AlertingManager) Stop() {
	if am.cancel != nil {
		am.cancel()
	}
}

// cleanupLoop periodically cleans up old alert tracking data
func (am *AlertingManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.cleanup()
		}
	}
}

// cleanup removes old alert tracking data
func (am *AlertingManager) cleanup() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()

	// Clean up old deduplication entries
	for key, timestamp := range am.recentAlerts {
		if now.Sub(timestamp) > am.config.DeduplicationWindow*2 {
			delete(am.recentAlerts, key)
		}
	}

	// Clean up old alert counts (keep last 5 minutes)
	cutoff := now.Add(-5 * time.Minute).Truncate(time.Minute)
	for key := range am.alertCounts {
		var keyTime int64
		fmt.Sscanf(key, "%d", &keyTime)
		if time.Unix(keyTime, 0).Before(cutoff) {
			delete(am.alertCounts, key)
		}
	}
}

// PerformanceAlertingHook creates a performance hook that generates alerts for threshold violations
func PerformanceAlertingHook(alertingManager *AlertingManager) PerformanceHook {
	return func(event string, metrics *PerformanceMetrics) {
		if alertingManager == nil || !alertingManager.config.EnablePerformanceAlerts {
			return
		}

		violations := metrics.CheckThresholds()
		for _, violation := range violations {
			alert := &Alert{
				ID:          fmt.Sprintf("perf-%s-%d", event, time.Now().Unix()),
				Name:        fmt.Sprintf("Performance threshold violation: %s", event),
				Description: violation,
				Severity:    AlertSeverityWarning,
				Timestamp:   time.Now(),
				Labels: map[string]string{
					"type":     "performance",
					"event":    event,
					"category": "threshold",
				},
				Annotations: map[string]string{
					"violation": violation,
					"event":     event,
				},
				Source: "performance",
			}

			if err := alertingManager.TriggerAlert(alert); err != nil {
				logger.Logger.Error("Failed to trigger performance alert",
					zap.String("event", event),
					zap.Error(err))
			}
		}
	}
}

// SLOAlertingHook creates a performance hook that generates alerts for SLO violations
func SLOAlertingHook(alertingManager *AlertingManager, sloMonitor *types.SLOMonitor) PerformanceHook {
	return func(event string, metrics *PerformanceMetrics) {
		if alertingManager == nil || !alertingManager.config.EnableSLOAlerts || sloMonitor == nil {
			return
		}

		statuses := sloMonitor.GetAllStatuses()
		for _, status := range statuses {
			var severity AlertSeverity
			var shouldAlert bool

			switch status.Status {
			case types.SLOStatusCritical:
				severity = AlertSeverityCritical
				shouldAlert = true
			case types.SLOStatusWarning:
				severity = AlertSeverityWarning
				shouldAlert = true
			case types.SLOStatusHealthy:
				continue
			default:
				continue
			}

			if !shouldAlert {
				continue
			}

			alert := &Alert{
				ID:          fmt.Sprintf("slo-%s-%d", status.Target.Name, time.Now().Unix()),
				Name:        fmt.Sprintf("SLO violation: %s", status.Target.Name),
				Description: fmt.Sprintf("SLO %s is at %.2f%% (target: %.2f%%)", status.Target.Description, status.CurrentValue, status.Target.Target),
				Severity:    severity,
				Timestamp:   time.Now(),
				Labels: map[string]string{
					"type":     "slo",
					"slo_name": status.Target.Name,
					"slo_type": string(status.Target.Type),
				},
				Annotations: map[string]string{
					"current_value":          fmt.Sprintf("%.2f%%", status.CurrentValue),
					"target_value":           fmt.Sprintf("%.2f%%", status.Target.Target),
					"error_budget_remaining": fmt.Sprintf("%.2f%%", status.ErrorBudgetRemaining),
					"slo_window":             status.Target.Window.String(),
				},
				Value:     status.CurrentValue,
				Threshold: status.Target.Target,
				Source:    "slo",
			}

			if err := alertingManager.TriggerAlert(alert); err != nil {
				logger.Logger.Error("Failed to trigger SLO alert",
					zap.String("slo", status.Target.Name),
					zap.Error(err))
			}
		}
	}
}

// LoggingAlertHandler creates an alert handler that logs alerts
func LoggingAlertHandler(alert *Alert) error {
	fields := []zap.Field{
		zap.String("alert_id", alert.ID),
		zap.String("name", alert.Name),
		zap.String("severity", string(alert.Severity)),
		zap.Time("timestamp", alert.Timestamp),
		zap.String("source", alert.Source),
		zap.Any("labels", alert.Labels),
	}

	switch alert.Severity {
	case AlertSeverityCritical:
		logger.Logger.Error(alert.Description, fields...)
	case AlertSeverityWarning:
		logger.Logger.Warn(alert.Description, fields...)
	default:
		logger.Logger.Info(alert.Description, fields...)
	}

	return nil
}

// MetricsAlertHandler creates an alert handler that emits metrics
func MetricsAlertHandler(metricsCollector types.MetricsCollector) AlertHandler {
	return func(alert *Alert) error {
		if metricsCollector == nil {
			return nil
		}

		metricsCollector.IncrementCounter("alert_handled_total", map[string]string{
			"severity": string(alert.Severity),
			"source":   alert.Source,
			"name":     alert.Name,
		})

		return nil
	}
}

// PrometheusAlertHandler creates an alert handler that exposes alerts as Prometheus metrics
func PrometheusAlertHandler(metricsCollector types.MetricsCollector) AlertHandler {
	return func(alert *Alert) error {
		if metricsCollector == nil {
			return nil
		}

		// Set gauge to 1 when alert is active
		metricsCollector.SetGauge("alert_active", 1, map[string]string{
			"alert_name": alert.Name,
			"severity":   string(alert.Severity),
			"source":     alert.Source,
		})

		// Record alert value if numeric
		if value, ok := alert.Value.(float64); ok {
			metricsCollector.SetGauge("alert_value", value, map[string]string{
				"alert_name": alert.Name,
				"source":     alert.Source,
			})
		}

		// Record threshold if numeric
		if threshold, ok := alert.Threshold.(float64); ok {
			metricsCollector.SetGauge("alert_threshold", threshold, map[string]string{
				"alert_name": alert.Name,
				"source":     alert.Source,
			})
		}

		return nil
	}
}
