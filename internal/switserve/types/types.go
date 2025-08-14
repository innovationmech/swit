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
