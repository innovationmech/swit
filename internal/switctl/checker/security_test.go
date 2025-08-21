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

package checker

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// SecurityCheckerTestSuite tests the security checker functionality
type SecurityCheckerTestSuite struct {
	suite.Suite
	tempDir         string
	securityChecker *SecurityChecker
	mockLogger      *MockLogger
}

func TestSecurityCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityCheckerTestSuite))
}

func (s *SecurityCheckerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "security-checker-test-*")
	s.Require().NoError(err)

	s.mockLogger = &MockLogger{messages: make([]LogMessage, 0)}
	s.securityChecker = NewSecurityChecker(s.tempDir, s.mockLogger)
}

func (s *SecurityCheckerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	if s.securityChecker != nil {
		s.securityChecker.Close()
	}
}

// TestSecurityCheckerInterface tests that SecurityChecker implements IndividualChecker
func (s *SecurityCheckerTestSuite) TestSecurityCheckerInterface() {
	var _ IndividualChecker = s.securityChecker
	s.NotNil(s.securityChecker)
}

// TestNewSecurityChecker tests the constructor
func (s *SecurityCheckerTestSuite) TestNewSecurityChecker() {
	checker := NewSecurityChecker("/test/path", s.mockLogger)

	s.NotNil(checker)
	s.Equal("/test/path", checker.workDir)
	s.Equal(s.mockLogger, checker.logger)
	s.Equal("security", checker.Name())
	s.Contains(checker.Description(), "security")
	s.Contains(checker.Description(), "gosec")
}

// TestDefaultSecurityConfig tests default configuration
func (s *SecurityCheckerTestSuite) TestDefaultSecurityConfig() {
	config := defaultSecurityConfig()

	s.True(config.GosecEnabled)
	s.False(config.NancyEnabled) // Requires setup
	s.False(config.TrivyEnabled) // Requires setup
	s.Equal("medium", config.SeverityThreshold)
	s.Equal("json", config.ReportFormat)
	s.Equal(50, config.MaxIssues)
	s.Equal("5m", config.Timeout)
	s.False(config.FailOnMedium)
	s.True(config.FailOnHigh)
	s.Contains(config.ExcludePatterns, "vendor/*")
}

// TestSetConfig tests configuration setting
func (s *SecurityCheckerTestSuite) TestSetConfig() {
	customConfig := SecurityConfig{
		GosecEnabled:      false,
		NancyEnabled:      true,
		TrivyEnabled:      true,
		SeverityThreshold: "high",
		MaxIssues:         100,
		Timeout:           "10m",
		FailOnMedium:      true,
		ExcludePatterns:   []string{"test/*"},
	}

	s.securityChecker.SetConfig(customConfig)
	s.Equal(customConfig, s.securityChecker.config)
}

// TestCheckWithCleanCode tests security check with clean code
func (s *SecurityCheckerTestSuite) TestCheckWithCleanCode() {
	// Create clean Go files
	cleanCode := `// Copyright © 2025 Test Author
package main

import (
	"crypto/rand"
	"fmt"
	"os"
)

func main() {
	key := os.Getenv("API_KEY")
	fmt.Println("API Key loaded:", len(key))
	
	// Use secure random
	data := make([]byte, 32)
	rand.Read(data)
}
`

	err := s.createTestFile("clean.go", cleanCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.Equal("security", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "successfully")
	s.True(result.Duration > 0)
	s.False(result.Fixable) // Security issues are not auto-fixable
}

// TestCheckWithHardcodedSecrets tests detection of hardcoded secrets
func (s *SecurityCheckerTestSuite) TestCheckWithHardcodedSecrets() {
	// Create code with hardcoded secrets
	insecureCode := `package main

import "fmt"

func main() {
	password := "secret123password"
	apiKey := "sk-1234567890abcdef"
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	
	fmt.Println(password, apiKey, token)
}
`

	err := s.createTestFile("secrets.go", insecureCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.Equal("security", result.Name)
	s.True(result.Status == interfaces.CheckStatusFail || result.Status == interfaces.CheckStatusWarning)
	s.NotEmpty(result.Details)

	// Should detect hardcoded secrets
	foundHardcodedSecret := false
	for _, detail := range result.Details {
		if detail.Rule == "HARDCODED_SECRET" {
			foundHardcodedSecret = true
			s.Contains(detail.Message, "hardcoded secret")
			s.Equal("high", detail.Severity)
			if securityIssue, ok := detail.Context.(interfaces.SecurityIssue); ok {
				s.Contains(securityIssue.Fix, "environment variables")
			}
		}
	}
	s.True(foundHardcodedSecret, "Should detect hardcoded secrets")
}

// TestCheckWithSQLInjectionRisk tests detection of SQL injection risks
func (s *SecurityCheckerTestSuite) TestCheckWithSQLInjectionRisk() {
	// Create code with SQL injection vulnerabilities
	sqlInjectionCode := `package main

import (
	"fmt"
	"database/sql"
)

func getUserData(db *sql.DB, userID string) {
	query := "SELECT * FROM users WHERE id = " + userID
	rows, err := db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()
	
	query2 := fmt.Sprintf("SELECT * FROM posts WHERE user_id = %s", userID)
	db.Query(query2)
}
`

	err := s.createTestFile("sql_injection.go", sqlInjectionCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.NotEmpty(result.Details)

	// Should detect SQL injection risk
	foundSQLInjection := false
	for _, detail := range result.Details {
		if detail.Rule == "SQL_INJECTION_RISK" {
			foundSQLInjection = true
			s.Contains(detail.Message, "SQL injection")
			s.Equal("high", detail.Severity)
			if securityIssue, ok := detail.Context.(interfaces.SecurityIssue); ok {
				s.Contains(securityIssue.Fix, "parameterized")
			}
		}
	}
	s.True(foundSQLInjection, "Should detect SQL injection risk")
}

// TestCheckWithUnsafeHTTP tests detection of insecure HTTP usage
func (s *SecurityCheckerTestSuite) TestCheckWithUnsafeHTTP() {
	// Create code with insecure HTTP usage
	unsafeHTTPCode := `package main

import (
	"net/http"
	"io/ioutil"
)

func fetchData() {
	resp, err := http.Get("http://api.example.com/data")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	_ = body
}
`

	err := s.createTestFile("unsafe_http.go", unsafeHTTPCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.NotEmpty(result.Details)

	// Should detect unsafe HTTP usage
	foundUnsafeHTTP := false
	for _, detail := range result.Details {
		if detail.Rule == "UNSAFE_HTTP" {
			foundUnsafeHTTP = true
			s.Contains(detail.Message, "HTTP used instead of HTTPS")
			s.Equal("medium", detail.Severity)
			if securityIssue, ok := detail.Context.(interfaces.SecurityIssue); ok {
				s.Contains(securityIssue.Fix, "HTTPS")
			}
		}
	}
	s.True(foundUnsafeHTTP, "Should detect unsafe HTTP usage")
}

// TestCheckWithWeakCrypto tests detection of weak cryptography
func (s *SecurityCheckerTestSuite) TestCheckWithWeakCrypto() {
	// Create code with weak cryptographic practices
	weakCryptoCode := `package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/des"
	"crypto/rc4"
	"math/rand"
)

func weakCrypto() {
	hasher1 := md5.New()
	hasher2 := sha1.New()
	
	key := []byte("12345678")
	cipher1, _ := des.NewCipher(key)
	cipher2, _ := rc4.NewCipher(key)
	
	data := make([]byte, 10)
	rand.Read(data) // Should use crypto/rand
	
	_, _, _ = hasher1, hasher2, cipher1
	_, _ = cipher2, data
}
`

	err := s.createTestFile("weak_crypto.go", weakCryptoCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.NotEmpty(result.Details)

	// Should detect weak crypto
	foundWeakCrypto := false
	for _, detail := range result.Details {
		if detail.Rule == "WEAK_CRYPTO" {
			foundWeakCrypto = true
			s.Contains(detail.Message, "cryptographic")
			s.Equal("medium", detail.Severity)
			if securityIssue, ok := detail.Context.(interfaces.SecurityIssue); ok {
				s.Contains(securityIssue.Fix, "strong")
			}
		}
	}
	s.True(foundWeakCrypto, "Should detect weak cryptography")
}

// TestGosecIntegration tests gosec tool integration
func (s *SecurityCheckerTestSuite) TestGosecIntegration() {
	// Test gosec output parsing
	gosecOutput := `{
		"Issues": [
			{
				"severity": "HIGH",
				"confidence": "HIGH",
				"cwe": {
					"id": "295",
					"url": "https://cwe.mitre.org/data/definitions/295.html"
				},
				"rule_id": "G101",
				"details": "Potential hardcoded credentials",
				"file": "main.go",
				"code": "password := \"secret123\"",
				"line": "10",
				"column": "2"
			}
		],
		"Stats": {
			"files": 5,
			"lines": 100
		}
	}`

	issues, filesScanned := s.securityChecker.parseGosecOutput(gosecOutput)

	s.Equal(5, filesScanned)
	s.Equal(1, len(issues))

	issue := issues[0]
	s.Equal("G101", issue.ID)
	s.Equal("high", issue.Severity)
	s.Equal("main.go", issue.File)
	s.Equal(10, issue.Line)
	s.Contains(issue.Description, "credentials")
	s.Contains(issue.Fix, "environment variables")
}

// TestNancyOutputParsing tests Nancy dependency scanner output parsing
func (s *SecurityCheckerTestSuite) TestNancyOutputParsing() {
	// Test Nancy output parsing
	nancyOutput := `{"vulnerability": {"id": "CVE-2021-1234", "title": "Test Vulnerability", "description": "A test vulnerability", "cvss_score": 7.5}}`

	issues := s.securityChecker.parseNancyOutput(nancyOutput)

	s.Equal(1, len(issues))

	issue := issues[0]
	s.Equal("NANCY-CVE-2021-1234", issue.ID)
	s.Equal("high", issue.Severity) // 7.5 CVSS maps to high
	s.Equal(7.5, issue.CVSS)
	s.Contains(issue.Title, "Dependency vulnerability")
	s.Contains(issue.Description, "test vulnerability")
}

// TestTrivyOutputParsing tests Trivy scanner output parsing
func (s *SecurityCheckerTestSuite) TestTrivyOutputParsing() {
	// Test Trivy output parsing
	trivyOutput := `{
		"Results": [
			{
				"Target": "go.mod",
				"Vulnerabilities": [
					{
						"VulnerabilityID": "CVE-2021-5678",
						"PkgName": "github.com/example/pkg",
						"Title": "Example vulnerability in package",
						"Description": "A vulnerability description",
						"Severity": "MEDIUM",
						"CVSS": {"Score": 5.5},
						"References": ["https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-5678"],
						"FixedVersion": "v1.2.3"
					}
				]
			}
		]
	}`

	issues := s.securityChecker.parseTrivyOutput(trivyOutput)

	s.Equal(1, len(issues))

	issue := issues[0]
	s.Equal("CVE-2021-5678", issue.ID)
	s.Equal("medium", issue.Severity)
	s.Equal(5.5, issue.CVSS)
	s.Contains(issue.Title, "Example vulnerability")
	s.Contains(issue.Fix, "v1.2.3")
}

// TestSeverityCalculation tests severity level calculations
func (s *SecurityCheckerTestSuite) TestSeverityCalculation() {
	tests := []struct {
		issues   []interfaces.SecurityIssue
		expected string
	}{
		{
			issues:   []interfaces.SecurityIssue{},
			expected: "none",
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "low"},
			},
			expected: "low",
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "medium"},
				{Severity: "low"},
			},
			expected: "medium",
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "high"},
				{Severity: "medium"},
			},
			expected: "high",
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "critical"},
				{Severity: "high"},
			},
			expected: "high",
		},
	}

	for _, test := range tests {
		result := s.securityChecker.calculateOverallSeverity(test.issues)
		s.Equal(test.expected, result)
	}
}

// TestSecurityScoreCalculation tests security score calculations
func (s *SecurityCheckerTestSuite) TestSecurityScoreCalculation() {
	tests := []struct {
		issues   []interfaces.SecurityIssue
		expected int
	}{
		{
			issues:   []interfaces.SecurityIssue{},
			expected: 100,
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "low"},
			},
			expected: 95,
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "medium"},
			},
			expected: 90,
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "high"},
			},
			expected: 80,
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "critical"},
			},
			expected: 70,
		},
		{
			issues: []interfaces.SecurityIssue{
				{Severity: "critical"},
				{Severity: "high"},
				{Severity: "medium"},
				{Severity: "low"},
			},
			expected: 35, // 100 - 30 - 20 - 10 - 5
		},
	}

	for _, test := range tests {
		result := s.securityChecker.calculateSecurityScore(test.issues)
		s.Equal(test.expected, result)
	}
}

// TestSeverityLevelMapping tests severity level numeric mapping
func (s *SecurityCheckerTestSuite) TestSeverityLevelMapping() {
	tests := []struct {
		severity string
		expected int
	}{
		{"critical", 4},
		{"high", 3},
		{"medium", 2},
		{"low", 1},
		{"info", 0},
		{"unknown", 0},
	}

	for _, test := range tests {
		result := s.securityChecker.getSeverityLevel(test.severity)
		s.Equal(test.expected, result)
	}
}

// TestIssueFiltering tests filtering and prioritization of issues
func (s *SecurityCheckerTestSuite) TestIssueFiltering() {
	// Create issues with different severities
	issues := []interfaces.SecurityIssue{
		{ID: "1", Severity: "low"},
		{ID: "2", Severity: "medium"},
		{ID: "3", Severity: "high"},
		{ID: "4", Severity: "critical"},
		{ID: "5", Severity: "low"},
	}

	// Set medium threshold
	config := s.securityChecker.config
	config.SeverityThreshold = "medium"
	config.MaxIssues = 3
	s.securityChecker.SetConfig(config)

	filtered := s.securityChecker.filterAndPrioritizeIssues(issues)

	// Should have filtered out low severity and limited to 3 issues
	s.Equal(3, len(filtered))

	// Should be sorted by severity (highest first)
	s.Equal("critical", filtered[0].Severity)
	s.Equal("high", filtered[1].Severity)
	s.Equal("medium", filtered[2].Severity)
}

// TestCVSSMapping tests CVSS score to severity mapping
func (s *SecurityCheckerTestSuite) TestCVSSMapping() {
	tests := []struct {
		cvssStr  string
		expected string
	}{
		{"9.5", "critical"},
		{"7.8", "high"},
		{"5.2", "medium"},
		{"2.1", "low"},
		{"invalid", "low"},
	}

	for _, test := range tests {
		result := s.securityChecker.mapNancySeverity(test.cvssStr)
		s.Equal(test.expected, result)
	}
}

// TestHardcodedSecretsDetection tests specific patterns for hardcoded secrets
func (s *SecurityCheckerTestSuite) TestHardcodedSecretsDetection() {
	tests := []struct {
		line     string
		expected bool
	}{
		{`password = "secret123"`, true},
		{`api_key := "sk-1234567890abcdef"`, true},
		{`token = "eyJhbGciOiJIUzI1NiI"`, true},
		{`private_key = "-----BEGIN"`, true},
		{`password = "test"`, false},    // Too short
		{`password = "example"`, false}, // Common test value
		{`var x = 123`, false},          // Not a secret pattern
	}

	for _, test := range tests {
		result := s.securityChecker.containsHardcodedSecrets(test.line)
		s.Equal(test.expected, result, "Line: %s", test.line)
	}
}

// TestSQLInjectionDetection tests SQL injection pattern detection
func (s *SecurityCheckerTestSuite) TestSQLInjectionDetection() {
	tests := []struct {
		line     string
		expected bool
	}{
		{`query := "SELECT * FROM users WHERE id = " + userID`, true},
		{`fmt.Sprintf("SELECT * FROM table WHERE x = %s", input)`, true},
		{`fmt.Sprintf("INSERT INTO table VALUES (%s)", data)`, true},
		{`query := "SELECT * FROM users WHERE id = ?"`, false}, // Parameterized
		{`var x = 123`, false}, // Not SQL
	}

	for _, test := range tests {
		result := s.securityChecker.containsSQLInjectionRisk(test.line)
		s.Equal(test.expected, result, "Line: %s", test.line)
	}
}

// TestWeakCryptoDetection tests weak cryptography pattern detection
func (s *SecurityCheckerTestSuite) TestWeakCryptoDetection() {
	tests := []struct {
		line     string
		expected bool
	}{
		{`md5.New()`, true},
		{`sha1.New()`, true},
		{`des.NewCipher(key)`, true},
		{`rc4.NewCipher(key)`, true},
		{`rand.Read(data)`, false},        // Note: Requires import-aware analysis for proper detection
		{`sha256.New()`, false},           // Secure
		{`crypto/rand.Read(data)`, false}, // Secure
	}

	for _, test := range tests {
		result := s.securityChecker.containsWeakCrypto(test.line)
		s.Equal(test.expected, result, "Line: %s", test.line)
	}
}

// TestFileExclusionPatterns tests file exclusion patterns
func (s *SecurityCheckerTestSuite) TestFileExclusionPatterns() {
	config := s.securityChecker.config
	config.ExcludePatterns = []string{"*_test.go", "vendor/*", "*.pb.go"}
	s.securityChecker.SetConfig(config)

	tests := []struct {
		filename string
		expected bool
	}{
		{"main.go", false},
		{"main_test.go", true},
		{"vendor/pkg/file.go", true},
		{"api.pb.go", true},
		{"src/main.go", false},
	}

	for _, test := range tests {
		result := s.securityChecker.isExcludedFile(test.filename)
		s.Equal(test.expected, result, "Filename: %s", test.filename)
	}
}

// TestGeneratedFileDetection tests detection of generated files
func (s *SecurityCheckerTestSuite) TestGeneratedFileDetection() {
	tests := []struct {
		filename string
		expected bool
	}{
		{"api.pb.go", true},
		{"api.pb.gw.go", true},
		{"types_gen.go", true},
		{"generated.go", true},
		{"main.go", false},
		{"handler.go", false},
	}

	for _, test := range tests {
		result := s.securityChecker.isGeneratedFile(test.filename)
		s.Equal(test.expected, result, "Filename: %s", test.filename)
	}
}

// TestEmptyProject tests handling of project with no Go files
func (s *SecurityCheckerTestSuite) TestEmptyProject() {
	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.Equal("security", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "successfully")
	s.Empty(result.Details) // No issues found
}

// TestContextCancellation tests handling of context cancellation
func (s *SecurityCheckerTestSuite) TestContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.securityChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.Canceled, err)
}

// TestContextTimeout tests handling of context timeout
func (s *SecurityCheckerTestSuite) TestContextTimeout() {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout

	_, err := s.securityChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.DeadlineExceeded, err)
}

// TestConcurrentExecution tests thread safety
func (s *SecurityCheckerTestSuite) TestConcurrentExecution() {
	// Create test file
	testCode := `package main
import "fmt"
func main() { fmt.Println("Hello") }`

	err := s.createTestFile("concurrent.go", testCode)
	s.Require().NoError(err)

	// Run multiple concurrent checks
	const numGoroutines = 5
	results := make([]interfaces.CheckResult, numGoroutines)
	errors := make([]error, numGoroutines)

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ctx := context.Background()
			results[index], errors[index] = s.securityChecker.Check(ctx)
		}(i)
	}

	wg.Wait()

	// Verify all checks completed successfully
	for i := 0; i < numGoroutines; i++ {
		s.NoError(errors[i], "Check %d should not error", i)
		s.Equal("security", results[i].Name)
	}
}

// TestTimeoutConfiguration tests timeout configuration parsing
func (s *SecurityCheckerTestSuite) TestTimeoutConfiguration() {
	tests := []struct {
		timeoutStr string
		expected   time.Duration
	}{
		{"", 5 * time.Minute},        // Default
		{"10s", 10 * time.Second},    // Valid duration
		{"2m", 2 * time.Minute},      // Valid duration
		{"invalid", 5 * time.Minute}, // Invalid falls back to default
	}

	for _, test := range tests {
		config := s.securityChecker.config
		config.Timeout = test.timeoutStr
		s.securityChecker.SetConfig(config)

		result := s.securityChecker.getTimeoutDuration()
		s.Equal(test.expected, result, "Timeout string: %s", test.timeoutStr)
	}
}

// TestRuleSuggestions tests fix suggestions for specific rules
func (s *SecurityCheckerTestSuite) TestRuleSuggestions() {
	tests := []struct {
		ruleID   string
		expected string
	}{
		{"G101", "Remove hardcoded credentials"},
		{"G201", "Use parameterized queries"},
		{"G401", "Use strong cryptographic hash"},
		{"G999", "Review and fix the security issue"}, // Default suggestion
	}

	for _, test := range tests {
		result := s.securityChecker.getSuggestionForRule(test.ruleID)
		s.Contains(result, test.expected, "Rule ID: %s", test.ruleID)
	}
}

// TestMultipleSecurityIssues tests handling of multiple different security issues
func (s *SecurityCheckerTestSuite) TestMultipleSecurityIssues() {
	// Create file with multiple security issues
	multipleIssuesCode := `package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"database/sql"
)

func insecureFunction() {
	// Hardcoded secret
	apiKey := "sk-1234567890abcdef"
	
	// Weak crypto
	hasher := md5.New()
	
	// Insecure HTTP
	resp, err := http.Get("http://api.example.com/data")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	// SQL injection risk
	userID := getUserInput()
	query := "SELECT * FROM users WHERE id = " + userID
	db.Query(query)
	
	_, _ = apiKey, hasher
}

func getUserInput() string { return "123" }
var db *sql.DB
`

	err := s.createTestFile("multiple_issues.go", multipleIssuesCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.securityChecker.Check(ctx)

	s.NoError(err)
	s.Equal("security", result.Name)
	s.True(result.Status == interfaces.CheckStatusFail || result.Status == interfaces.CheckStatusWarning)
	s.NotEmpty(result.Details)

	// Should detect multiple types of issues
	issueTypes := make(map[string]bool)
	for _, detail := range result.Details {
		issueTypes[detail.Rule] = true
	}

	// Expect multiple different types of security issues
	s.True(len(issueTypes) >= 2, "Should detect multiple types of security issues")
}

// TestClose tests proper cleanup
func (s *SecurityCheckerTestSuite) TestClose() {
	err := s.securityChecker.Close()
	s.NoError(err)
}

// TestConfigWithCustomRules tests custom security rules
func (s *SecurityCheckerTestSuite) TestConfigWithCustomRules() {
	customConfig := SecurityConfig{
		GosecEnabled:      true,
		NancyEnabled:      true,
		TrivyEnabled:      true,
		SeverityThreshold: "low",
		ExcludeRules:      []string{"G101", "G201"},
		IncludeRules:      []string{"G401", "G501"},
		CustomRulesPath:   "/custom/rules",
		MaxIssues:         10,
		FailOnMedium:      true,
		FailOnHigh:        true,
	}

	s.securityChecker.SetConfig(customConfig)

	s.Equal(customConfig.GosecEnabled, s.securityChecker.config.GosecEnabled)
	s.Equal(customConfig.ExcludeRules, s.securityChecker.config.ExcludeRules)
	s.Equal(customConfig.IncludeRules, s.securityChecker.config.IncludeRules)
	s.Equal(customConfig.MaxIssues, s.securityChecker.config.MaxIssues)
}

// Helper methods

func (s *SecurityCheckerTestSuite) createTestFile(filename, content string) error {
	filePath := filepath.Join(s.tempDir, filename)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// Benchmark tests for performance

func BenchmarkSecurityCheck(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "security-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	checker := NewSecurityChecker(tempDir, logger)
	defer checker.Close()

	// Create test file
	code := `package main
import "fmt"
func main() { fmt.Println("Hello") }`

	err = os.WriteFile(filepath.Join(tempDir, "bench.go"), []byte(code), 0644)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSecurityCheckWithIssues(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "security-issues-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	logger := &MockLogger{}
	checker := NewSecurityChecker(tempDir, logger)
	defer checker.Close()

	// Create file with multiple security issues
	code := `package main
import (
	"crypto/md5"
	"net/http"
)
func main() {
	password := "secret123"
	hasher := md5.New()
	resp, _ := http.Get("http://api.example.com/data")
	_, _, _ = password, hasher, resp
}`

	err = os.WriteFile(filepath.Join(tempDir, "issues.go"), []byte(code), 0644)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHardcodedSecretDetection(b *testing.B) {
	checker := &SecurityChecker{}
	line := `apiKey := "sk-1234567890abcdefghijklmnopqrstuvwxyz"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker.containsHardcodedSecrets(line)
	}
}
