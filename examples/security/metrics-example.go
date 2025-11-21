// Copyright 2025 Swit. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/innovationmech/swit/pkg/security"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Example demonstrates how to integrate security metrics into your application
func main() {
	// Create security metrics collector
	metrics, err := security.NewSecurityMetrics(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to create security metrics: %v", err))
	}

	// Simulate authentication attempts
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Successful authentication
			start := time.Now()
			metrics.RecordAuthAttempt("jwt", "success")
			metrics.RecordAuthDuration("jwt", time.Since(start).Seconds())

			// Token validation
			metrics.RecordTokenValidation("success")

			// Simulate occasional failures
			if time.Now().Unix()%5 == 0 {
				metrics.RecordAuthAttempt("jwt", "failure")
			}
		}
	}()

	// Simulate policy evaluations
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			start := time.Now()
			metrics.RecordPolicyEvaluation("rbac", "allow")
			metrics.RecordPolicyEvaluationDuration("rbac", time.Since(start).Seconds())

			// Simulate occasional denials
			if time.Now().Unix()%4 == 0 {
				metrics.RecordPolicyEvaluation("rbac", "deny")
				metrics.RecordAccessDenied("insufficient_permissions", "/api/admin")
			}
		}
	}()

	// Simulate TLS connections
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			start := time.Now()
			metrics.RecordTLSConnection("1.3", "TLS_AES_128_GCM_SHA256")
			metrics.RecordTLSHandshakeDuration("1.3", time.Since(start).Seconds())
		}
	}()

	// Simulate security events
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			metrics.RecordSecurityEvent("suspicious_activity", "medium")
		}
	}()

	// Simulate vulnerability scanning
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			metrics.RecordSecurityScan("gosec", "success")
			metrics.SetVulnerabilitiesFound("high", "gosec", 3)
			metrics.SetVulnerabilitiesFound("medium", "gosec", 10)
		}
	}()

	// Simulate session management
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		sessions := 0
		for range ticker.C {
			// Simulate session creation/destruction
			if sessions < 50 && time.Now().Unix()%2 == 0 {
				metrics.IncreaseActiveSessions()
				metrics.RecordSessionCreated("user")
				sessions++
			} else if sessions > 0 && time.Now().Unix()%3 == 0 {
				metrics.DecreaseActiveSessions()
				metrics.RecordSessionRevoked("logout")
				sessions--
			}
		}
	}()

	// Expose metrics endpoint
	http.Handle("/metrics", promhttp.HandlerFor(
		metrics.GetRegistry(),
		promhttp.HandlerOpts{},
	))

	fmt.Println("Security metrics server started on :9090")
	fmt.Println("Visit http://localhost:9090/metrics to see metrics")
	fmt.Println("\nExample metrics available:")
	fmt.Println("- security_auth_attempts_total")
	fmt.Println("- security_auth_duration_seconds")
	fmt.Println("- security_token_validations_total")
	fmt.Println("- security_policy_evaluations_total")
	fmt.Println("- security_policy_evaluation_duration_seconds")
	fmt.Println("- security_access_denied_total")
	fmt.Println("- security_security_events_total")
	fmt.Println("- security_vulnerabilities_found")
	fmt.Println("- security_security_scans_total")
	fmt.Println("- security_tls_connections_total")
	fmt.Println("- security_tls_handshake_duration_seconds")
	fmt.Println("- security_active_sessions")
	fmt.Println("- security_session_created_total")
	fmt.Println("- security_session_revoked_total")

	if err := http.ListenAndServe(":9090", nil); err != nil {
		panic(fmt.Sprintf("Failed to start server: %v", err))
	}
}
