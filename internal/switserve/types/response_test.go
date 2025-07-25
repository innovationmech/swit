// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"net/http"
	"reflect"
	"testing"
)

func TestNewResponseStatus(t *testing.T) {
	tests := []struct {
		name        string
		code        int
		message     string
		wantSuccess bool
	}{
		{"success 200", 200, "OK", true},
		{"success 201", 201, "Created", true},
		{"client error 400", 400, "Bad Request", false},
		{"server error 500", 500, "Internal Server Error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewResponseStatus(tt.code, tt.message)
			if got.Code != tt.code {
				t.Errorf("NewResponseStatus().Code = %v, want %v", got.Code, tt.code)
			}
			if got.Message != tt.message {
				t.Errorf("NewResponseStatus().Message = %v, want %v", got.Message, tt.message)
			}
			if got.Success != tt.wantSuccess {
				t.Errorf("NewResponseStatus().Success = %v, want %v", got.Success, tt.wantSuccess)
			}
		})
	}
}

func TestNewSuccessStatus(t *testing.T) {
	message := "Operation successful"
	got := NewSuccessStatus(message)

	if got.Code != http.StatusOK {
		t.Errorf("NewSuccessStatus().Code = %v, want %v", got.Code, http.StatusOK)
	}
	if got.Message != message {
		t.Errorf("NewSuccessStatus().Message = %v, want %v", got.Message, message)
	}
	if !got.Success {
		t.Errorf("NewSuccessStatus().Success = %v, want %v", got.Success, true)
	}
}

func TestNewErrorStatus(t *testing.T) {
	code := http.StatusBadRequest
	message := "Invalid input"
	got := NewErrorStatus(code, message)

	if got.Code != code {
		t.Errorf("NewErrorStatus().Code = %v, want %v", got.Code, code)
	}
	if got.Message != message {
		t.Errorf("NewErrorStatus().Message = %v, want %v", got.Message, message)
	}
	if got.Success {
		t.Errorf("NewErrorStatus().Success = %v, want %v", got.Success, false)
	}
}

func TestNewSuccessResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	got := NewSuccessResponse(data)

	if got.Status == nil {
		t.Error("NewSuccessResponse().Status is nil")
	}
	if got.Status.Code != http.StatusOK {
		t.Errorf("NewSuccessResponse().Status.Code = %v, want %v", got.Status.Code, http.StatusOK)
	}
	if !got.Status.Success {
		t.Errorf("NewSuccessResponse().Status.Success = %v, want %v", got.Status.Success, true)
	}
	if !reflect.DeepEqual(got.Data, data) {
		t.Errorf("NewSuccessResponse().Data = %v, want %v", got.Data, data)
	}
	if got.Error != nil {
		t.Errorf("NewSuccessResponse().Error = %v, want nil", got.Error)
	}
}

func TestNewErrorResponse(t *testing.T) {
	code := http.StatusBadRequest
	errorCode := "VALIDATION_ERROR"
	message := "Invalid input"
	details := "Field 'name' is required"

	got := NewErrorResponse(code, errorCode, message, details)

	if got.Status == nil {
		t.Error("NewErrorResponse().Status is nil")
	}
	if got.Status.Code != code {
		t.Errorf("NewErrorResponse().Status.Code = %v, want %v", got.Status.Code, code)
	}
	if got.Status.Success {
		t.Errorf("NewErrorResponse().Status.Success = %v, want %v", got.Status.Success, false)
	}
	if got.Error == nil {
		t.Error("NewErrorResponse().Error is nil")
	}
	if got.Error.Code != errorCode {
		t.Errorf("NewErrorResponse().Error.Code = %v, want %v", got.Error.Code, errorCode)
	}
	if got.Error.Message != message {
		t.Errorf("NewErrorResponse().Error.Message = %v, want %v", got.Error.Message, message)
	}
	if got.Error.Details != details {
		t.Errorf("NewErrorResponse().Error.Details = %v, want %v", got.Error.Details, details)
	}
}
