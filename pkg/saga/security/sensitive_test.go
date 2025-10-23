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

package security

import (
	"encoding/json"
	"testing"
)

func TestNewSensitiveMarker(t *testing.T) {
	marker := NewSensitiveMarker()
	if marker == nil {
		t.Fatal("NewSensitiveMarker() returned nil")
	}
}

func TestSensitiveMarker_MarkField(t *testing.T) {
	marker := NewSensitiveMarker()

	tests := []struct {
		name      string
		fieldName string
		dataType  SensitiveDataType
		wantErr   bool
	}{
		{
			name:      "mark phone field",
			fieldName: "phone",
			dataType:  SensitiveTypePhone,
			wantErr:   false,
		},
		{
			name:      "mark email field",
			fieldName: "email",
			dataType:  SensitiveTypeEmail,
			wantErr:   false,
		},
		{
			name:      "mark password field",
			fieldName: "password",
			dataType:  SensitiveTypePassword,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := marker.MarkField(tt.fieldName, tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarkField() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !marker.IsSensitive(tt.fieldName) {
				t.Errorf("Field %s should be marked as sensitive", tt.fieldName)
			}
		})
	}
}

func TestSensitiveMarker_GetSensitiveFields(t *testing.T) {
	marker := NewSensitiveMarker()

	marker.MarkField("phone", SensitiveTypePhone)
	marker.MarkField("email", SensitiveTypeEmail)
	marker.MarkField("password", SensitiveTypePassword)

	fields := marker.GetSensitiveFields()
	if len(fields) != 3 {
		t.Errorf("Expected 3 sensitive fields, got %d", len(fields))
	}
}

func TestSensitiveMarker_ShouldEncrypt(t *testing.T) {
	marker := NewSensitiveMarker()

	marker.MarkField("password", SensitiveTypePassword)
	marker.MarkField("email", SensitiveTypeEmail)

	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{
			name:      "password should be encrypted",
			fieldName: "password",
			expected:  true,
		},
		{
			name:      "email should not be encrypted by default",
			fieldName: "email",
			expected:  false,
		},
		{
			name:      "non-sensitive field",
			fieldName: "name",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marker.ShouldEncrypt(tt.fieldName)
			if result != tt.expected {
				t.Errorf("ShouldEncrypt(%s) = %v, want %v", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestSensitiveMarker_ShouldMask(t *testing.T) {
	marker := NewSensitiveMarker()

	marker.MarkField("phone", SensitiveTypePhone)

	if !marker.ShouldMask("phone") {
		t.Error("Phone field should be masked")
	}

	if marker.ShouldMask("name") {
		t.Error("Name field should not be masked")
	}
}

func TestNewSensitiveData(t *testing.T) {
	type UserData struct {
		Name     string `json:"name"`
		Phone    string `json:"phone" sensitive:"phone"`
		Email    string `json:"email" sensitive:"email"`
		Password string `json:"password" sensitive:"password"`
	}

	user := UserData{
		Name:     "张三",
		Phone:    "13812345678",
		Email:    "test@example.com",
		Password: "secret123",
	}

	sd := NewSensitiveData(user)
	if sd == nil {
		t.Fatal("NewSensitiveData() returned nil")
	}

	if !sd.Marker().IsSensitive("Phone") {
		t.Error("Phone should be marked as sensitive")
	}

	if !sd.Marker().IsSensitive("Email") {
		t.Error("Email should be marked as sensitive")
	}

	if !sd.Marker().IsSensitive("Password") {
		t.Error("Password should be marked as sensitive")
	}
}

func TestSensitiveData_MarshalJSON(t *testing.T) {
	type UserData struct {
		Name     string `json:"name"`
		Phone    string `json:"phone" sensitive:"phone"`
		Email    string `json:"email" sensitive:"email"`
		Password string `json:"password" sensitive:"password"`
	}

	user := UserData{
		Name:     "张三",
		Phone:    "13812345678",
		Email:    "test@example.com",
		Password: "secret123",
	}

	sd := NewSensitiveData(user)

	// Marshal to JSON
	jsonData, err := json.Marshal(sd)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify sensitive fields are masked
	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Check that sensitive fields are masked
	if phone, ok := result["phone"].(string); ok {
		if phone == "13812345678" {
			t.Error("Phone number should be masked in JSON")
		}
		if phone != "138****5678" {
			t.Errorf("Phone number masking incorrect: got %s, want 138****5678", phone)
		}
	}

	if password, ok := result["password"].(string); ok {
		if password == "secret123" {
			t.Error("Password should be masked in JSON")
		}
	}
}

func TestSensitiveData_WithPointer(t *testing.T) {
	type UserData struct {
		Name  string `json:"name"`
		Phone string `json:"phone" sensitive:"phone"`
	}

	user := &UserData{
		Name:  "测试",
		Phone: "13812345678",
	}

	sd := NewSensitiveData(user)
	if sd == nil {
		t.Fatal("NewSensitiveData() with pointer returned nil")
	}

	if !sd.Marker().IsSensitive("Phone") {
		t.Error("Phone should be marked as sensitive")
	}
}

func TestExtractSensitiveFields(t *testing.T) {
	marker := NewSensitiveMarker()
	marker.MarkField("phone", SensitiveTypePhone)
	marker.MarkField("password", SensitiveTypePassword)

	tests := []struct {
		name     string
		data     interface{}
		expected int
	}{
		{
			name: "map data",
			data: map[string]interface{}{
				"name":     "张三",
				"phone":    "13812345678",
				"email":    "test@example.com",
				"password": "secret",
			},
			expected: 2,
		},
		{
			name: "struct data",
			data: struct {
				name     string
				phone    string
				email    string
				password string
			}{
				name:     "张三",
				phone:    "13812345678",
				email:    "test@example.com",
				password: "secret",
			},
			expected: 0, // 小写字段不可导出，无法访问
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractSensitiveFields(tt.data, marker)
			if len(result) != tt.expected {
				t.Errorf("ExtractSensitiveFields() returned %d fields, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestSensitiveData_NilValue(t *testing.T) {
	sd := NewSensitiveData(nil)
	if sd == nil {
		t.Fatal("NewSensitiveData(nil) returned nil")
	}

	fields := sd.Marker().GetSensitiveFields()
	if len(fields) != 0 {
		t.Errorf("Expected 0 sensitive fields for nil value, got %d", len(fields))
	}
}

func TestSensitiveData_NonStructValue(t *testing.T) {
	// Test with non-struct value (string)
	sd := NewSensitiveData("test string")
	if sd == nil {
		t.Fatal("NewSensitiveData() with string returned nil")
	}

	fields := sd.Marker().GetSensitiveFields()
	if len(fields) != 0 {
		t.Errorf("Expected 0 sensitive fields for non-struct value, got %d", len(fields))
	}
}

// TestSensitiveDataIntegration demonstrates complete integration scenario
func TestSensitiveDataIntegration(t *testing.T) {
	// Define a user order with sensitive data
	type OrderData struct {
		OrderID      string  `json:"order_id"`
		CustomerName string  `json:"customer_name" sensitive:"name"`
		Phone        string  `json:"phone" sensitive:"phone"`
		Email        string  `json:"email" sensitive:"email"`
		CreditCard   string  `json:"credit_card" sensitive:"bankcard"`
		Amount       float64 `json:"amount"`
	}

	order := OrderData{
		OrderID:      "ORD-12345",
		CustomerName: "李明",
		Phone:        "13912345678",
		Email:        "liming@example.com",
		CreditCard:   "6222021234567890",
		Amount:       999.99,
	}

	// Create sensitive data wrapper
	sd := NewSensitiveData(order)

	// Verify sensitive fields are detected
	sensitiveFields := sd.Marker().GetSensitiveFields()
	if len(sensitiveFields) != 4 {
		t.Errorf("Expected 4 sensitive fields, got %d", len(sensitiveFields))
	}

	// Marshal to JSON and verify masking
	jsonData, err := json.Marshal(sd)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify non-sensitive data is preserved
	if result["order_id"] != "ORD-12345" {
		t.Error("OrderID should not be masked")
	}

	if result["amount"] != 999.99 {
		t.Error("Amount should not be masked")
	}

	// Verify sensitive data is masked
	if phone, ok := result["phone"].(string); ok {
		if phone == "13912345678" {
			t.Error("Phone should be masked")
		}
	}

	t.Logf("Masked JSON: %s", string(jsonData))
}
