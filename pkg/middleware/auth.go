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
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/discovery"
)

// AuthConfig 认证中间件配置
type AuthConfig struct {
	// WhiteList 白名单路径，这些路径不需要认证
	WhiteList []string
	// AuthServiceName 认证服务名称，用于服务发现
	AuthServiceName string
	// AuthEndpoint 认证端点，默认为 "/auth/validate"
	AuthEndpoint string
	// ServiceDiscoveryAddress 服务发现地址
	ServiceDiscoveryAddress string
}

// DefaultAuthConfig 默认认证配置
var DefaultAuthConfig = &AuthConfig{
	WhiteList: []string{
		"/health",
		"/stop",
	},
	AuthServiceName: "switauth",
	AuthEndpoint:    "/auth/validate",
}

// AuthMiddleware 创建认证中间件
// 使用默认配置的认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return AuthMiddlewareWithConfig(DefaultAuthConfig)
}

// AuthMiddlewareWithConfig 使用指定配置创建认证中间件
func AuthMiddlewareWithConfig(config *AuthConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultAuthConfig
	}

	return func(ctx *gin.Context) {
		// 检查是否在白名单中
		if isWhitelisted(ctx.Request.URL.Path, config.WhiteList) {
			ctx.Next()
			return
		}

		// 验证Authorization头
		token, err := extractTokenFromHeader(ctx.GetHeader("Authorization"))
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// 验证token
		if err := validateToken(ctx, token, config); err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx.Next()
	}
}

// AuthMiddlewareWithWhiteList 使用自定义白名单创建认证中间件
func AuthMiddlewareWithWhiteList(whiteList []string) gin.HandlerFunc {
	config := &AuthConfig{
		WhiteList:       whiteList,
		AuthServiceName: DefaultAuthConfig.AuthServiceName,
		AuthEndpoint:    DefaultAuthConfig.AuthEndpoint,
	}
	return AuthMiddlewareWithConfig(config)
}

// isWhitelisted 检查路径是否在白名单中
func isWhitelisted(path string, whiteList []string) bool {
	for _, whitelistPath := range whiteList {
		if path == whitelistPath {
			return true
		}
	}
	return false
}

// extractTokenFromHeader 从Authorization头中提取token
func extractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authentication header required")
	}

	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authentication header format")
	}

	return tokenParts[1], nil
}

// validateToken 验证token
func validateToken(ctx *gin.Context, token string, config *AuthConfig) error {
	// 获取服务发现实例
	var sd *discovery.ServiceDiscovery
	var err error

	if config.ServiceDiscoveryAddress != "" {
		sd, err = discovery.GetServiceDiscoveryByAddress(config.ServiceDiscoveryAddress)
	} else {
		sd, err = discovery.GetDefaultServiceDiscovery()
	}

	if err != nil {
		return fmt.Errorf("failed to initialize service discovery: %v", err)
	}

	// 发现认证服务
	serviceURL, err := sd.GetInstanceRoundRobin(config.AuthServiceName)
	if err != nil {
		return fmt.Errorf("failed to discover %s service: %v", config.AuthServiceName, err)
	}

	// 创建验证请求
	endpoint := config.AuthEndpoint
	if endpoint == "" {
		endpoint = "/auth/validate"
	}

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx.Request.Context(), "GET", fmt.Sprintf("%s%s", serviceURL, endpoint), nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %v", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	// 发送验证请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("validation request failed: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			// 日志记录错误但不返回
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid token")
	}

	return nil
}
