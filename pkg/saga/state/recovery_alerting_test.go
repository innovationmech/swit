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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// mockAlertNotifier is a mock implementation of AlertNotifier for testing.
type mockAlertNotifier struct {
	mu       sync.Mutex
	alerts   []*RecoveryAlert
	shouldFail bool
}

func (m *mockAlertNotifier) Notify(ctx context.Context, alert *RecoveryAlert) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shouldFail {
		return assert.AnError
	}

	m.alerts = append(m.alerts, alert)
	return nil
}

func (m *mockAlertNotifier) getAlerts() []*RecoveryAlert {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*RecoveryAlert, len(m.alerts))
	copy(result, m.alerts)
	return result
}

func (m *mockAlertNotifier) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alerts = nil
}

func TestNewRecoveryAlertingManager(t *testing.T) {
	logger := zaptest.NewLogger(t)

	tests := []struct {
		name    string
		config  *RecoveryAlertingConfig
		metrics *RecoveryMetrics
		wantNil bool
	}{
		{
			name:    "default config",
			config:  nil,
			metrics: nil,
			wantNil: false,
		},
		{
			name:    "custom config",
			config:  DefaultRecoveryAlertingConfig(),
			metrics: nil,
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewRecoveryAlertingManager(tt.config, tt.metrics, logger)

			if tt.wantNil {
				assert.Nil(t, am)
			} else {
				assert.NotNil(t, am)
				assert.NotNil(t, am.config)
				assert.NotEmpty(t, am.rules) // Should have default rules
			}
		})
	}
}

func TestRecoveryAlertingManager_AddRule(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewRecoveryAlertingManager(nil, nil, logger)

	initialRules := len(am.rules)

	rule := &AlertRule{
		Name:        "TestRule",
		Description: "Test rule",
		Severity:    AlertSeverityWarning,
		Enabled:     true,
		CheckFunc: func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			return false, nil, nil
		},
	}

	am.AddRule(rule)
	assert.Equal(t, initialRules+1, len(am.rules))
}

func TestRecoveryAlertingManager_AddNotifier(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewRecoveryAlertingManager(nil, nil, logger)

	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	assert.Equal(t, 1, len(am.notifiers))
}

func TestRecoveryAlertingManager_HighFailureRateAlert(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	config := DefaultRecoveryAlertingConfig()
	config.HighFailureRateThreshold = 0.3 // 30% threshold

	am := NewRecoveryAlertingManager(config, metrics, logger)
	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	// Record metrics with high failure rate (40%)
	for i := 0; i < 6; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoverySuccess("resume", "test_saga", time.Second)
	}
	for i := 0; i < 4; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoveryFailure("resume", "test_saga", "timeout", time.Second)
	}

	// Check rules
	am.checkAlertRules()

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	// Should trigger high failure rate alert
	alerts := notifier.getAlerts()
	assert.NotEmpty(t, alerts)

	found := false
	for _, alert := range alerts {
		if alert.Name == "HighRecoveryFailureRate" {
			found = true
			assert.Equal(t, AlertSeverityWarning, alert.Severity)
			break
		}
	}
	assert.True(t, found, "HighRecoveryFailureRate alert should be triggered")
}

func TestRecoveryAlertingManager_TooManyStuckSagasAlert(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	config := DefaultRecoveryAlertingConfig()
	config.TooManyStuckSagasThreshold = 5

	am := NewRecoveryAlertingManager(config, metrics, logger)
	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	// Record many stuck/timeout failures
	for i := 0; i < 6; i++ {
		metrics.RecordDetectedFailure("stuck")
	}

	// Check rules
	am.checkAlertRules()

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	// Should trigger too many stuck Sagas alert
	alerts := notifier.getAlerts()
	assert.NotEmpty(t, alerts)

	found := false
	for _, alert := range alerts {
		if alert.Name == "TooManyStuckSagas" {
			found = true
			assert.Equal(t, AlertSeverityCritical, alert.Severity)
			break
		}
	}
	assert.True(t, found, "TooManyStuckSagas alert should be triggered")
}

func TestRecoveryAlertingManager_SlowRecoveryAlert(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	config := DefaultRecoveryAlertingConfig()
	config.SlowRecoveryThreshold = 10 * time.Second

	am := NewRecoveryAlertingManager(config, metrics, logger)
	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	// Record slow recoveries
	for i := 0; i < 5; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
		metrics.RecordRecoverySuccess("resume", "test_saga", 15*time.Second)
	}

	// Check rules
	am.checkAlertRules()

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	// Should trigger slow recovery alert
	alerts := notifier.getAlerts()
	assert.NotEmpty(t, alerts)

	found := false
	for _, alert := range alerts {
		if alert.Name == "SlowRecovery" {
			found = true
			assert.Equal(t, AlertSeverityWarning, alert.Severity)
			break
		}
	}
	assert.True(t, found, "SlowRecovery alert should be triggered")
}

func TestRecoveryAlertingManager_Deduplication(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	config := DefaultRecoveryAlertingConfig()
	config.DeduplicationWindow = 1 * time.Second

	am := NewRecoveryAlertingManager(config, metrics, logger)
	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	// Create an alert
	alert := &RecoveryAlert{
		Name:      "TestAlert",
		Severity:  AlertSeverityWarning,
		Timestamp: time.Now(),
	}

	// Send the same alert multiple times
	am.sendAlert(alert)
	am.sendAlert(alert)
	am.sendAlert(alert)

	// Wait for async notifications
	time.Sleep(100 * time.Millisecond)

	// Should only send one alert due to deduplication
	alerts := notifier.getAlerts()
	assert.Equal(t, 1, len(alerts))

	// Wait for deduplication window to expire
	time.Sleep(1100 * time.Millisecond)

	// Now should allow sending the same alert again
	notifier.reset()
	am.sendAlert(alert)

	time.Sleep(100 * time.Millisecond)
	alerts = notifier.getAlerts()
	assert.Equal(t, 1, len(alerts))
}

func TestRecoveryAlertingManager_RateLimiting(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	config := DefaultRecoveryAlertingConfig()
	config.MaxAlertsPerMinute = 3
	config.DeduplicationWindow = 0 // Disable deduplication for this test

	am := NewRecoveryAlertingManager(config, metrics, logger)
	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	// Send more alerts than the limit
	for i := 0; i < 5; i++ {
		alert := &RecoveryAlert{
			Name:      "TestAlert" + string(rune(i)), // Different names to bypass deduplication
			Severity:  AlertSeverityWarning,
			Timestamp: time.Now(),
		}
		am.sendAlert(alert)
	}

	// Wait for async notifications
	time.Sleep(100 * time.Millisecond)

	// Should only send up to the limit
	alerts := notifier.getAlerts()
	assert.LessOrEqual(t, len(alerts), config.MaxAlertsPerMinute)
}

func TestRecoveryAlertingManager_GetAlertHistory(t *testing.T) {
	logger := zaptest.NewLogger(t)
	am := NewRecoveryAlertingManager(nil, nil, logger)

	// Add some alerts to history
	for i := 0; i < 5; i++ {
		alert := &RecoveryAlert{
			Name:      "TestAlert",
			Timestamp: time.Now(),
		}
		am.addToAlertHistory(alert)
	}

	// Get history
	history := am.GetAlertHistory(0) // 0 means all
	assert.Equal(t, 5, len(history))

	// Get limited history
	history = am.GetAlertHistory(3)
	assert.Equal(t, 3, len(history))
}

func TestRecoveryAlertingManager_StartStop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultRecoveryAlertingConfig()
	config.CheckInterval = 100 * time.Millisecond

	am := NewRecoveryAlertingManager(config, nil, logger)

	// Start
	am.Start()
	time.Sleep(150 * time.Millisecond)

	// Stop
	am.Stop()

	// Should not panic
}

func TestRecoveryAlertingManager_DisabledAlerting(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultRecoveryAlertingConfig()
	config.Enabled = false

	am := NewRecoveryAlertingManager(config, nil, logger)

	// Start should not start the loop
	am.Start()
	time.Sleep(100 * time.Millisecond)

	// Stop should not panic
	am.Stop()
}

func TestLogAlertNotifier(t *testing.T) {
	logger := zaptest.NewLogger(t)
	notifier := NewLogAlertNotifier(logger)

	alert := &RecoveryAlert{
		ID:          "test-123",
		Name:        "TestAlert",
		Description: "Test description",
		Severity:    AlertSeverityWarning,
		Timestamp:   time.Now(),
		Value:       0.5,
		Threshold:   0.7,
	}

	ctx := context.Background()
	err := notifier.Notify(ctx, alert)
	assert.NoError(t, err)
}

func TestRecoveryAlertingManager_CustomRule(t *testing.T) {
	logger := zaptest.NewLogger(t)
	metrics, err := NewRecoveryMetrics(nil)
	require.NoError(t, err)

	am := NewRecoveryAlertingManager(nil, metrics, logger)
	notifier := &mockAlertNotifier{}
	am.AddNotifier(notifier)

	// Add a custom rule
	triggered := false
	am.AddRule(&AlertRule{
		Name:        "CustomRule",
		Description: "Custom test rule",
		Severity:    AlertSeverityInfo,
		Enabled:     true,
		CheckFunc: func(snapshot *RecoveryMetricsSnapshot) (bool, interface{}, interface{}) {
			if snapshot.TotalAttempts > 5 {
				triggered = true
				return true, snapshot.TotalAttempts, int64(5)
			}
			return false, nil, nil
		},
	})

	// Record metrics to trigger the rule
	for i := 0; i < 10; i++ {
		metrics.RecordRecoveryAttempt("resume", "test_saga")
	}

	// Check rules
	am.checkAlertRules()

	// Wait for async notification
	time.Sleep(100 * time.Millisecond)

	// Should trigger custom rule
	assert.True(t, triggered)
	alerts := notifier.getAlerts()
	assert.NotEmpty(t, alerts)

	found := false
	for _, alert := range alerts {
		if alert.Name == "CustomRule" {
			found = true
			assert.Equal(t, AlertSeverityInfo, alert.Severity)
			break
		}
	}
	assert.True(t, found, "CustomRule alert should be triggered")
}

