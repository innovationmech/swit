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

package health

import "github.com/gin-gonic/gin"

// HealthRouteRegistrar 健康检查路由注册器
type HealthRouteRegistrar struct{}

// NewHealthRouteRegistrar 创建健康检查路由注册器
func NewHealthRouteRegistrar() *HealthRouteRegistrar {
	return &HealthRouteRegistrar{}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (hrr *HealthRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	rg.GET("/health", HealthHandler)
	return nil
}

// GetName 实现 RouteRegistrar 接口
func (hrr *HealthRouteRegistrar) GetName() string {
	return "health-check"
}

// GetVersion 实现 RouteRegistrar 接口
func (hrr *HealthRouteRegistrar) GetVersion() string {
	return "root" // 健康检查不需要版本前缀
}

// GetPrefix 实现 RouteRegistrar 接口
func (hrr *HealthRouteRegistrar) GetPrefix() string {
	return ""
}
