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
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// QualityCheckerTestSuite tests for QualityChecker (types.go)
type QualityCheckerTestSuite struct {
	suite.Suite
	checker *QualityChecker
	mockFS  *MockFileSystem
}

func (s *QualityCheckerTestSuite) SetupTest() {
	s.mockFS = NewMockFileSystem()
	s.checker = NewQualityChecker(s.mockFS)
}

func TestQualityCheckerTestSuite(t *testing.T) {
	suite.Run(t, new(QualityCheckerTestSuite))
}

// TestNewQualityChecker tests the constructor
func (s *QualityCheckerTestSuite) TestNewQualityChecker() {
	checker := NewQualityChecker(s.mockFS)
	s.NotNil(checker)
	s.Equal(s.mockFS, checker.fs)
	s.Equal(DefaultTimeout, checker.timeout)
	s.Equal(DefaultMaxGoroutines, checker.maxGoroutines)
	s.False(checker.enableCache)
}

// TestNewQualityCheckerWithOptions tests constructor with custom options
func (s *QualityCheckerTestSuite) TestNewQualityCheckerWithOptions() {
	options := CheckerOptions{
		Timeout:       DefaultTimeout * 2,
		MaxGoroutines: 8,
		EnableCache:   true,
	}

	checker := NewQualityCheckerWithOptions(s.mockFS, options)
	s.NotNil(checker)
	s.Equal(s.mockFS, checker.fs)
	s.Equal(options.Timeout, checker.timeout)
	s.Equal(options.MaxGoroutines, checker.maxGoroutines)
	s.Equal(options.EnableCache, checker.enableCache)
}

// TestRegisterChecker tests registering individual checkers
func (s *QualityCheckerTestSuite) TestRegisterChecker() {
	mockChecker := NewMockIndividualChecker()
	mockChecker.On("Name").Return("test-checker")

	s.checker.RegisterChecker("test-checker", mockChecker)

	// Verify checker was registered
	checkers := s.checker.ListCheckers()
	s.Contains(checkers, "test-checker")
}

// TestUnregisterChecker tests unregistering individual checkers
func (s *QualityCheckerTestSuite) TestUnregisterChecker() {
	mockChecker := NewMockIndividualChecker()
	mockChecker.On("Name").Return("test-checker")

	s.checker.RegisterChecker("test-checker", mockChecker)
	s.checker.UnregisterChecker("test-checker")

	// Verify checker was unregistered
	checkers := s.checker.ListCheckers()
	s.NotContains(checkers, "test-checker")
}

// TestListCheckers tests listing all registered checkers
func (s *QualityCheckerTestSuite) TestListCheckers() {
	mockChecker1 := NewMockIndividualChecker()
	mockChecker1.On("Name").Return("checker1")
	mockChecker2 := NewMockIndividualChecker()
	mockChecker2.On("Name").Return("checker2")

	s.checker.RegisterChecker("checker1", mockChecker1)
	s.checker.RegisterChecker("checker2", mockChecker2)

	checkers := s.checker.ListCheckers()
	s.Contains(checkers, "checker1")
	s.Contains(checkers, "checker2")
	s.Len(checkers, 2)
}

// TestRunAllChecks tests running all registered checkers
func (s *QualityCheckerTestSuite) TestRunAllChecks() {
	mockChecker := NewMockIndividualChecker()
	mockChecker.On("Name").Return("test-checker")
	mockChecker.On("Check", mock.Anything).Return(CreateTestCheckResult("test-checker", interfaces.CheckStatusPass, "Test passed"), nil)

	s.checker.RegisterChecker("test-checker", mockChecker)

	ctx := context.Background()
	summary, err := s.checker.RunAllChecks(ctx)

	s.NoError(err)
	s.NotNil(summary)
	s.Equal(1, summary.TotalChecks)
	s.Equal(1, summary.PassedChecks)
	s.Equal(0, summary.FailedChecks)
}

// TestRunSpecificCheck tests running a specific checker
func (s *QualityCheckerTestSuite) TestRunSpecificCheck() {
	mockChecker := NewMockIndividualChecker()
	mockChecker.On("Name").Return("test-checker")
	mockChecker.On("Check", mock.Anything).Return(CreateTestCheckResult("test-checker", interfaces.CheckStatusPass, "Test passed"), nil)

	s.checker.RegisterChecker("test-checker", mockChecker)

	ctx := context.Background()
	result, err := s.checker.RunSpecificCheck(ctx, "test-checker")

	s.NoError(err)
	s.NotNil(result)
	s.Equal("test-checker", result.Name)
	s.Equal(interfaces.CheckStatusPass, result.Status)
}

// TestRunSpecificCheckNotFound tests running a non-existent checker
func (s *QualityCheckerTestSuite) TestRunSpecificCheckNotFound() {
	ctx := context.Background()
	result, err := s.checker.RunSpecificCheck(ctx, "nonexistent")

	s.Error(err)
	s.Equal(interfaces.CheckResult{}, result)
}

// TestClose tests closing the quality checker
func (s *QualityCheckerTestSuite) TestClose() {
	mockChecker := NewMockIndividualChecker()
	mockChecker.On("Name").Return("test-checker").Maybe()
	mockChecker.On("Close").Return(nil)

	s.checker.RegisterChecker("test-checker", mockChecker)

	err := s.checker.Close()
	s.NoError(err)

	mockChecker.AssertExpectations(s.T())
}
