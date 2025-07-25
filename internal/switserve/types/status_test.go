// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
	}{
		{"active", StatusActive, "active"},
		{"inactive", StatusInactive, "inactive"},
		{"pending", StatusPending, "pending"},
		{"deleted", StatusDeleted, "deleted"},
		{"suspended", StatusSuspended, "suspended"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   bool
	}{
		{"valid active", StatusActive, true},
		{"valid inactive", StatusInactive, true},
		{"valid pending", StatusPending, true},
		{"valid deleted", StatusDeleted, true},
		{"valid suspended", StatusSuspended, true},
		{"invalid empty", Status(""), false},
		{"invalid unknown", Status("unknown"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.want {
				t.Errorf("Status.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
