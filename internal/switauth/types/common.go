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
	"fmt"
	"net/http"
)

// Status represents a general status with code and message
type Status struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ResponseStatus represents the status of an API response
type ResponseStatus struct {
	Success bool   `json:"success"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// Common status constants
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusPending = "pending"
)

// HTTP status code constants
const (
	StatusCodeOK                  = http.StatusOK
	StatusCodeCreated             = http.StatusCreated
	StatusCodeBadRequest          = http.StatusBadRequest
	StatusCodeUnauthorized        = http.StatusUnauthorized
	StatusCodeForbidden           = http.StatusForbidden
	StatusCodeNotFound            = http.StatusNotFound
	StatusCodeConflict            = http.StatusConflict
	StatusCodeInternalServerError = http.StatusInternalServerError
)

// Error types for authentication
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Error codes
const (
	ErrorCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrorCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrorCodeTokenInvalid       = "TOKEN_INVALID"
	ErrorCodeUserNotFound       = "USER_NOT_FOUND"
	ErrorCodeUserInactive       = "USER_INACTIVE"
	ErrorCodeInternalError      = "INTERNAL_ERROR"
	ErrorCodeValidationError    = "VALIDATION_ERROR"
	ErrorCodeInvalidRequest     = "INVALID_REQUEST"
)

// Error factory functions
func NewInvalidCredentialsError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeInvalidCredentials,
		Message: "Invalid username or password",
		Details: details,
	}
}

func NewTokenExpiredError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeTokenExpired,
		Message: "Token has expired",
		Details: details,
	}
}

func NewTokenInvalidError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeTokenInvalid,
		Message: "Token is invalid",
		Details: details,
	}
}

func NewUserNotFoundError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeUserNotFound,
		Message: "User not found",
		Details: details,
	}
}

func NewUserInactiveError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeUserInactive,
		Message: "User account is inactive",
		Details: details,
	}
}

func NewInternalError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeInternalError,
		Message: "Internal server error",
		Details: details,
	}
}

func NewValidationError(details string) *AuthError {
	return &AuthError{
		Code:    ErrorCodeValidationError,
		Message: "Validation error",
		Details: details,
	}
}

func NewAuthError(code, message string, err error) *AuthError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return &AuthError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// NewSuccessStatus creates a success response status
func NewSuccessStatus(message string) *ResponseStatus {
	return &ResponseStatus{
		Success: true,
		Code:    StatusCodeOK,
		Message: message,
	}
}

// NewErrorStatus creates an error response status
func NewErrorStatus(code int, message, error string) *ResponseStatus {
	return &ResponseStatus{
		Success: false,
		Code:    code,
		Message: message,
		Error:   error,
	}
}
