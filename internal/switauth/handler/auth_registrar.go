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

package handler

import (
	"github.com/gin-gonic/gin"
)

// AuthRouteRegistrar 认证路由注册器
type AuthRouteRegistrar struct {
	controller *AuthController
}

// NewAuthRouteRegistrar 创建认证路由注册器
func NewAuthRouteRegistrar(controller *AuthController) *AuthRouteRegistrar {
	return &AuthRouteRegistrar{
		controller: controller,
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (arr *AuthRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	authGroup := rg.Group("/auth")
	{
		authGroup.POST("/login", arr.controller.Login)
		authGroup.POST("/logout", arr.controller.Logout)
		authGroup.POST("/refresh", arr.controller.RefreshToken)
		authGroup.GET("/validate", arr.controller.ValidateToken)
	}

	return nil
}

// GetName 实现 RouteRegistrar 接口
func (arr *AuthRouteRegistrar) GetName() string {
	return "auth-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (arr *AuthRouteRegistrar) GetVersion() string {
	return "root" // 认证API不需要版本前缀
}

// GetPrefix 实现 RouteRegistrar 接口
func (arr *AuthRouteRegistrar) GetPrefix() string {
	return ""
}
