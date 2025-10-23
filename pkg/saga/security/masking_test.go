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
	"regexp"
	"testing"
)

func TestNewMasker(t *testing.T) {
	tests := []struct {
		name    string
		config  *MaskConfig
		wantErr bool
	}{
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name:    "custom config",
			config:  DefaultMaskConfig(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masker, err := NewMasker(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMasker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && masker == nil {
				t.Error("NewMasker() returned nil masker")
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "valid Chinese mobile",
			phone:    "13812345678",
			expected: "138****5678",
		},
		{
			name:     "phone with spaces",
			phone:    "138 1234 5678",
			expected: "138****5678",
		},
		{
			name:     "phone with dashes",
			phone:    "138-1234-5678",
			expected: "138****5678",
		},
		{
			name:     "empty phone",
			phone:    "",
			expected: "",
		},
		{
			name:     "invalid format",
			phone:    "123456",
			expected: "1****6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskPhone(tt.phone)
			if result != tt.expected {
				t.Errorf("MaskPhone() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "valid email",
			email:    "test@example.com",
			expected: "te**@e*****e.com",
		},
		{
			name:     "short email",
			email:    "a@b.com",
			expected: "*@*.com",
		},
		{
			name:     "long email",
			email:    "testuser@example.com",
			expected: "te******@e*****e.com",
		},
		{
			name:     "empty email",
			email:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskEmail(tt.email)
			if result != tt.expected {
				t.Errorf("MaskEmail() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskBankCard(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name       string
		cardNumber string
		expected   string
	}{
		{
			name:       "16 digit card",
			cardNumber: "1234567890123456",
			expected:   "1234********3456",
		},
		{
			name:       "19 digit card",
			cardNumber: "1234567890123456789",
			expected:   "1234***********6789",
		},
		{
			name:       "card with spaces",
			cardNumber: "1234 5678 9012 3456",
			expected:   "1234********3456",
		},
		{
			name:       "empty card",
			cardNumber: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskBankCard(tt.cardNumber)
			if result != tt.expected {
				t.Errorf("MaskBankCard() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskIDCard(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		idCard   string
		expected string
	}{
		{
			name:     "18 digit ID card",
			idCard:   "110101199001011234",
			expected: "1101**********1234",
		},
		{
			name:     "18 digit ID card with X",
			idCard:   "11010119900101123X",
			expected: "1101**********123X",
		},
		{
			name:     "15 digit ID card",
			idCard:   "110101900101123",
			expected: "1101*******1123",
		},
		{
			name:     "empty ID card",
			idCard:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskIDCard(tt.idCard)
			if result != tt.expected {
				t.Errorf("MaskIDCard() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskName(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "two character name",
			input:    "张三",
			expected: "张*",
		},
		{
			name:     "three character name",
			input:    "李四五",
			expected: "李**",
		},
		{
			name:     "four character name",
			input:    "欧阳修之",
			expected: "欧***",
		},
		{
			name:     "single character",
			input:    "王",
			expected: "*",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "",
		},
		{
			name:     "English name",
			input:    "John",
			expected: "J***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskName(tt.input)
			if result != tt.expected {
				t.Errorf("MaskName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskAddress(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "long address",
			address:  "北京市朝阳区建国路88号SOHO现代城",
			expected: "北京市朝阳区*************",
		},
		{
			name:     "short address",
			address:  "北京市",
			expected: "北**",
		},
		{
			name:     "empty address",
			address:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskAddress(tt.address)
			if result != tt.expected {
				t.Errorf("MaskAddress() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskPassword(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		password string
		expected string
	}{
		{
			name:     "normal password",
			password: "MyPassword123!",
			expected: "********",
		},
		{
			name:     "short password",
			password: "123",
			expected: "********",
		},
		{
			name:     "long password",
			password: "VeryLongPasswordWith123!@#$%",
			expected: "********",
		},
		{
			name:     "empty password",
			password: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskPassword(tt.password)
			if result != tt.expected {
				t.Errorf("MaskPassword() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMask(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		data     string
		dataType SensitiveDataType
		expected string
	}{
		{
			name:     "mask phone",
			data:     "13812345678",
			dataType: SensitiveTypePhone,
			expected: "138****5678",
		},
		{
			name:     "mask email",
			data:     "test@example.com",
			dataType: SensitiveTypeEmail,
			expected: "te**@e*****e.com",
		},
		{
			name:     "mask bank card",
			data:     "1234567890123456",
			dataType: SensitiveTypeBankCard,
			expected: "1234********3456",
		},
		{
			name:     "mask ID card",
			data:     "110101199001011234",
			dataType: SensitiveTypeIDCard,
			expected: "1101**********1234",
		},
		{
			name:     "mask name",
			data:     "张三",
			dataType: SensitiveTypeName,
			expected: "张*",
		},
		{
			name:     "mask password",
			data:     "MyPassword123",
			dataType: SensitiveTypePassword,
			expected: "********",
		},
		{
			name:     "empty data",
			data:     "",
			dataType: SensitiveTypePhone,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.Mask(tt.data, tt.dataType)
			if result != tt.expected {
				t.Errorf("Mask() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAddCustomRule(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	tests := []struct {
		name     string
		ruleName string
		rule     *MaskRule
		wantErr  bool
	}{
		{
			name:     "valid rule",
			ruleName: "custom1",
			rule: &MaskRule{
				Pattern:    regexp.MustCompile(`^\d{6}$`),
				KeepPrefix: 2,
				KeepSuffix: 2,
				MaskChar:   '#',
			},
			wantErr: false,
		},
		{
			name:     "nil rule",
			ruleName: "custom2",
			rule:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := masker.AddCustomRule(tt.ruleName, tt.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddCustomRule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMaskCustom(t *testing.T) {
	masker, err := NewMasker(nil)
	if err != nil {
		t.Fatalf("Failed to create masker: %v", err)
	}

	// Add custom rule
	rule := &MaskRule{
		Pattern:    regexp.MustCompile(`^\d{6}$`),
		KeepPrefix: 2,
		KeepSuffix: 2,
		MaskChar:   '#',
	}
	err = masker.AddCustomRule("postal_code", rule)
	if err != nil {
		t.Fatalf("Failed to add custom rule: %v", err)
	}

	tests := []struct {
		name     string
		data     string
		ruleName string
		expected string
	}{
		{
			name:     "valid postal code",
			data:     "100000",
			ruleName: "postal_code",
			expected: "10##00",
		},
		{
			name:     "non-existent rule",
			data:     "123456",
			ruleName: "non_existent",
			expected: "12**56",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := masker.MaskCustom(tt.data, tt.ruleName)
			if result != tt.expected {
				t.Errorf("MaskCustom() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		dataType SensitiveDataType
	}{
		{
			name:     "mask phone",
			data:     "13812345678",
			dataType: SensitiveTypePhone,
		},
		{
			name:     "mask email",
			data:     "test@example.com",
			dataType: SensitiveTypeEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskString(tt.data, tt.dataType)
			if result == "" {
				t.Error("MaskString() returned empty string")
			}
		})
	}
}

func TestMaskMap(t *testing.T) {
	marker := NewSensitiveMarker()
	marker.MarkField("phone", SensitiveTypePhone)
	marker.MarkField("email", SensitiveTypeEmail)

	tests := []struct {
		name     string
		data     map[string]interface{}
		marker   SensitiveMarker
		expected map[string]interface{}
	}{
		{
			name: "mask sensitive fields",
			data: map[string]interface{}{
				"phone": "13812345678",
				"email": "test@example.com",
				"name":  "张三",
			},
			marker: marker,
			expected: map[string]interface{}{
				"phone": "138****5678",
				"email": "te**@e*****e.com",
				"name":  "张三",
			},
		},
		{
			name:     "nil data",
			data:     nil,
			marker:   marker,
			expected: nil,
		},
		{
			name: "nil marker",
			data: map[string]interface{}{
				"phone": "13812345678",
			},
			marker:   nil,
			expected: map[string]interface{}{"phone": "13812345678"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskMap(tt.data, tt.marker)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("MaskMap() = %v, want nil", result)
				}
				return
			}

			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("MaskMap() field %s = %v, want %v", key, result[key], expectedValue)
				}
			}
		})
	}
}

func TestMaskJSON(t *testing.T) {
	marker := NewSensitiveMarker()
	marker.MarkField("phone", SensitiveTypePhone)

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "map data",
			data: map[string]interface{}{
				"phone": "13812345678",
			},
		},
		{
			name: "array data",
			data: []interface{}{
				map[string]interface{}{
					"phone": "13812345678",
				},
			},
		},
		{
			name: "string data",
			data: "test",
		},
		{
			name: "nil data",
			data: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskJSON(tt.data, marker)
			if tt.data == nil && result != nil {
				t.Errorf("MaskJSON() = %v, want nil", result)
			}
		})
	}
}

func TestDefaultMaskConfig(t *testing.T) {
	config := DefaultMaskConfig()

	if config == nil {
		t.Fatal("DefaultMaskConfig() returned nil")
	}

	if config.DefaultMaskChar != '*' {
		t.Errorf("DefaultMaskChar = %c, want *", config.DefaultMaskChar)
	}

	if config.PhoneKeepPrefix != 3 {
		t.Errorf("PhoneKeepPrefix = %d, want 3", config.PhoneKeepPrefix)
	}

	if config.PhoneKeepSuffix != 4 {
		t.Errorf("PhoneKeepSuffix = %d, want 4", config.PhoneKeepSuffix)
	}
}

func BenchmarkMaskPhone(b *testing.B) {
	masker, err := NewMasker(nil)
	if err != nil {
		b.Fatalf("Failed to create masker: %v", err)
	}

	phone := "13812345678"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		masker.MaskPhone(phone)
	}
}

func BenchmarkMaskEmail(b *testing.B) {
	masker, err := NewMasker(nil)
	if err != nil {
		b.Fatalf("Failed to create masker: %v", err)
	}

	email := "test@example.com"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		masker.MaskEmail(email)
	}
}

func BenchmarkMaskBankCard(b *testing.B) {
	masker, err := NewMasker(nil)
	if err != nil {
		b.Fatalf("Failed to create masker: %v", err)
	}

	card := "1234567890123456"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		masker.MaskBankCard(card)
	}
}
