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

package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/internal/switserve/config"
)

// 创建一个白名单路径切片
var whiteList = []string{
	"/users/create",
	"/health",
	"/stop",
}

// 修改 AuthMiddleware 函数
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 检查当前路径是否在白名单中
		for _, path := range whiteList {
			if ctx.Request.URL.Path == path {
				ctx.Next()
				return
			}
		}

		// 如果不在白名单中，进行认证
		tokenString := ctx.GetHeader("Authorization")
		if tokenString == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "需要认证头"})
			return
		}
		cfg := config.GetConfig()
		resp, err := http.Get(cfg.AuthServer + "/validate")
		if err != nil || resp.StatusCode != http.StatusOK {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
			return
		}
		ctx.Next()
	}
}
