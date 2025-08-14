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
