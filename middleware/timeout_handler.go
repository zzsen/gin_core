// Package middleware 提供Gin框架的中间件功能
// 本文件实现了请求超时处理器中间件，用于控制API请求的执行时间并记录响应时长
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"
)

// TimeoutHandler 创建一个同时处理超时和记录请求响应时长的中间件
// 该中间件使用 context.WithTimeout 实现协作式超时控制，避免在 goroutine 中操作
// gin.Context 导致的并发安全问题。
//
// 超时上下文会传播到下游的 DB 查询、HTTP 调用等 context-aware 操作，
// 处理函数应通过 c.Request.Context() 获取上下文并传递给 I/O 操作，以实现超时自动中断。
//
// 执行流程：
// 1. 从配置中获取 API 超时时间，若为 0 则跳过超时控制
// 2. 通过 context.WithTimeout 为请求设置截止时间
// 3. 同步调用 c.Next() 执行后续处理链
// 4. 检查上下文是否超时，未写入响应时返回 408
// 5. 记录响应时长，对接近超时的请求发出警告
//
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func TimeoutHandler() gin.HandlerFunc {
	timeout := time.Duration(app.BaseConfig.Service.ApiTimeout) * time.Second

	return func(c *gin.Context) {
		// 1. 超时时间无效时跳过超时控制，直接执行后续处理链
		if timeout <= 0 {
			c.Next()
			return
		}

		// 2. 通过 context.WithTimeout 为请求设置截止时间
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		startTime := time.Now()

		// 3. 同步调用 c.Next()，在同一 goroutine 中执行，避免并发访问 gin.Context
		c.Next()

		duration := time.Since(startTime)

		// 4. 检查上下文是否超时
		if ctx.Err() == context.DeadlineExceeded {
			if !c.Writer.Written() {
				c.Abort()
				logger.Error("[timeout] Request to %s timeout (%v)", c.Request.URL.Path, timeout)
				response.Result(c, http.StatusRequestTimeout, nil, "Request timed out")
			} else {
				logger.Error("[timeout] Request to %s timeout (%v), but response already written", c.Request.URL.Path, timeout)
			}
			return
		}

		// 5. 响应时长超过 80% 的超时时间，记录警告日志
		if duration > timeout*8/10 {
			logger.Warn("[timeout] Request to %s took %d ms, which is more than 80%% of the timeout (%v)", c.Request.URL.Path, duration.Milliseconds(), timeout)
		}
	}
}
