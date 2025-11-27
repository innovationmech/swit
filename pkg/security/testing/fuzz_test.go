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

package testing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// Fuzz Testing for JWT Tokens
// ============================================================================

// FuzzJWTToken fuzzes JWT token parsing and validation.
func FuzzJWTToken(f *testing.F) {
	// Add seed corpus
	seeds := []string{
		// Valid JWT structure
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
		// Empty parts
		"..",
		"...",
		// Invalid base64
		"!!!.!!!.!!!",
		// Very long token
		strings.Repeat("a", 10000),
		// Unicode characters
		"eyJ\u0000bGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.test",
		// None algorithm attack
		"eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwiYWRtaW4iOnRydWV9.",
		// Algorithm confusion
		"eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.test",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, token string) {
		// Test token parsing doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with token: %s, panic: %v", truncate(token, 100), r)
			}
		}()

		// Parse and validate the token
		_ = parseJWTForFuzz(token)
	})
}

// FuzzJWTHeader fuzzes JWT header parsing.
func FuzzJWTHeader(f *testing.F) {
	seeds := []string{
		`{"alg":"HS256","typ":"JWT"}`,
		`{"alg":"none"}`,
		`{"alg":"RS256","kid":"key1"}`,
		`{}`,
		`{"alg":""}`,
		`{"alg":null}`,
		`{"alg":123}`,
		`{"alg":"HS256","typ":"JWT","kid":"../../../etc/passwd"}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, header string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with header: %s, panic: %v", truncate(header, 100), r)
			}
		}()

		// Encode header and create token
		encodedHeader := base64.RawURLEncoding.EncodeToString([]byte(header))
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"test"}`))
		token := encodedHeader + "." + payload + ".signature"

		_ = parseJWTForFuzz(token)
	})
}

// FuzzJWTClaims fuzzes JWT claims parsing.
func FuzzJWTClaims(f *testing.F) {
	seeds := []string{
		`{"sub":"user","exp":9999999999}`,
		`{"sub":"","exp":0}`,
		`{"sub":null}`,
		`{"sub":123}`,
		`{"exp":"not-a-number"}`,
		`{"roles":["admin"]}`,
		`{"roles":"admin"}`,
		`{}`,
		`{"__proto__":{"admin":true}}`,
		`{"constructor":{"prototype":{"admin":true}}}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, claims string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with claims: %s, panic: %v", truncate(claims, 100), r)
			}
		}()

		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		encodedClaims := base64.RawURLEncoding.EncodeToString([]byte(claims))
		token := header + "." + encodedClaims + ".signature"

		_ = parseJWTForFuzz(token)
	})
}

// ============================================================================
// Fuzz Testing for HTTP Endpoints
// ============================================================================

// FuzzHTTPPath fuzzes HTTP path handling.
func FuzzHTTPPath(f *testing.F) {
	seeds := []string{
		"/api/users",
		"/api/users/123",
		"/../../../etc/passwd",
		"/api/users%2f..%2f..%2fetc%2fpasswd",
		"/api/users;id=123",
		"/api/users?id=123",
		"/api/users#fragment",
		"/api/users\x00",
		"/api/users\r\nX-Injected: header",
		strings.Repeat("/a", 1000),
		"/api/users/" + strings.Repeat("a", 10000),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	router := setupFuzzRouter()

	f.Fuzz(func(t *testing.T, path string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with path: %s, panic: %v", truncate(path, 100), r)
			}
		}()

		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify response doesn't leak sensitive information
		validateFuzzResponse(t, w, path)
	})
}

// FuzzHTTPHeaders fuzzes HTTP header handling.
func FuzzHTTPHeaders(f *testing.F) {
	seeds := []string{
		"Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.test",
		"Bearer ",
		"Bearer\x00token",
		"Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
		strings.Repeat("A", 10000),
		"Bearer " + strings.Repeat(".", 1000),
		"Bearer\r\nX-Injected: value",
		"Bearer\ttoken",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	router := setupFuzzRouter()

	f.Fuzz(func(t *testing.T, authHeader string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with auth header: %s, panic: %v", truncate(authHeader, 100), r)
			}
		}()

		req := httptest.NewRequest("GET", "/api/protected", nil)
		req.Header.Set("Authorization", authHeader)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		validateFuzzResponse(t, w, authHeader)
	})
}

// FuzzJSONBody fuzzes JSON request body parsing.
func FuzzJSONBody(f *testing.F) {
	seeds := []string{
		`{"name":"test"}`,
		`{}`,
		`[]`,
		`null`,
		`""`,
		`123`,
		`{"name":null}`,
		`{"name":"` + strings.Repeat("a", 10000) + `"}`,
		`{"a":{"b":{"c":{"d":{"e":"deep"}}}}}`,
		`{"__proto__":{"admin":true}}`,
		`{"constructor":{"prototype":{"admin":true}}}`,
		`{"name":"<script>alert('xss')</script>"}`,
		`{"name":"'; DROP TABLE users; --"}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	router := setupFuzzRouter()

	f.Fuzz(func(t *testing.T, body string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with body: %s, panic: %v", truncate(body, 100), r)
			}
		}()

		req := httptest.NewRequest("POST", "/api/data", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		validateFuzzResponse(t, w, body)
	})
}

// FuzzQueryParams fuzzes query parameter handling.
func FuzzQueryParams(f *testing.F) {
	seeds := []string{
		"search=test",
		"search=<script>alert('xss')</script>",
		"search=' OR '1'='1",
		"search=" + strings.Repeat("a", 10000),
		"search=test&search=test2",
		"search[]=test&search[]=test2",
		"search[$ne]=",
		"search=test%00null",
		"search=test%0d%0aX-Injected: header",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	router := setupFuzzRouter()

	f.Fuzz(func(t *testing.T, queryString string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with query: %s, panic: %v", truncate(queryString, 100), r)
			}
		}()

		req := httptest.NewRequest("GET", "/api/search?"+queryString, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		validateFuzzResponse(t, w, queryString)
	})
}

// ============================================================================
// Fuzz Testing for Input Validation
// ============================================================================

// FuzzEmailValidation fuzzes email validation.
func FuzzEmailValidation(f *testing.F) {
	seeds := []string{
		"user@example.com",
		"user+tag@example.com",
		"user@sub.example.com",
		"user@",
		"@example.com",
		"user@.com",
		"user@example.",
		"user@example.com\x00",
		"user@example.com\r\n",
		strings.Repeat("a", 100) + "@example.com",
		"user@" + strings.Repeat("a", 100) + ".com",
		"<script>@example.com",
		"user'--@example.com",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, email string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with email: %s, panic: %v", truncate(email, 100), r)
			}
		}()

		_ = validateEmailForFuzz(email)
	})
}

// FuzzPasswordValidation fuzzes password validation.
func FuzzPasswordValidation(f *testing.F) {
	seeds := []string{
		"password123",
		"",
		strings.Repeat("a", 1000),
		"pass\x00word",
		"pass\r\nword",
		"<script>alert('xss')</script>",
		"'; DROP TABLE users; --",
		"パスワード",
		"🔐🔑🔒",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, password string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with password length: %d, panic: %v", len(password), r)
			}
		}()

		_ = validatePasswordForFuzz(password)
	})
}

// FuzzUsernameValidation fuzzes username validation.
func FuzzUsernameValidation(f *testing.F) {
	seeds := []string{
		"user123",
		"admin",
		"root",
		"",
		strings.Repeat("a", 1000),
		"user\x00name",
		"user\r\nname",
		"<script>alert('xss')</script>",
		"'; DROP TABLE users; --",
		"../../../etc/passwd",
		"user@domain",
		"user+tag",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, username string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with username: %s, panic: %v", truncate(username, 100), r)
			}
		}()

		_ = validateUsernameForFuzz(username)
	})
}

// ============================================================================
// Fuzz Testing for Cryptographic Operations
// ============================================================================

// FuzzBase64Decode fuzzes base64 decoding.
func FuzzBase64Decode(f *testing.F) {
	seeds := []string{
		"dGVzdA==",
		"dGVzdA",
		"!!!",
		"",
		strings.Repeat("A", 10000),
		"dGVzdA==\x00",
		"dGVzdA==\r\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with base64 input: %s, panic: %v", truncate(input, 100), r)
			}
		}()

		// Try standard base64
		_, _ = base64.StdEncoding.DecodeString(input)

		// Try URL-safe base64
		_, _ = base64.URLEncoding.DecodeString(input)

		// Try raw URL-safe base64
		_, _ = base64.RawURLEncoding.DecodeString(input)
	})
}

// FuzzHexDecode fuzzes hex decoding.
func FuzzHexDecode(f *testing.F) {
	seeds := []string{
		"48656c6c6f",
		"",
		"0",
		"GG",
		strings.Repeat("00", 10000),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with hex input: %s, panic: %v", truncate(input, 100), r)
			}
		}()

		_ = decodeHexForFuzz(input)
	})
}

// ============================================================================
// Fuzz Testing for Policy Evaluation
// ============================================================================

// FuzzPolicyInput fuzzes OPA policy input.
func FuzzPolicyInput(f *testing.F) {
	seeds := []string{
		`{"user":{"id":"user1","roles":["admin"]}}`,
		`{"user":{"id":"","roles":[]}}`,
		`{"user":null}`,
		`{}`,
		`{"user":{"id":"user1","roles":["` + strings.Repeat("a", 1000) + `"]}}`,
		`{"user":{"id":"../../../etc/passwd"}}`,
		`{"user":{"id":"<script>alert('xss')</script>"}}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic occurred with policy input: %s, panic: %v", truncate(input, 100), r)
			}
		}()

		var policyInput map[string]interface{}
		_ = json.Unmarshal([]byte(input), &policyInput)
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func parseJWTForFuzz(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid token format")
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return err
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return err
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return err
	}

	return nil
}

func setupFuzzRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Recovery middleware to catch panics
	router.Use(gin.Recovery())

	// Add test endpoints
	router.GET("/api/protected", func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/api/search", func(c *gin.Context) {
		query := c.Query("search")
		c.JSON(http.StatusOK, gin.H{"query": query})
	})

	router.POST("/api/data", func(c *gin.Context) {
		var data map[string]interface{}
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"received": data})
	})

	router.GET("/api/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	return router
}

func validateFuzzResponse(t *testing.T, w *httptest.ResponseRecorder, input string) {
	body := w.Body.String()

	// Check for sensitive information leaks
	sensitivePatterns := []string{
		"panic",
		"goroutine",
		"runtime error",
		"/home/",
		"/Users/",
		"password",
		"secret",
		"stack trace",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(strings.ToLower(body), strings.ToLower(pattern)) {
			t.Errorf("Sensitive information leaked: %s (input: %s)", pattern, truncate(input, 50))
		}
	}

	// Check for reflection of dangerous characters without escaping
	if strings.Contains(input, "<script>") && strings.Contains(body, "<script>") {
		t.Errorf("XSS payload reflected without escaping (input: %s)", truncate(input, 50))
	}
}

func validateEmailForFuzz(email string) bool {
	if len(email) > 254 {
		return false
	}
	if !utf8.ValidString(email) {
		return false
	}
	if strings.Contains(email, "\x00") || strings.Contains(email, "\r") || strings.Contains(email, "\n") {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	return len(parts[0]) > 0 && len(parts[1]) > 0
}

func validatePasswordForFuzz(password string) bool {
	if len(password) < 8 || len(password) > 128 {
		return false
	}
	if !utf8.ValidString(password) {
		return false
	}
	if strings.Contains(password, "\x00") {
		return false
	}
	return true
}

func validateUsernameForFuzz(username string) bool {
	if len(username) < 3 || len(username) > 64 {
		return false
	}
	if !utf8.ValidString(username) {
		return false
	}
	// Check for dangerous characters
	dangerous := []string{"\x00", "\r", "\n", "/", "\\", "..", "<", ">", "'", "\""}
	for _, d := range dangerous {
		if strings.Contains(username, d) {
			return false
		}
	}
	return true
}

func decodeHexForFuzz(input string) []byte {
	if len(input)%2 != 0 {
		return nil
	}
	result := make([]byte, len(input)/2)
	for i := 0; i < len(input); i += 2 {
		var b byte
		_, err := fmt.Sscanf(input[i:i+2], "%02x", &b)
		if err != nil {
			return nil
		}
		result[i/2] = b
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ============================================================================
// Fuzz Test Helpers for External Use
// ============================================================================

// FuzzTestConfig configures fuzz testing behavior.
type FuzzTestConfig struct {
	MaxInputSize int
	Timeout      int // seconds
	Corpus       []string
}

// DefaultFuzzTestConfig returns default fuzz test configuration.
func DefaultFuzzTestConfig() *FuzzTestConfig {
	return &FuzzTestConfig{
		MaxInputSize: 1024 * 1024, // 1MB
		Timeout:      60,
		Corpus:       []string{},
	}
}

// RunFuzzHTTPEndpoint is a helper to fuzz test an HTTP endpoint.
// Note: This is a helper function, not a fuzz test itself.
func RunFuzzHTTPEndpoint(f *testing.F, router *gin.Engine, method, path string, config *FuzzTestConfig) {
	if config == nil {
		config = DefaultFuzzTestConfig()
	}

	// Add corpus
	for _, seed := range config.Corpus {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > config.MaxInputSize {
			return
		}

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Panic with input length %d: %v", len(input), r)
			}
		}()

		var req *http.Request
		if method == "GET" {
			req = httptest.NewRequest(method, path+"?input="+input, nil)
		} else {
			req = httptest.NewRequest(method, path, bytes.NewReader([]byte(input)))
			req.Header.Set("Content-Type", "application/json")
		}

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		validateFuzzResponse(t, w, input)
	})
}

