// Copyright © 2023 jackelyj <dreamerlyj@gmail.com>
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

package repository

import (
	"context"

	"github.com/innovationmech/swit/internal/switblog/model"
	"gorm.io/gorm"
)

type BlogRepository interface {
	CreateBlog(ctx context.Context, blog *model.Blog) (*model.Blog, error)
	GetBlog(ctx context.Context, id string) (*model.Blog, error)
	UpdateBlog(ctx context.Context, blog *model.Blog) (*model.Blog, error)
	DeleteBlog(ctx context.Context, id string) error
}

type blogRepository struct {
	db *gorm.DB
}

func (br *blogRepository) CreateBlog(ctx context.Context, blog *model.Blog) (*model.Blog, error) {
	return nil, nil
}

func (br *blogRepository) GetBlog(ctx context.Context, id string) (*model.Blog, error) {
	return nil, nil
}

func (br *blogRepository) UpdateBlog(ctx context.Context, blog *model.Blog) (*model.Blog, error) {
	return nil, nil
}

func (br *blogRepository) DeleteBlog(ctx context.Context, id string) error {
	return nil
}

func NewBlogRepository(db *gorm.DB) BlogRepository {
	return &blogRepository{db: db}
}
