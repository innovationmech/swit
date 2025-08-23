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
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/innovationmech/swit/internal/switctl/interfaces"
)

// SecurityBasicTestSuite tests for basic SecurityChecker methods
type SecurityBasicTestSuite struct {
	suite.Suite
	checker *SecurityChecker
	logger  interfaces.Logger
}

func (s *SecurityBasicTestSuite) SetupTest() {
	s.logger = &MockLogger{}
	s.checker = NewSecurityChecker("/tmp", s.logger)
}

func TestSecurityBasicTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityBasicTestSuite))
}

// TestName tests the Name method
func (s *SecurityBasicTestSuite) TestName() {
	name := s.checker.Name()
	s.Equal("security", name)
}

// TestDescription tests the Description method
func (s *SecurityBasicTestSuite) TestDescription() {
	description := s.checker.Description()
	s.Contains(description, "security")
	s.Contains(description, "vulnerability")
}

// TestClose tests the Close method
func (s *SecurityBasicTestSuite) TestClose() {
	err := s.checker.Close()
	s.NoError(err)
}

// TestGetSuggestionForRule tests the getSuggestionForRule function
func (s *SecurityBasicTestSuite) TestGetSuggestionForRule() {
	suggestion := s.checker.getSuggestionForRule("G101")
	s.Contains(suggestion, "environment variables")

	suggestion = s.checker.getSuggestionForRule("G201")
	s.Contains(suggestion, "parameterized queries")

	suggestion = s.checker.getSuggestionForRule("G401")
	s.Contains(suggestion, "cryptographic")

	suggestion = s.checker.getSuggestionForRule("UNKNOWN_RULE")
	s.Equal("Review and fix the security issue according to best practices", suggestion)
}

// TestGetSeverityLevel tests the getSeverityLevel function
func (s *SecurityBasicTestSuite) TestGetSeverityLevel() {
	level := s.checker.getSeverityLevel("critical")
	s.Equal(4, level)

	level = s.checker.getSeverityLevel("high")
	s.Equal(3, level)

	level = s.checker.getSeverityLevel("medium")
	s.Equal(2, level)

	level = s.checker.getSeverityLevel("low")
	s.Equal(1, level)

	level = s.checker.getSeverityLevel("unknown")
	s.Equal(0, level)
}
