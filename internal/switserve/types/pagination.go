// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"math"
)

// PaginationRequest represents common pagination parameters
type PaginationRequest struct {
	Page     int `json:"page" form:"page" binding:"min=1"`
	PageSize int `json:"page_size" form:"page_size" binding:"min=1,max=100"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewPaginationRequest creates a new pagination request with defaults
func NewPaginationRequest(page, pageSize int) *PaginationRequest {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return &PaginationRequest{
		Page:     page,
		PageSize: pageSize,
	}
}

// GetOffset calculates the offset for database queries
func (p *PaginationRequest) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

// GetLimit returns the page size as limit
func (p *PaginationRequest) GetLimit() int {
	return p.PageSize
}

// NewPaginationResponse creates a new pagination response
func NewPaginationResponse(page, pageSize int, total int64) *PaginationResponse {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	return &PaginationResponse{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Status     *ResponseStatus     `json:"status"`
	Data       interface{}         `json:"data"`
	Pagination *PaginationResponse `json:"pagination"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse(data interface{}, pagination *PaginationResponse) *PaginatedResponse {
	return &PaginatedResponse{
		Status:     NewSuccessStatus("Success"),
		Data:       data,
		Pagination: pagination,
	}
}
