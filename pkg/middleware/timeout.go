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
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware 创建一个真正的超时中间件，具备并发安全保护
// timeout: 超时时间
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 创建并发安全的响应包装器
		tw := &timeoutWriter{
			ResponseWriter: c.Writer,
			headers:        make(http.Header),
			statusCode:     http.StatusOK,
		}

		// 用于同步的通道
		done := make(chan struct{})
		panicChan := make(chan interface{}, 1)

		// 设置新的writer和request
		c.Writer = tw
		c.Request = c.Request.WithContext(ctx)

		// 在新的goroutine中执行后续处理
		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicChan <- p
				}
				close(done)
			}()
			c.Next()
		}()

		// 等待处理完成或超时
		var panicValue interface{}
		var completed bool

		for !completed {
			select {
			case <-done:
				// 检查是否有panic需要处理
				select {
				case p := <-panicChan:
					panicValue = p
				default:
					// 没有panic，正常完成
				}
				completed = true

			case p := <-panicChan:
				// 收到panic，等待goroutine完成
				panicValue = p
				select {
				case <-done:
					// goroutine已完成
					completed = true
				case <-time.After(5 * time.Second):
					// 等待goroutine完成超时，强制退出
					completed = true
				}

			case <-ctx.Done():
				// 超时处理
				tw.mu.Lock()
				if !tw.hasWritten {
					tw.timedOut = true
					// 直接写入超时响应
					tw.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
					tw.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
					tw.ResponseWriter.Write([]byte(`{"error":"Request timeout"}`))
					tw.hasWritten = true
				}
				tw.mu.Unlock()

				// 等待handler完成，但设置超时避免goroutine泄露
				select {
				case <-done:
					// goroutine正常完成
				case <-time.After(5 * time.Second):
					// 等待超时，可能存在goroutine泄露，但避免无限等待
				}

				// 检查是否有panic需要处理
				select {
				case p := <-panicChan:
					panicValue = p
				default:
					// 没有panic
				}
				completed = true
			}
		}

		// 处理panic或正常完成
		if panicValue != nil {
			// 处理panic
			tw.mu.Lock()
			if !tw.hasWritten && !tw.timedOut {
				tw.timedOut = true
				tw.mu.Unlock()

				// 返回500错误
				tw.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
				tw.ResponseWriter.WriteHeader(http.StatusInternalServerError)
				tw.ResponseWriter.Write([]byte(`{"error":"Internal server error"}`))
			} else {
				tw.mu.Unlock()
			}
			panic(panicValue)
		} else {
			// 正常完成
			tw.mu.Lock()
			defer tw.mu.Unlock()

			if !tw.hasWritten && !tw.timedOut {
				// 将缓存的headers写入真正的ResponseWriter
				tw.writeHeaders()
				tw.ResponseWriter.WriteHeader(tw.statusCode)
			}
		}
	}
}

// timeoutWriter 包装gin.ResponseWriter以处理超时和并发安全
type timeoutWriter struct {
	gin.ResponseWriter
	headers    http.Header
	statusCode int
	hasWritten bool
	timedOut   bool
	mu         sync.Mutex
}

// Header 返回header map，允许直接修改
func (tw *timeoutWriter) Header() http.Header {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.headers
}

// Write 写入响应body，并发安全
func (tw *timeoutWriter) Write(data []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut {
		// 已超时，忽略写入，返回写入的字节数以避免错误
		return len(data), nil
	}

	if !tw.hasWritten {
		// 第一次写入时，先写入headers和status
		tw.writeHeaders()
		tw.ResponseWriter.WriteHeader(tw.statusCode)
		tw.hasWritten = true
	}

	return tw.ResponseWriter.Write(data)
}

// WriteHeader 写入状态码，并发安全
func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.timedOut || tw.hasWritten {
		return
	}

	// 缓存状态码，等到真正写入时再使用
	tw.statusCode = code
}

// WriteString 实现gin.ResponseWriter接口
func (tw *timeoutWriter) WriteString(s string) (int, error) {
	return tw.Write([]byte(s))
}

// writeHeaders 将缓存的headers写入真正的ResponseWriter
func (tw *timeoutWriter) writeHeaders() {
	for k, v := range tw.headers {
		tw.ResponseWriter.Header()[k] = v
	}
}

// Status 返回当前状态码
func (tw *timeoutWriter) Status() int {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.hasWritten {
		return tw.ResponseWriter.Status()
	}
	return tw.statusCode
}

// Size 返回已写入的字节数
func (tw *timeoutWriter) Size() int {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if tw.hasWritten {
		return tw.ResponseWriter.Size()
	}
	return 0
}

// Written 返回是否已写入
func (tw *timeoutWriter) Written() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.hasWritten
}
