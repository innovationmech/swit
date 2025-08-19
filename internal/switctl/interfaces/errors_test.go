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

package interfaces

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ErrorTestSuite tests error types and functions
type ErrorTestSuite struct {
	suite.Suite
}

func TestErrorTestSuite(t *testing.T) {
	suite.Run(t, new(ErrorTestSuite))
}

func (s *ErrorTestSuite) TestSwitctlError_Error() {
	tests := []struct {
		name     string
		err      *SwitctlError
		expected string
	}{
		{
			name: "error with details",
			err: &SwitctlError{
				Code:    ErrCodeInvalidInput,
				Message: "Invalid input provided",
				Details: "Missing required field",
			},
			expected: "INVALID_INPUT: Invalid input provided (Missing required field)",
		},
		{
			name: "error without details",
			err: &SwitctlError{
				Code:    ErrCodeFileNotFound,
				Message: "File not found",
			},
			expected: "FILE_NOT_FOUND: File not found",
		},
		{
			name: "error with empty details",
			err: &SwitctlError{
				Code:    ErrCodeInternalError,
				Message: "Internal error occurred",
				Details: "",
			},
			expected: "INTERNAL_ERROR: Internal error occurred",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, tt.err.Error())
		})
	}
}

func (s *ErrorTestSuite) TestSwitctlError_Unwrap() {
	cause := errors.New("original error")

	tests := []struct {
		name     string
		err      *SwitctlError
		expected error
	}{
		{
			name: "error with cause",
			err: &SwitctlError{
				Code:    ErrCodeInternalError,
				Message: "Internal error",
				Cause:   cause,
			},
			expected: cause,
		},
		{
			name: "error without cause",
			err: &SwitctlError{
				Code:    ErrCodeInvalidInput,
				Message: "Invalid input",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, tt.err.Unwrap())
		})
	}
}

func (s *ErrorTestSuite) TestNewInvalidInputError() {
	tests := []struct {
		name            string
		message         string
		details         []string
		expectedCode    string
		expectedMessage string
		expectedDetails string
	}{
		{
			name:            "with details",
			message:         "Invalid service name",
			details:         []string{"Service name must be alphanumeric"},
			expectedCode:    ErrCodeInvalidInput,
			expectedMessage: "Invalid service name",
			expectedDetails: "Service name must be alphanumeric",
		},
		{
			name:            "without details",
			message:         "Missing required field",
			details:         []string{},
			expectedCode:    ErrCodeInvalidInput,
			expectedMessage: "Missing required field",
			expectedDetails: "",
		},
		{
			name:            "nil details",
			message:         "Invalid format",
			details:         nil,
			expectedCode:    ErrCodeInvalidInput,
			expectedMessage: "Invalid format",
			expectedDetails: "",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			err := NewInvalidInputError(tt.message, tt.details...)
			assert.Equal(s.T(), tt.expectedCode, err.Code)
			assert.Equal(s.T(), tt.expectedMessage, err.Message)
			assert.Equal(s.T(), tt.expectedDetails, err.Details)
			assert.Nil(s.T(), err.Cause)
		})
	}
}

func (s *ErrorTestSuite) TestNewFileNotFoundError() {
	path := "/path/to/missing/file.txt"
	err := NewFileNotFoundError(path)

	assert.Equal(s.T(), ErrCodeFileNotFound, err.Code)
	assert.Equal(s.T(), "File not found", err.Message)
	assert.Equal(s.T(), path, err.Details)
	assert.Nil(s.T(), err.Cause)
}

func (s *ErrorTestSuite) TestNewPermissionDeniedError() {
	path := "/restricted/path"
	err := NewPermissionDeniedError(path)

	assert.Equal(s.T(), ErrCodePermissionDenied, err.Code)
	assert.Equal(s.T(), "Permission denied", err.Message)
	assert.Equal(s.T(), path, err.Details)
	assert.Nil(s.T(), err.Cause)
}

func (s *ErrorTestSuite) TestNewTemplateError() {
	cause := errors.New("template parse error")
	message := "Failed to parse template"
	err := NewTemplateError(message, cause)

	assert.Equal(s.T(), ErrCodeTemplateRenderError, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), cause, err.Cause)
}

func (s *ErrorTestSuite) TestNewGenerationError() {
	message := "Code generation failed"
	details := "Missing template file"
	err := NewGenerationError(message, details)

	assert.Equal(s.T(), ErrCodeGenerationFailed, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), details, err.Details)
}

func (s *ErrorTestSuite) TestNewValidationError() {
	message := "Validation failed"
	details := "Required field missing"
	err := NewValidationError(message, details)

	assert.Equal(s.T(), ErrCodeValidationFailed, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), details, err.Details)
}

func (s *ErrorTestSuite) TestNewPluginError() {
	code := ErrCodePluginLoadError
	message := "Plugin load failed"
	cause := errors.New("file not found")
	err := NewPluginError(code, message, cause)

	assert.Equal(s.T(), code, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), cause, err.Cause)
}

func (s *ErrorTestSuite) TestNewConfigError() {
	code := ErrCodeConfigParseError
	message := "Config parse failed"
	cause := errors.New("yaml parse error")
	err := NewConfigError(code, message, cause)

	assert.Equal(s.T(), code, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), cause, err.Cause)
}

func (s *ErrorTestSuite) TestNewInternalError() {
	message := "Internal system error"
	cause := errors.New("database connection failed")
	err := NewInternalError(message, cause)

	assert.Equal(s.T(), ErrCodeInternalError, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), cause, err.Cause)
}

func (s *ErrorTestSuite) TestNewNotImplementedError() {
	feature := "advanced templating"
	err := NewNotImplementedError(feature)

	assert.Equal(s.T(), ErrCodeNotImplemented, err.Code)
	assert.Equal(s.T(), "Feature not implemented", err.Message)
	assert.Equal(s.T(), feature, err.Details)
	assert.Nil(s.T(), err.Cause)
}

func (s *ErrorTestSuite) TestErrorSeverity_String() {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected string
	}{
		{"info severity", SeverityInfo, "INFO"},
		{"warning severity", SeverityWarning, "WARNING"},
		{"error severity", SeverityError, "ERROR"},
		{"fatal severity", SeverityFatal, "FATAL"},
		{"unknown severity", ErrorSeverity(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, tt.severity.String())
		})
	}
}

func (s *ErrorTestSuite) TestNewDetailedError() {
	code := ErrCodeValidationFailed
	message := "Validation error"
	severity := SeverityError

	err := NewDetailedError(code, message, severity)

	assert.Equal(s.T(), code, err.Code)
	assert.Equal(s.T(), message, err.Message)
	assert.Equal(s.T(), severity, err.Severity)
	assert.NotNil(s.T(), err.Context)
	assert.Empty(s.T(), err.Context)
}

func (s *ErrorTestSuite) TestDetailedError_Chaining() {
	cause := errors.New("underlying error")
	err := NewDetailedError(ErrCodeInternalError, "Internal error", SeverityError).
		WithDetails("Additional details").
		WithCause(cause).
		WithContext("component", "user-service").
		WithContext("operation", "create-user").
		WithSuggestion("Check input parameters").
		WithHelpURL("https://docs.example.com/errors/internal")

	assert.Equal(s.T(), "Additional details", err.Details)
	assert.Equal(s.T(), cause, err.Cause)
	assert.Equal(s.T(), "user-service", err.Context["component"])
	assert.Equal(s.T(), "create-user", err.Context["operation"])
	assert.Equal(s.T(), "Check input parameters", err.Suggestion)
	assert.Equal(s.T(), "https://docs.example.com/errors/internal", err.HelpURL)
}

func (s *ErrorTestSuite) TestDetailedError_WithContext_NilMap() {
	err := &DetailedError{
		SwitctlError: &SwitctlError{
			Code:    ErrCodeInternalError,
			Message: "Test error",
		},
		Context: nil,
	}

	result := err.WithContext("key", "value")

	assert.NotNil(s.T(), result.Context)
	assert.Equal(s.T(), "value", result.Context["key"])
}

func (s *ErrorTestSuite) TestIsRecoverable() {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "invalid input error - recoverable",
			err:      NewInvalidInputError("Invalid input"),
			expected: true,
		},
		{
			name:     "missing argument error - recoverable",
			err:      &SwitctlError{Code: ErrCodeMissingArgument, Message: "Missing arg"},
			expected: true,
		},
		{
			name:     "invalid format error - recoverable",
			err:      &SwitctlError{Code: ErrCodeInvalidFormat, Message: "Invalid format"},
			expected: true,
		},
		{
			name:     "file not found error - recoverable",
			err:      NewFileNotFoundError("/path/to/file"),
			expected: true,
		},
		{
			name:     "directory not found error - recoverable",
			err:      &SwitctlError{Code: ErrCodeDirectoryNotFound, Message: "Dir not found"},
			expected: true,
		},
		{
			name:     "config parse error - recoverable",
			err:      &SwitctlError{Code: ErrCodeConfigParseError, Message: "Parse error"},
			expected: true,
		},
		{
			name:     "config validation error - recoverable",
			err:      &SwitctlError{Code: ErrCodeConfigValidation, Message: "Validation error"},
			expected: true,
		},
		{
			name:     "internal error - not recoverable",
			err:      NewInternalError("Internal error", nil),
			expected: false,
		},
		{
			name:     "generation failed error - not recoverable",
			err:      &SwitctlError{Code: ErrCodeGenerationFailed, Message: "Generation failed"},
			expected: false,
		},
		{
			name:     "non-switctl error - not recoverable",
			err:      errors.New("regular error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, IsRecoverable(tt.err))
		})
	}
}

func (s *ErrorTestSuite) TestGetErrorCode() {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "switctl error",
			err:      NewInvalidInputError("Invalid input"),
			expected: ErrCodeInvalidInput,
		},
		{
			name:     "detailed error",
			err:      NewDetailedError(ErrCodeValidationFailed, "Validation failed", SeverityError),
			expected: ErrCodeValidationFailed,
		},
		{
			name:     "regular error",
			err:      errors.New("regular error"),
			expected: ErrCodeInternalError,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: ErrCodeInternalError,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			assert.Equal(s.T(), tt.expected, GetErrorCode(tt.err))
		})
	}
}

// Benchmarks for performance testing
func BenchmarkSwitctlError_Error(b *testing.B) {
	err := &SwitctlError{
		Code:    ErrCodeInvalidInput,
		Message: "Invalid input provided",
		Details: "Missing required field",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkNewInvalidInputError(b *testing.B) {
	message := "Invalid input"
	details := "Missing field"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewInvalidInputError(message, details)
	}
}

func BenchmarkIsRecoverable(b *testing.B) {
	err := NewInvalidInputError("Invalid input")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsRecoverable(err)
	}
}

func BenchmarkGetErrorCode(b *testing.B) {
	err := NewInvalidInputError("Invalid input")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetErrorCode(err)
	}
}

// Edge case tests
func (s *ErrorTestSuite) TestSwitctlError_ErrorWithNilFields() {
	err := &SwitctlError{}
	expected := ": "
	assert.Equal(s.T(), expected, err.Error())
}

func (s *ErrorTestSuite) TestDetailedError_Inheritance() {
	detailedErr := NewDetailedError(ErrCodeInternalError, "Test error", SeverityError)

	// Test that DetailedError implements the error interface
	var genericError error = detailedErr
	assert.Contains(s.T(), genericError.Error(), "INTERNAL_ERROR")

	// Test unwrapping
	cause := errors.New("underlying cause")
	detailedErr.WithCause(cause)
	assert.Equal(s.T(), cause, errors.Unwrap(detailedErr))
}

func (s *ErrorTestSuite) TestErrorConstants() {
	// Test that all error code constants are defined and not empty
	errorCodes := []string{
		ErrCodeInvalidInput, ErrCodeMissingArgument, ErrCodeInvalidFormat,
		ErrCodeInvalidServiceName, ErrCodeInvalidPort,
		ErrCodeFileNotFound, ErrCodeDirectoryNotFound, ErrCodePermissionDenied,
		ErrCodeFileExists, ErrCodeReadError, ErrCodeWriteError,
		ErrCodeTemplateNotFound, ErrCodeTemplateParseError, ErrCodeTemplateRenderError,
		ErrCodeTemplateLoadError,
		ErrCodeGenerationFailed, ErrCodeServiceExists, ErrCodeAPIExists,
		ErrCodeModelExists, ErrCodeInvalidConfig,
		ErrCodeValidationFailed, ErrCodeTestsFailed, ErrCodeCoverageTooLow,
		ErrCodeLintErrors, ErrCodeSecurityIssues,
		ErrCodePluginNotFound, ErrCodePluginLoadError, ErrCodePluginInitError,
		ErrCodePluginExecError,
		ErrCodeConfigNotFound, ErrCodeConfigParseError, ErrCodeConfigValidation,
		ErrCodeConfigSaveError,
		ErrCodeDependencyError, ErrCodeVersionConflict, ErrCodeUpdateFailed,
		ErrCodeAuditFailed,
		ErrCodeNetworkError, ErrCodeDownloadFailed, ErrCodeUploadFailed,
		ErrCodeTimeoutError,
		ErrCodeNotProjectRoot, ErrCodeProjectExists, ErrCodeInvalidProject,
		ErrCodeMigrationFailed,
		ErrCodeInternalError, ErrCodeNotImplemented, ErrCodeResourceExhausted,
		ErrCodeCancelled,
	}

	for _, code := range errorCodes {
		s.Run("error_code_"+code, func() {
			assert.NotEmpty(s.T(), code, "Error code should not be empty")
			assert.True(s.T(), len(code) > 0, "Error code should have length > 0")
		})
	}
}
