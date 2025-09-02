// Package middleware 提供Gin框架的中间件功能
// 本文件实现了请求超时处理器中间件，用于控制API请求的执行时间并记录响应时长
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"
)

// TimeoutHandler 创建一个同时处理超时和记录请求响应时长的中间件
// 该中间件会：
// 1. 从配置中获取API超时时间
// 2. 记录请求开始时间
// 3. 在独立的goroutine中处理请求
// 4. 监控请求执行时间，超时时自动终止
// 5. 记录响应时长，对接近超时的请求发出警告
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func TimeoutHandler() gin.HandlerFunc {
	// 从配置中获取API超时时间，转换为time.Duration类型
	timeout := time.Duration(app.BaseConfig.Service.ApiTimeout) * time.Second

	return func(c *gin.Context) {
		// 记录请求开始时间，用于计算响应时长
		startTime := time.Now()

		// 创建通道用于接收处理结果和错误信息
		done := make(chan struct{}, 1) // 处理完成信号通道
		errorChan := make(chan any, 1) // 错误信息通道

		// 启动一个 goroutine 来处理请求，避免阻塞主流程
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 处理可能的 panic，将错误信息发送到错误通道
					errorChan <- r
				}
				// 处理完成后向完成通道发送信号
				done <- struct{}{}
			}()
			// 调用下一个处理函数，继续执行请求处理链
			c.Next()
		}()

		// 使用select语句监控多个通道，实现超时控制
		select {
		case <-done:
			// 请求在超时时间内处理完成
			// 计算请求响应时长
			duration := time.Since(startTime)
			// 响应时长超过 80% 的超时时间，记录警告日志
			if duration > timeout*8/10 {
				logger.Warn("[timeout] Request to %s took %d ms, which is more than 80%% of the timeout (%d)", c.Request.URL.Path, duration.Milliseconds(), timeout)
			}
		case err := <-errorChan:
			// 处理 panic, 抛给exception_handler中间件处理
			panic(err)
		case <-time.After(timeout):
			// 请求超时，触发超时处理逻辑
			// 终止当前请求的处理链，不再执行后续中间件和处理器
			c.Abort()
			// 记录超时错误日志
			logger.Error("[timeout] Request to %s timeout (%d)", c.Request.URL.Path, timeout)
			// 返回超时响应，使用统一的响应格式
			response.Result(c, http.StatusRequestTimeout, nil, "Request timed out")
		}
	}
}
