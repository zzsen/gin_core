// Package middleware 提供Gin框架的中间件功能
// 本文件实现了请求追踪ID处理器中间件，为每个HTTP请求生成唯一的追踪标识符
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// traceHeaders 定义了从上游请求中读取 trace ID 时依次检查的请求头，按优先级排序
var traceHeaders = []string{
	"X-Trace-ID",
	"X-Request-ID",
}

// TraceIdHandler 请求追踪ID处理器中间件
// 该中间件会：
// 1. 尝试从上游请求头中读取已有的 trace ID（支持 X-Trace-ID、X-Request-ID）
// 2. 若上游未传递 trace ID，则生成新的 UUID 作为追踪ID
// 3. 将追踪ID存储在Gin上下文中，供后续中间件和处理器使用
// 4. 将追踪ID添加到HTTP响应头中，方便客户端进行请求追踪
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func TraceIdHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 尝试从上游请求头中读取 trace ID，实现跨服务追踪传播
		traceID := extractTraceID(c)

		// 2. 若上游未传递，则生成新的 UUID
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// 3. 将 Trace ID 存储在 gin.Context 中，供后续中间件和处理器访问
		c.Set("traceId", traceID)

		// 4. 将 Trace ID 添加到响应头中，方便客户端跟踪和调试
		c.Writer.Header().Set("X-Trace-ID", traceID)

		c.Next()
	}
}

// extractTraceID 从请求头中提取 trace ID
// 按 traceHeaders 定义的优先级依次检查，返回第一个非空值
func extractTraceID(c *gin.Context) string {
	for _, header := range traceHeaders {
		if val := strings.TrimSpace(c.GetHeader(header)); val != "" {
			return val
		}
	}
	return ""
}
