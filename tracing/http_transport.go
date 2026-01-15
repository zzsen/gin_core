// Package tracing 提供基于 OpenTelemetry 的分布式链路追踪功能
// 本文件实现了 HTTP 客户端的追踪传输层，支持跨服务追踪传播
package tracing

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TracingTransport 带链路追踪的 HTTP Transport
// 实现了 http.RoundTripper 接口，用于追踪出站 HTTP 请求
// 并将追踪上下文传播到下游服务
type TracingTransport struct {
	// Base 底层的 HTTP Transport
	Base http.RoundTripper
}

// NewTracingTransport 创建新的追踪 Transport
// 参数：
//   - base: 底层的 http.RoundTripper，如果为 nil 则使用 http.DefaultTransport
//
// 返回：
//   - *TracingTransport: 追踪 Transport 实例
func NewTracingTransport(base http.RoundTripper) *TracingTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &TracingTransport{Base: base}
}

// RoundTrip 执行 HTTP 请求并追踪
// 实现 http.RoundTripper 接口
// 该函数会：
// 1. 创建出站 HTTP 请求的 Span
// 2. 将追踪上下文注入到请求头中（用于跨服务传播）
// 3. 执行请求并记录响应状态
// 4. 如果发生错误，记录错误信息
func (t *TracingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !IsHTTPClientTracingEnabled() {
		return t.Base.RoundTrip(req)
	}

	ctx := req.Context()

	// 构建 Span 名称
	spanName := fmt.Sprintf("HTTP %s %s", req.Method, req.URL.Path)

	// 创建出站 Span
	ctx, span := StartSpan(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.HTTPMethod(req.Method),
			semconv.HTTPURL(req.URL.String()),
			semconv.NetPeerName(req.URL.Host),
			attribute.String("http.scheme", req.URL.Scheme),
			attribute.String("http.path", req.URL.Path),
		),
	)
	defer span.End()

	// 将追踪上下文注入到请求头中
	// 这样下游服务可以继续追踪链路
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	// 使用新的上下文执行请求
	req = req.WithContext(ctx)
	resp, err := t.Base.RoundTrip(req)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return resp, err
	}

	// 记录响应状态码
	span.SetAttributes(semconv.HTTPStatusCode(resp.StatusCode))

	// 标记错误状态
	if resp.StatusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	return resp, nil
}

// NewTracingHTTPClient 创建带追踪功能的 HTTP 客户端
// 返回：
//   - *http.Client: 配置了追踪 Transport 的 HTTP 客户端
func NewTracingHTTPClient() *http.Client {
	return &http.Client{
		Transport: NewTracingTransport(nil),
	}
}

// WrapHTTPClient 为现有的 HTTP 客户端添加追踪功能
// 参数：
//   - client: 现有的 HTTP 客户端
//
// 返回：
//   - *http.Client: 添加了追踪功能的 HTTP 客户端
func WrapHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return NewTracingHTTPClient()
	}
	client.Transport = NewTracingTransport(client.Transport)
	return client
}
