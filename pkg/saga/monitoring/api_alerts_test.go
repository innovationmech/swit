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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/saga/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAlertManager is a mock implementation of AlertManager for testing.
type mockAlertManager struct {
	alerts []*state.RecoveryAlert
	rules  []*state.AlertRule
	stats  AlertStats
}

func (m *mockAlertManager) GetAlertHistory(limit int) []*state.RecoveryAlert {
	if limit <= 0 || limit > len(m.alerts) {
		return m.alerts
	}
	return m.alerts[:limit]
}

func (m *mockAlertManager) GetAlertRules() []*state.AlertRule {
	return m.rules
}

func (m *mockAlertManager) GetAlertStats() AlertStats {
	return m.stats
}

// createTestAlertsAPI creates a test AlertsAPI with mock data.
func createTestAlertsAPI() (*AlertsAPI, *mockAlertManager) {
	now := time.Now()
	alerts := []*state.RecoveryAlert{
		{
			ID:          "alert-1",
			Name:        "HighRecoveryFailureRate",
			Description: "Recovery failure rate exceeds threshold",
			Severity:    state.AlertSeverityWarning,
			Timestamp:   now.Add(-1 * time.Hour),
			Labels: map[string]string{
				"rule": "HighRecoveryFailureRate",
			},
			Annotations: map[string]string{
				"description": "Recovery failure rate exceeds threshold",
			},
			Value:     0.15,
			Threshold: 0.10,
		},
		{
			ID:          "alert-2",
			Name:        "TooManyStuckSagas",
			Description: "Number of stuck Sagas exceeds threshold",
			Severity:    state.AlertSeverityCritical,
			Timestamp:   now.Add(-30 * time.Minute),
			Labels: map[string]string{
				"rule": "TooManyStuckSagas",
			},
			Annotations: map[string]string{
				"description": "Number of stuck Sagas exceeds threshold",
			},
			Value:     15,
			Threshold: 10,
		},
		{
			ID:          "alert-3",
			Name:        "SlowRecovery",
			Description: "Average recovery time exceeds threshold",
			Severity:    state.AlertSeverityInfo,
			Timestamp:   now.Add(-15 * time.Minute),
			Labels: map[string]string{
				"rule": "SlowRecovery",
			},
			Annotations: map[string]string{
				"description": "Average recovery time exceeds threshold",
			},
			Value:     35 * time.Second,
			Threshold: 30 * time.Second,
		},
	}

	rules := []*state.AlertRule{
		{
			Name:        "HighRecoveryFailureRate",
			Description: "Recovery failure rate exceeds threshold",
			Severity:    state.AlertSeverityWarning,
			Enabled:     true,
		},
		{
			Name:        "TooManyStuckSagas",
			Description: "Number of stuck Sagas exceeds threshold",
			Severity:    state.AlertSeverityCritical,
			Enabled:     true,
		},
		{
			Name:        "SlowRecovery",
			Description: "Average recovery time exceeds threshold",
			Severity:    state.AlertSeverityWarning,
			Enabled:     true,
		},
	}

	stats := AlertStats{
		TotalAlerts:      100,
		ActiveAlerts:     5,
		CriticalAlerts:   2,
		WarningAlerts:    3,
		InfoAlerts:       0,
		AlertRatePerHour: 4.5,
		LastAlertTime:    &now,
		AlertsByRule: map[string]int64{
			"HighRecoveryFailureRate": 40,
			"TooManyStuckSagas":       30,
			"SlowRecovery":            30,
		},
	}

	manager := &mockAlertManager{
		alerts: alerts,
		rules:  rules,
		stats:  stats,
	}

	return NewAlertsAPI(manager), manager
}

func TestAlertsAPI_GetAlerts(t *testing.T) {
	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
		checkResponse  func(t *testing.T, resp AlertListResponse)
	}{
		{
			name:           "Get all alerts",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  3,
			checkResponse: func(t *testing.T, resp AlertListResponse) {
				assert.Equal(t, 3, resp.Total)
				assert.Len(t, resp.Alerts, 3)
				assert.NotNil(t, resp.Stats)
			},
		},
		{
			name:           "Filter by severity - warning",
			queryParams:    "severity=warning",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			checkResponse: func(t *testing.T, resp AlertListResponse) {
				assert.Equal(t, 1, resp.Total)
				assert.Equal(t, "warning", resp.Alerts[0].Severity)
			},
		},
		{
			name:           "Filter by severity - critical",
			queryParams:    "severity=critical",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			checkResponse: func(t *testing.T, resp AlertListResponse) {
				assert.Equal(t, 1, resp.Total)
				assert.Equal(t, "critical", resp.Alerts[0].Severity)
			},
		},
		{
			name:           "Filter by rule name",
			queryParams:    "rule=TooManyStuckSagas",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			checkResponse: func(t *testing.T, resp AlertListResponse) {
				assert.Equal(t, 1, resp.Total)
				assert.Equal(t, "TooManyStuckSagas", resp.Alerts[0].Name)
			},
		},
		{
			name:           "Limit results",
			queryParams:    "limit=2",
			expectedStatus: http.StatusOK,
			expectedCount:  2,
			checkResponse: func(t *testing.T, resp AlertListResponse) {
				assert.Equal(t, 2, resp.Total)
			},
		},
		{
			name:           "Invalid severity",
			queryParams:    "severity=invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			router := gin.New()
			alertsAPI, _ := createTestAlertsAPI()

			router.GET("/api/alerts", alertsAPI.GetAlerts)

			// Create request
			req, err := http.NewRequest(http.MethodGet, "/api/alerts?"+tt.queryParams, nil)
			require.NoError(t, err)

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response body for successful requests
			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp AlertListResponse
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestAlertsAPI_GetAlertRules(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	alertsAPI, _ := createTestAlertsAPI()

	router.GET("/api/alerts/rules", alertsAPI.GetAlertRules)

	// Create request
	req, err := http.NewRequest(http.MethodGet, "/api/alerts/rules", nil)
	require.NoError(t, err)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp AlertRulesResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, 3, resp.Total)
	assert.Len(t, resp.Rules, 3)

	// Check first rule
	assert.Equal(t, "HighRecoveryFailureRate", resp.Rules[0].Name)
	assert.Equal(t, "warning", resp.Rules[0].Severity)
	assert.True(t, resp.Rules[0].Enabled)
}

func TestAlertsAPI_GetAlertStats(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	alertsAPI, _ := createTestAlertsAPI()

	router.GET("/api/alerts/stats", alertsAPI.GetAlertStats)

	// Create request
	req, err := http.NewRequest(http.MethodGet, "/api/alerts/stats", nil)
	require.NoError(t, err)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp AlertStatsResponse
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.Equal(t, int64(100), resp.Stats.TotalAlerts)
	assert.Equal(t, int64(5), resp.Stats.ActiveAlerts)
	assert.Equal(t, int64(2), resp.Stats.CriticalAlerts)
	assert.Equal(t, int64(3), resp.Stats.WarningAlerts)
	assert.Equal(t, float64(4.5), resp.Stats.AlertRatePerHour)
	assert.NotNil(t, resp.Stats.LastAlertTime)
}

func TestAlertsAPI_GetAlert(t *testing.T) {
	tests := []struct {
		name           string
		alertID        string
		expectedStatus int
		checkResponse  func(t *testing.T, resp AlertDTO)
	}{
		{
			name:           "Get existing alert",
			alertID:        "alert-1",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp AlertDTO) {
				assert.Equal(t, "alert-1", resp.ID)
				assert.Equal(t, "HighRecoveryFailureRate", resp.Name)
				assert.Equal(t, "warning", resp.Severity)
			},
		},
		{
			name:           "Get non-existent alert",
			alertID:        "non-existent",
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			gin.SetMode(gin.TestMode)
			router := gin.New()
			alertsAPI, _ := createTestAlertsAPI()

			router.GET("/api/alerts/:id", alertsAPI.GetAlert)

			// Create request
			req, err := http.NewRequest(http.MethodGet, "/api/alerts/"+tt.alertID, nil)
			require.NoError(t, err)

			// Perform request
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response body for successful requests
			if tt.expectedStatus == http.StatusOK && tt.checkResponse != nil {
				var resp AlertDTO
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestAlertsAPI_AcknowledgeAlert(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.New()
	alertsAPI, _ := createTestAlertsAPI()

	router.POST("/api/alerts/:id/acknowledge", alertsAPI.AcknowledgeAlert)

	// Create request
	req, err := http.NewRequest(http.MethodPost, "/api/alerts/alert-1/acknowledge", nil)
	require.NoError(t, err)

	// Perform request
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	assert.True(t, resp["success"].(bool))
	assert.NotEmpty(t, resp["message"])
}

func TestFilterAlerts(t *testing.T) {
	now := time.Now()
	alerts := []*state.RecoveryAlert{
		{
			ID:        "alert-1",
			Name:      "Rule1",
			Severity:  state.AlertSeverityWarning,
			Timestamp: now.Add(-1 * time.Hour),
		},
		{
			ID:        "alert-2",
			Name:      "Rule2",
			Severity:  state.AlertSeverityCritical,
			Timestamp: now.Add(-30 * time.Minute),
		},
		{
			ID:        "alert-3",
			Name:      "Rule1",
			Severity:  state.AlertSeverityInfo,
			Timestamp: now.Add(-15 * time.Minute),
		},
	}

	tests := []struct {
		name          string
		req           *AlertListRequest
		expectedCount int
		checkResults  func(t *testing.T, results []*state.RecoveryAlert)
	}{
		{
			name: "No filters",
			req: &AlertListRequest{
				Limit: 50,
			},
			expectedCount: 3,
			checkResults:  nil,
		},
		{
			name: "Filter by severity - warning",
			req: &AlertListRequest{
				Severity: "warning",
				Limit:    50,
			},
			expectedCount: 1,
			checkResults: func(t *testing.T, results []*state.RecoveryAlert) {
				assert.Equal(t, state.AlertSeverityWarning, results[0].Severity)
			},
		},
		{
			name: "Filter by rule name",
			req: &AlertListRequest{
				Rule:  "Rule1",
				Limit: 50,
			},
			expectedCount: 2,
			checkResults: func(t *testing.T, results []*state.RecoveryAlert) {
				for _, alert := range results {
					assert.Equal(t, "Rule1", alert.Name)
				}
			},
		},
		{
			name: "Filter by time - since 45 minutes ago",
			req: &AlertListRequest{
				Since: func() *time.Time { t := now.Add(-45 * time.Minute); return &t }(),
				Limit: 50,
			},
			expectedCount: 2,
			checkResults: func(t *testing.T, results []*state.RecoveryAlert) {
				for _, alert := range results {
					assert.True(t, alert.Timestamp.After(now.Add(-45*time.Minute)))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := filterAlerts(alerts, tt.req)
			assert.Len(t, results, tt.expectedCount)
			if tt.checkResults != nil {
				tt.checkResults(t, results)
			}
		})
	}
}

func TestAlertListRequest_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		req      AlertListRequest
		expected AlertListRequest
	}{
		{
			name: "Apply defaults - empty request",
			req:  AlertListRequest{},
			expected: AlertListRequest{
				Limit: 50,
			},
		},
		{
			name: "Apply defaults - with custom limit",
			req: AlertListRequest{
				Limit: 10,
			},
			expected: AlertListRequest{
				Limit: 10,
			},
		},
		{
			name: "Apply defaults - limit too high",
			req: AlertListRequest{
				Limit: 1000,
			},
			expected: AlertListRequest{
				Limit: 500, // Max limit
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.req.ApplyDefaults()
			assert.Equal(t, tt.expected.Limit, tt.req.Limit)
		})
	}
}

func TestFromRecoveryAlert(t *testing.T) {
	now := time.Now()
	alert := &state.RecoveryAlert{
		ID:          "test-alert-1",
		Name:        "TestAlert",
		Description: "Test alert description",
		Severity:    state.AlertSeverityWarning,
		Timestamp:   now,
		Labels: map[string]string{
			"key1": "value1",
		},
		Annotations: map[string]string{
			"key2": "value2",
		},
		Value:     100,
		Threshold: 50,
	}

	dto := fromRecoveryAlert(alert)

	assert.Equal(t, alert.ID, dto.ID)
	assert.Equal(t, alert.Name, dto.Name)
	assert.Equal(t, alert.Description, dto.Description)
	assert.Equal(t, string(alert.Severity), dto.Severity)
	assert.Equal(t, alert.Timestamp, dto.Timestamp)
	assert.Equal(t, alert.Labels, dto.Labels)
	assert.Equal(t, alert.Annotations, dto.Annotations)
	assert.Equal(t, alert.Value, dto.Value)
	assert.Equal(t, alert.Threshold, dto.Threshold)
}
