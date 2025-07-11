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

package stop

import "github.com/gin-gonic/gin"

// StopRouteRegistrar 停止服务路由注册器
type StopRouteRegistrar struct {
	shutdownFunc func()
}

// NewStopRouteRegistrar 创建停止服务路由注册器
func NewStopRouteRegistrar(shutdownFunc func()) *StopRouteRegistrar {
	return &StopRouteRegistrar{
		shutdownFunc: shutdownFunc,
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (srr *StopRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	handler := NewStopHandler(srr.shutdownFunc)
	rg.POST("/stop", handler.Stop)
	return nil
}

// GetName 实现 RouteRegistrar 接口
func (srr *StopRouteRegistrar) GetName() string {
	return "stop-service"
}

// GetVersion 实现 RouteRegistrar 接口
func (srr *StopRouteRegistrar) GetVersion() string {
	return "root" // 停止服务不需要版本前缀
}

// GetPrefix 实现 RouteRegistrar 接口
func (srr *StopRouteRegistrar) GetPrefix() string {
	return ""
}
