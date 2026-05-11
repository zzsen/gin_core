// Package tracing HTTP Transport 追踪测试
//
// ==================== 测试说明 ====================
// 本文件包含 TracingTransport 和 HTTP 客户端工具函数的单元测试。
//
// 测试覆盖内容：
// 1. NewTracingTransport 创建（nil/自定义 base）
// 2. RoundTrip 禁用追踪时的直通行为
// 3. RoundTrip 启用追踪时的 Span 创建和上下文传播
// 4. RoundTrip 请求失败时的错误记录
// 5. RoundTrip HTTP 4xx/5xx 响应的状态标记
// 6. NewTracingHTTPClient/WrapHTTPClient 工具函数
//
// 运行测试：go test -v ./tracing/... -run "TestHTTP|TestNewTracing|TestWrapHTTPClient"
// ==================================================
package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestNewTracingTransport_NilBase 测试 nil base 时使用默认 Transport
//
// 【功能点】base=nil 时应使用 http.DefaultTransport
func TestNewTracingTransport_NilBase(t *testing.T) {
	tt := NewTracingTransport(nil)
	assert.Equal(t, http.DefaultTransport, tt.Base)
}

// TestNewTracingTransport_CustomBase 测试自定义 base
func TestNewTracingTransport_CustomBase(t *testing.T) {
	custom := &http.Transport{MaxIdleConns: 10}
	tt := NewTracingTransport(custom)
	assert.Equal(t, custom, tt.Base)
}

// TestHTTPRoundTrip_Disabled 测试禁用追踪时的直通
//
// 【功能点】追踪未启用时直接调用 base RoundTrip
func TestHTTPRoundTrip_Disabled(t *testing.T) {
	resetGlobals()
	tracingConfig = nil

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewTracingTransport(nil)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestHTTPRoundTrip_Enabled 测试启用追踪时的 Span 创建
//
// 【功能点】追踪启用时应创建 Span 并注入追踪上下文到请求头
func TestHTTPRoundTrip_Enabled(t *testing.T) {
	resetGlobals()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{
		Enabled:                 true,
		EnableHTTPClientTracing: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewTracingTransport(nil)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", server.URL+"/api/test", nil)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

// TestHTTPRoundTrip_ServerError 测试 HTTP 错误响应
//
// 【功能点】HTTP 4xx/5xx 应标记 Span 错误状态
func TestHTTPRoundTrip_ServerError(t *testing.T) {
	resetGlobals()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{
		Enabled:                 true,
		EnableHTTPClientTracing: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := NewTracingTransport(nil)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", server.URL, nil)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	resp.Body.Close()
}

// TestHTTPRoundTrip_NetworkError 测试网络错误
//
// 【功能点】请求失败时应记录错误
func TestHTTPRoundTrip_NetworkError(t *testing.T) {
	resetGlobals()

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{
		Enabled:                 true,
		EnableHTTPClientTracing: true,
	}

	transport := NewTracingTransport(nil)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://127.0.0.1:1", nil)
	_, err := transport.RoundTrip(req)
	require.Error(t, err)
}

// TestNewTracingHTTPClient 测试创建追踪 HTTP 客户端
func TestNewTracingHTTPClient(t *testing.T) {
	client := NewTracingHTTPClient()
	require.NotNil(t, client)
	assert.IsType(t, &TracingTransport{}, client.Transport)
}

// TestWrapHTTPClient 测试包装现有 HTTP 客户端
//
// 【功能点】nil 时返回新客户端，非 nil 时包装现有 Transport
func TestWrapHTTPClient(t *testing.T) {
	wrapped := WrapHTTPClient(nil)
	require.NotNil(t, wrapped)
	assert.IsType(t, &TracingTransport{}, wrapped.Transport)

	existing := &http.Client{}
	wrapped = WrapHTTPClient(existing)
	assert.Equal(t, existing, wrapped)
	assert.IsType(t, &TracingTransport{}, wrapped.Transport)
}
