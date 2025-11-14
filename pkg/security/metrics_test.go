// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package security

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewSecurityMetrics(t *testing.T) {
	tests := []struct {
		name      string
		config    *SecurityMetricsConfig
		wantErr   bool
		checkFunc func(t *testing.T, m *SecurityMetrics)
	}{
		{
			name:    "default config",
			config:  nil,
			wantErr: false,
			checkFunc: func(t *testing.T, m *SecurityMetrics) {
				if m == nil {
					t.Fatal("expected metrics to be created")
				}
				if m.registry == nil {
					t.Error("expected registry to be set")
				}
			},
		},
		{
			name: "custom config",
			config: &SecurityMetricsConfig{
				Namespace: "custom",
				Subsystem: "test",
				Registry:  prometheus.NewRegistry(),
			},
			wantErr: false,
			checkFunc: func(t *testing.T, m *SecurityMetrics) {
				if m == nil {
					t.Fatal("expected metrics to be created")
				}
				if m.registry == nil {
					t.Error("expected registry to be set")
				}
			},
		},
		{
			name: "custom duration buckets",
			config: &SecurityMetricsConfig{
				Namespace:       "security",
				DurationBuckets: []float64{0.1, 0.5, 1.0},
			},
			wantErr: false,
			checkFunc: func(t *testing.T, m *SecurityMetrics) {
				if m == nil {
					t.Fatal("expected metrics to be created")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSecurityMetrics(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSecurityMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, got)
			}
		})
	}
}

func TestSecurityMetrics_AuthenticationMetrics(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	t.Run("RecordAuthAttempt", func(t *testing.T) {
		metrics.RecordAuthAttempt("jwt", "success")
		metrics.RecordAuthAttempt("jwt", "success")
		metrics.RecordAuthAttempt("jwt", "failure")
		metrics.RecordAuthAttempt("oauth2", "success")

		// Check counter values
		count := testutil.ToFloat64(metrics.authAttemptsTotal.WithLabelValues("success", "jwt"))
		if count != 2 {
			t.Errorf("expected 2 successful jwt attempts, got %f", count)
		}

		count = testutil.ToFloat64(metrics.authAttemptsTotal.WithLabelValues("failure", "jwt"))
		if count != 1 {
			t.Errorf("expected 1 failed jwt attempt, got %f", count)
		}

		count = testutil.ToFloat64(metrics.authAttemptsTotal.WithLabelValues("success", "oauth2"))
		if count != 1 {
			t.Errorf("expected 1 successful oauth2 attempt, got %f", count)
		}
	})

	t.Run("RecordAuthDuration", func(t *testing.T) {
		metrics.RecordAuthDuration("jwt", 0.05)
		metrics.RecordAuthDuration("jwt", 0.1)
		metrics.RecordAuthDuration("oauth2", 0.2)

		// Check histogram count using the collector directly
		metricFamilies, err := metrics.registry.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "security_auth_duration_seconds" {
				for _, m := range mf.GetMetric() {
					labels := make(map[string]string)
					for _, lp := range m.GetLabel() {
						labels[lp.GetName()] = lp.GetValue()
					}
					if labels["method"] == "jwt" {
						found = true
						count := m.GetHistogram().GetSampleCount()
						if count != 2 {
							t.Errorf("expected 2 jwt auth duration observations, got %d", count)
						}
					}
				}
			}
		}
		if !found {
			t.Error("expected to find jwt auth duration metric")
		}
	})

	t.Run("RecordTokenValidation", func(t *testing.T) {
		metrics.RecordTokenValidation("success")
		metrics.RecordTokenValidation("success")
		metrics.RecordTokenValidation("expired")
		metrics.RecordTokenValidation("invalid")

		count := testutil.ToFloat64(metrics.tokenValidationsTotal.WithLabelValues("success"))
		if count != 2 {
			t.Errorf("expected 2 successful validations, got %f", count)
		}

		count = testutil.ToFloat64(metrics.tokenValidationsTotal.WithLabelValues("expired"))
		if count != 1 {
			t.Errorf("expected 1 expired validation, got %f", count)
		}
	})

	t.Run("RecordTokenRefresh", func(t *testing.T) {
		metrics.RecordTokenRefresh("success")
		metrics.RecordTokenRefresh("failure")

		count := testutil.ToFloat64(metrics.tokenRefreshesTotal.WithLabelValues("success"))
		if count != 1 {
			t.Errorf("expected 1 successful refresh, got %f", count)
		}
	})
}

func TestSecurityMetrics_AuthorizationMetrics(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	t.Run("RecordPolicyEvaluation", func(t *testing.T) {
		metrics.RecordPolicyEvaluation("rbac", "allow")
		metrics.RecordPolicyEvaluation("rbac", "allow")
		metrics.RecordPolicyEvaluation("rbac", "deny")
		metrics.RecordPolicyEvaluation("abac", "allow")

		count := testutil.ToFloat64(metrics.policyEvaluationsTotal.WithLabelValues("allow", "rbac"))
		if count != 2 {
			t.Errorf("expected 2 rbac allow evaluations, got %f", count)
		}

		count = testutil.ToFloat64(metrics.policyEvaluationsTotal.WithLabelValues("deny", "rbac"))
		if count != 1 {
			t.Errorf("expected 1 rbac deny evaluation, got %f", count)
		}
	})

	t.Run("RecordPolicyEvaluationDuration", func(t *testing.T) {
		metrics.RecordPolicyEvaluationDuration("rbac", 0.01)
		metrics.RecordPolicyEvaluationDuration("rbac", 0.02)
		metrics.RecordPolicyEvaluationDuration("abac", 0.05)

		// Check histogram count using the collector directly
		metricFamilies, err := metrics.registry.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "security_policy_evaluation_duration_seconds" {
				for _, m := range mf.GetMetric() {
					labels := make(map[string]string)
					for _, lp := range m.GetLabel() {
						labels[lp.GetName()] = lp.GetValue()
					}
					if labels["policy"] == "rbac" {
						found = true
						count := m.GetHistogram().GetSampleCount()
						if count != 2 {
							t.Errorf("expected 2 rbac duration observations, got %d", count)
						}
					}
				}
			}
		}
		if !found {
			t.Error("expected to find rbac policy evaluation duration metric")
		}
	})

	t.Run("RecordAccessDenied", func(t *testing.T) {
		metrics.RecordAccessDenied("insufficient_permissions", "/api/users")
		metrics.RecordAccessDenied("insufficient_permissions", "/api/users")
		metrics.RecordAccessDenied("invalid_token", "/api/admin")

		count := testutil.ToFloat64(metrics.accessDeniedTotal.WithLabelValues("insufficient_permissions", "/api/users"))
		if count != 2 {
			t.Errorf("expected 2 access denied events, got %f", count)
		}

		count = testutil.ToFloat64(metrics.accessDeniedTotal.WithLabelValues("invalid_token", "/api/admin"))
		if count != 1 {
			t.Errorf("expected 1 access denied event, got %f", count)
		}
	})
}

func TestSecurityMetrics_SecurityEventMetrics(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	t.Run("RecordSecurityEvent", func(t *testing.T) {
		metrics.RecordSecurityEvent("brute_force", "high")
		metrics.RecordSecurityEvent("brute_force", "high")
		metrics.RecordSecurityEvent("suspicious_activity", "medium")

		count := testutil.ToFloat64(metrics.securityEventsTotal.WithLabelValues("brute_force", "high"))
		if count != 2 {
			t.Errorf("expected 2 brute_force high events, got %f", count)
		}

		count = testutil.ToFloat64(metrics.securityEventsTotal.WithLabelValues("suspicious_activity", "medium"))
		if count != 1 {
			t.Errorf("expected 1 suspicious_activity medium event, got %f", count)
		}
	})
}

func TestSecurityMetrics_VulnerabilityMetrics(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	t.Run("SetVulnerabilitiesFound", func(t *testing.T) {
		metrics.SetVulnerabilitiesFound("high", "gosec", 5)
		metrics.SetVulnerabilitiesFound("medium", "gosec", 10)
		metrics.SetVulnerabilitiesFound("low", "govulncheck", 3)

		value := testutil.ToFloat64(metrics.vulnerabilitiesFound.WithLabelValues("high", "gosec"))
		if value != 5 {
			t.Errorf("expected 5 high gosec vulnerabilities, got %f", value)
		}

		value = testutil.ToFloat64(metrics.vulnerabilitiesFound.WithLabelValues("medium", "gosec"))
		if value != 10 {
			t.Errorf("expected 10 medium gosec vulnerabilities, got %f", value)
		}

		// Update existing metric
		metrics.SetVulnerabilitiesFound("high", "gosec", 3)
		value = testutil.ToFloat64(metrics.vulnerabilitiesFound.WithLabelValues("high", "gosec"))
		if value != 3 {
			t.Errorf("expected 3 high gosec vulnerabilities after update, got %f", value)
		}
	})

	t.Run("RecordSecurityScan", func(t *testing.T) {
		metrics.RecordSecurityScan("gosec", "success")
		metrics.RecordSecurityScan("gosec", "success")
		metrics.RecordSecurityScan("govulncheck", "failure")

		count := testutil.ToFloat64(metrics.securityScansTotal.WithLabelValues("gosec", "success"))
		if count != 2 {
			t.Errorf("expected 2 successful gosec scans, got %f", count)
		}

		count = testutil.ToFloat64(metrics.securityScansTotal.WithLabelValues("govulncheck", "failure"))
		if count != 1 {
			t.Errorf("expected 1 failed govulncheck scan, got %f", count)
		}
	})
}

func TestSecurityMetrics_TLSMetrics(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	t.Run("RecordTLSConnection", func(t *testing.T) {
		metrics.RecordTLSConnection("1.3", "TLS_AES_128_GCM_SHA256")
		metrics.RecordTLSConnection("1.3", "TLS_AES_128_GCM_SHA256")
		metrics.RecordTLSConnection("1.2", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256")

		count := testutil.ToFloat64(metrics.tlsConnectionsTotal.WithLabelValues("1.3", "TLS_AES_128_GCM_SHA256"))
		if count != 2 {
			t.Errorf("expected 2 TLS 1.3 connections, got %f", count)
		}

		count = testutil.ToFloat64(metrics.tlsConnectionsTotal.WithLabelValues("1.2", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"))
		if count != 1 {
			t.Errorf("expected 1 TLS 1.2 connection, got %f", count)
		}
	})

	t.Run("RecordTLSHandshakeDuration", func(t *testing.T) {
		metrics.RecordTLSHandshakeDuration("1.3", 0.05)
		metrics.RecordTLSHandshakeDuration("1.3", 0.06)
		metrics.RecordTLSHandshakeDuration("1.2", 0.1)

		// Check histogram count using the collector directly
		metricFamilies, err := metrics.registry.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		found := false
		for _, mf := range metricFamilies {
			if mf.GetName() == "security_tls_handshake_duration_seconds" {
				for _, m := range mf.GetMetric() {
					labels := make(map[string]string)
					for _, lp := range m.GetLabel() {
						labels[lp.GetName()] = lp.GetValue()
					}
					if labels["version"] == "1.3" {
						found = true
						count := m.GetHistogram().GetSampleCount()
						if count != 2 {
							t.Errorf("expected 2 TLS 1.3 handshake observations, got %d", count)
						}
					}
				}
			}
		}
		if !found {
			t.Error("expected to find TLS 1.3 handshake duration metric")
		}
	})

	t.Run("SetCertificateExpiry", func(t *testing.T) {
		metrics.SetCertificateExpiry("example.com", 86400*30)     // 30 days
		metrics.SetCertificateExpiry("api.example.com", 86400*60) // 60 days

		value := testutil.ToFloat64(metrics.certificateExpiry.WithLabelValues("example.com"))
		expected := float64(86400 * 30)
		if value != expected {
			t.Errorf("expected %f seconds until expiry, got %f", expected, value)
		}

		// Update expiry
		metrics.SetCertificateExpiry("example.com", 86400*29) // 29 days
		value = testutil.ToFloat64(metrics.certificateExpiry.WithLabelValues("example.com"))
		expected = float64(86400 * 29)
		if value != expected {
			t.Errorf("expected %f seconds until expiry after update, got %f", expected, value)
		}
	})
}

func TestSecurityMetrics_SessionMetrics(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	t.Run("SetActiveSessions", func(t *testing.T) {
		metrics.SetActiveSessions(10)
		value := testutil.ToFloat64(metrics.activeSessionsGauge)
		if value != 10 {
			t.Errorf("expected 10 active sessions, got %f", value)
		}

		metrics.SetActiveSessions(5)
		value = testutil.ToFloat64(metrics.activeSessionsGauge)
		if value != 5 {
			t.Errorf("expected 5 active sessions after update, got %f", value)
		}
	})

	t.Run("IncreaseDecreaseActiveSessions", func(t *testing.T) {
		metrics.SetActiveSessions(0)

		metrics.IncreaseActiveSessions()
		metrics.IncreaseActiveSessions()
		value := testutil.ToFloat64(metrics.activeSessionsGauge)
		if value != 2 {
			t.Errorf("expected 2 active sessions after increments, got %f", value)
		}

		metrics.DecreaseActiveSessions()
		value = testutil.ToFloat64(metrics.activeSessionsGauge)
		if value != 1 {
			t.Errorf("expected 1 active session after decrement, got %f", value)
		}
	})

	t.Run("RecordSessionCreated", func(t *testing.T) {
		metrics.RecordSessionCreated("user")
		metrics.RecordSessionCreated("user")
		metrics.RecordSessionCreated("service")

		count := testutil.ToFloat64(metrics.sessionCreatedTotal.WithLabelValues("user"))
		if count != 2 {
			t.Errorf("expected 2 user sessions created, got %f", count)
		}

		count = testutil.ToFloat64(metrics.sessionCreatedTotal.WithLabelValues("service"))
		if count != 1 {
			t.Errorf("expected 1 service session created, got %f", count)
		}
	})

	t.Run("RecordSessionRevoked", func(t *testing.T) {
		metrics.RecordSessionRevoked("logout")
		metrics.RecordSessionRevoked("logout")
		metrics.RecordSessionRevoked("timeout")

		count := testutil.ToFloat64(metrics.sessionRevokedTotal.WithLabelValues("logout"))
		if count != 2 {
			t.Errorf("expected 2 logout revocations, got %f", count)
		}

		count = testutil.ToFloat64(metrics.sessionRevokedTotal.WithLabelValues("timeout"))
		if count != 1 {
			t.Errorf("expected 1 timeout revocation, got %f", count)
		}
	})
}

func TestSecurityMetrics_GetRegistry(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	registry := metrics.GetRegistry()
	if registry == nil {
		t.Error("expected non-nil registry")
	}

	// Record some metrics to ensure they appear in the registry output
	metrics.RecordAuthAttempt("jwt", "success")
	metrics.RecordAuthDuration("jwt", 0.1)
	metrics.RecordTokenValidation("success")
	metrics.RecordPolicyEvaluation("rbac", "allow")
	metrics.SetVulnerabilitiesFound("high", "gosec", 5)
	metrics.RecordTLSConnection("1.3", "TLS_AES_128_GCM_SHA256")
	metrics.SetActiveSessions(10)

	// Verify registry contains our metrics
	metricFamilies, err := registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Error("expected registry to contain metrics")
	}

	// Check for some expected metric names
	expectedMetrics := []string{
		"security_auth_attempts_total",
		"security_auth_duration_seconds",
		"security_token_validations_total",
		"security_policy_evaluations_total",
		"security_vulnerabilities_found",
		"security_tls_connections_total",
		"security_active_sessions",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[mf.GetName()] = true
	}

	for _, expected := range expectedMetrics {
		if !foundMetrics[expected] {
			t.Errorf("expected metric %s not found in registry", expected)
		}
	}
}

func TestSecurityMetrics_MetricNaming(t *testing.T) {
	// Test with custom namespace and subsystem
	config := &SecurityMetricsConfig{
		Namespace: "custom",
		Subsystem: "test",
		Registry:  prometheus.NewRegistry(),
	}

	metrics, err := NewSecurityMetrics(config)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// Record some metrics
	metrics.RecordAuthAttempt("jwt", "success")

	// Gather and check metric names have correct namespace/subsystem
	metricFamilies, err := metrics.registry.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	found := false
	for _, mf := range metricFamilies {
		name := mf.GetName()
		if strings.Contains(name, "auth_attempts_total") {
			found = true
			if !strings.HasPrefix(name, "custom_test_") {
				t.Errorf("expected metric name to start with 'custom_test_', got %s", name)
			}
		}
	}

	if !found {
		t.Error("expected to find auth_attempts_total metric")
	}
}

func TestDefaultSecurityMetricsConfig(t *testing.T) {
	config := DefaultSecurityMetricsConfig()

	if config.Namespace != "security" {
		t.Errorf("expected namespace 'security', got %s", config.Namespace)
	}

	if config.Subsystem != "" {
		t.Errorf("expected empty subsystem, got %s", config.Subsystem)
	}

	if config.Registry == nil {
		t.Error("expected non-nil registry")
	}

	if len(config.DurationBuckets) == 0 {
		t.Error("expected non-empty duration buckets")
	}
}

func TestSecurityMetrics_ConcurrentAccess(t *testing.T) {
	metrics, err := NewSecurityMetrics(nil)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// Test concurrent writes to metrics
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				metrics.RecordAuthAttempt("jwt", "success")
				metrics.RecordTokenValidation("success")
				metrics.IncreaseActiveSessions()
				metrics.DecreaseActiveSessions()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics are consistent
	count := testutil.ToFloat64(metrics.authAttemptsTotal.WithLabelValues("success", "jwt"))
	if count != 1000 {
		t.Errorf("expected 1000 auth attempts, got %f", count)
	}

	count = testutil.ToFloat64(metrics.tokenValidationsTotal.WithLabelValues("success"))
	if count != 1000 {
		t.Errorf("expected 1000 token validations, got %f", count)
	}
}
