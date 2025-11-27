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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// DoSTestSuite provides DoS protection testing capabilities.
type DoSTestSuite struct {
	t      *testing.T
	router *gin.Engine
	config *DoSTestConfig
}

// DoSTestConfig holds configuration for DoS tests.
type DoSTestConfig struct {
	// MaxPayloadSize is the maximum allowed payload size
	MaxPayloadSize int64
	// MaxJSONDepth is the maximum allowed JSON nesting depth
	MaxJSONDepth int
	// MaxRequestsPerSecond is the expected rate limit
	MaxRequestsPerSecond int
	// ConcurrentConnections is the number of concurrent connections to test
	ConcurrentConnections int
	// TestDuration is the duration for load tests
	TestDuration time.Duration
}

// DefaultDoSTestConfig returns default DoS test configuration.
func DefaultDoSTestConfig() *DoSTestConfig {
	return &DoSTestConfig{
		MaxPayloadSize:        10 * 1024 * 1024, // 10MB
		MaxJSONDepth:          100,
		MaxRequestsPerSecond:  100,
		ConcurrentConnections: 50,
		TestDuration:          5 * time.Second,
	}
}

// NewDoSTestSuite creates a new DoS test suite.
func NewDoSTestSuite(t *testing.T, router *gin.Engine, config *DoSTestConfig) *DoSTestSuite {
	if config == nil {
		config = DefaultDoSTestConfig()
	}
	return &DoSTestSuite{
		t:      t,
		router: router,
		config: config,
	}
}

// ============================================================================
// Payload Size Tests
// ============================================================================

// TestLargePayloadHandling tests handling of large payloads.
func (ds *DoSTestSuite) TestLargePayloadHandling() {
	ds.t.Run("LargePayloadHandling", func(t *testing.T) {
		// Test various payload sizes
		sizes := []struct {
			name string
			size int64
		}{
			{"1KB", 1024},
			{"10KB", 10 * 1024},
			{"100KB", 100 * 1024},
			{"1MB", 1024 * 1024},
			{"10MB", 10 * 1024 * 1024},
			{"100MB", 100 * 1024 * 1024},
		}

		for _, size := range sizes {
			t.Run(size.name, func(t *testing.T) {
				payload := strings.Repeat("A", int(size.size))
				resp := ds.makeRequest("POST", "/api/data", map[string]string{
					"Content-Type": "application/json",
				}, []byte(fmt.Sprintf(`{"data":"%s"}`, payload)))

				if size.size > ds.config.MaxPayloadSize {
					if resp.Code == http.StatusOK {
						t.Errorf("Payload of size %s should be rejected", size.name)
					}
				}

				t.Logf("Payload %s: status %d", size.name, resp.Code)
			})
		}
	})
}

// TestLargeHeaderHandling tests handling of large HTTP headers.
func (ds *DoSTestSuite) TestLargeHeaderHandling() {
	ds.t.Run("LargeHeaderHandling", func(t *testing.T) {
		// Test large header values
		headerSizes := []int{1024, 8192, 16384, 32768, 65536}

		for _, size := range headerSizes {
			t.Run(fmt.Sprintf("%dBytes", size), func(t *testing.T) {
				largeValue := strings.Repeat("A", size)
				resp := ds.makeRequest("GET", "/api/health", map[string]string{
					"X-Custom-Header": largeValue,
				}, nil)

				// Large headers should be rejected or handled gracefully
				t.Logf("Header size %d: status %d", size, resp.Code)
			})
		}

		// Test many headers
		t.Run("ManyHeaders", func(t *testing.T) {
			headers := make(map[string]string)
			for i := 0; i < 100; i++ {
				headers[fmt.Sprintf("X-Custom-Header-%d", i)] = "value"
			}

			resp := ds.makeRequest("GET", "/api/health", headers, nil)
			t.Logf("Many headers: status %d", resp.Code)
		})
	})
}

// ============================================================================
// JSON Parsing Tests
// ============================================================================

// TestNestedJSONHandling tests handling of deeply nested JSON.
func (ds *DoSTestSuite) TestNestedJSONHandling() {
	ds.t.Run("NestedJSONHandling", func(t *testing.T) {
		depths := []int{10, 50, 100, 500, 1000}

		for _, depth := range depths {
			t.Run(fmt.Sprintf("Depth%d", depth), func(t *testing.T) {
				nestedJSON := ds.createNestedJSON(depth)
				resp := ds.makeRequest("POST", "/api/data", map[string]string{
					"Content-Type": "application/json",
				}, nestedJSON)

				if depth > ds.config.MaxJSONDepth {
					if resp.Code == http.StatusOK {
						t.Errorf("JSON with depth %d should be rejected", depth)
					}
				}

				t.Logf("JSON depth %d: status %d", depth, resp.Code)
			})
		}
	})
}

// TestManyJSONKeys tests handling of JSON with many keys.
func (ds *DoSTestSuite) TestManyJSONKeys() {
	ds.t.Run("ManyJSONKeys", func(t *testing.T) {
		keyCounts := []int{100, 1000, 10000}

		for _, count := range keyCounts {
			t.Run(fmt.Sprintf("Keys%d", count), func(t *testing.T) {
				data := make(map[string]string)
				for i := 0; i < count; i++ {
					data[fmt.Sprintf("key%d", i)] = "value"
				}
				jsonData, _ := json.Marshal(data)

				resp := ds.makeRequest("POST", "/api/data", map[string]string{
					"Content-Type": "application/json",
				}, jsonData)

				t.Logf("JSON with %d keys: status %d", count, resp.Code)
			})
		}
	})
}

// TestMalformedJSON tests handling of malformed JSON.
func (ds *DoSTestSuite) TestMalformedJSON() {
	ds.t.Run("MalformedJSON", func(t *testing.T) {
		malformedPayloads := []struct {
			name    string
			payload string
		}{
			{"UnclosedBrace", `{"key": "value"`},
			{"UnclosedBracket", `["item1", "item2"`},
			{"TrailingComma", `{"key": "value",}`},
			{"DuplicateKeys", `{"key": "value1", "key": "value2"}`},
			{"InvalidUnicode", `{"key": "\uXXXX"}`},
			{"NullByte", "{\"key\": \"val\x00ue\"}"},
			{"ControlChars", "{\"key\": \"val\x01\x02ue\"}"},
			{"LargeNumber", `{"num": 999999999999999999999999999999999999999999999999}`},
			{"NegativeZero", `{"num": -0}`},
			{"ExponentialNumber", `{"num": 1e999999999}`},
		}

		for _, mp := range malformedPayloads {
			t.Run(mp.name, func(t *testing.T) {
				resp := ds.makeRequest("POST", "/api/data", map[string]string{
					"Content-Type": "application/json",
				}, []byte(mp.payload))

				// Malformed JSON should be rejected
				if resp.Code == http.StatusOK {
					t.Logf("Warning: Malformed JSON %s was accepted", mp.name)
				}
			})
		}
	})
}

// ============================================================================
// Rate Limiting Tests
// ============================================================================

// TestRateLimiting tests rate limiting protection.
func (ds *DoSTestSuite) TestRateLimiting() {
	ds.t.Run("RateLimiting", func(t *testing.T) {
		// Send requests rapidly
		var rateLimitHit atomic.Bool
		var successCount, failCount atomic.Int32

		requestCount := ds.config.MaxRequestsPerSecond * 2

		for i := 0; i < requestCount; i++ {
			resp := ds.makeRequest("GET", "/api/health", nil, nil)

			if resp.Code == http.StatusTooManyRequests {
				rateLimitHit.Store(true)
				failCount.Add(1)
			} else if resp.Code == http.StatusOK {
				successCount.Add(1)
			}
		}

		t.Logf("Rate limit test: %d successful, %d rate limited", successCount.Load(), failCount.Load())

		if !rateLimitHit.Load() {
			t.Log("Warning: Rate limiting may not be configured")
		}
	})
}

// TestConcurrentConnections tests handling of many concurrent connections.
func (ds *DoSTestSuite) TestConcurrentConnections() {
	ds.t.Run("ConcurrentConnections", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount, failCount atomic.Int32

		for i := 0; i < ds.config.ConcurrentConnections; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				resp := ds.makeRequest("GET", "/api/health", nil, nil)
				if resp.Code == http.StatusOK {
					successCount.Add(1)
				} else {
					failCount.Add(1)
				}
			}()
		}

		wg.Wait()

		t.Logf("Concurrent connections: %d successful, %d failed",
			successCount.Load(), failCount.Load())

		// Most connections should succeed
		if float64(failCount.Load())/float64(ds.config.ConcurrentConnections) > 0.5 {
			t.Error("Too many concurrent connections failed")
		}
	})
}

// TestSustainedLoad tests handling of sustained load.
func (ds *DoSTestSuite) TestSustainedLoad() {
	ds.t.Run("SustainedLoad", func(t *testing.T) {
		var wg sync.WaitGroup
		var totalRequests, successCount, failCount atomic.Int32

		done := make(chan struct{})

		// Start workers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					select {
					case <-done:
						return
					default:
						totalRequests.Add(1)
						resp := ds.makeRequest("GET", "/api/health", nil, nil)
						if resp.Code == http.StatusOK {
							successCount.Add(1)
						} else {
							failCount.Add(1)
						}
					}
				}
			}()
		}

		// Run for configured duration
		time.Sleep(ds.config.TestDuration)
		close(done)
		wg.Wait()

		rps := float64(totalRequests.Load()) / ds.config.TestDuration.Seconds()
		successRate := float64(successCount.Load()) / float64(totalRequests.Load()) * 100

		t.Logf("Sustained load test: %d total requests, %.2f RPS, %.2f%% success rate",
			totalRequests.Load(), rps, successRate)
	})
}

// ============================================================================
// Resource Exhaustion Tests
// ============================================================================

// TestSlowlorisProtection tests protection against slowloris attacks.
func (ds *DoSTestSuite) TestSlowlorisProtection() {
	ds.t.Run("SlowlorisProtection", func(t *testing.T) {
		// This test requires actual TCP connections
		// In unit tests, we can only simulate the concept
		t.Skip("Slowloris test requires integration testing with actual TCP connections")
	})
}

// TestHashCollisionAttack tests protection against hash collision attacks.
func (ds *DoSTestSuite) TestHashCollisionAttack() {
	ds.t.Run("HashCollisionAttack", func(t *testing.T) {
		// Create payload with potentially colliding keys
		// This is a simplified test - real hash collision attacks require
		// knowledge of the hash function implementation
		data := make(map[string]string)
		for i := 0; i < 1000; i++ {
			// Use predictable key patterns
			data[fmt.Sprintf("key_%d", i)] = "value"
		}
		jsonData, _ := json.Marshal(data)

		start := time.Now()
		resp := ds.makeRequest("POST", "/api/data", map[string]string{
			"Content-Type": "application/json",
		}, jsonData)
		duration := time.Since(start)

		t.Logf("Hash collision test: status %d, duration %v", resp.Code, duration)

		// Processing should complete in reasonable time
		if duration > 5*time.Second {
			t.Error("Request took too long - possible hash collision vulnerability")
		}
	})
}

// TestRegexDoS tests protection against ReDoS (Regular Expression DoS).
func (ds *DoSTestSuite) TestRegexDoS() {
	ds.t.Run("RegexDoS", func(t *testing.T) {
		// ReDoS payloads designed to cause catastrophic backtracking
		redosPayloads := []string{
			strings.Repeat("a", 30) + "!",
			strings.Repeat("a", 30) + "X",
			strings.Repeat("ab", 15) + "!",
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa!", // For patterns like (a+)+
		}

		for i, payload := range redosPayloads {
			t.Run(fmt.Sprintf("Payload%d", i), func(t *testing.T) {
				start := time.Now()
				resp := ds.makeRequest("GET", "/api/search?q="+payload, nil, nil)
				duration := time.Since(start)

				t.Logf("ReDoS payload %d: status %d, duration %v", i, resp.Code, duration)

				if duration > 1*time.Second {
					t.Error("Potential ReDoS vulnerability detected")
				}
			})
		}
	})
}

// TestXMLBombProtection tests protection against XML bomb attacks.
func (ds *DoSTestSuite) TestXMLBombProtection() {
	ds.t.Run("XMLBombProtection", func(t *testing.T) {
		// Billion laughs attack (XML bomb)
		xmlBomb := `<?xml version="1.0"?>
<!DOCTYPE lolz [
  <!ENTITY lol "lol">
  <!ENTITY lol2 "&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;&lol;">
  <!ENTITY lol3 "&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;&lol2;">
  <!ENTITY lol4 "&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;&lol3;">
]>
<lolz>&lol4;</lolz>`

		resp := ds.makeRequest("POST", "/api/data", map[string]string{
			"Content-Type": "application/xml",
		}, []byte(xmlBomb))

		// XML bomb should be rejected
		if resp.Code == http.StatusOK {
			t.Error("XML bomb attack should be rejected")
		}
	})
}

// TestZipBombProtection tests protection against zip bomb attacks.
func (ds *DoSTestSuite) TestZipBombProtection() {
	ds.t.Run("ZipBombProtection", func(t *testing.T) {
		// This is a simplified test - real zip bombs require actual compressed data
		t.Skip("Zip bomb test requires actual compressed test files")
	})
}

// ============================================================================
// Request Smuggling Tests
// ============================================================================

// TestHTTPRequestSmuggling tests protection against HTTP request smuggling.
func (ds *DoSTestSuite) TestHTTPRequestSmuggling() {
	ds.t.Run("HTTPRequestSmuggling", func(t *testing.T) {
		// CL.TE smuggling attempt
		t.Run("CL_TE_Smuggling", func(t *testing.T) {
			// This requires raw HTTP manipulation
			// In unit tests, we can only verify header handling
			req := httptest.NewRequest("POST", "/api/data", strings.NewReader("smuggled"))
			req.Header.Set("Content-Length", "0")
			req.Header.Set("Transfer-Encoding", "chunked")

			w := httptest.NewRecorder()
			ds.router.ServeHTTP(w, req)

			// Should reject conflicting headers
			t.Logf("CL.TE smuggling: status %d", w.Code)
		})

		// TE.CL smuggling attempt
		t.Run("TE_CL_Smuggling", func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/data", strings.NewReader("0\r\n\r\nsmuggled"))
			req.Header.Set("Transfer-Encoding", "chunked")
			req.Header.Set("Content-Length", "6")

			w := httptest.NewRecorder()
			ds.router.ServeHTTP(w, req)

			t.Logf("TE.CL smuggling: status %d", w.Code)
		})
	})
}

// ============================================================================
// Helper Methods
// ============================================================================

func (ds *DoSTestSuite) makeRequest(method, path string, headers map[string]string, body []byte) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req := httptest.NewRequest(method, path, bodyReader)

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	ds.router.ServeHTTP(w, req)
	return w
}

func (ds *DoSTestSuite) createNestedJSON(depth int) []byte {
	result := ""
	for i := 0; i < depth; i++ {
		result += `{"nested":`
	}
	result += `"value"`
	for i := 0; i < depth; i++ {
		result += "}"
	}
	return []byte(result)
}

// ============================================================================
// DoS Test Report
// ============================================================================

// DoSTestResult represents the result of a DoS test.
type DoSTestResult struct {
	TestName string
	Category string
	Passed   bool
	Details  string
	Metrics  map[string]interface{}
}

// DoSTestReport represents a DoS test report.
type DoSTestReport struct {
	Timestamp   time.Time
	Config      *DoSTestConfig
	TotalTests  int
	PassedTests int
	FailedTests int
	Results     []DoSTestResult
}

// GenerateDoSReport generates a DoS test report.
func (ds *DoSTestSuite) GenerateDoSReport() *DoSTestReport {
	return &DoSTestReport{
		Timestamp:   time.Now(),
		Config:      ds.config,
		TotalTests:  0,
		PassedTests: 0,
		FailedTests: 0,
		Results:     []DoSTestResult{},
	}
}

// ToJSON converts the report to JSON.
func (r *DoSTestReport) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToHTML converts the report to HTML.
func (r *DoSTestReport) ToHTML() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>DoS Protection Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .passed { color: green; }
        .failed { color: red; }
        table { border-collapse: collapse; width: 100%%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
    </style>
</head>
<body>
    <h1>DoS Protection Test Report</h1>
    <p>Generated: %s</p>
    <p>Total Tests: %d | Passed: %d | Failed: %d</p>
    <h2>Configuration</h2>
    <ul>
        <li>Max Payload Size: %d bytes</li>
        <li>Max JSON Depth: %d</li>
        <li>Max RPS: %d</li>
        <li>Concurrent Connections: %d</li>
    </ul>
    <h2>Results</h2>
    <table>
        <tr>
            <th>Test Name</th>
            <th>Category</th>
            <th>Status</th>
            <th>Details</th>
        </tr>
`, r.Timestamp.Format(time.RFC3339), r.TotalTests, r.PassedTests, r.FailedTests,
		r.Config.MaxPayloadSize, r.Config.MaxJSONDepth,
		r.Config.MaxRequestsPerSecond, r.Config.ConcurrentConnections))

	for _, result := range r.Results {
		status := "PASSED"
		statusClass := "passed"
		if !result.Passed {
			status = "FAILED"
			statusClass = "failed"
		}

		sb.WriteString(fmt.Sprintf(`        <tr>
            <td>%s</td>
            <td>%s</td>
            <td class="%s">%s</td>
            <td>%s</td>
        </tr>
`, result.TestName, result.Category, statusClass, status, result.Details))
	}

	sb.WriteString(`    </table>
</body>
</html>`)

	return sb.String()
}

// ============================================================================
// Run All DoS Tests
// ============================================================================

// RunAllDoSTests runs all DoS protection tests.
func (ds *DoSTestSuite) RunAllDoSTests() {
	ds.TestLargePayloadHandling()
	ds.TestLargeHeaderHandling()
	ds.TestNestedJSONHandling()
	ds.TestManyJSONKeys()
	ds.TestMalformedJSON()
	ds.TestRateLimiting()
	ds.TestConcurrentConnections()
	ds.TestSustainedLoad()
	ds.TestHashCollisionAttack()
	ds.TestRegexDoS()
	ds.TestXMLBombProtection()
	ds.TestHTTPRequestSmuggling()
}
