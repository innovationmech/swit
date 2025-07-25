// Copyright 2024 Innovation Mechanism. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"
)

func TestNewPaginationRequest(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{"valid values", 2, 20, 2, 20},
		{"zero page", 0, 20, 1, 20},
		{"negative page", -1, 20, 1, 20},
		{"zero page size", 2, 0, 2, 10},
		{"negative page size", 2, -5, 2, 10},
		{"oversized page size", 2, 150, 2, 100},
		{"both invalid", 0, 0, 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPaginationRequest(tt.page, tt.pageSize)
			if got.Page != tt.wantPage {
				t.Errorf("NewPaginationRequest().Page = %v, want %v", got.Page, tt.wantPage)
			}
			if got.PageSize != tt.wantPageSize {
				t.Errorf("NewPaginationRequest().PageSize = %v, want %v", got.PageSize, tt.wantPageSize)
			}
		})
	}
}

func TestPaginationRequest_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		want     int
	}{
		{"first page", 1, 10, 0},
		{"second page", 2, 10, 10},
		{"third page", 3, 20, 40},
		{"large page", 10, 5, 45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaginationRequest{Page: tt.page, PageSize: tt.pageSize}
			if got := p.GetOffset(); got != tt.want {
				t.Errorf("PaginationRequest.GetOffset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPaginationRequest_GetLimit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		want     int
	}{
		{"small page size", 5, 5},
		{"medium page size", 20, 20},
		{"large page size", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PaginationRequest{PageSize: tt.pageSize}
			if got := p.GetLimit(); got != tt.want {
				t.Errorf("PaginationRequest.GetLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewPaginationResponse(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		total          int64
		wantTotalPages int
		wantHasNext    bool
		wantHasPrev    bool
	}{
		{"first page with more", 1, 10, 25, 3, true, false},
		{"middle page", 2, 10, 25, 3, true, true},
		{"last page", 3, 10, 25, 3, false, true},
		{"single page", 1, 10, 5, 1, false, false},
		{"exact multiple", 2, 10, 20, 2, false, true},
		{"empty result", 1, 10, 0, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPaginationResponse(tt.page, tt.pageSize, tt.total)
			if got.Page != tt.page {
				t.Errorf("NewPaginationResponse().Page = %v, want %v", got.Page, tt.page)
			}
			if got.PageSize != tt.pageSize {
				t.Errorf("NewPaginationResponse().PageSize = %v, want %v", got.PageSize, tt.pageSize)
			}
			if got.Total != tt.total {
				t.Errorf("NewPaginationResponse().Total = %v, want %v", got.Total, tt.total)
			}
			if got.TotalPages != tt.wantTotalPages {
				t.Errorf("NewPaginationResponse().TotalPages = %v, want %v", got.TotalPages, tt.wantTotalPages)
			}
			if got.HasNext != tt.wantHasNext {
				t.Errorf("NewPaginationResponse().HasNext = %v, want %v", got.HasNext, tt.wantHasNext)
			}
			if got.HasPrev != tt.wantHasPrev {
				t.Errorf("NewPaginationResponse().HasPrev = %v, want %v", got.HasPrev, tt.wantHasPrev)
			}
		})
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	data := []string{"item1", "item2"}
	pagination := NewPaginationResponse(1, 10, 2)

	got := NewPaginatedResponse(data, pagination)

	if got.Status == nil {
		t.Error("NewPaginatedResponse().Status is nil")
	}
	if !got.Status.Success {
		t.Errorf("NewPaginatedResponse().Status.Success = %v, want %v", got.Status.Success, true)
	}
	if got.Pagination != pagination {
		t.Errorf("NewPaginatedResponse().Pagination = %v, want %v", got.Pagination, pagination)
	}
}
