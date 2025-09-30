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

package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/innovationmech/swit/pkg/middleware"
)

func main() {
	// 创建 Gin 路由
	router := gin.Default()

	// 示例 1: 使用默认 Token Bucket 中间件
	router.GET("/api/v1/default", middleware.TokenBucketMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "使用默认 token bucket 配置",
		})
	})

	// 示例 2: 自定义 Token Bucket 配置
	tokenBucketConfig := middleware.ThrottleConfig{
		Strategy: middleware.ThrottleStrategyTokenBucket,
		Capacity: 10,  // 允许 10 个突发请求
		Rate:     5.0, // 每秒补充 5 个令牌
	}
	tokenBucketMiddleware, err := middleware.NewThrottleMiddleware(tokenBucketConfig)
	if err != nil {
		log.Fatalf("创建 token bucket 中间件失败: %v", err)
	}
	defer tokenBucketMiddleware.Stop()

	router.GET("/api/v1/token-bucket", tokenBucketMiddleware.Handler(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "使用自定义 token bucket 配置",
			"config": gin.H{
				"capacity": 10,
				"rate":     5.0,
			},
		})
	})

	// 示例 3: 使用 Leaky Bucket 策略
	leakyBucketConfig := middleware.ThrottleConfig{
		Strategy: middleware.ThrottleStrategyLeakyBucket,
		Capacity: 20,   // 最大队列大小 20
		Rate:     10.0, // 每秒处理 10 个请求
	}
	leakyBucketMiddleware, err := middleware.NewThrottleMiddleware(leakyBucketConfig)
	if err != nil {
		log.Fatalf("创建 leaky bucket 中间件失败: %v", err)
	}
	defer leakyBucketMiddleware.Stop()

	router.GET("/api/v1/leaky-bucket", leakyBucketMiddleware.Handler(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "使用 leaky bucket 配置",
			"config": gin.H{
				"capacity": 20,
				"rate":     10.0,
			},
		})
	})

	// 示例 4: 基于用户 ID 的限流（自定义 KeyFunc）
	userBasedConfig := middleware.ThrottleConfig{
		Strategy: middleware.ThrottleStrategyTokenBucket,
		Capacity: 30,
		Rate:     10.0,
		KeyFunc: func(c *gin.Context) string {
			// 使用 Authorization header 或 query 参数中的 user_id
			userID := c.GetHeader("X-User-ID")
			if userID == "" {
				userID = c.Query("user_id")
			}
			if userID == "" {
				userID = c.ClientIP() // 回退到 IP
			}
			return userID
		},
	}
	userBasedMiddleware, err := middleware.NewThrottleMiddleware(userBasedConfig)
	if err != nil {
		log.Fatalf("创建用户限流中间件失败: %v", err)
	}
	defer userBasedMiddleware.Stop()

	router.GET("/api/v1/user-based", userBasedMiddleware.Handler(), func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = c.Query("user_id")
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "基于用户的限流",
			"user_id": userID,
		})
	})

	// 示例 5: 严格的 API 限流（用于敏感操作）
	strictConfig := middleware.ThrottleConfig{
		Strategy:        middleware.ThrottleStrategyLeakyBucket,
		Capacity:        5,
		Rate:            1.0, // 每秒只允许 1 个请求
		CleanupInterval: 2 * time.Minute,
		MaxIdleTime:     5 * time.Minute,
		ErrorHandler: func(c *gin.Context) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate_limit_exceeded",
				"message":     "请求过于频繁，请稍后再试",
				"retry_after": 1,
			})
			c.Abort()
		},
	}
	strictMiddleware, err := middleware.NewThrottleMiddleware(strictConfig)
	if err != nil {
		log.Fatalf("创建严格限流中间件失败: %v", err)
	}
	defer strictMiddleware.Stop()

	router.POST("/api/v1/sensitive", strictMiddleware.Handler(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "敏感操作成功",
		})
	})

	// 示例 6: 路由组限流
	apiV2 := router.Group("/api/v2")
	{
		// 为整个路由组应用限流
		groupMiddleware, err := middleware.NewTokenBucketMiddleware(50, 20.0)
		if err != nil {
			log.Fatalf("创建路由组中间件失败: %v", err)
		}
		defer groupMiddleware.Stop()
		apiV2.Use(groupMiddleware.Handler())

		apiV2.GET("/users", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []string{"user1", "user2"}})
		})

		apiV2.GET("/posts", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []string{"post1", "post2"}})
		})
	}

	// 健康检查端点（不限流）
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// 打印使用说明
	log.Println("服务器启动在 http://localhost:8080")
	log.Println("\n可用端点:")
	log.Println("  GET  /api/v1/default        - 默认 token bucket 配置")
	log.Println("  GET  /api/v1/token-bucket   - 自定义 token bucket (10 容量, 5 req/s)")
	log.Println("  GET  /api/v1/leaky-bucket   - Leaky bucket (20 容量, 10 req/s)")
	log.Println("  GET  /api/v1/user-based     - 基于用户 ID 的限流")
	log.Println("  POST /api/v1/sensitive      - 严格限流 (1 req/s)")
	log.Println("  GET  /api/v2/users          - 路由组限流")
	log.Println("  GET  /api/v2/posts          - 路由组限流")
	log.Println("  GET  /health                - 健康检查（不限流）")
	log.Println("\n测试命令:")
	log.Println("  # 测试 token bucket:")
	log.Println("  for i in {1..15}; do curl http://localhost:8080/api/v1/token-bucket; echo; done")
	log.Println("\n  # 测试基于用户的限流:")
	log.Println("  curl -H 'X-User-ID: user123' http://localhost:8080/api/v1/user-based")
	log.Println("\n  # 测试严格限流:")
	log.Println("  for i in {1..5}; do curl -X POST http://localhost:8080/api/v1/sensitive; echo; done")

	// 启动服务器
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
