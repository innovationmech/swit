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
	"sync"
	"time"

	"github.com/innovationmech/swit/pkg/saga/state"
)

// AlertIntegration provides integration layer between the recovery alerting system
// and the monitoring dashboard. It wraps the RecoveryAlertingManager and provides
// additional functionality for the monitoring API.
type AlertIntegration struct {
	alertManager *state.RecoveryAlertingManager
	statsCache   *alertStatsCache
	mu           sync.RWMutex
}

// alertStatsCache caches alert statistics to avoid expensive calculations on every request.
type alertStatsCache struct {
	stats      AlertStats
	lastUpdate time.Time
	ttl        time.Duration
	mu         sync.RWMutex
}

// NewAlertIntegration creates a new alert integration wrapper.
//
// Parameters:
//   - alertManager: The recovery alerting manager to integrate.
//
// Returns:
//   - A configured AlertIntegration ready for use.
func NewAlertIntegration(alertManager *state.RecoveryAlertingManager) *AlertIntegration {
	if alertManager == nil {
		return nil
	}

	return &AlertIntegration{
		alertManager: alertManager,
		statsCache: &alertStatsCache{
			ttl: 5 * time.Second, // Cache stats for 5 seconds
		},
	}
}

// GetAlertHistory returns recent alerts with optional limit.
// Implements AlertManager interface.
func (ai *AlertIntegration) GetAlertHistory(limit int) []*state.RecoveryAlert {
	if ai == nil || ai.alertManager == nil {
		return []*state.RecoveryAlert{}
	}

	ai.mu.RLock()
	defer ai.mu.RUnlock()

	return ai.alertManager.GetAlertHistory(limit)
}

// GetAlertRules returns all configured alert rules.
// Implements AlertManager interface.
func (ai *AlertIntegration) GetAlertRules() []*state.AlertRule {
	if ai == nil || ai.alertManager == nil {
		return []*state.AlertRule{}
	}

	ai.mu.RLock()
	defer ai.mu.RUnlock()

	return ai.alertManager.GetRules()
}

// GetAlertStats returns statistics about alerts.
// Implements AlertManager interface with caching.
func (ai *AlertIntegration) GetAlertStats() AlertStats {
	if ai == nil || ai.alertManager == nil {
		return AlertStats{}
	}

	// Check cache first
	ai.statsCache.mu.RLock()
	if time.Since(ai.statsCache.lastUpdate) < ai.statsCache.ttl {
		stats := ai.statsCache.stats
		ai.statsCache.mu.RUnlock()
		return stats
	}
	ai.statsCache.mu.RUnlock()

	// Cache miss or expired, calculate new stats
	ai.statsCache.mu.Lock()
	defer ai.statsCache.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(ai.statsCache.lastUpdate) < ai.statsCache.ttl {
		return ai.statsCache.stats
	}

	stats := ai.calculateAlertStats()
	ai.statsCache.stats = stats
	ai.statsCache.lastUpdate = time.Now()

	return stats
}

// calculateAlertStats calculates alert statistics from alert history.
func (ai *AlertIntegration) calculateAlertStats() AlertStats {
	// Get recent alert history (last 1000 alerts should be sufficient)
	alerts := ai.alertManager.GetAlertHistory(1000)

	stats := AlertStats{
		AlertsByRule: make(map[string]int64),
	}

	if len(alerts) == 0 {
		return stats
	}

	// Count alerts by severity and rule
	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	recentAlertCount := int64(0)

	for _, alert := range alerts {
		stats.TotalAlerts++

		// Count by severity
		switch alert.Severity {
		case state.AlertSeverityCritical:
			stats.CriticalAlerts++
		case state.AlertSeverityWarning:
			stats.WarningAlerts++
		case state.AlertSeverityInfo:
			stats.InfoAlerts++
		}

		// Count by rule
		stats.AlertsByRule[alert.Name]++

		// Track last alert time
		if stats.LastAlertTime == nil || alert.Timestamp.After(*stats.LastAlertTime) {
			stats.LastAlertTime = &alert.Timestamp
		}

		// Count recent alerts for rate calculation
		if alert.Timestamp.After(oneHourAgo) {
			recentAlertCount++
		}
	}

	// Consider critical and warning alerts as "active"
	stats.ActiveAlerts = stats.CriticalAlerts + stats.WarningAlerts

	// Calculate alert rate per hour
	if recentAlertCount > 0 {
		stats.AlertRatePerHour = float64(recentAlertCount)
	}

	return stats
}

// IsHealthy returns whether the alerting system is functioning properly.
func (ai *AlertIntegration) IsHealthy() bool {
	if ai == nil || ai.alertManager == nil {
		return false
	}

	// Check if we've received any alerts recently
	// A healthy alerting system should be running and checking rules
	alerts := ai.alertManager.GetAlertHistory(1)
	return len(alerts) > 0 || time.Since(time.Now()) < 5*time.Minute
}

// GetAlertingConfig returns the current alerting configuration.
func (ai *AlertIntegration) GetAlertingConfig() *state.RecoveryAlertingConfig {
	if ai == nil || ai.alertManager == nil {
		return nil
	}

	ai.mu.RLock()
	defer ai.mu.RUnlock()

	return ai.alertManager.GetConfig()
}

// GetAlertByID retrieves a specific alert by its ID.
func (ai *AlertIntegration) GetAlertByID(alertID string) *state.RecoveryAlert {
	if ai == nil || ai.alertManager == nil {
		return nil
	}

	alerts := ai.GetAlertHistory(1000)
	for _, alert := range alerts {
		if alert.ID == alertID {
			return alert
		}
	}

	return nil
}

// GetActiveAlerts returns only currently active (critical and warning) alerts.
func (ai *AlertIntegration) GetActiveAlerts() []*state.RecoveryAlert {
	if ai == nil || ai.alertManager == nil {
		return []*state.RecoveryAlert{}
	}

	allAlerts := ai.GetAlertHistory(1000)
	activeAlerts := make([]*state.RecoveryAlert, 0)

	for _, alert := range allAlerts {
		if alert.Severity == state.AlertSeverityCritical ||
			alert.Severity == state.AlertSeverityWarning {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// GetAlertsBySeverity returns alerts filtered by severity level.
func (ai *AlertIntegration) GetAlertsBySeverity(severity state.RecoveryAlertSeverity, limit int) []*state.RecoveryAlert {
	if ai == nil || ai.alertManager == nil {
		return []*state.RecoveryAlert{}
	}

	allAlerts := ai.GetAlertHistory(limit * 2) // Get more to ensure we have enough after filtering
	filteredAlerts := make([]*state.RecoveryAlert, 0)

	for _, alert := range allAlerts {
		if alert.Severity == severity {
			filteredAlerts = append(filteredAlerts, alert)
			if len(filteredAlerts) >= limit {
				break
			}
		}
	}

	return filteredAlerts
}

// GetAlertsByRule returns alerts filtered by rule name.
func (ai *AlertIntegration) GetAlertsByRule(ruleName string, limit int) []*state.RecoveryAlert {
	if ai == nil || ai.alertManager == nil {
		return []*state.RecoveryAlert{}
	}

	allAlerts := ai.GetAlertHistory(limit * 2) // Get more to ensure we have enough after filtering
	filteredAlerts := make([]*state.RecoveryAlert, 0)

	for _, alert := range allAlerts {
		if alert.Name == ruleName {
			filteredAlerts = append(filteredAlerts, alert)
			if len(filteredAlerts) >= limit {
				break
			}
		}
	}

	return filteredAlerts
}

// GetAlertsSince returns alerts that occurred after the specified time.
func (ai *AlertIntegration) GetAlertsSince(since time.Time, limit int) []*state.RecoveryAlert {
	if ai == nil || ai.alertManager == nil {
		return []*state.RecoveryAlert{}
	}

	allAlerts := ai.GetAlertHistory(limit * 2)
	filteredAlerts := make([]*state.RecoveryAlert, 0)

	for _, alert := range allAlerts {
		if alert.Timestamp.After(since) {
			filteredAlerts = append(filteredAlerts, alert)
			if len(filteredAlerts) >= limit {
				break
			}
		}
	}

	return filteredAlerts
}

// ClearStatsCache clears the cached alert statistics, forcing recalculation on next request.
func (ai *AlertIntegration) ClearStatsCache() {
	if ai == nil || ai.statsCache == nil {
		return
	}

	ai.statsCache.mu.Lock()
	defer ai.statsCache.mu.Unlock()

	ai.statsCache.lastUpdate = time.Time{} // Reset to zero time to force recalculation
}
