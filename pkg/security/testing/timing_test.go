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

package testing

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TimingTestSuite provides timing attack detection capabilities.
type TimingTestSuite struct {
	t         *testing.T
	router    *gin.Engine
	jwtSecret string
	config    *TimingTestConfig
}

// TimingTestConfig holds configuration for timing tests.
type TimingTestConfig struct {
	// Iterations is the number of measurements to take
	Iterations int
	// WarmupIterations is the number of warmup iterations before measurement
	WarmupIterations int
	// MaxVariancePercent is the maximum allowed variance percentage
	MaxVariancePercent float64
	// JWTSecret is the secret used for JWT signing
	JWTSecret string
}

// DefaultTimingTestConfig returns default timing test configuration.
func DefaultTimingTestConfig() *TimingTestConfig {
	return &TimingTestConfig{
		Iterations:         100,
		WarmupIterations:   10,
		MaxVariancePercent: 20.0,
		JWTSecret:          "test-secret-key",
	}
}

// NewTimingTestSuite creates a new timing test suite.
func NewTimingTestSuite(t *testing.T, router *gin.Engine, config *TimingTestConfig) *TimingTestSuite {
	if config == nil {
		config = DefaultTimingTestConfig()
	}
	return &TimingTestSuite{
		t:         t,
		router:    router,
		jwtSecret: config.JWTSecret,
		config:    config,
	}
}

// ============================================================================
// Timing Attack Tests
// ============================================================================

// TestTokenValidationTiming tests for timing side-channels in token validation.
func (ts *TimingTestSuite) TestTokenValidationTiming() {
	ts.t.Run("TokenValidationTiming", func(t *testing.T) {
		// Test timing difference between valid and invalid tokens
		t.Run("ValidVsInvalidToken", func(t *testing.T) {
			validToken := ts.createValidToken()
			invalidToken := ts.createInvalidToken()

			validTimes := ts.measureTokenValidation(validToken, ts.config.Iterations)
			invalidTimes := ts.measureTokenValidation(invalidToken, ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Valid vs Invalid Token", validTimes, invalidTimes)
		})

		// Test timing difference for signature verification
		t.Run("SignatureVerificationTiming", func(t *testing.T) {
			validToken := ts.createValidToken()
			wrongSigToken := ts.createTokenWithWrongSignature()

			validTimes := ts.measureTokenValidation(validToken, ts.config.Iterations)
			wrongSigTimes := ts.measureTokenValidation(wrongSigToken, ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Valid vs Wrong Signature", validTimes, wrongSigTimes)
		})

		// Test timing difference for expired tokens
		t.Run("ExpiredTokenTiming", func(t *testing.T) {
			validToken := ts.createValidToken()
			expiredToken := ts.createExpiredToken()

			validTimes := ts.measureTokenValidation(validToken, ts.config.Iterations)
			expiredTimes := ts.measureTokenValidation(expiredToken, ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Valid vs Expired Token", validTimes, expiredTimes)
		})
	})
}

// TestStringComparisonTiming tests for timing side-channels in string comparisons.
func (ts *TimingTestSuite) TestStringComparisonTiming() {
	ts.t.Run("StringComparisonTiming", func(t *testing.T) {
		// Test early vs late mismatch
		t.Run("EarlyVsLateMismatch", func(t *testing.T) {
			secret := "correct-secret-value"

			// Mismatch at position 0
			earlyMismatch := "Xorrect-secret-value"
			// Mismatch at last position
			lateMismatch := "correct-secret-valuX"

			earlyTimes := ts.measureStringComparison(secret, earlyMismatch, ts.config.Iterations)
			lateTimes := ts.measureStringComparison(secret, lateMismatch, ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Early vs Late Mismatch", earlyTimes, lateTimes)
		})

		// Test length-based timing
		t.Run("LengthBasedTiming", func(t *testing.T) {
			secret := "correct-secret-value"

			shortInput := "short"
			correctLengthInput := strings.Repeat("X", len(secret))

			shortTimes := ts.measureStringComparison(secret, shortInput, ts.config.Iterations)
			correctLengthTimes := ts.measureStringComparison(secret, correctLengthInput, ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Short vs Correct Length", shortTimes, correctLengthTimes)
		})
	})
}

// TestPasswordComparisonTiming tests for timing side-channels in password comparisons.
func (ts *TimingTestSuite) TestPasswordComparisonTiming() {
	ts.t.Run("PasswordComparisonTiming", func(t *testing.T) {
		// Test password hash comparison
		t.Run("PasswordHashComparison", func(t *testing.T) {
			correctHash := sha256.Sum256([]byte("correct-password"))
			wrongHash := sha256.Sum256([]byte("wrong-password"))

			correctTimes := ts.measureHashComparison(correctHash[:], correctHash[:], ts.config.Iterations)
			wrongTimes := ts.measureHashComparison(correctHash[:], wrongHash[:], ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Correct vs Wrong Hash", correctTimes, wrongTimes)
		})
	})
}

// TestHMACVerificationTiming tests for timing side-channels in HMAC verification.
func (ts *TimingTestSuite) TestHMACVerificationTiming() {
	ts.t.Run("HMACVerificationTiming", func(t *testing.T) {
		message := []byte("test message")
		key := []byte(ts.jwtSecret)

		h := hmac.New(sha256.New, key)
		h.Write(message)
		correctMAC := h.Sum(nil)

		// Wrong MAC with early mismatch
		earlyWrongMAC := make([]byte, len(correctMAC))
		copy(earlyWrongMAC, correctMAC)
		earlyWrongMAC[0] ^= 0xFF

		// Wrong MAC with late mismatch
		lateWrongMAC := make([]byte, len(correctMAC))
		copy(lateWrongMAC, correctMAC)
		lateWrongMAC[len(lateWrongMAC)-1] ^= 0xFF

		correctTimes := ts.measureHMACVerification(key, message, correctMAC, ts.config.Iterations)
		earlyWrongTimes := ts.measureHMACVerification(key, message, earlyWrongMAC, ts.config.Iterations)
		lateWrongTimes := ts.measureHMACVerification(key, message, lateWrongMAC, ts.config.Iterations)

		ts.analyzeTimingDifference(t, "Correct vs Early Wrong MAC", correctTimes, earlyWrongTimes)
		ts.analyzeTimingDifference(t, "Correct vs Late Wrong MAC", correctTimes, lateWrongTimes)
		ts.analyzeTimingDifference(t, "Early vs Late Wrong MAC", earlyWrongTimes, lateWrongTimes)
	})
}

// TestUserEnumerationTiming tests for timing side-channels that enable user enumeration.
func (ts *TimingTestSuite) TestUserEnumerationTiming() {
	ts.t.Run("UserEnumerationTiming", func(t *testing.T) {
		// Test login endpoint timing for existing vs non-existing users
		t.Run("ExistingVsNonExistingUser", func(t *testing.T) {
			existingUserTimes := ts.measureLoginAttempt("existing-user", "wrong-password", ts.config.Iterations)
			nonExistingUserTimes := ts.measureLoginAttempt("non-existing-user", "wrong-password", ts.config.Iterations)

			ts.analyzeTimingDifference(t, "Existing vs Non-Existing User", existingUserTimes, nonExistingUserTimes)
		})
	})
}

// ============================================================================
// Measurement Functions
// ============================================================================

func (ts *TimingTestSuite) measureTokenValidation(token string, iterations int) []time.Duration {
	// Warmup
	for i := 0; i < ts.config.WarmupIterations; i++ {
		ts.validateToken(token)
	}

	times := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		ts.validateToken(token)
		times[i] = time.Since(start)
	}

	return times
}

func (ts *TimingTestSuite) measureStringComparison(a, b string, iterations int) []time.Duration {
	// Warmup
	for i := 0; i < ts.config.WarmupIterations; i++ {
		_ = subtle.ConstantTimeCompare([]byte(a), []byte(b))
	}

	times := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_ = subtle.ConstantTimeCompare([]byte(a), []byte(b))
		times[i] = time.Since(start)
	}

	return times
}

func (ts *TimingTestSuite) measureHashComparison(a, b []byte, iterations int) []time.Duration {
	// Warmup
	for i := 0; i < ts.config.WarmupIterations; i++ {
		_ = subtle.ConstantTimeCompare(a, b)
	}

	times := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		_ = subtle.ConstantTimeCompare(a, b)
		times[i] = time.Since(start)
	}

	return times
}

func (ts *TimingTestSuite) measureHMACVerification(key, message, mac []byte, iterations int) []time.Duration {
	// Warmup
	for i := 0; i < ts.config.WarmupIterations; i++ {
		h := hmac.New(sha256.New, key)
		h.Write(message)
		expectedMAC := h.Sum(nil)
		_ = hmac.Equal(expectedMAC, mac)
	}

	times := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		h := hmac.New(sha256.New, key)
		h.Write(message)
		expectedMAC := h.Sum(nil)
		_ = hmac.Equal(expectedMAC, mac)
		times[i] = time.Since(start)
	}

	return times
}

func (ts *TimingTestSuite) measureLoginAttempt(username, password string, iterations int) []time.Duration {
	// Warmup
	for i := 0; i < ts.config.WarmupIterations; i++ {
		ts.attemptLogin(username, password)
	}

	times := make([]time.Duration, iterations)
	for i := 0; i < iterations; i++ {
		start := time.Now()
		ts.attemptLogin(username, password)
		times[i] = time.Since(start)
	}

	return times
}

// ============================================================================
// Analysis Functions
// ============================================================================

func (ts *TimingTestSuite) analyzeTimingDifference(t *testing.T, name string, times1, times2 []time.Duration) {
	stats1 := calculateStats(times1)
	stats2 := calculateStats(times2)

	// Calculate timing difference
	avgDiff := math.Abs(float64(stats1.Mean-stats2.Mean)) / float64(stats1.Mean) * 100

	t.Logf("%s Analysis:", name)
	t.Logf("  Set 1: mean=%v, median=%v, stddev=%v", stats1.Mean, stats1.Median, stats1.StdDev)
	t.Logf("  Set 2: mean=%v, median=%v, stddev=%v", stats2.Mean, stats2.Median, stats2.StdDev)
	t.Logf("  Difference: %.2f%%", avgDiff)

	// Check if timing difference exceeds threshold
	if avgDiff > ts.config.MaxVariancePercent {
		t.Logf("  WARNING: Potential timing side-channel detected (%.2f%% > %.2f%%)",
			avgDiff, ts.config.MaxVariancePercent)
	} else {
		t.Logf("  OK: Timing variance within acceptable range")
	}
}

// TimingStats holds statistical data about timing measurements.
type TimingStats struct {
	Mean   time.Duration
	Median time.Duration
	StdDev time.Duration
	Min    time.Duration
	Max    time.Duration
}

func calculateStats(times []time.Duration) TimingStats {
	if len(times) == 0 {
		return TimingStats{}
	}

	// Sort for median
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	// Calculate mean
	var sum time.Duration
	for _, t := range times {
		sum += t
	}
	mean := sum / time.Duration(len(times))

	// Calculate median
	var median time.Duration
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		median = sorted[len(sorted)/2]
	}

	// Calculate standard deviation
	var sumSquares float64
	for _, t := range times {
		diff := float64(t - mean)
		sumSquares += diff * diff
	}
	stdDev := time.Duration(math.Sqrt(sumSquares / float64(len(times))))

	return TimingStats{
		Mean:   mean,
		Median: median,
		StdDev: stdDev,
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

func (ts *TimingTestSuite) validateToken(token string) bool {
	resp := ts.makeRequest("GET", "/api/protected", map[string]string{
		"Authorization": "Bearer " + token,
	}, nil)
	return resp.Code == http.StatusOK
}

func (ts *TimingTestSuite) attemptLogin(username, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	return ts.makeRequest("POST", "/api/auth/login", map[string]string{
		"Content-Type": "application/json",
	}, body)
}

func (ts *TimingTestSuite) makeRequest(method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	ts.router.ServeHTTP(w, req)
	return w
}

func (ts *TimingTestSuite) createValidToken() string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"roles": []string{"user"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(ts.jwtSecret))
	return tokenString
}

func (ts *TimingTestSuite) createInvalidToken() string {
	return "invalid.token.here"
}

func (ts *TimingTestSuite) createTokenWithWrongSignature() string {
	token := ts.createValidToken()
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		parts[2] = base64.RawURLEncoding.EncodeToString([]byte("wrong-signature"))
		return strings.Join(parts, ".")
	}
	return token
}

func (ts *TimingTestSuite) createExpiredToken() string {
	claims := jwt.MapClaims{
		"sub":   "test-user",
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"iat":   time.Now().Add(-2 * time.Hour).Unix(),
		"roles": []string{"user"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(ts.jwtSecret))
	return tokenString
}

// ============================================================================
// Constant-Time Comparison Utilities
// ============================================================================

// ConstantTimeStringCompare performs constant-time string comparison.
func ConstantTimeStringCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ConstantTimeByteCompare performs constant-time byte slice comparison.
func ConstantTimeByteCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// SecureHMACVerify performs constant-time HMAC verification.
func SecureHMACVerify(key, message, mac []byte) bool {
	h := hmac.New(sha256.New, key)
	h.Write(message)
	expectedMAC := h.Sum(nil)
	return hmac.Equal(expectedMAC, mac)
}

// ============================================================================
// Timing Attack Report
// ============================================================================

// TimingTestResult represents the result of a timing test.
type TimingTestResult struct {
	TestName          string
	Stats1            TimingStats
	Stats2            TimingStats
	DifferencePercent float64
	Vulnerable        bool
	Details           string
}

// TimingTestReport represents a timing test report.
type TimingTestReport struct {
	Timestamp       time.Time
	Config          *TimingTestConfig
	TotalTests      int
	VulnerableTests int
	Results         []TimingTestResult
}

// GenerateTimingReport generates a timing test report.
func (ts *TimingTestSuite) GenerateTimingReport() *TimingTestReport {
	return &TimingTestReport{
		Timestamp:       time.Now(),
		Config:          ts.config,
		TotalTests:      0,
		VulnerableTests: 0,
		Results:         []TimingTestResult{},
	}
}

// ToJSON converts the report to JSON.
func (r *TimingTestReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToHTML converts the report to HTML.
func (r *TimingTestReport) ToHTML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Timing Attack Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .safe { color: green; }
        .vulnerable { color: red; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
    </style>
</head>
<body>
    <h1>Timing Attack Test Report</h1>
    <p>Generated: %s</p>
    <p>Total Tests: %d | Vulnerable: %d</p>
    <h2>Configuration</h2>
    <ul>
        <li>Iterations: %d</li>
        <li>Warmup Iterations: %d</li>
        <li>Max Variance: %.2f%%</li>
    </ul>
    <h2>Results</h2>
    <table>
        <tr>
            <th>Test Name</th>
            <th>Difference</th>
            <th>Status</th>
            <th>Details</th>
        </tr>
`, r.Timestamp.Format(time.RFC3339), r.TotalTests, r.VulnerableTests,
		r.Config.Iterations, r.Config.WarmupIterations, r.Config.MaxVariancePercent))

	for _, result := range r.Results {
		status := "SAFE"
		statusClass := "safe"
		if result.Vulnerable {
			status = "VULNERABLE"
			statusClass = "vulnerable"
		}

		sb.WriteString(fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>%.2f%%</td>
            <td class="%s">%s</td>
            <td>%s</td>
        </tr>
`, result.TestName, result.DifferencePercent, statusClass, status, result.Details))
	}

	sb.WriteString(`    </table>
</body>
</html>`)

	return sb.String()
}

// ============================================================================
// Run All Timing Tests
// ============================================================================

// RunAllTimingTests runs all timing attack tests.
func (ts *TimingTestSuite) RunAllTimingTests() {
	ts.TestTokenValidationTiming()
	ts.TestStringComparisonTiming()
	ts.TestPasswordComparisonTiming()
	ts.TestHMACVerificationTiming()
	ts.TestUserEnumerationTiming()
}
