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

package checker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// StyleCheckerTestSuite tests the style checker functionality
type StyleCheckerTestSuite struct {
	suite.Suite
	tempDir      string
	styleChecker *StyleChecker
	mockFS       *MockFileSystem
}

func TestStyleCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(StyleCheckerTestSuite))
}

func (s *StyleCheckerTestSuite) SetupTest() {
	var err error
	s.tempDir, err = os.MkdirTemp("", "style-checker-test-*")
	s.Require().NoError(err)

	s.mockFS = NewMockFileSystem()
	s.styleChecker = NewStyleChecker(s.tempDir, StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   true,
		EnableImportCheck:    true,
		EnableCopyrightCheck: true,
		GolangciLintConfig:   ".golangci.yml",
	})
}

func (s *StyleCheckerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
	if s.styleChecker != nil {
		s.styleChecker.Close()
	}
}

// TestStyleCheckerInterface tests that StyleChecker implements IndividualChecker
func (s *StyleCheckerTestSuite) TestStyleCheckerInterface() {
	var _ IndividualChecker = s.styleChecker
	s.NotNil(s.styleChecker)
}

// TestNewStyleChecker tests the constructor
func (s *StyleCheckerTestSuite) TestNewStyleChecker() {
	checker := NewStyleChecker("/test/path", StyleCheckerOptions{
		EnableGofmt: true,
	})

	s.NotNil(checker)
	s.Equal("/test/path", checker.projectPath)
	s.True(checker.options.EnableGofmt)
	s.Equal("code_style", checker.Name())
	s.Contains(checker.Description(), "code style")
}

// TestCheckWithValidGoCode tests style checking with properly formatted Go code
func (s *StyleCheckerTestSuite) TestCheckWithValidGoCode() {
	// Create valid Go files
	validCode := `// Copyright © 2025 Test Author
//
// Licensed under MIT License

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Hello, World!")
	os.Exit(0)
}
`

	err := s.createTestFile("main.go", validCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.styleChecker.Check(ctx)

	s.NoError(err)
	s.Equal("code_style", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "passed")
	s.True(result.Duration > 0)
	s.True(result.Score >= 80) // Should have high score for valid code
}

// TestCheckWithFormattingIssues tests detection of formatting issues
func (s *StyleCheckerTestSuite) TestCheckWithFormattingIssues() {
	// Create poorly formatted Go code
	badlyFormattedCode := `package main
import"fmt"
func main(){fmt.Println("bad formatting")}
`

	err := s.createTestFile("bad.go", badlyFormattedCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.styleChecker.Check(ctx)

	s.NoError(err)
	s.Equal("code_style", result.Name)
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.Contains(result.Message, "formatting")
	s.NotEmpty(result.Details)
	s.True(result.Fixable) // gofmt issues are fixable
}

// TestCheckWithLintIssues tests golangci-lint integration
func (s *StyleCheckerTestSuite) TestCheckWithLintIssues() {
	// Create code with lint issues
	lintIssueCode := `// Copyright © 2025 Test Author
package main

import (
	"fmt"
	"os" // unused import
)

func main() {
	x := 42 // unused variable
	fmt.Println("Hello")
}
`

	err := s.createTestFile("lint_issues.go", lintIssueCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.styleChecker.Check(ctx)

	s.NoError(err)
	s.Equal("code_style", result.Name)
	s.True(result.Status == interfaces.CheckStatusFail || result.Status == interfaces.CheckStatusWarning)
	s.NotEmpty(result.Details)

	// Should detect unused variable and import issues
	foundUnusedVar := false
	foundUnusedImport := false
	for _, detail := range result.Details {
		if detail.Rule == "ineffassign" || detail.Rule == "unused" {
			foundUnusedVar = true
		}
		if detail.Message == "unused import" {
			foundUnusedImport = true
		}
	}
	s.True(foundUnusedVar || foundUnusedImport, "Should detect lint issues")
}

// TestCheckImportOrganization tests import organization checking
func (s *StyleCheckerTestSuite) TestCheckImportOrganization() {
	// Create code with poorly organized imports
	poorImportsCode := `package main

import (
	"os"
	"github.com/stretchr/testify/assert"
	"fmt"
	"context"
)

func main() {
	fmt.Println("Hello")
}
`

	err := s.createTestFile("poor_imports.go", poorImportsCode)
	s.Require().NoError(err)

	result := s.styleChecker.checkImportOrganization()

	s.Equal(interfaces.CheckStatusWarning, result.Status)
	s.Contains(result.Message, "import")
	s.NotEmpty(result.Details)

	// Should suggest proper import organization
	foundImportDetail := false
	for _, detail := range result.Details {
		if detail.Rule == "import-organization" {
			foundImportDetail = true
		}
	}
	s.True(foundImportDetail)
}

// TestCheckCopyrightHeaders tests copyright header validation
func (s *StyleCheckerTestSuite) TestCheckCopyrightHeaders() {
	// Test with missing copyright header
	noCopyrightCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`

	err := s.createTestFile("no_copyright.go", noCopyrightCode)
	s.Require().NoError(err)

	result := s.styleChecker.checkCopyrightHeaders()

	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.Contains(result.Message, "copyright")
	s.NotEmpty(result.Details)

	// Should identify files missing copyright
	foundMissingCopyright := false
	for _, detail := range result.Details {
		if detail.Rule == "missing-copyright" && detail.File == "no_copyright.go" {
			foundMissingCopyright = true
		}
	}
	s.True(foundMissingCopyright)
}

// TestCheckCopyrightHeadersWithValidHeaders tests valid copyright headers
func (s *StyleCheckerTestSuite) TestCheckCopyrightHeadersWithValidHeaders() {
	validCopyrightCode := `// Copyright © 2025 Test Author
//
// Licensed under MIT License

package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`

	err := s.createTestFile("valid_copyright.go", validCopyrightCode)
	s.Require().NoError(err)

	result := s.styleChecker.checkCopyrightHeaders()

	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "valid")
}

// TestGofmtCheck tests gofmt checking specifically
func (s *StyleCheckerTestSuite) TestGofmtCheck() {
	// Test properly formatted code
	formattedCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	err := s.createTestFile("formatted.go", formattedCode)
	s.Require().NoError(err)

	result := s.styleChecker.runGofmtCheck()

	s.Equal(interfaces.CheckStatusPass, result.Status)
	s.Contains(result.Message, "formatted")

	// Test unformatted code
	unformattedCode := `package main
import"fmt"
func main(){fmt.Println("Hello")}`

	err = s.createTestFile("unformatted.go", unformattedCode)
	s.Require().NoError(err)

	result = s.styleChecker.runGofmtCheck()

	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.Contains(result.Message, "formatting")
	s.True(result.Fixable)
}

// TestGolangciLintIntegration tests golangci-lint integration
func (s *StyleCheckerTestSuite) TestGolangciLintIntegration() {
	// Create golangci-lint config
	configContent := `run:
  timeout: 5m
  tests: true

linters:
  enable:
    - gofmt
    - golint
    - govet
    - ineffassign
    - misspell

linters-settings:
  gofmt:
    simplify: true
`

	err := s.createTestFile(".golangci.yml", configContent)
	s.Require().NoError(err)

	// Create code with various issues
	issueCode := `package main

import (
	"fmt"
	"os" // unused import
)

func main() {
	var unused_var int // unused variable, bad naming
	fmt.Println("Hello worlld") // typo
}
`

	err = s.createTestFile("issues.go", issueCode)
	s.Require().NoError(err)

	result := s.styleChecker.runGolangciLint()

	s.NotEqual(interfaces.CheckStatusPass, result.Status)
	s.NotEmpty(result.Details)
}

// TestDisabledCheckers tests behavior when individual checkers are disabled
func (s *StyleCheckerTestSuite) TestDisabledCheckers() {
	// Create a Go file so the checker doesn't exit early with "no Go files found"
	err := s.createTestFile("test.go", `package main

func main() {
    fmt.Println("Hello, World!")
}
`)
	s.Require().NoError(err)

	checker := NewStyleChecker(s.tempDir, StyleCheckerOptions{
		EnableGofmt:          false,
		EnableGolangciLint:   false,
		EnableImportCheck:    false,
		EnableCopyrightCheck: false,
	})
	defer checker.Close()

	ctx := context.Background()
	result, err := checker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusSkip, result.Status)
	s.Contains(result.Message, "disabled")
}

// TestPartiallyDisabledCheckers tests mixed enabled/disabled checkers
func (s *StyleCheckerTestSuite) TestPartiallyDisabledCheckers() {
	checker := NewStyleChecker(s.tempDir, StyleCheckerOptions{
		EnableGofmt:          true,
		EnableGolangciLint:   false,
		EnableImportCheck:    true,
		EnableCopyrightCheck: false,
	})
	defer checker.Close()

	// Create valid code
	validCode := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`

	err := s.createTestFile("partial.go", validCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := checker.Check(ctx)

	s.NoError(err)
	// Should pass since only enabled checks (gofmt and imports) pass
	s.Equal(interfaces.CheckStatusPass, result.Status)
}

// TestContextCancellation tests proper handling of context cancellation
func (s *StyleCheckerTestSuite) TestContextCancellation() {
	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.styleChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.Canceled, err)
}

// TestContextTimeout tests proper handling of context timeout
func (s *StyleCheckerTestSuite) TestContextTimeout() {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure timeout

	_, err := s.styleChecker.Check(ctx)
	s.Error(err)
	s.Equal(context.DeadlineExceeded, err)
}

// TestLargeCodebase tests performance with larger codebase
func (s *StyleCheckerTestSuite) TestLargeCodebase() {
	// Create multiple files to simulate larger codebase
	for i := 0; i < 10; i++ {
		code := `// Copyright © 2025 Test Author
package main

import "fmt"

func main() {
	fmt.Println("File %d")
}
`
		err := s.createTestFile(fmt.Sprintf("file%d.go", i), code)
		s.Require().NoError(err)
	}

	ctx := context.Background()
	start := time.Now()
	result, err := s.styleChecker.Check(ctx)
	duration := time.Since(start)

	s.NoError(err)
	s.NotNil(result)
	s.True(duration < 30*time.Second, "Should complete within reasonable time")
}

// TestInvalidGoFiles tests handling of invalid Go files
func (s *StyleCheckerTestSuite) TestInvalidGoFiles() {
	invalidCode := `this is not valid go code at all`

	err := s.createTestFile("invalid.go", invalidCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.styleChecker.Check(ctx)

	s.NoError(err) // Should not error, but report issues
	s.Equal(interfaces.CheckStatusFail, result.Status)
	s.NotEmpty(result.Details)
}

// TestEmptyProject tests handling of empty project directory
func (s *StyleCheckerTestSuite) TestEmptyProject() {
	emptyChecker := NewStyleChecker(s.tempDir, StyleCheckerOptions{
		EnableGofmt: true,
	})
	defer emptyChecker.Close()

	ctx := context.Background()
	result, err := emptyChecker.Check(ctx)

	s.NoError(err)
	s.Equal(interfaces.CheckStatusSkip, result.Status)
	s.Contains(result.Message, "no Go files")
}

// TestNonExistentProject tests handling of non-existent project path
func (s *StyleCheckerTestSuite) TestNonExistentProject() {
	nonExistentChecker := NewStyleChecker("/nonexistent/path", StyleCheckerOptions{
		EnableGofmt: true,
	})
	defer nonExistentChecker.Close()

	ctx := context.Background()
	result, err := nonExistentChecker.Check(ctx)

	s.NoError(err) // Should not error, but report skip
	s.Equal(interfaces.CheckStatusSkip, result.Status)
	s.Contains(result.Message, "not exist")
}

// TestClose tests proper cleanup
func (s *StyleCheckerTestSuite) TestClose() {
	err := s.styleChecker.Close()
	s.NoError(err)
}

// TestConcurrentChecks tests thread safety
func (s *StyleCheckerTestSuite) TestConcurrentChecks() {
	// Create valid test file
	validCode := `package main
import "fmt"
func main() { fmt.Println("Hello") }`

	err := s.createTestFile("concurrent.go", validCode)
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
			results[index], errors[index] = s.styleChecker.Check(ctx)
		}(i)
	}

	wg.Wait()

	// Verify all checks completed without errors
	for i := 0; i < numGoroutines; i++ {
		s.NoError(errors[i], "Check %d should not error", i)
		s.Equal("code_style", results[i].Name)
	}
}

// TestResultDetailStructure tests the structure of check details
func (s *StyleCheckerTestSuite) TestResultDetailStructure() {
	// Create code with specific issues
	issueCode := `package main
import"fmt"
func main(){
unused:=42
fmt.Println("test")
}
`

	err := s.createTestFile("detail_test.go", issueCode)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.styleChecker.Check(ctx)

	s.NoError(err)
	s.NotEmpty(result.Details)

	// Verify detail structure
	for _, detail := range result.Details {
		s.NotEmpty(detail.File)
		s.NotEmpty(detail.Message)
		s.NotEmpty(detail.Rule)
		s.NotEmpty(detail.Severity)
		if detail.Line > 0 {
			s.True(detail.Line > 0)
		}
		if detail.Column > 0 {
			s.True(detail.Column > 0)
		}
	}
}

// Helper methods

func (s *StyleCheckerTestSuite) createTestFile(filename, content string) error {
	filePath := filepath.Join(s.tempDir, filename)

	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// Benchmark tests for performance

func BenchmarkStyleCheck(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "style-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	code := `package main
import "fmt"
func main() { fmt.Println("Hello") }`

	err = os.WriteFile(filepath.Join(tempDir, "bench.go"), []byte(code), 0644)
	if err != nil {
		b.Fatal(err)
	}

	checker := NewStyleChecker(tempDir, StyleCheckerOptions{
		EnableGofmt: true,
	})
	defer checker.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLargeCodebaseStyleCheck(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "style-large-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple test files
	for i := 0; i < 50; i++ {
		code := fmt.Sprintf(`package main
import "fmt"
func main() { 
	fmt.Println("File %d")
}`, i)

		err = os.WriteFile(filepath.Join(tempDir, fmt.Sprintf("file%d.go", i)), []byte(code), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	checker := NewStyleChecker(tempDir, StyleCheckerOptions{
		EnableGofmt:       true,
		EnableImportCheck: true,
	})
	defer checker.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := checker.Check(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
