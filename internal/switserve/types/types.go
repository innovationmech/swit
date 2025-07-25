// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package types provides shared type definitions for the switserve service.
// This package contains common data structures, constants, and utility types
// that are used across different layers of the application to ensure consistency
// and reduce code duplication.
//
// The package includes:
//   - Status types and constants for entity states
//   - Response structures for API responses
//   - Pagination types for paginated queries
//   - Error types with factory functions
//   - Health check related types
//
// Usage:
//
//	import "github.com/innovationmech/swit/internal/switserve/types"
//
//	// Using status types
//	user.Status = types.StatusActive
//
//	// Creating API responses
//	response := types.NewSuccessResponse(data)
//
//	// Creating paginated responses
//	pagination := types.NewPaginationResponse(1, 10, 100)
//	response := types.NewPaginatedResponse(users, pagination)
//
//	// Creating service errors
//	err := types.ErrNotFound("user")
package types
