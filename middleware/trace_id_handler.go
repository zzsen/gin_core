// Package middleware 提供Gin框架的中间件功能
// 本文件实现了请求追踪ID处理器中间件，为每个HTTP请求生成唯一的追踪标识符
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TraceIdHandler 请求追踪ID处理器中间件
// 该中间件会：
// 1. 为每个HTTP请求生成唯一的UUID作为追踪ID
// 2. 将追踪ID存储在Gin上下文中，供后续中间件和处理器使用
// 3. 将追踪ID添加到HTTP响应头中，方便客户端进行请求追踪
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func TraceIdHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用UUID库生成唯一的 Trace ID
		traceID := uuid.New().String()

		// 将 Trace ID 存储在 gin.Context 中，供后续中间件和处理器访问
		c.Set("traceId", traceID)

		// 将 Trace ID 添加到响应头中，方便客户端跟踪和调试
		// 使用标准的X-Trace-ID头字段，符合HTTP扩展头的命名规范
		c.Writer.Header().Set("X-Trace-ID", traceID)

		// 继续执行下一个中间件或处理器
		c.Next()
	}
}
