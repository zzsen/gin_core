// Package middleware 提供Gin框架的中间件功能
// 本文件实现了 OpenTelemetry 链路追踪中间件，为每个 HTTP 请求创建追踪 Span
package middleware

import (
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/zzsen/gin_core/tracing"
)

// OtelTraceHandler OpenTelemetry 链路追踪中间件
// 该中间件会：
// 1. 从请求头中提取上游服务的追踪上下文（支持 W3C Trace Context 标准）
// 2. 为当前请求创建新的 Span
// 3. 将追踪信息存储到 Gin 上下文中
// 4. 将追踪 ID 添加到响应头中
// 5. 记录请求的关键信息（方法、路径、状态码等）
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func OtelTraceHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查追踪是否已启用
		if !tracing.IsEnabled() {
			c.Next()
			return
		}

		// 从请求头提取上游的追踪上下文
		// 支持 W3C Trace Context 标准（traceparent, tracestate 头）
		propagator := otel.GetTextMapPropagator()
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 确定 Span 名称
		// 优先使用完整路由路径（带参数模板），如 /api/users/:id
		// 如果没有匹配的路由，则使用原始请求路径
		spanName := c.FullPath()
		if spanName == "" {
			spanName = c.Request.URL.Path
		}

		// 创建当前请求的 Span
		ctx, span := tracing.StartSpan(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				// HTTP 语义属性
				semconv.HTTPMethod(c.Request.Method),
				semconv.HTTPURL(c.Request.URL.String()),
				semconv.HTTPRoute(c.FullPath()),
				semconv.HTTPScheme(getScheme(c)),
				semconv.HTTPTarget(c.Request.URL.RequestURI()),
				// 网络属性
				attribute.String("net.peer.ip", c.ClientIP()),
				semconv.NetHostName(c.Request.Host),
				// 自定义属性
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.request_id", c.GetString("requestId")),
			),
		)
		defer span.End()

		// 更新请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 获取追踪 ID 和 Span ID
		traceID := tracing.GetTraceID(ctx)
		spanID := tracing.GetSpanID(ctx)

		// 向后兼容：将追踪信息存储到 Gin 上下文中
		// 这样其他中间件和处理器可以通过 c.Get("traceId") 获取
		c.Set("traceId", traceID)
		c.Set("spanId", spanID)

		// 设置响应头，方便客户端进行请求追踪
		c.Writer.Header().Set("X-Trace-ID", traceID)
		c.Writer.Header().Set("X-Span-ID", spanID)

		// 继续处理请求
		c.Next()

		// 记录响应状态
		statusCode := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCode(statusCode))

		// 记录响应体大小
		span.SetAttributes(attribute.Int("http.response_content_length", c.Writer.Size()))

		// 根据状态码设置 Span 状态
		if statusCode >= 500 {
			span.SetStatus(codes.Error, "Internal Server Error")
		} else if statusCode >= 400 {
			span.SetStatus(codes.Error, "Client Error")
		}

		// 记录错误信息（如果有）
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
			span.SetStatus(codes.Error, c.Errors.String())
		}
	}
}

// getScheme 获取请求的协议方案（http 或 https）
func getScheme(c *gin.Context) string {
	// 检查 X-Forwarded-Proto 头（用于反向代理场景）
	if scheme := c.GetHeader("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}

	// 检查 TLS 连接
	if c.Request.TLS != nil {
		return "https"
	}

	return "http"
}
