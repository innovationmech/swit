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

package types

import (
	"errors"
	"fmt"
	"net/http"
)

// Common error codes
const (
	ErrCodeValidation         = "VALIDATION_ERROR"
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ServiceError represents a service-level error with HTTP status code
type ServiceError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	HTTPStatus int    `json:"-"`
	Cause      error  `json:"-"`
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// NewServiceError creates a new service error
func NewServiceError(code, message string, httpStatus int) *ServiceError {
	return &ServiceError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// NewServiceErrorWithDetails creates a new service error with details
func NewServiceErrorWithDetails(code, message, details string, httpStatus int) *ServiceError {
	return &ServiceError{
		Code:       code,
		Message:    message,
		Details:    details,
		HTTPStatus: httpStatus,
	}
}

// NewServiceErrorWithCause creates a new service error with underlying cause
func NewServiceErrorWithCause(code, message string, httpStatus int, cause error) *ServiceError {
	return &ServiceError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Cause:      cause,
	}
}

// Error factory functions

// ErrValidation creates a validation error
func ErrValidation(message string) *ServiceError {
	return NewServiceError(ErrCodeValidation, message, http.StatusBadRequest)
}

// ErrNotFound creates a not found error
func ErrNotFound(resource string) *ServiceError {
	return NewServiceError(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

// ErrUnauthorized creates an unauthorized error
func ErrUnauthorized(message string) *ServiceError {
	return NewServiceError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

// ErrForbidden creates a forbidden error
func ErrForbidden(message string) *ServiceError {
	return NewServiceError(ErrCodeForbidden, message, http.StatusForbidden)
}

// ErrConflict creates a conflict error
func ErrConflict(message string) *ServiceError {
	return NewServiceError(ErrCodeConflict, message, http.StatusConflict)
}

// ErrInternal creates an internal server error
func ErrInternal(message string) *ServiceError {
	return NewServiceError(ErrCodeInternal, message, http.StatusInternalServerError)
}

// ErrInternalWithCause creates an internal server error with cause
func ErrInternalWithCause(message string, cause error) *ServiceError {
	return NewServiceErrorWithCause(ErrCodeInternal, message, http.StatusInternalServerError, cause)
}

// ErrBadRequest creates a bad request error
func ErrBadRequest(message string) *ServiceError {
	return NewServiceError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

// ErrServiceUnavailable creates a service unavailable error
func ErrServiceUnavailable(message string) *ServiceError {
	return NewServiceError(ErrCodeServiceUnavailable, message, http.StatusServiceUnavailable)
}

// IsServiceError checks if an error is a ServiceError
func IsServiceError(err error) bool {
	var serviceErr *ServiceError
	return errors.As(err, &serviceErr)
}

// GetServiceError extracts ServiceError from error
func GetServiceError(err error) *ServiceError {
	var serviceErr *ServiceError
	if errors.As(err, &serviceErr) {
		return serviceErr
	}
	return nil
}
