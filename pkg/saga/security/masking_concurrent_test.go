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
	"sync"
	"testing"
)

// TestMaskerConcurrentConfigSharing tests that multiple maskers sharing the same config
// don't cause concurrent map access issues
func TestMaskerConcurrentConfigSharing(t *testing.T) {
	// Create a shared config with some custom rules
	sharedConfig := DefaultMaskConfig()
	
	// Add some custom rules to the shared config
	postalCodeRule := &MaskRule{
		Pattern:   regexp.MustCompile(`^\d{6}$`),
		KeepPrefix: 2,
		KeepSuffix: 2,
		MaskChar:  '*',
	}
	sharedConfig.CustomRules["postal_code"] = postalCodeRule

	// Create multiple maskers using the same config
	numMaskers := 10
	maskers := make([]*DefaultMasker, numMaskers)
	
	for i := 0; i < numMaskers; i++ {
		masker, err := NewMasker(sharedConfig)
		if err != nil {
			t.Fatalf("Failed to create masker %d: %v", i, err)
		}
		maskers[i] = masker
	}

	// Test concurrent access to custom rules
	var wg sync.WaitGroup
	numGoroutines := 100
	
	// Test concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			masker := maskers[id%numMaskers]
			result := masker.MaskCustom("123456", "postal_code")
			expected := "12**56"
			if result != expected {
				t.Errorf("MaskCustom returned %q, expected %q", result, expected)
			}
		}(i)
	}

	// Test concurrent writes (adding new rules)
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			masker := maskers[id%numMaskers]
			ruleName := "test_rule"
			rule := &MaskRule{
				Pattern:   regexp.MustCompile(`^test\d+$`),
				KeepPrefix: 2,
				KeepSuffix: 1,
				MaskChar:  '#',
			}
			err := masker.AddCustomRule(ruleName, rule)
			if err != nil {
				t.Errorf("Failed to add custom rule: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify that each masker has its own CustomRules map
	// and modifications to one don't affect others
	for i, masker := range maskers {
		// Each masker should have the original postal_code rule
		result := masker.MaskCustom("123456", "postal_code")
		expected := "12**56"
		if result != expected {
			t.Errorf("Masker %d: MaskCustom returned %q, expected %q", i, result, expected)
		}

		// Check that the masker has its own CustomRules map
		// We can't compare maps directly, so we'll add a rule to one and check it doesn't affect the other
		originalLen := len(sharedConfig.CustomRules)
		masker.config.CustomRules["temp_test"] = &MaskRule{MaskChar: 'X'}
		if len(sharedConfig.CustomRules) != originalLen {
			t.Errorf("Masker %d is sharing the same CustomRules map as the shared config", i)
		}
		// Clean up the temp rule
		delete(masker.config.CustomRules, "temp_test")

		// The shared config should only have the original rule
		if len(sharedConfig.CustomRules) != 1 {
			t.Errorf("Shared config has %d rules, expected 1", len(sharedConfig.CustomRules))
		}
		if _, exists := sharedConfig.CustomRules["postal_code"]; !exists {
			t.Error("Shared config missing original postal_code rule")
		}
	}
}

// TestCopyMaskConfig tests the copyMaskConfig function
func TestCopyMaskConfig(t *testing.T) {
	// Test copying nil config
	copied := copyMaskConfig(nil)
	if copied == nil {
		t.Error("copyMaskConfig(nil) should not return nil")
	}
	if copied.CustomRules == nil {
		t.Error("CustomRules map should be initialized")
	}

	// Test copying a config with custom rules
	original := &MaskConfig{
		DefaultMaskChar: '#',
		PhoneKeepPrefix: 2,
		CustomRules: make(map[string]*MaskRule),
	}
	
	rule := &MaskRule{
		Pattern:   regexp.MustCompile(`^\d{6}$`),
		KeepPrefix: 2,
		KeepSuffix: 2,
		MaskChar:  '*',
	}
	original.CustomRules["test"] = rule

	copied = copyMaskConfig(original)

	// Verify the copy
	if copied.DefaultMaskChar != original.DefaultMaskChar {
		t.Errorf("DefaultMaskChar not copied correctly")
	}
	if copied.PhoneKeepPrefix != original.PhoneKeepPrefix {
		t.Errorf("PhoneKeepPrefix not copied correctly")
	}
	if len(copied.CustomRules) != len(original.CustomRules) {
		t.Errorf("CustomRules length mismatch")
	}

	// Verify that CustomRules maps are different
	// We can't compare maps directly, so we'll modify one and check the other is unaffected
	originalLen := len(original.CustomRules)
	copied.CustomRules["temp_test"] = &MaskRule{MaskChar: 'X'}
	if len(original.CustomRules) != originalLen {
		t.Error("CustomRules maps should be different instances")
	}
	// Clean up
	delete(copied.CustomRules, "temp_test")

	// Verify that MaskRule instances are different
	copiedRule := copied.CustomRules["test"]
	if copiedRule == original.CustomRules["test"] {
		t.Error("MaskRule instances should be different")
	}

	// Verify that the regex pattern is the same (shared reference is fine for regex)
	if copiedRule.Pattern != original.CustomRules["test"].Pattern {
		t.Error("Regex patterns should be the same instance")
	}
}

// TestMaskerIsolation tests that modifications to one masker's config
// don't affect other maskers
func TestMaskerIsolation(t *testing.T) {
	// Create a shared config
	sharedConfig := DefaultMaskConfig()

	// Create two maskers with the same config
	masker1, err := NewMasker(sharedConfig)
	if err != nil {
		t.Fatalf("Failed to create masker1: %v", err)
	}

	masker2, err := NewMasker(sharedConfig)
	if err != nil {
		t.Fatalf("Failed to create masker2: %v", err)
	}

	// Add a custom rule to masker1
	rule := &MaskRule{
		Pattern:   regexp.MustCompile(`^\d{6}$`),
		KeepPrefix: 2,
		KeepSuffix: 2,
		MaskChar:  '#',
	}
	err = masker1.AddCustomRule("test_rule", rule)
	if err != nil {
		t.Fatalf("Failed to add custom rule to masker1: %v", err)
	}

	// Verify that masker1 has the rule
	result1 := masker1.MaskCustom("123456", "test_rule")
	expected := "12##56"
	if result1 != expected {
		t.Errorf("masker1 MaskCustom returned %q, expected %q", result1, expected)
	}

	// Verify that masker2 doesn't have the rule
	result2 := masker2.MaskCustom("123456", "test_rule")
	// Should use default masking since the rule doesn't exist in masker2
	if result2 == expected {
		t.Errorf("masker2 should not have the custom rule, but got %q", result2)
	}

	// Verify that the shared config is unchanged
	if len(sharedConfig.CustomRules) != 0 {
		t.Errorf("Shared config should be empty, but has %d rules", len(sharedConfig.CustomRules))
	}
}
