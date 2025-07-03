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

package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	middleware2 "github.com/innovationmech/swit/pkg/middleware"
)

// GlobalMiddlewareRegistrar 全局中间件注册器
type GlobalMiddlewareRegistrar struct{}

// NewGlobalMiddlewareRegistrar 创建全局中间件注册器
func NewGlobalMiddlewareRegistrar() *GlobalMiddlewareRegistrar {
	return &GlobalMiddlewareRegistrar{}
}

// RegisterMiddleware 实现 MiddlewareRegistrar 接口
func (gmr *GlobalMiddlewareRegistrar) RegisterMiddleware(router *gin.Engine) error {
	// Register timeout middleware first for request timeout control
	timeoutConfig := DefaultTimeoutConfig()
	timeoutConfig.Timeout = 30 * time.Second
	timeoutConfig.SkipPaths = []string{"/health", "/metrics", "/debug", "/stop"}
	router.Use(TimeoutWithConfig(timeoutConfig))

	// Register other global middlewares
	router.Use(middleware2.Logger(), middleware2.CORSMiddleware())
	return nil
}

// GetName 实现 MiddlewareRegistrar 接口
func (gmr *GlobalMiddlewareRegistrar) GetName() string {
	return "global-middleware"
}

// GetPriority 实现 MiddlewareRegistrar 接口
func (gmr *GlobalMiddlewareRegistrar) GetPriority() int {
	return 1 // 最高优先级
}
