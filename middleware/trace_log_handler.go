// Package middleware 提供Gin框架的中间件功能
// 本文件实现了请求追踪日志处理器中间件，用于记录HTTP请求的详细信息和执行时间
package middleware

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/logger"
)

// TraceLogHandler 请求追踪日志处理器中间件
// 该中间件会：
// 1. 记录请求开始时间
// 2. 执行请求处理流程
// 3. 记录请求结束时间并计算响应时长
// 4. 收集请求的详细信息（方法、URL、状态码、客户端IP等）
// 5. 解析请求表单数据并转换为JSON格式
// 6. 获取追踪ID和请求ID
// 7. 收集错误信息
// 8. 使用结构化日志记录所有信息
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func TraceLogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 记录请求开始时间，用于计算响应时长
		startTime := time.Now()

		// 执行请求处理流程，调用后续的中间件和处理器
		c.Next()

		// 记录请求结束时间
		endTime := time.Now()

		// 计算请求执行时间，用于性能监控和分析
		responseTime := endTime.Sub(startTime)

		// 获取请求相关信息，用于日志记录和问题排查
		reqMethod := c.Request.Method                                     // 请求方式（GET、POST等）
		reqUrl := c.Request.RequestURI                                    // 请求路由路径
		statusCode := c.Writer.Status()                                   // HTTP响应状态码
		clientIP := c.ClientIP()                                          // 客户端IP地址
		header := c.GetHeader("User-Agent") + "@@" + c.GetHeader("token") // 用户代理和认证令牌的组合

		// 解析多部分表单数据，限制内存使用为128MB
		_ = c.Request.ParseMultipartForm(128)
		reqForm := c.Request.Form
		var reqJsonStr string

		// 将请求表单数据转换为 JSON 字符串，便于日志记录和分析
		if len(reqForm) > 0 {
			reqJsonByte, _ := json.Marshal(reqForm)
			reqJsonStr = string(reqJsonByte)
		}

		// 获取请求中的 requestId，用于关联同一请求的不同操作
		requestId := c.GetString("requestId")

		// 获取请求中的 traceId，用于分布式追踪
		traceId, exists := c.Get("traceId")
		if !exists {
			traceId = ""
		}

		// 获取 Gin 中间件中的错误信息，收集所有中间件产生的错误
		var errorsStr string
		for _, err := range c.Errors.Errors() {
			errorsStr += err + "; "
		}

		// 使用结构化日志记录所有请求信息，便于日志分析和问题排查
		// 敏感字段（如token）会自动脱敏
		logger.TraceWithFields(map[string]any{
			"traceId":      traceId,      // 追踪ID，用于分布式追踪
			"requestId":    requestId,    // 请求ID，用于关联同一请求的不同操作
			"statusCode":   statusCode,   // HTTP状态码，用于判断请求处理结果
			"responseTime": responseTime, // 响应时间，用于性能监控
			"clientIp":     clientIP,     // 客户端IP，用于访问控制和问题排查
			"reqMethod":    reqMethod,    // 请求方法，用于了解请求类型
			"uaToken":      header,       // 用户代理和令牌，用于用户识别和认证
			"reqUri":       reqUrl,       // 请求URI，用于路由分析
			"body":         reqJsonStr,   // 请求体数据，用于调试和审计
			"errStr":       errorsStr,    // 错误信息，用于问题排查
		}, "请求日志")
	}
}
