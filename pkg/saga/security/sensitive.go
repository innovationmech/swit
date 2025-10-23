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

// Package security provides sensitive data marking and protection functionality.
package security

import (
	"encoding/json"
	"reflect"
	"strings"
)

// SensitiveDataType represents the type of sensitive data
type SensitiveDataType string

const (
	// SensitiveTypePhone represents phone numbers
	SensitiveTypePhone SensitiveDataType = "phone"

	// SensitiveTypeIDCard represents ID card numbers
	SensitiveTypeIDCard SensitiveDataType = "idcard"

	// SensitiveTypeBankCard represents bank card numbers
	SensitiveTypeBankCard SensitiveDataType = "bankcard"

	// SensitiveTypeEmail represents email addresses
	SensitiveTypeEmail SensitiveDataType = "email"

	// SensitiveTypeName represents personal names
	SensitiveTypeName SensitiveDataType = "name"

	// SensitiveTypeAddress represents addresses
	SensitiveTypeAddress SensitiveDataType = "address"

	// SensitiveTypePassword represents passwords
	SensitiveTypePassword SensitiveDataType = "password"

	// SensitiveTypeCustom represents custom sensitive data
	SensitiveTypeCustom SensitiveDataType = "custom"
)

// SensitiveField defines a field that contains sensitive data
type SensitiveField struct {
	// FieldName is the name of the field
	FieldName string

	// DataType is the type of sensitive data
	DataType SensitiveDataType

	// Encrypted indicates if the field should be encrypted
	Encrypted bool

	// Masked indicates if the field should be masked in logs
	Masked bool
}

// SensitiveMarker is an interface for marking sensitive data
type SensitiveMarker interface {
	// MarkField marks a field as sensitive
	MarkField(fieldName string, dataType SensitiveDataType) error

	// IsSensitive checks if a field is marked as sensitive
	IsSensitive(fieldName string) bool

	// GetSensitiveFields returns all sensitive fields
	GetSensitiveFields() []SensitiveField

	// ShouldEncrypt checks if a field should be encrypted
	ShouldEncrypt(fieldName string) bool

	// ShouldMask checks if a field should be masked
	ShouldMask(fieldName string) bool
}

// DefaultSensitiveMarker implements SensitiveMarker
type DefaultSensitiveMarker struct {
	fields map[string]SensitiveField
}

// NewSensitiveMarker creates a new sensitive data marker
func NewSensitiveMarker() *DefaultSensitiveMarker {
	return &DefaultSensitiveMarker{
		fields: make(map[string]SensitiveField),
	}
}

// MarkField marks a field as sensitive
func (m *DefaultSensitiveMarker) MarkField(fieldName string, dataType SensitiveDataType) error {
	field := SensitiveField{
		FieldName: fieldName,
		DataType:  dataType,
		Encrypted: shouldEncryptByDefault(dataType),
		Masked:    true, // Always mask by default
	}

	m.fields[fieldName] = field
	return nil
}

// IsSensitive checks if a field is marked as sensitive
func (m *DefaultSensitiveMarker) IsSensitive(fieldName string) bool {
	_, exists := m.fields[fieldName]
	return exists
}

// GetSensitiveFields returns all sensitive fields
func (m *DefaultSensitiveMarker) GetSensitiveFields() []SensitiveField {
	fields := make([]SensitiveField, 0, len(m.fields))
	for _, field := range m.fields {
		fields = append(fields, field)
	}
	return fields
}

// ShouldEncrypt checks if a field should be encrypted
func (m *DefaultSensitiveMarker) ShouldEncrypt(fieldName string) bool {
	if field, exists := m.fields[fieldName]; exists {
		return field.Encrypted
	}
	return false
}

// ShouldMask checks if a field should be masked
func (m *DefaultSensitiveMarker) ShouldMask(fieldName string) bool {
	if field, exists := m.fields[fieldName]; exists {
		return field.Masked
	}
	return false
}

// shouldEncryptByDefault determines if a data type should be encrypted by default
func shouldEncryptByDefault(dataType SensitiveDataType) bool {
	switch dataType {
	case SensitiveTypePassword, SensitiveTypeBankCard, SensitiveTypeIDCard:
		return true
	default:
		return false
	}
}

// SensitiveData is a struct tag-based approach to mark sensitive fields
type SensitiveData struct {
	value interface{}
	mask  SensitiveMarker
}

// NewSensitiveData creates a new SensitiveData wrapper
func NewSensitiveData(value interface{}) *SensitiveData {
	sd := &SensitiveData{
		value: value,
		mask:  NewSensitiveMarker(),
	}

	// Auto-detect sensitive fields from struct tags
	sd.detectSensitiveFields()

	return sd
}

// detectSensitiveFields scans struct tags to identify sensitive fields
func (sd *SensitiveData) detectSensitiveFields() {
	if sd.value == nil {
		return
	}

	val := reflect.ValueOf(sd.value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Check for "sensitive" tag
		if tag := field.Tag.Get("sensitive"); tag != "" {
			dataType := SensitiveDataType(tag)
			sd.mask.MarkField(field.Name, dataType)
		}
	}
}

// Value returns the underlying value
func (sd *SensitiveData) Value() interface{} {
	return sd.value
}

// Marker returns the sensitive marker
func (sd *SensitiveData) Marker() SensitiveMarker {
	return sd.mask
}

// MarshalJSON implements json.Marshaler to handle sensitive data during JSON serialization
func (sd *SensitiveData) MarshalJSON() ([]byte, error) {
	if sd.value == nil {
		return json.Marshal(nil)
	}

	// Create a masked version of the data
	masked := sd.createMaskedCopy()
	return json.Marshal(masked)
}

// createMaskedCopy creates a copy of the data with sensitive fields masked
func (sd *SensitiveData) createMaskedCopy() interface{} {
	if sd.value == nil {
		return nil
	}

	val := reflect.ValueOf(sd.value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return sd.value
	}

	// Create a new map to hold masked values
	result := make(map[string]interface{})

	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Get JSON field name
		jsonTag := field.Tag.Get("json")
		fieldName := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Check if field should be masked
		if sd.mask.ShouldMask(field.Name) {
			// Get the masker for this field type
			if fieldValue.CanInterface() {
				originalValue := fieldValue.Interface()
				if str, ok := originalValue.(string); ok {
					maskedValue := MaskString(str, sd.getSensitiveType(field.Name))
					result[fieldName] = maskedValue
				} else {
					result[fieldName] = "***MASKED***"
				}
			} else {
				result[fieldName] = "***MASKED***"
			}
		} else {
			if fieldValue.CanInterface() {
				result[fieldName] = fieldValue.Interface()
			}
		}
	}

	return result
}

// getSensitiveType gets the sensitive data type for a field
func (sd *SensitiveData) getSensitiveType(fieldName string) SensitiveDataType {
	for _, field := range sd.mask.GetSensitiveFields() {
		if field.FieldName == fieldName {
			return field.DataType
		}
	}
	return SensitiveTypeCustom
}

// ExtractSensitiveFields extracts sensitive fields from a map or struct
func ExtractSensitiveFields(data interface{}, marker SensitiveMarker) map[string]interface{} {
	result := make(map[string]interface{})

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Map {
		// Handle map
		for _, key := range val.MapKeys() {
			keyStr := key.String()
			if marker.IsSensitive(keyStr) {
				result[keyStr] = val.MapIndex(key).Interface()
			}
		}
	} else if val.Kind() == reflect.Struct {
		// Handle struct
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if marker.IsSensitive(field.Name) && val.Field(i).CanInterface() {
				result[field.Name] = val.Field(i).Interface()
			}
		}
	}

	return result
}
