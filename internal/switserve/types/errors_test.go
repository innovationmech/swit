// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"errors"
	"net/http"
	"testing"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ServiceError
		want string
	}{
		{
			"without details",
			&ServiceError{Code: "TEST_ERROR", Message: "Test message"},
			"TEST_ERROR: Test message",
		},
		{
			"with details",
			&ServiceError{Code: "TEST_ERROR", Message: "Test message", Details: "Additional info"},
			"TEST_ERROR: Test message (Additional info)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ServiceError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &ServiceError{
		Code:    "TEST_ERROR",
		Message: "Test message",
		Cause:   cause,
	}

	if got := err.Unwrap(); got != cause {
		t.Errorf("ServiceError.Unwrap() = %v, want %v", got, cause)
	}
}

func TestNewServiceError(t *testing.T) {
	code := "TEST_ERROR"
	message := "Test message"
	httpStatus := http.StatusBadRequest

	got := NewServiceError(code, message, httpStatus)

	if got.Code != code {
		t.Errorf("NewServiceError().Code = %v, want %v", got.Code, code)
	}
	if got.Message != message {
		t.Errorf("NewServiceError().Message = %v, want %v", got.Message, message)
	}
	if got.HTTPStatus != httpStatus {
		t.Errorf("NewServiceError().HTTPStatus = %v, want %v", got.HTTPStatus, httpStatus)
	}
	if got.Details != "" {
		t.Errorf("NewServiceError().Details = %v, want empty string", got.Details)
	}
	if got.Cause != nil {
		t.Errorf("NewServiceError().Cause = %v, want nil", got.Cause)
	}
}

func TestNewServiceErrorWithDetails(t *testing.T) {
	code := "TEST_ERROR"
	message := "Test message"
	details := "Additional details"
	httpStatus := http.StatusBadRequest

	got := NewServiceErrorWithDetails(code, message, details, httpStatus)

	if got.Code != code {
		t.Errorf("NewServiceErrorWithDetails().Code = %v, want %v", got.Code, code)
	}
	if got.Message != message {
		t.Errorf("NewServiceErrorWithDetails().Message = %v, want %v", got.Message, message)
	}
	if got.Details != details {
		t.Errorf("NewServiceErrorWithDetails().Details = %v, want %v", got.Details, details)
	}
	if got.HTTPStatus != httpStatus {
		t.Errorf("NewServiceErrorWithDetails().HTTPStatus = %v, want %v", got.HTTPStatus, httpStatus)
	}
}

func TestNewServiceErrorWithCause(t *testing.T) {
	code := "TEST_ERROR"
	message := "Test message"
	httpStatus := http.StatusInternalServerError
	cause := errors.New("underlying error")

	got := NewServiceErrorWithCause(code, message, httpStatus, cause)

	if got.Code != code {
		t.Errorf("NewServiceErrorWithCause().Code = %v, want %v", got.Code, code)
	}
	if got.Message != message {
		t.Errorf("NewServiceErrorWithCause().Message = %v, want %v", got.Message, message)
	}
	if got.HTTPStatus != httpStatus {
		t.Errorf("NewServiceErrorWithCause().HTTPStatus = %v, want %v", got.HTTPStatus, httpStatus)
	}
	if got.Cause != cause {
		t.Errorf("NewServiceErrorWithCause().Cause = %v, want %v", got.Cause, cause)
	}
}

func TestErrorFactoryFunctions(t *testing.T) {
	tests := []struct {
		name       string
		factory    func(string) *ServiceError
		message    string
		wantCode   string
		wantStatus int
	}{
		{"ErrValidation", ErrValidation, "Invalid input", ErrCodeValidation, http.StatusBadRequest},
		{"ErrNotFound", func(msg string) *ServiceError { return ErrNotFound("user") }, "user not found", ErrCodeNotFound, http.StatusNotFound},
		{"ErrUnauthorized", ErrUnauthorized, "Access denied", ErrCodeUnauthorized, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, "Forbidden", ErrCodeForbidden, http.StatusForbidden},
		{"ErrConflict", ErrConflict, "Resource exists", ErrCodeConflict, http.StatusConflict},
		{"ErrInternal", ErrInternal, "Internal error", ErrCodeInternal, http.StatusInternalServerError},
		{"ErrBadRequest", ErrBadRequest, "Bad request", ErrCodeBadRequest, http.StatusBadRequest},
		{"ErrServiceUnavailable", ErrServiceUnavailable, "Service down", ErrCodeServiceUnavailable, http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.factory(tt.message)
			if got.Code != tt.wantCode {
				t.Errorf("%s().Code = %v, want %v", tt.name, got.Code, tt.wantCode)
			}
			if got.HTTPStatus != tt.wantStatus {
				t.Errorf("%s().HTTPStatus = %v, want %v", tt.name, got.HTTPStatus, tt.wantStatus)
			}
		})
	}
}

func TestErrInternalWithCause(t *testing.T) {
	message := "Database error"
	cause := errors.New("connection failed")

	got := ErrInternalWithCause(message, cause)

	if got.Code != ErrCodeInternal {
		t.Errorf("ErrInternalWithCause().Code = %v, want %v", got.Code, ErrCodeInternal)
	}
	if got.Message != message {
		t.Errorf("ErrInternalWithCause().Message = %v, want %v", got.Message, message)
	}
	if got.HTTPStatus != http.StatusInternalServerError {
		t.Errorf("ErrInternalWithCause().HTTPStatus = %v, want %v", got.HTTPStatus, http.StatusInternalServerError)
	}
	if got.Cause != cause {
		t.Errorf("ErrInternalWithCause().Cause = %v, want %v", got.Cause, cause)
	}
}

func TestIsServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"service error", ErrValidation("test"), true},
		{"wrapped service error", errors.New("wrapped"), false},
		{"nil error", nil, false},
		{"regular error", errors.New("regular"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsServiceError(tt.err); got != tt.want {
				t.Errorf("IsServiceError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetServiceError(t *testing.T) {
	serviceErr := ErrValidation("test")
	regularErr := errors.New("regular")

	tests := []struct {
		name string
		err  error
		want *ServiceError
	}{
		{"service error", serviceErr, serviceErr},
		{"regular error", regularErr, nil},
		{"nil error", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetServiceError(tt.err)
			if got != tt.want {
				t.Errorf("GetServiceError() = %v, want %v", got, tt.want)
			}
		})
	}
}
