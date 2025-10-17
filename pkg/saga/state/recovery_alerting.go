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

package state

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RecoveryAlertSeverity represents the severity level of a recovery alert.
type RecoveryAlertSeverity string

const (
	// AlertSeverityInfo indicates informational alerts.
	AlertSeverityInfo RecoveryAlertSeverity = "info"

	// AlertSeverityWarning indicates warning-level alerts.
	AlertSeverityWarning RecoveryAlertSeverity = "warning"

	// AlertSeverityCritical indicates critical alerts.
	AlertSeverityCritical RecoveryAlertSeverity = "critical"
)

// RecoveryAlert represents a recovery system alert.
type RecoveryAlert struct {
	// ID is a unique identifier for the alert.
	ID string `json:"id"`

	// Name is the alert name.
	Name string `json:"name"`

	// Description provides details about the alert.
	Description string `json:"description"`

	// Severity indicates the alert severity.
	Severity RecoveryAlertSeverity `json:"severity"`

	// Timestamp when the alert was triggered.
	Timestamp time.Time `json:"timestamp"`

	// Labels for categorization and routing.
	Labels map[string]string `json:"labels"`

	// Annotations provide additional context.
	Annotations map[string]string `json:"annotations"`

	// Value that triggered the alert.
	Value interface{} `json:"value"`

	// Threshold that was exceeded.
	Threshold interface{} `json:"threshold"`
}

// AlertNotifier is an interface for alert notification handlers.
type AlertNotifier interface {
	// Notify sends an alert notification.
	Notify(ctx context.Context, alert *RecoveryAlert) error
}

// AlertRule defines a condition that triggers an alert.
type AlertRule struct {
	// Name is the rule name.
	Name string

	// Description describes what this rule checks.
	Description string

	// Severity is the alert severity if triggered.
	Severity RecoveryAlertSeverity

	// CheckFunc is the function that evaluates the rule.
	// Returns true if the alert should be triggered.
	CheckFunc func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{})

	// Enabled determines if the rule is active.
	Enabled bool
}

// RecoveryAlertingConfig holds configuration for the alerting system.
type RecoveryAlertingConfig struct {
	// Enabled determines if alerting is active.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// CheckInterval is how often to check alert rules.
	CheckInterval time.Duration `yaml:"check_interval" json:"check_interval"`

	// DeduplicationWindow is the time window to deduplicate alerts.
	DeduplicationWindow time.Duration `yaml:"deduplication_window" json:"deduplication_window"`

	// MaxAlertsPerMinute limits alert rate to prevent alert storms.
	MaxAlertsPerMinute int `yaml:"max_alerts_per_minute" json:"max_alerts_per_minute"`

	// HighFailureRateThreshold is the threshold for high failure rate alerts (0-1).
	HighFailureRateThreshold float64 `yaml:"high_failure_rate_threshold" json:"high_failure_rate_threshold"`

	// TooManyStuckSagasThreshold is the threshold for too many stuck Sagas.
	TooManyStuckSagasThreshold int64 `yaml:"too_many_stuck_sagas_threshold" json:"too_many_stuck_sagas_threshold"`

	// SlowRecoveryThreshold is the threshold for slow recovery alerts.
	SlowRecoveryThreshold time.Duration `yaml:"slow_recovery_threshold" json:"slow_recovery_threshold"`
}

// DefaultRecoveryAlertingConfig returns default alerting configuration.
func DefaultRecoveryAlertingConfig() *RecoveryAlertingConfig {
	return &RecoveryAlertingConfig{
		Enabled:                    true,
		CheckInterval:              30 * time.Second,
		DeduplicationWindow:        5 * time.Minute,
		MaxAlertsPerMinute:         10,
		HighFailureRateThreshold:   0.1, // 10% failure rate
		TooManyStuckSagasThreshold: 10,  // 10 stuck Sagas
		SlowRecoveryThreshold:      30 * time.Second,
	}
}

// RecoveryAlertingManager manages alerts for the recovery system.
type RecoveryAlertingManager struct {
	config    *RecoveryAlertingConfig
	rules     []*AlertRule
	notifiers []AlertNotifier
	metrics   *RecoveryMetrics
	logger    *zap.Logger

	// Alert deduplication
	recentAlerts map[string]time.Time
	alertMu      sync.RWMutex

	// Rate limiting
	alertCounts   map[string]int // minute -> count
	alertCountsMu sync.Mutex
	lastMinute    time.Time

	// Alert history
	alertHistory    []*RecoveryAlert
	alertHistoryMu  sync.RWMutex
	maxAlertHistory int

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}
	doneCh chan struct{}
}

// NewRecoveryAlertingManager creates a new alerting manager.
func NewRecoveryAlertingManager(
	config *RecoveryAlertingConfig,
	metrics *RecoveryMetrics,
	logger *zap.Logger,
) *RecoveryAlertingManager {
	if config == nil {
		config = DefaultRecoveryAlertingConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	am := &RecoveryAlertingManager{
		config:          config,
		rules:           make([]*AlertRule, 0),
		notifiers:       make([]AlertNotifier, 0),
		metrics:         metrics,
		logger:          logger.With(zap.String("component", "recovery_alerting")),
		recentAlerts:    make(map[string]time.Time),
		alertCounts:     make(map[string]int),
		alertHistory:    make([]*RecoveryAlert, 0),
		maxAlertHistory: 100, // Keep last 100 alerts
		ctx:             ctx,
		cancel:          cancel,
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}

	// Register default alert rules
	am.registerDefaultRules()

	return am
}

// registerDefaultRules registers built-in alert rules.
func (am *RecoveryAlertingManager) registerDefaultRules() {
	// High recovery failure rate alert
	am.AddRule(&AlertRule{
		Name:        "HighRecoveryFailureRate",
		Description: "Recovery failure rate exceeds threshold",
		Severity:    AlertSeverityWarning,
		Enabled:     true,
		CheckFunc: func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			if snapshot.TotalAttempts < 10 {
				return false, nil, nil // Need sufficient data
			}
			return snapshot.SuccessRate < (1 - am.config.HighFailureRateThreshold),
				snapshot.SuccessRate,
				(1 - am.config.HighFailureRateThreshold)
		},
	})

	// Too many stuck Sagas alert
	am.AddRule(&AlertRule{
		Name:        "TooManyStuckSagas",
		Description: "Number of stuck Sagas exceeds threshold",
		Severity:    AlertSeverityCritical,
		Enabled:     true,
		CheckFunc: func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			total := snapshot.DetectedStuck + snapshot.DetectedTimeouts
			return total > am.config.TooManyStuckSagasThreshold,
				total,
				am.config.TooManyStuckSagasThreshold
		},
	})

	// Slow recovery alert
	am.AddRule(&AlertRule{
		Name:        "SlowRecovery",
		Description: "Average recovery time exceeds threshold",
		Severity:    AlertSeverityWarning,
		Enabled:     true,
		CheckFunc: func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			if snapshot.TotalAttempts < 5 {
				return false, nil, nil // Need sufficient data
			}
			return snapshot.AverageDuration > am.config.SlowRecoveryThreshold,
				snapshot.AverageDuration,
				am.config.SlowRecoveryThreshold
		},
	})

	// High manual intervention rate alert
	am.AddRule(&AlertRule{
		Name:        "HighManualInterventionRate",
		Description: "Manual intervention rate is high",
		Severity:    AlertSeverityWarning,
		Enabled:     true,
		CheckFunc: func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			if snapshot.TotalAttempts < 10 {
				return false, nil, nil
			}
			rate := float64(snapshot.ManualInterventions) / float64(snapshot.TotalAttempts)
			threshold := 0.2 // 20% manual intervention rate
			return rate > threshold, rate, threshold
		},
	})
}

// AddRule registers a new alert rule.
func (am *RecoveryAlertingManager) AddRule(rule *AlertRule) {
	am.rules = append(am.rules, rule)
	am.logger.Info("alert rule registered",
		zap.String("rule", rule.Name),
		zap.String("severity", string(rule.Severity)),
	)
}

// AddNotifier registers a new alert notifier.
func (am *RecoveryAlertingManager) AddNotifier(notifier AlertNotifier) {
	am.notifiers = append(am.notifiers, notifier)
	am.logger.Info("alert notifier registered")
}

// Start begins the alerting check loop.
func (am *RecoveryAlertingManager) Start() {
	if !am.config.Enabled {
		am.logger.Info("alerting disabled, not starting check loop")
		return
	}

	go am.alertingLoop()
	am.logger.Info("alerting manager started",
		zap.Duration("check_interval", am.config.CheckInterval),
	)
}

// Stop stops the alerting check loop.
func (am *RecoveryAlertingManager) Stop() {
	if !am.config.Enabled {
		return
	}

	close(am.stopCh)
	<-am.doneCh
	am.cancel()

	am.logger.Info("alerting manager stopped")
}

// alertingLoop is the main loop that periodically checks alert rules.
func (am *RecoveryAlertingManager) alertingLoop() {
	defer close(am.doneCh)

	ticker := time.NewTicker(am.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-am.stopCh:
			return
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.checkAlertRules()
		}
	}
}

// checkAlertRules evaluates all alert rules.
func (am *RecoveryAlertingManager) checkAlertRules() {
	if am.metrics == nil {
		return
	}

	snapshot := am.metrics.GetSnapshot()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		triggered, value, threshold := rule.CheckFunc(&snapshot)
		if triggered {
			alert := &RecoveryAlert{
				ID:          fmt.Sprintf("%s-%d", rule.Name, time.Now().UnixNano()),
				Name:        rule.Name,
				Description: rule.Description,
				Severity:    rule.Severity,
				Timestamp:   time.Now(),
				Labels: map[string]string{
					"rule": rule.Name,
				},
				Annotations: map[string]string{
					"description": rule.Description,
				},
				Value:     value,
				Threshold: threshold,
			}

			am.sendAlert(alert)
		}
	}
}

// sendAlert sends an alert if it passes deduplication and rate limiting.
func (am *RecoveryAlertingManager) sendAlert(alert *RecoveryAlert) {
	// Check deduplication
	if !am.shouldSendAlert(alert) {
		return
	}

	// Check rate limiting
	if !am.checkRateLimit() {
		am.logger.Warn("alert rate limit exceeded, dropping alert",
			zap.String("alert", alert.Name),
		)
		return
	}

	// Store in history
	am.addToAlertHistory(alert)

	// Send to notifiers
	am.logger.Info("sending alert",
		zap.String("alert", alert.Name),
		zap.String("severity", string(alert.Severity)),
		zap.Any("value", alert.Value),
		zap.Any("threshold", alert.Threshold),
	)

	ctx, cancel := context.WithTimeout(am.ctx, 5*time.Second)
	defer cancel()

	for _, notifier := range am.notifiers {
		go func(n AlertNotifier) {
			if err := n.Notify(ctx, alert); err != nil {
				am.logger.Error("failed to send alert",
					zap.String("alert", alert.Name),
					zap.Error(err),
				)
			}
		}(notifier)
	}
}

// shouldSendAlert checks if an alert should be sent based on deduplication.
func (am *RecoveryAlertingManager) shouldSendAlert(alert *RecoveryAlert) bool {
	am.alertMu.Lock()
	defer am.alertMu.Unlock()

	key := alert.Name
	lastSent, exists := am.recentAlerts[key]

	if exists && time.Since(lastSent) < am.config.DeduplicationWindow {
		return false // Skip duplicate
	}

	am.recentAlerts[key] = time.Now()

	// Cleanup old entries
	for k, t := range am.recentAlerts {
		if time.Since(t) > am.config.DeduplicationWindow {
			delete(am.recentAlerts, k)
		}
	}

	return true
}

// checkRateLimit checks if we're within the alert rate limit.
func (am *RecoveryAlertingManager) checkRateLimit() bool {
	am.alertCountsMu.Lock()
	defer am.alertCountsMu.Unlock()

	now := time.Now()
	currentMinute := now.Truncate(time.Minute)

	// Reset counter if we're in a new minute
	if !currentMinute.Equal(am.lastMinute) {
		am.lastMinute = currentMinute
		am.alertCounts = make(map[string]int)
	}

	key := currentMinute.Format(time.RFC3339)
	count := am.alertCounts[key]

	if count >= am.config.MaxAlertsPerMinute {
		return false
	}

	am.alertCounts[key]++
	return true
}

// addToAlertHistory adds an alert to the history.
func (am *RecoveryAlertingManager) addToAlertHistory(alert *RecoveryAlert) {
	am.alertHistoryMu.Lock()
	defer am.alertHistoryMu.Unlock()

	am.alertHistory = append(am.alertHistory, alert)

	// Trim history if it exceeds max size
	if len(am.alertHistory) > am.maxAlertHistory {
		am.alertHistory = am.alertHistory[len(am.alertHistory)-am.maxAlertHistory:]
	}
}

// GetAlertHistory returns recent alerts.
func (am *RecoveryAlertingManager) GetAlertHistory(limit int) []*RecoveryAlert {
	am.alertHistoryMu.RLock()
	defer am.alertHistoryMu.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	// Return most recent alerts
	result := make([]*RecoveryAlert, limit)
	startIdx := len(am.alertHistory) - limit
	copy(result, am.alertHistory[startIdx:])

	return result
}

// LogAlertNotifier is a simple notifier that logs alerts.
type LogAlertNotifier struct {
	logger *zap.Logger
}

// NewLogAlertNotifier creates a new log-based alert notifier.
func NewLogAlertNotifier(logger *zap.Logger) *LogAlertNotifier {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LogAlertNotifier{logger: logger}
}

// Notify logs the alert.
func (n *LogAlertNotifier) Notify(ctx context.Context, alert *RecoveryAlert) error {
	logFunc := n.logger.Info
	switch alert.Severity {
	case AlertSeverityWarning:
		logFunc = n.logger.Warn
	case AlertSeverityCritical:
		logFunc = n.logger.Error
	}

	logFunc("recovery alert",
		zap.String("alert_id", alert.ID),
		zap.String("alert_name", alert.Name),
		zap.String("description", alert.Description),
		zap.String("severity", string(alert.Severity)),
		zap.Any("value", alert.Value),
		zap.Any("threshold", alert.Threshold),
		zap.Time("timestamp", alert.Timestamp),
	)

	return nil
}
