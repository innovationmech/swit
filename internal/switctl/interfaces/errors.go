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
	"fmt"
)

// SwitctlError represents a switctl-specific error with detailed information.
type SwitctlError struct {
	Code    string `json:"code"`              // Error code
	Message string `json:"message"`           // Human-readable message
	Details string `json:"details,omitempty"` // Additional details
	Cause   error  `json:"-"`                 // Underlying cause
}

// Error implements the error interface.
func (e *SwitctlError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error.
func (e *SwitctlError) Unwrap() error {
	return e.Cause
}

// Error code constants for different error categories.
const (
	// Input validation errors
	ErrCodeInvalidInput       = "INVALID_INPUT"
	ErrCodeMissingArgument    = "MISSING_ARGUMENT"
	ErrCodeInvalidFormat      = "INVALID_FORMAT"
	ErrCodeInvalidServiceName = "INVALID_SERVICE_NAME"
	ErrCodeInvalidPort        = "INVALID_PORT"

	// File system errors
	ErrCodeFileNotFound      = "FILE_NOT_FOUND"
	ErrCodeDirectoryNotFound = "DIRECTORY_NOT_FOUND"
	ErrCodePermissionDenied  = "PERMISSION_DENIED"
	ErrCodeFileExists        = "FILE_EXISTS"
	ErrCodeReadError         = "READ_ERROR"
	ErrCodeWriteError        = "WRITE_ERROR"

	// Template errors
	ErrCodeTemplateNotFound    = "TEMPLATE_NOT_FOUND"
	ErrCodeTemplateParseError  = "TEMPLATE_PARSE_ERROR"
	ErrCodeTemplateRenderError = "TEMPLATE_RENDER_ERROR"
	ErrCodeTemplateLoadError   = "TEMPLATE_LOAD_ERROR"

	// Generation errors
	ErrCodeGenerationFailed = "GENERATION_FAILED"
	ErrCodeServiceExists    = "SERVICE_EXISTS"
	ErrCodeAPIExists        = "API_EXISTS"
	ErrCodeModelExists      = "MODEL_EXISTS"
	ErrCodeInvalidConfig    = "INVALID_CONFIG"

	// Validation errors
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeTestsFailed      = "TESTS_FAILED"
	ErrCodeCoverageTooLow   = "COVERAGE_TOO_LOW"
	ErrCodeLintErrors       = "LINT_ERRORS"
	ErrCodeSecurityIssues   = "SECURITY_ISSUES"

	// Plugin errors
	ErrCodePluginNotFound  = "PLUGIN_NOT_FOUND"
	ErrCodePluginLoadError = "PLUGIN_LOAD_ERROR"
	ErrCodePluginInitError = "PLUGIN_INIT_ERROR"
	ErrCodePluginExecError = "PLUGIN_EXEC_ERROR"

	// Configuration errors
	ErrCodeConfigNotFound   = "CONFIG_NOT_FOUND"
	ErrCodeConfigParseError = "CONFIG_PARSE_ERROR"
	ErrCodeConfigValidation = "CONFIG_VALIDATION"
	ErrCodeConfigSaveError  = "CONFIG_SAVE_ERROR"

	// Dependency errors
	ErrCodeDependencyError = "DEPENDENCY_ERROR"
	ErrCodeVersionConflict = "VERSION_CONFLICT"
	ErrCodeUpdateFailed    = "UPDATE_FAILED"
	ErrCodeAuditFailed     = "AUDIT_FAILED"

	// Network errors
	ErrCodeNetworkError   = "NETWORK_ERROR"
	ErrCodeDownloadFailed = "DOWNLOAD_FAILED"
	ErrCodeUploadFailed   = "UPLOAD_FAILED"
	ErrCodeTimeoutError   = "TIMEOUT_ERROR"

	// Project errors
	ErrCodeNotProjectRoot  = "NOT_PROJECT_ROOT"
	ErrCodeProjectExists   = "PROJECT_EXISTS"
	ErrCodeInvalidProject  = "INVALID_PROJECT"
	ErrCodeMigrationFailed = "MIGRATION_FAILED"

	// Internal errors
	ErrCodeInternalError     = "INTERNAL_ERROR"
	ErrCodeNotImplemented    = "NOT_IMPLEMENTED"
	ErrCodeResourceExhausted = "RESOURCE_EXHAUSTED"
	ErrCodeCancelled         = "CANCELLED"
)

// Error constructor functions for common error types.

// NewInvalidInputError creates an error for invalid input.
func NewInvalidInputError(message string, details ...string) *SwitctlError {
	detail := ""
	if len(details) > 0 {
		detail = details[0]
	}
	return &SwitctlError{
		Code:    ErrCodeInvalidInput,
		Message: message,
		Details: detail,
	}
}

// NewFileNotFoundError creates an error for file not found.
func NewFileNotFoundError(path string) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodeFileNotFound,
		Message: "File not found",
		Details: path,
	}
}

// NewPermissionDeniedError creates an error for permission denied.
func NewPermissionDeniedError(path string) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodePermissionDenied,
		Message: "Permission denied",
		Details: path,
	}
}

// NewTemplateError creates an error for template operations.
func NewTemplateError(message string, cause error) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodeTemplateRenderError,
		Message: message,
		Cause:   cause,
	}
}

// NewGenerationError creates an error for code generation.
func NewGenerationError(message string, details string) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodeGenerationFailed,
		Message: message,
		Details: details,
	}
}

// NewValidationError creates an error for validation failures.
func NewValidationError(message string, details string) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodeValidationFailed,
		Message: message,
		Details: details,
	}
}

// NewPluginError creates an error for plugin operations.
func NewPluginError(code, message string, cause error) *SwitctlError {
	return &SwitctlError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewConfigError creates an error for configuration operations.
func NewConfigError(code, message string, cause error) *SwitctlError {
	return &SwitctlError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewInternalError creates an error for internal system errors.
func NewInternalError(message string, cause error) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodeInternalError,
		Message: message,
		Cause:   cause,
	}
}

// NewNotImplementedError creates an error for unimplemented features.
func NewNotImplementedError(feature string) *SwitctlError {
	return &SwitctlError{
		Code:    ErrCodeNotImplemented,
		Message: "Feature not implemented",
		Details: feature,
	}
}

// ErrorSeverity represents the severity level of an error.
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityFatal
)

// String returns the string representation of the error severity.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// DetailedError provides additional context and severity information.
type DetailedError struct {
	*SwitctlError
	Severity   ErrorSeverity     `json:"severity"`
	Context    map[string]string `json:"context,omitempty"`
	Suggestion string            `json:"suggestion,omitempty"`
	HelpURL    string            `json:"help_url,omitempty"`
}

// NewDetailedError creates a detailed error with additional context.
func NewDetailedError(code, message string, severity ErrorSeverity) *DetailedError {
	return &DetailedError{
		SwitctlError: &SwitctlError{
			Code:    code,
			Message: message,
		},
		Severity: severity,
		Context:  make(map[string]string),
	}
}

// WithDetails adds details to the error.
func (e *DetailedError) WithDetails(details string) *DetailedError {
	e.Details = details
	return e
}

// WithCause sets the underlying cause.
func (e *DetailedError) WithCause(cause error) *DetailedError {
	e.Cause = cause
	return e
}

// WithContext adds context information.
func (e *DetailedError) WithContext(key, value string) *DetailedError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion for fixing the error.
func (e *DetailedError) WithSuggestion(suggestion string) *DetailedError {
	e.Suggestion = suggestion
	return e
}

// WithHelpURL adds a help URL for more information.
func (e *DetailedError) WithHelpURL(url string) *DetailedError {
	e.HelpURL = url
	return e
}

// IsRecoverable checks if an error is recoverable.
func IsRecoverable(err error) bool {
	if switctlErr, ok := err.(*SwitctlError); ok {
		switch switctlErr.Code {
		case ErrCodeInvalidInput, ErrCodeMissingArgument, ErrCodeInvalidFormat:
			return true
		case ErrCodeFileNotFound, ErrCodeDirectoryNotFound:
			return true
		case ErrCodeConfigParseError, ErrCodeConfigValidation:
			return true
		default:
			return false
		}
	}
	return false
}

// GetErrorCode extracts the error code from an error.
func GetErrorCode(err error) string {
	if switctlErr, ok := err.(*SwitctlError); ok {
		return switctlErr.Code
	}
	if detailedErr, ok := err.(*DetailedError); ok {
		return detailedErr.Code
	}
	return ErrCodeInternalError
}
