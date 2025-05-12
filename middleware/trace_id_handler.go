package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TraceIdHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成唯一的 Trace ID
		traceID := uuid.New().String()
		// 将 Trace ID 存储在 gin.Context 中
		c.Set("traceId", traceID)
		// 将 Trace ID 添加到响应头中，方便客户端跟踪
		c.Writer.Header().Set("X-Trace-ID", traceID)
		c.Next()
	}
}
