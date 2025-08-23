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
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// CompositeIndividualTestSuite tests CompositeChecker as IndividualChecker
type CompositeIndividualTestSuite struct {
	suite.Suite
	composite *CompositeChecker
	tempDir   string
	logger    interfaces.Logger
}

func (s *CompositeIndividualTestSuite) SetupTest() {
	s.logger = &MockLogger{}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "composite_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	s.composite = NewCompositeChecker(s.tempDir, s.logger)
}

func (s *CompositeIndividualTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func TestCompositeIndividualTestSuite(t *testing.T) {
	suite.Run(t, new(CompositeIndividualTestSuite))
}

// TestIndividualCheckerInterface tests that CompositeChecker implements IndividualChecker
func (s *CompositeIndividualTestSuite) TestIndividualCheckerInterface() {
	var _ IndividualChecker = s.composite
	s.NotNil(s.composite)
}

// TestName tests the Name method
func (s *CompositeIndividualTestSuite) TestName() {
	name := s.composite.Name()
	s.Equal("composite", name)
}

// TestDescription tests the Description method
func (s *CompositeIndividualTestSuite) TestDescription() {
	description := s.composite.Description()
	s.Contains(description, "quality")
	s.Contains(description, "checker")
}

// TestClose tests the Close method
func (s *CompositeIndividualTestSuite) TestClose() {
	err := s.composite.Close()
	s.NoError(err)
}

// TestCheck tests the Check method (comprehensive test)
func (s *CompositeIndividualTestSuite) TestCheck() {
	// Create some basic test files
	testFile := `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}
`
	err := os.WriteFile(s.tempDir+"/main.go", []byte(testFile), 0644)
	s.Require().NoError(err)

	ctx := context.Background()
	result, err := s.composite.Check(ctx)

	s.NoError(err)
	s.NotNil(result)
	s.Equal("composite", result.Name)
	s.NotZero(result.Duration)
	// Status should be one of the valid statuses
	s.Contains([]interfaces.CheckStatus{
		interfaces.CheckStatusPass,
		interfaces.CheckStatusWarning,
		interfaces.CheckStatusFail,
		interfaces.CheckStatusError,
	}, result.Status)
}

// TestCheckWithCanceledContext tests Check with canceled context
func (s *CompositeIndividualTestSuite) TestCheckWithCanceledContext() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := s.composite.Check(ctx)

	s.Error(err)
	s.Equal(context.Canceled, err)
	s.Equal(interfaces.CheckResult{}, result)
}

// TestSetConfigMethods tests the various Set*Config methods
func (s *CompositeIndividualTestSuite) TestSetConfigMethods() {
	// Test SetStyleConfig
	styleConfig := StyleCheckerOptions{
		EnableGofmt:          false,
		EnableGolangciLint:   false,
		EnableImportCheck:    false,
		EnableCopyrightCheck: false,
	}
	s.composite.SetStyleConfig(styleConfig)

	// Test SetTestConfig - using struct from test.go
	testConfig := TestRunnerOptions{
		EnableCoverage:      false,
		EnableBenchmarks:    false,
		CoverageThreshold:   50.0,
		EnableRaceDetection: false,
	}
	s.composite.SetTestConfig(testConfig)

	// Test SetSecurityConfig
	securityConfig := SecurityConfig{
		GosecEnabled:      false,
		NancyEnabled:      false,
		TrivyEnabled:      false,
		SeverityThreshold: "medium",
	}
	s.composite.SetSecurityConfig(securityConfig)

	// Test SetConfigValidationConfig
	configConfig := ConfigValidationConfig{
		YAMLValidation:  false,
		JSONValidation:  false,
		ProtoValidation: false,
	}
	s.composite.SetConfigValidationConfig(configConfig)

	// These methods don't return anything, but we can verify they don't panic
	s.NotNil(s.composite)
}

// TestCheckCoverage tests the CheckCoverage method
func (s *CompositeIndividualTestSuite) TestCheckCoverage() {
	result := s.composite.CheckCoverage()
	s.NotNil(result)
	// Result should have some basic structure
	s.GreaterOrEqual(result.Coverage, 0.0)
}

// TestCheckInterfaces tests the CheckInterfaces method
func (s *CompositeIndividualTestSuite) TestCheckInterfaces() {
	result := s.composite.CheckInterfaces()
	s.NotNil(result)
	s.Equal("interfaces", result.Name)
}

// TestCheckImports tests the CheckImports method
func (s *CompositeIndividualTestSuite) TestCheckImports() {
	result := s.composite.CheckImports()
	s.NotNil(result)
	s.Equal("imports", result.Name)
}

// TestCheckCopyright tests the CheckCopyright method
func (s *CompositeIndividualTestSuite) TestCheckCopyright() {
	result := s.composite.CheckCopyright()
	s.NotNil(result)
	s.Equal("copyright", result.Name)
}

// TestFixIssues tests the FixIssues method
func (s *CompositeIndividualTestSuite) TestFixIssues() {
	// Create a Go file with some content to fix
	testFile := `package main
import "fmt"
func main(){fmt.Println("Hello")}
`
	err := os.WriteFile(s.tempDir+"/fix_test.go", []byte(testFile), 0644)
	s.Require().NoError(err)

	err = s.composite.FixIssues()
	s.NoError(err) // Should not error even if no actual fixes are implemented
}

// TestGenerateReports tests the GenerateReports method
func (s *CompositeIndividualTestSuite) TestGenerateReports() {
	outputDir := s.tempDir + "/reports"
	err := s.composite.GenerateReports(outputDir)
	s.NoError(err) // Should not error even if no actual reports are generated
}

// Note: MockLogger is already defined in checker_test.go
