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

package security

import (
	"github.com/prometheus/client_golang/prometheus"
)

// SecurityMetrics implements Prometheus metrics collection for security-related operations.
// It provides comprehensive monitoring of authentication, authorization, security events,
// vulnerabilities, and TLS connections.
type SecurityMetrics struct {
	// Authentication metrics
	authAttemptsTotal     *prometheus.CounterVec
	authDuration          *prometheus.HistogramVec
	tokenValidationsTotal *prometheus.CounterVec
	tokenRefreshesTotal   *prometheus.CounterVec

	// Authorization metrics
	policyEvaluationsTotal   *prometheus.CounterVec
	policyEvaluationDuration *prometheus.HistogramVec
	accessDeniedTotal        *prometheus.CounterVec

	// Security events
	securityEventsTotal *prometheus.CounterVec

	// Vulnerability metrics
	vulnerabilitiesFound *prometheus.GaugeVec
	securityScansTotal   *prometheus.CounterVec

	// TLS/Connection metrics
	tlsConnectionsTotal  *prometheus.CounterVec
	tlsHandshakeDuration *prometheus.HistogramVec
	certificateExpiry    *prometheus.GaugeVec

	// Session metrics
	activeSessionsGauge prometheus.Gauge
	sessionCreatedTotal *prometheus.CounterVec
	sessionRevokedTotal *prometheus.CounterVec

	// Prometheus registry
	registry *prometheus.Registry
}

// SecurityMetricsConfig contains configuration for security metrics.
type SecurityMetricsConfig struct {
	// Namespace is the Prometheus namespace for all metrics (default: "security")
	Namespace string

	// Subsystem is the Prometheus subsystem for all metrics (default: "")
	Subsystem string

	// Registry is the Prometheus registry to use. If nil, a new registry is created.
	Registry *prometheus.Registry

	// DurationBuckets defines the buckets for duration histograms.
	// If nil, default buckets optimized for security operations are used.
	DurationBuckets []float64
}

// DefaultSecurityMetricsConfig returns a default configuration for security metrics.
func DefaultSecurityMetricsConfig() *SecurityMetricsConfig {
	return &SecurityMetricsConfig{
		Namespace: "security",
		Subsystem: "",
		Registry:  prometheus.NewRegistry(),
		// Duration buckets optimized for security operations (milliseconds to seconds)
		DurationBuckets: []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
		},
	}
}

// NewSecurityMetrics creates a new Prometheus-based security metrics collector.
// It initializes all security metrics and registers them with the provided or default registry.
//
// Parameters:
//   - config: Configuration for the metrics collector. If nil, default config is used.
//
// Returns:
//   - A configured SecurityMetrics ready to collect metrics.
//   - An error if metric registration fails.
//
// Example:
//
//	metrics, err := NewSecurityMetrics(nil)
//	if err != nil {
//	    return err
//	}
//	metrics.RecordAuthAttempt("jwt", "success")
func NewSecurityMetrics(config *SecurityMetricsConfig) (*SecurityMetrics, error) {
	if config == nil {
		config = DefaultSecurityMetricsConfig()
	}

	if config.Registry == nil {
		config.Registry = prometheus.NewRegistry()
	}

	if config.Namespace == "" {
		config.Namespace = "security"
	}

	if config.DurationBuckets == nil {
		config.DurationBuckets = []float64{
			0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0,
		}
	}

	metrics := &SecurityMetrics{
		registry: config.Registry,
	}

	// Initialize authentication metrics
	metrics.authAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "auth_attempts_total",
			Help:      "Total number of authentication attempts by result and method",
		},
		[]string{"result", "method"},
	)

	metrics.authDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "auth_duration_seconds",
			Help:      "Authentication operation duration in seconds by method",
			Buckets:   config.DurationBuckets,
		},
		[]string{"method"},
	)

	metrics.tokenValidationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "token_validations_total",
			Help:      "Total number of token validation attempts by result",
		},
		[]string{"result"},
	)

	metrics.tokenRefreshesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "token_refreshes_total",
			Help:      "Total number of token refresh attempts by result",
		},
		[]string{"result"},
	)

	// Initialize authorization metrics
	metrics.policyEvaluationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "policy_evaluations_total",
			Help:      "Total number of policy evaluations by decision and policy",
		},
		[]string{"decision", "policy"},
	)

	metrics.policyEvaluationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "policy_evaluation_duration_seconds",
			Help:      "Policy evaluation duration in seconds by policy",
			Buckets:   config.DurationBuckets,
		},
		[]string{"policy"},
	)

	metrics.accessDeniedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "access_denied_total",
			Help:      "Total number of access denied events by reason and resource",
		},
		[]string{"reason", "resource"},
	)

	// Initialize security event metrics
	metrics.securityEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "security_events_total",
			Help:      "Total number of security events by type and severity",
		},
		[]string{"type", "severity"},
	)

	// Initialize vulnerability metrics
	metrics.vulnerabilitiesFound = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "vulnerabilities_found",
			Help:      "Current number of vulnerabilities found by severity and tool",
		},
		[]string{"severity", "tool"},
	)

	metrics.securityScansTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "security_scans_total",
			Help:      "Total number of security scans by tool and result",
		},
		[]string{"tool", "result"},
	)

	// Initialize TLS metrics
	metrics.tlsConnectionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "tls_connections_total",
			Help:      "Total number of TLS connections by version and cipher suite",
		},
		[]string{"version", "cipher_suite"},
	)

	metrics.tlsHandshakeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "tls_handshake_duration_seconds",
			Help:      "TLS handshake duration in seconds by version",
			Buckets:   config.DurationBuckets,
		},
		[]string{"version"},
	)

	metrics.certificateExpiry = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "certificate_expiry_seconds",
			Help:      "Seconds until certificate expiry by common name",
		},
		[]string{"common_name"},
	)

	// Initialize session metrics
	metrics.activeSessionsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "active_sessions",
			Help:      "Current number of active sessions",
		},
	)

	metrics.sessionCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "session_created_total",
			Help:      "Total number of sessions created by type",
		},
		[]string{"type"},
	)

	metrics.sessionRevokedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: config.Namespace,
			Subsystem: config.Subsystem,
			Name:      "session_revoked_total",
			Help:      "Total number of sessions revoked by reason",
		},
		[]string{"reason"},
	)

	// Register all metrics with the registry
	collectors := []prometheus.Collector{
		metrics.authAttemptsTotal,
		metrics.authDuration,
		metrics.tokenValidationsTotal,
		metrics.tokenRefreshesTotal,
		metrics.policyEvaluationsTotal,
		metrics.policyEvaluationDuration,
		metrics.accessDeniedTotal,
		metrics.securityEventsTotal,
		metrics.vulnerabilitiesFound,
		metrics.securityScansTotal,
		metrics.tlsConnectionsTotal,
		metrics.tlsHandshakeDuration,
		metrics.certificateExpiry,
		metrics.activeSessionsGauge,
		metrics.sessionCreatedTotal,
		metrics.sessionRevokedTotal,
	}

	for _, collector := range collectors {
		if err := config.Registry.Register(collector); err != nil {
			return nil, err
		}
	}

	return metrics, nil
}

// Authentication Metrics

// RecordAuthAttempt records an authentication attempt.
// Parameters:
//   - method: Authentication method (e.g., "jwt", "oauth2", "basic")
//   - result: Result of the attempt ("success", "failure", "error")
func (sm *SecurityMetrics) RecordAuthAttempt(method, result string) {
	sm.authAttemptsTotal.WithLabelValues(result, method).Inc()
}

// RecordAuthDuration records the duration of an authentication operation.
// Parameters:
//   - method: Authentication method (e.g., "jwt", "oauth2", "basic")
//   - durationSeconds: Duration in seconds
func (sm *SecurityMetrics) RecordAuthDuration(method string, durationSeconds float64) {
	sm.authDuration.WithLabelValues(method).Observe(durationSeconds)
}

// RecordTokenValidation records a token validation attempt.
// Parameters:
//   - result: Result of the validation ("success", "expired", "invalid", "error")
func (sm *SecurityMetrics) RecordTokenValidation(result string) {
	sm.tokenValidationsTotal.WithLabelValues(result).Inc()
}

// RecordTokenRefresh records a token refresh attempt.
// Parameters:
//   - result: Result of the refresh ("success", "failure", "error")
func (sm *SecurityMetrics) RecordTokenRefresh(result string) {
	sm.tokenRefreshesTotal.WithLabelValues(result).Inc()
}

// Authorization Metrics

// RecordPolicyEvaluation records a policy evaluation.
// Parameters:
//   - policy: Policy name or identifier
//   - decision: Evaluation decision ("allow", "deny", "error")
func (sm *SecurityMetrics) RecordPolicyEvaluation(policy, decision string) {
	sm.policyEvaluationsTotal.WithLabelValues(decision, policy).Inc()
}

// RecordPolicyEvaluationDuration records the duration of a policy evaluation.
// Parameters:
//   - policy: Policy name or identifier
//   - durationSeconds: Duration in seconds
func (sm *SecurityMetrics) RecordPolicyEvaluationDuration(policy string, durationSeconds float64) {
	sm.policyEvaluationDuration.WithLabelValues(policy).Observe(durationSeconds)
}

// RecordAccessDenied records an access denied event.
// Parameters:
//   - reason: Reason for denial (e.g., "insufficient_permissions", "invalid_token")
//   - resource: Resource that was attempted to be accessed
func (sm *SecurityMetrics) RecordAccessDenied(reason, resource string) {
	sm.accessDeniedTotal.WithLabelValues(reason, resource).Inc()
}

// Security Event Metrics

// RecordSecurityEvent records a security event.
// Parameters:
//   - eventType: Type of security event (e.g., "brute_force", "suspicious_activity")
//   - severity: Severity level ("low", "medium", "high", "critical")
func (sm *SecurityMetrics) RecordSecurityEvent(eventType, severity string) {
	sm.securityEventsTotal.WithLabelValues(eventType, severity).Inc()
}

// Vulnerability Metrics

// SetVulnerabilitiesFound sets the current count of vulnerabilities found.
// Parameters:
//   - severity: Vulnerability severity ("low", "medium", "high", "critical")
//   - tool: Scanner tool name (e.g., "gosec", "govulncheck", "trivy")
//   - count: Number of vulnerabilities
func (sm *SecurityMetrics) SetVulnerabilitiesFound(severity, tool string, count float64) {
	sm.vulnerabilitiesFound.WithLabelValues(severity, tool).Set(count)
}

// RecordSecurityScan records a security scan execution.
// Parameters:
//   - tool: Scanner tool name (e.g., "gosec", "govulncheck", "trivy")
//   - result: Scan result ("success", "failure", "error")
func (sm *SecurityMetrics) RecordSecurityScan(tool, result string) {
	sm.securityScansTotal.WithLabelValues(tool, result).Inc()
}

// TLS/Connection Metrics

// RecordTLSConnection records a TLS connection establishment.
// Parameters:
//   - version: TLS version (e.g., "1.2", "1.3")
//   - cipherSuite: Cipher suite used (e.g., "TLS_AES_128_GCM_SHA256")
func (sm *SecurityMetrics) RecordTLSConnection(version, cipherSuite string) {
	sm.tlsConnectionsTotal.WithLabelValues(version, cipherSuite).Inc()
}

// RecordTLSHandshakeDuration records the duration of a TLS handshake.
// Parameters:
//   - version: TLS version (e.g., "1.2", "1.3")
//   - durationSeconds: Duration in seconds
func (sm *SecurityMetrics) RecordTLSHandshakeDuration(version string, durationSeconds float64) {
	sm.tlsHandshakeDuration.WithLabelValues(version).Observe(durationSeconds)
}

// SetCertificateExpiry sets the time until certificate expiry.
// Parameters:
//   - commonName: Certificate common name
//   - secondsUntilExpiry: Seconds until the certificate expires
func (sm *SecurityMetrics) SetCertificateExpiry(commonName string, secondsUntilExpiry float64) {
	sm.certificateExpiry.WithLabelValues(commonName).Set(secondsUntilExpiry)
}

// Session Metrics

// SetActiveSessions sets the current number of active sessions.
// Parameters:
//   - count: Number of active sessions
func (sm *SecurityMetrics) SetActiveSessions(count float64) {
	sm.activeSessionsGauge.Set(count)
}

// IncreaseActiveSessions increments the active sessions count.
func (sm *SecurityMetrics) IncreaseActiveSessions() {
	sm.activeSessionsGauge.Inc()
}

// DecreaseActiveSessions decrements the active sessions count.
func (sm *SecurityMetrics) DecreaseActiveSessions() {
	sm.activeSessionsGauge.Dec()
}

// RecordSessionCreated records a session creation.
// Parameters:
//   - sessionType: Type of session (e.g., "user", "service", "temporary")
func (sm *SecurityMetrics) RecordSessionCreated(sessionType string) {
	sm.sessionCreatedTotal.WithLabelValues(sessionType).Inc()
}

// RecordSessionRevoked records a session revocation.
// Parameters:
//   - reason: Reason for revocation (e.g., "logout", "timeout", "admin_action")
func (sm *SecurityMetrics) RecordSessionRevoked(reason string) {
	sm.sessionRevokedTotal.WithLabelValues(reason).Inc()
}

// GetRegistry returns the Prometheus registry used by this metrics collector.
// This can be used to expose metrics via an HTTP endpoint.
func (sm *SecurityMetrics) GetRegistry() *prometheus.Registry {
	return sm.registry
}
