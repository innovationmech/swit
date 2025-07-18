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

package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// ExampleRateLimiter demonstrates proper usage of RateLimiter with cleanup
func ExampleRateLimiter() {
	// Create a rate limiter that allows 10 requests per minute
	rl := NewRateLimiter(10, time.Minute)

	// IMPORTANT: Always call Stop() to prevent goroutine leaks
	defer rl.Stop()

	// Use with Gin router
	router := gin.New()
	router.Use(rl.Middleware())

	// Your routes here
	router.GET("/api/data", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	fmt.Println("Rate limiter configured with proper cleanup")
	// Output: Rate limiter configured with proper cleanup
}

// ExampleNewAuthRateLimiter demonstrates auth-specific rate limiting
func ExampleNewAuthRateLimiter() {
	// Create an auth rate limiter (5 requests per minute)
	authRL := NewAuthRateLimiter()

	// IMPORTANT: Always call Stop() to prevent goroutine leaks
	defer authRL.Stop()

	// Use for authentication endpoints
	router := gin.New()
	auth := router.Group("/auth")
	auth.Use(authRL.Middleware())

	auth.POST("/login", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "login endpoint"})
	})

	fmt.Println("Auth rate limiter configured")
	// Output: Auth rate limiter configured
}

// ExampleRateLimiter_Stop demonstrates graceful shutdown
func ExampleRateLimiter_Stop() {
	rl := NewRateLimiter(5, time.Minute)

	// Simulate some usage
	rl.allow("192.168.1.1")

	// Gracefully stop the rate limiter
	rl.Stop()

	fmt.Println("Rate limiter stopped gracefully")
	// Output: Rate limiter stopped gracefully
}
