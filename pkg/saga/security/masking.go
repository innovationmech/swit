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

// Package security provides data masking functionality for sensitive information.
package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/innovationmech/swit/pkg/logger"
	"go.uber.org/zap"
)

// MaskConfig defines configuration for data masking
type MaskConfig struct {
	// DefaultMaskChar is the character used for masking
	DefaultMaskChar rune

	// PhoneKeepPrefix keeps the first N digits of phone numbers
	PhoneKeepPrefix int

	// PhoneKeepSuffix keeps the last N digits of phone numbers
	PhoneKeepSuffix int

	// EmailKeepPrefix keeps the first N characters of email local part
	EmailKeepPrefix int

	// BankCardKeepPrefix keeps the first N digits of bank card
	BankCardKeepPrefix int

	// BankCardKeepSuffix keeps the last N digits of bank card
	BankCardKeepSuffix int

	// IDCardKeepPrefix keeps the first N digits of ID card
	IDCardKeepPrefix int

	// IDCardKeepSuffix keeps the last N digits of ID card
	IDCardKeepSuffix int

	// NameKeepFirst keeps the first character of name
	NameKeepFirst bool

	// CustomRules for custom masking patterns
	CustomRules map[string]*MaskRule
}

// MaskRule defines a custom masking rule
type MaskRule struct {
	// Pattern is the regex pattern to match
	Pattern *regexp.Regexp

	// KeepPrefix keeps the first N characters
	KeepPrefix int

	// KeepSuffix keeps the last N characters
	KeepSuffix int

	// MaskChar is the character used for masking
	MaskChar rune
}

// DefaultMaskConfig returns the default masking configuration
func DefaultMaskConfig() *MaskConfig {
	return &MaskConfig{
		DefaultMaskChar:    '*',
		PhoneKeepPrefix:    3,
		PhoneKeepSuffix:    4,
		EmailKeepPrefix:    2,
		BankCardKeepPrefix: 4,
		BankCardKeepSuffix: 4,
		IDCardKeepPrefix:   4,
		IDCardKeepSuffix:   4,
		NameKeepFirst:      true,
		CustomRules:        make(map[string]*MaskRule),
	}
}

// copyMaskConfig creates a deep copy of MaskConfig to avoid concurrent map access issues
func copyMaskConfig(original *MaskConfig) *MaskConfig {
	if original == nil {
		return DefaultMaskConfig()
	}

	copy := &MaskConfig{
		DefaultMaskChar:    original.DefaultMaskChar,
		PhoneKeepPrefix:    original.PhoneKeepPrefix,
		PhoneKeepSuffix:    original.PhoneKeepSuffix,
		EmailKeepPrefix:    original.EmailKeepPrefix,
		BankCardKeepPrefix: original.BankCardKeepPrefix,
		BankCardKeepSuffix: original.BankCardKeepSuffix,
		IDCardKeepPrefix:   original.IDCardKeepPrefix,
		IDCardKeepSuffix:   original.IDCardKeepSuffix,
		NameKeepFirst:      original.NameKeepFirst,
		CustomRules:        make(map[string]*MaskRule),
	}

	// Deep copy CustomRules
	for name, rule := range original.CustomRules {
		if rule != nil {
			copy.CustomRules[name] = &MaskRule{
				Pattern:   rule.Pattern,
				KeepPrefix: rule.KeepPrefix,
				KeepSuffix: rule.KeepSuffix,
				MaskChar:  rule.MaskChar,
			}
		}
	}

	return copy
}

// Masker is an interface for data masking
type Masker interface {
	// Mask masks data based on its type
	Mask(data string, dataType SensitiveDataType) string

	// MaskPhone masks phone numbers
	MaskPhone(phone string) string

	// MaskEmail masks email addresses
	MaskEmail(email string) string

	// MaskBankCard masks bank card numbers
	MaskBankCard(cardNumber string) string

	// MaskIDCard masks ID card numbers
	MaskIDCard(idCard string) string

	// MaskName masks names
	MaskName(name string) string

	// MaskAddress masks addresses
	MaskAddress(address string) string

	// MaskPassword completely masks passwords
	MaskPassword(password string) string

	// MaskCustom applies custom masking rules
	MaskCustom(data string, ruleName string) string

	// AddCustomRule adds a custom masking rule
	AddCustomRule(name string, rule *MaskRule) error
}

// DefaultMasker implements the Masker interface
type DefaultMasker struct {
	config *MaskConfig
	logger *zap.Logger
	mu     sync.RWMutex

	// Compiled regex patterns
	phonePattern    *regexp.Regexp
	emailPattern    *regexp.Regexp
	bankCardPattern *regexp.Regexp
	idCardPattern   *regexp.Regexp
}

// NewMasker creates a new data masker
func NewMasker(config *MaskConfig) (*DefaultMasker, error) {
	// Create a deep copy of the config to avoid concurrent map access issues
	maskerConfig := copyMaskConfig(config)

	masker := &DefaultMasker{
		config: maskerConfig,
		logger: logger.Logger,
	}

	if masker.logger == nil {
		masker.logger = zap.NewNop()
	}

	// Compile regex patterns
	var err error

	// Phone: 11 digits (Chinese mobile)
	masker.phonePattern, err = regexp.Compile(`^1[3-9]\d{9}$`)
	if err != nil {
		return nil, err
	}

	// Email: standard email pattern
	masker.emailPattern, err = regexp.Compile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if err != nil {
		return nil, err
	}

	// Bank card: 16-19 digits
	masker.bankCardPattern, err = regexp.Compile(`^\d{16,19}$`)
	if err != nil {
		return nil, err
	}

	// ID card: 15 or 18 digits (Chinese ID card)
	masker.idCardPattern, err = regexp.Compile(`^(\d{15}|\d{17}[\dXx])$`)
	if err != nil {
		return nil, err
	}

	return masker, nil
}

// Mask masks data based on its type
func (m *DefaultMasker) Mask(data string, dataType SensitiveDataType) string {
	if data == "" {
		return data
	}

	switch dataType {
	case SensitiveTypePhone:
		return m.MaskPhone(data)
	case SensitiveTypeEmail:
		return m.MaskEmail(data)
	case SensitiveTypeBankCard:
		return m.MaskBankCard(data)
	case SensitiveTypeIDCard:
		return m.MaskIDCard(data)
	case SensitiveTypeName:
		return m.MaskName(data)
	case SensitiveTypeAddress:
		return m.MaskAddress(data)
	case SensitiveTypePassword:
		return m.MaskPassword(data)
	default:
		// For custom types, mask middle portion
		return m.maskGeneric(data, 2, 2)
	}
}

// MaskPhone masks phone numbers
func (m *DefaultMasker) MaskPhone(phone string) string {
	if phone == "" {
		return phone
	}

	// Remove spaces and dashes
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	// Validate phone format
	if !m.phonePattern.MatchString(phone) {
		// If not standard format, use generic masking
		return m.maskGeneric(phone, 3, 4)
	}

	return m.maskGeneric(phone, m.config.PhoneKeepPrefix, m.config.PhoneKeepSuffix)
}

// MaskEmail masks email addresses
func (m *DefaultMasker) MaskEmail(email string) string {
	if email == "" {
		return email
	}

	// Validate email format
	if !m.emailPattern.MatchString(email) {
		return m.maskGeneric(email, 2, 0)
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return m.maskGeneric(email, 2, 0)
	}

	localPart := parts[0]
	domain := parts[1]

	// Mask local part
	maskedLocal := m.maskGeneric(localPart, m.config.EmailKeepPrefix, 0)

	// Mask domain partially
	domainParts := strings.Split(domain, ".")
	if len(domainParts) > 1 {
		// Keep the TLD, mask the domain name
		domainName := domainParts[0]
		tld := strings.Join(domainParts[1:], ".")

		maskedDomain := m.maskGeneric(domainName, 1, 1) + "." + tld
		return maskedLocal + "@" + maskedDomain
	}

	return maskedLocal + "@" + domain
}

// MaskBankCard masks bank card numbers
func (m *DefaultMasker) MaskBankCard(cardNumber string) string {
	if cardNumber == "" {
		return cardNumber
	}

	// Remove spaces and dashes
	cardNumber = strings.ReplaceAll(cardNumber, " ", "")
	cardNumber = strings.ReplaceAll(cardNumber, "-", "")

	// Validate card number format
	if !m.bankCardPattern.MatchString(cardNumber) {
		return m.maskGeneric(cardNumber, 4, 4)
	}

	return m.maskGeneric(cardNumber, m.config.BankCardKeepPrefix, m.config.BankCardKeepSuffix)
}

// MaskIDCard masks ID card numbers
func (m *DefaultMasker) MaskIDCard(idCard string) string {
	if idCard == "" {
		return idCard
	}

	// Validate ID card format
	if !m.idCardPattern.MatchString(idCard) {
		return m.maskGeneric(idCard, 4, 4)
	}

	return m.maskGeneric(idCard, m.config.IDCardKeepPrefix, m.config.IDCardKeepSuffix)
}

// MaskName masks names
func (m *DefaultMasker) MaskName(name string) string {
	if name == "" {
		return name
	}

	// For Chinese names (usually 2-4 characters)
	length := utf8.RuneCountInString(name)

	if length <= 1 {
		return string(m.config.DefaultMaskChar)
	}

	if m.config.NameKeepFirst {
		if length == 2 {
			// For 2-char names: keep first, mask second
			return m.maskGeneric(name, 1, 0)
		}
		// For longer names: keep first, mask rest
		return m.maskGeneric(name, 1, 0)
	}

	// Mask middle character(s)
	return m.maskGeneric(name, 1, 1)
}

// MaskAddress masks addresses
func (m *DefaultMasker) MaskAddress(address string) string {
	if address == "" {
		return address
	}

	length := utf8.RuneCountInString(address)

	// Keep first 6 characters (usually province/city)
	keepPrefix := 6
	if length < 10 {
		keepPrefix = length / 3
	}

	return m.maskGeneric(address, keepPrefix, 0)
}

// MaskPassword completely masks passwords
func (m *DefaultMasker) MaskPassword(password string) string {
	if password == "" {
		return password
	}

	// Always return fixed length mask for passwords
	return "********"
}

// MaskCustom applies custom masking rules
func (m *DefaultMasker) MaskCustom(data string, ruleName string) string {
	m.mu.RLock()
	rule, exists := m.config.CustomRules[ruleName]
	m.mu.RUnlock()

	if !exists {
		m.logger.Warn("Custom rule not found, using default masking",
			zap.String("rule_name", ruleName))
		return m.maskGeneric(data, 2, 2)
	}

	// Apply custom rule
	if rule.Pattern != nil && !rule.Pattern.MatchString(data) {
		m.logger.Debug("Data does not match custom pattern",
			zap.String("rule_name", ruleName))
	}

	maskChar := rule.MaskChar
	if maskChar == 0 {
		maskChar = m.config.DefaultMaskChar
	}

	return m.maskGenericWithChar(data, rule.KeepPrefix, rule.KeepSuffix, maskChar)
}

// AddCustomRule adds a custom masking rule
func (m *DefaultMasker) AddCustomRule(name string, rule *MaskRule) error {
	if rule == nil {
		return ErrInvalidMaskRule
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.CustomRules[name] = rule

	m.logger.Info("Added custom masking rule",
		zap.String("rule_name", name))

	return nil
}

// maskGeneric provides generic masking logic
func (m *DefaultMasker) maskGeneric(data string, keepPrefix, keepSuffix int) string {
	return m.maskGenericWithChar(data, keepPrefix, keepSuffix, m.config.DefaultMaskChar)
}

// maskGenericWithChar provides generic masking logic with custom mask character
func (m *DefaultMasker) maskGenericWithChar(data string, keepPrefix, keepSuffix int, maskChar rune) string {
	if data == "" {
		return data
	}

	runes := []rune(data)
	length := len(runes)

	// Validate parameters
	if keepPrefix < 0 {
		keepPrefix = 0
	}
	if keepSuffix < 0 {
		keepSuffix = 0
	}

	// If keep prefix + suffix >= length, mask middle character(s)
	if keepPrefix+keepSuffix >= length {
		if length == 1 {
			return string(maskChar)
		}
		keepPrefix = 1
		keepSuffix = 1
		if length == 2 {
			keepSuffix = 0
		}
	}

	// Build masked string
	var result strings.Builder
	result.Grow(length)

	// Keep prefix
	for i := 0; i < keepPrefix && i < length; i++ {
		result.WriteRune(runes[i])
	}

	// Mask middle
	maskLength := length - keepPrefix - keepSuffix
	for i := 0; i < maskLength; i++ {
		result.WriteRune(maskChar)
	}

	// Keep suffix
	for i := length - keepSuffix; i < length; i++ {
		result.WriteRune(runes[i])
	}

	return result.String()
}

// ErrInvalidMaskRule is returned when a mask rule is invalid
var ErrInvalidMaskRule = fmt.Errorf("invalid mask rule")

// MaskString is a convenience function to mask a string based on data type
func MaskString(data string, dataType SensitiveDataType) string {
	masker, err := NewMasker(nil)
	if err != nil {
		return data
	}
	return masker.Mask(data, dataType)
}

// MaskMap masks sensitive fields in a map
func MaskMap(data map[string]interface{}, marker SensitiveMarker) map[string]interface{} {
	if data == nil || marker == nil {
		return data
	}

	masker, err := NewMasker(nil)
	if err != nil {
		return data
	}

	result := make(map[string]interface{})

	for key, value := range data {
		if marker.IsSensitive(key) {
			// Get sensitive field info
			var dataType SensitiveDataType
			for _, field := range marker.GetSensitiveFields() {
				if field.FieldName == key {
					dataType = field.DataType
					break
				}
			}

			// Mask the value if it's a string
			if str, ok := value.(string); ok {
				result[key] = masker.Mask(str, dataType)
			} else {
				result[key] = "***MASKED***"
			}
		} else {
			result[key] = value
		}
	}

	return result
}

// MaskJSON masks sensitive fields in a JSON-like structure
func MaskJSON(data interface{}, marker SensitiveMarker) interface{} {
	if data == nil || marker == nil {
		return data
	}

	switch v := data.(type) {
	case map[string]interface{}:
		return MaskMap(v, marker)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = MaskJSON(item, marker)
		}
		return result
	case string:
		// Check if this string is a sensitive value
		// This requires context which we don't have here
		return v
	default:
		return v
	}
}
