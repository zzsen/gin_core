package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"
)

// TimeoutHandler 创建一个同时处理超时和记录请求响应时长的中间件
func TimeoutHandler() gin.HandlerFunc {
	timeout := time.Duration(global.BaseConfig.Service.ApiTimeout) * time.Second
	return func(c *gin.Context) {
		// 记录请求开始时间
		startTime := time.Now()

		// 创建一个通道用于接收处理结果
		done := make(chan struct{}, 1)
		errorChan := make(chan any, 1)

		// 启动一个 goroutine 来处理请求
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 处理可能的 panic
					errorChan <- r
				}
				// 处理完成后向通道发送信号
				done <- struct{}{}
			}()
			// 调用下一个处理函数
			c.Next()
		}()

		// 创建一个定时器，设置超时时间
		select {
		case <-done:
			// 请求在超时时间内处理完成
			// 计算请求响应时长
			duration := time.Since(startTime)
			// 响应时长超过 80% 的超时时间，记录日志
			if duration > timeout*8/10 {
				logger.Warn("[timeout] Request to %s took %d ms, which is more than 80%% of the timeout (%d)", c.Request.URL.Path, duration.Milliseconds(), timeout)
			}
		case err := <-errorChan:
			// 处理 panic, 抛给exception_handler
			panic(err)
		case <-time.After(timeout):
			// 请求超时
			// 终止当前请求的处理链
			c.Abort()
			logger.Error("[timeout] Request to %s timeout (%d)", c.Request.URL.Path, timeout)
			// 返回超时响应
			response.Result(c, http.StatusRequestTimeout, nil, "Request timed out")
		}
	}
}
