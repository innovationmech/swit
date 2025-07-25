// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

// Status represents common status types across the system
type Status string

const (
	// StatusActive represents an active entity
	StatusActive Status = "active"
	// StatusInactive represents an inactive entity
	StatusInactive Status = "inactive"
	// StatusPending represents a pending entity
	StatusPending Status = "pending"
	// StatusDeleted represents a deleted entity
	StatusDeleted Status = "deleted"
	// StatusSuspended represents a suspended entity
	StatusSuspended Status = "suspended"
	// StatusShutdownInitiated represents a shutdown initiated state
	StatusShutdownInitiated Status = "shutdown_initiated"
)

// String returns the string representation of the status
func (s Status) String() string {
	return string(s)
}

// IsValid checks if the status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusActive, StatusInactive, StatusPending, StatusDeleted, StatusSuspended, StatusShutdownInitiated:
		return true
	default:
		return false
	}
}
