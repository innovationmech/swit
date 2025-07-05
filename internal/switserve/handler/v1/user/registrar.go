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

package user

import (
	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/middleware"
)

// UserRouteRegistrar 用户路由注册器
type UserRouteRegistrar struct {
	controller *UserController
}

// NewUserRouteRegistrar 创建用户路由注册器
func NewUserRouteRegistrar() *UserRouteRegistrar {
	return &UserRouteRegistrar{
		controller: NewUserController(),
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (urr *UserRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	// 需要认证的用户API路由
	userGroup := rg.Group("/users")
	// 配置认证中间件，包含 switserve 特定的白名单
	authWhiteList := []string{"/users/create", "/health", "/stop"}
	userGroup.Use(middleware.AuthMiddlewareWithWhiteList(authWhiteList))
	{
		userGroup.POST("/create", urr.controller.CreateUser)
		userGroup.GET("/username/:username", urr.controller.GetUserByUsername)
		userGroup.GET("/email/:email", urr.controller.GetUserByEmail)
		userGroup.DELETE("/:id", urr.controller.DeleteUser)
	}

	return nil
}

// GetName 实现 RouteRegistrar 接口
func (urr *UserRouteRegistrar) GetName() string {
	return "user-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (urr *UserRouteRegistrar) GetVersion() string {
	return "v1"
}

// GetPrefix 实现 RouteRegistrar 接口
func (urr *UserRouteRegistrar) GetPrefix() string {
	return ""
}

// UserInternalRouteRegistrar 用户内部API路由注册器
type UserInternalRouteRegistrar struct {
	controller *UserController
}

// NewUserInternalRouteRegistrar 创建用户内部API路由注册器
func NewUserInternalRouteRegistrar() *UserInternalRouteRegistrar {
	return &UserInternalRouteRegistrar{
		controller: NewUserController(),
	}
}

// RegisterRoutes 实现 RouteRegistrar 接口
func (uirr *UserInternalRouteRegistrar) RegisterRoutes(rg *gin.RouterGroup) error {
	// 内部API路由，不需要认证中间件
	internal := rg.Group("/internal")
	{
		internal.POST("/validate-user", uirr.controller.ValidateUserCredentials)
	}

	return nil
}

// GetName 实现 RouteRegistrar 接口
func (uirr *UserInternalRouteRegistrar) GetName() string {
	return "user-internal-api"
}

// GetVersion 实现 RouteRegistrar 接口
func (uirr *UserInternalRouteRegistrar) GetVersion() string {
	return "root" // 不需要版本前缀
}

// GetPrefix 实现 RouteRegistrar 接口
func (uirr *UserInternalRouteRegistrar) GetPrefix() string {
	return ""
}
