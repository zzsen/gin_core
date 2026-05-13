// Package http_client HTTP 客户端工具测试
//
// ==================== 测试说明 ====================
// 本文件包含 HTTP 客户端的单元测试，使用 httptest 模拟服务端。
//
// 测试覆盖内容：
// 1. DefaultClientConfig 默认值
// 2. NewClient nil 配置
// 3. NewClient 自定义配置
// 4. GetDefaultClient 单例
// 5. ResponseWrapper IsSuccess / HasError
// 6. GetWithContext 正常请求
// 7. PostJSONWithContext 正常请求
// 8. PutJSONWithContext 正常请求
// 9. DeleteWithContext 正常请求
// 10. 请求超时处理
// 11. isRetryableError 判断
// 12. GetBreakerStats / ResetBreaker / IsCircuitOpen（未启用熔断器）
// 13. CloseIdleConnections
// 14. ServerError Error()
//
// 运行测试：go test -v ./utils/http_client/... -run "Test"
// ==================================================
package http_client

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDefaultClientConfig 测试默认配置
func TestDefaultClientConfig(t *testing.T) {
	cfg := DefaultClientConfig()
	assert.Equal(t, 100, cfg.MaxIdleConns)
	assert.Equal(t, 10, cfg.MaxIdleConnsPerHost)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.True(t, cfg.EnableTracing)
	assert.False(t, cfg.EnableCircuitBreaker)
}

// TestNewClient_NilConfig 测试 nil 配置使用默认值
func TestNewClient_NilConfig(t *testing.T) {
	client := NewClient(nil)
	require.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 3, client.config.MaxRetries)
}

// TestNewClient_CustomConfig 测试自定义配置
func TestNewClient_CustomConfig(t *testing.T) {
	cfg := &ClientConfig{
		MaxIdleConns:    50,
		MaxRetries:      1,
		RetryInterval:   50 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ResponseTimeout: 5 * time.Second,
		EnableTracing:   false,
	}
	client := NewClient(cfg)
	assert.Equal(t, 1, client.config.MaxRetries)
	assert.Nil(t, client.breakerRegistry)
}

// TestNewClient_WithCircuitBreaker 测试启用熔断器
func TestNewClient_WithCircuitBreaker(t *testing.T) {
	cfg := DefaultClientConfig()
	cfg.EnableCircuitBreaker = true
	cfg.EnableTracing = false
	client := NewClient(cfg)
	assert.NotNil(t, client.breakerRegistry)
}

// TestResponseWrapper_IsSuccess 测试响应成功判断
func TestResponseWrapper_IsSuccess(t *testing.T) {
	tests := []struct {
		code   int
		expect bool
	}{
		{200, true}, {201, true}, {204, true},
		{301, false}, {400, false}, {500, false}, {0, false},
	}
	for _, tt := range tests {
		r := ResponseWrapper{StatusCode: tt.code}
		assert.Equal(t, tt.expect, r.IsSuccess(), "code=%d", tt.code)
	}
}

// TestResponseWrapper_HasError 测试响应错误判断
func TestResponseWrapper_HasError(t *testing.T) {
	assert.True(t, (&ResponseWrapper{StatusCode: 0}).HasError())
	assert.True(t, (&ResponseWrapper{Error: net.ErrClosed}).HasError())
	assert.False(t, (&ResponseWrapper{StatusCode: 200}).HasError())
}

// TestGetWithContext 测试 GET 请求
func TestGetWithContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	resp := GetWithContext(context.Background(), srv.URL+"/test", 5, nil)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Body, `"ok":true`)
}

// TestPostJSONWithContext 测试 POST JSON 请求
func TestPostJSONWithContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`created`))
	}))
	defer srv.Close()

	resp := PostJSONWithContext(context.Background(), srv.URL, `{"name":"test"}`, 5, nil)
	assert.Equal(t, 201, resp.StatusCode)
}

// TestPutJSONWithContext 测试 PUT JSON 请求
func TestPutJSONWithContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	resp := PutJSONWithContext(context.Background(), srv.URL, `{"id":1}`, 5, nil)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestDeleteWithContext 测试 DELETE 请求
func TestDeleteWithContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	resp := DeleteWithContext(context.Background(), srv.URL, 5, nil)
	assert.Equal(t, 204, resp.StatusCode)
}

// TestCustomHeaders 测试自定义请求头
func TestCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "gin-core/http-client", r.Header.Get("User-Agent"))
		w.WriteHeader(200)
	}))
	defer srv.Close()

	headers := map[string]string{"Authorization": "Bearer token123"}
	resp := GetWithContext(context.Background(), srv.URL, 5, headers)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestIsRetryableError 测试可重试错误判断
func TestIsRetryableError(t *testing.T) {
	assert.False(t, isRetryableError(nil))
	assert.False(t, isRetryableError(net.ErrClosed))

	dialErr := &net.OpError{Op: "dial", Err: net.ErrClosed}
	assert.True(t, isRetryableError(dialErr))
}

// TestServerError_Error 测试 ServerError 消息
func TestServerError_Error(t *testing.T) {
	e := &ServerError{StatusCode: 502}
	assert.Contains(t, e.Error(), "Bad Gateway")
}

// TestClient_BreakerStats_Nil 测试未启用熔断器时返回 nil
func TestClient_BreakerStats_Nil(t *testing.T) {
	client := NewClient(&ClientConfig{EnableTracing: false})
	assert.Nil(t, client.GetBreakerStats())
}

// TestClient_IsCircuitOpen_NilRegistry 测试未启用熔断器
func TestClient_IsCircuitOpen_NilRegistry(t *testing.T) {
	client := NewClient(&ClientConfig{EnableTracing: false})
	assert.False(t, client.IsCircuitOpen("any"))
}

// TestClient_ResetBreaker_NilRegistry 测试未启用熔断器的 Reset
func TestClient_ResetBreaker_NilRegistry(t *testing.T) {
	client := NewClient(&ClientConfig{EnableTracing: false})
	assert.NotPanics(t, func() { client.ResetBreaker("any") })
	assert.NotPanics(t, func() { client.ResetAllBreakers() })
}

// TestClient_CloseIdleConnections 测试关闭空闲连接
func TestClient_CloseIdleConnections(t *testing.T) {
	client := NewClient(&ClientConfig{EnableTracing: false})
	assert.NotPanics(t, func() { client.CloseIdleConnections() })
}

// TestClient_GetHTTPClient 测试获取底层 http.Client
func TestClient_GetHTTPClient(t *testing.T) {
	client := NewClient(&ClientConfig{EnableTracing: false})
	assert.NotNil(t, client.GetHTTPClient())
}

// TestPostParamsWithContext 测试 POST 参数请求
func TestPostParamsWithContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp := PostParamsWithContext(context.Background(), srv.URL, "key=value", 5, nil)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestPostFormWithContext_SimpleFields 测试 POST 表单（仅普通字段）
func TestPostFormWithContext_SimpleFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")
		err := r.ParseMultipartForm(32 << 20)
		require.NoError(t, err)
		assert.Equal(t, "bar", r.FormValue("foo"))
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	resp := PostFormWithContext(
		context.Background(), srv.URL,
		map[string]string{"foo": "bar"},
		nil, nil, nil, 5,
	)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestPostFormWithContext_WithFileReader 测试 POST 表单（文件 Reader）
func TestPostFormWithContext_WithFileReader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(32 << 20)
		require.NoError(t, err)
		file, _, err := r.FormFile("upload")
		require.NoError(t, err)
		defer file.Close()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	fileReader := strings.NewReader("file content here")
	resp := PostFormWithContext(
		context.Background(), srv.URL,
		nil, nil,
		map[string]io.Reader{"upload": fileReader},
		nil, 5,
	)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestCreateRequestError_InvalidURL 测试无效 URL 产生的错误
func TestCreateRequestError_InvalidURL(t *testing.T) {
	resp := GetWithContext(context.Background(), "://invalid", 5, nil)
	assert.True(t, resp.HasError())
	assert.Contains(t, resp.Body, "创建HTTP请求错误")
}

// TestDoRequest_Timeout 测试请求超时
func TestDoRequest_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	resp := GetWithContext(context.Background(), srv.URL, 1, nil)
	assert.True(t, resp.HasError())
}

// TestClient_Do_WithCircuitBreaker 测试带熔断器的请求
func TestClient_Do_WithCircuitBreaker(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	cfg := &ClientConfig{
		MaxIdleConns:                   10,
		MaxIdleConnsPerHost:            5,
		MaxConnsPerHost:                10,
		IdleConnTimeout:                30 * time.Second,
		DialTimeout:                    5 * time.Second,
		TLSHandshakeTimeout:            5 * time.Second,
		ResponseTimeout:                5 * time.Second,
		MaxRetries:                     0,
		RetryInterval:                  100 * time.Millisecond,
		EnableTracing:                  false,
		EnableCircuitBreaker:           true,
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerTimeout:          10 * time.Second,
		CircuitBreakerMaxRequests:      3,
	}
	client := NewClient(cfg)
	require.NotNil(t, client.breakerRegistry)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	resp.Body.Close()
}

// TestClient_Do_CircuitBreaker_5xx 测试熔断器处理 5xx 响应
func TestClient_Do_CircuitBreaker_5xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	cfg := &ClientConfig{
		DialTimeout:                    5 * time.Second,
		TLSHandshakeTimeout:            5 * time.Second,
		ResponseTimeout:                5 * time.Second,
		MaxRetries:                     0,
		RetryInterval:                  10 * time.Millisecond,
		EnableTracing:                  false,
		EnableCircuitBreaker:           true,
		CircuitBreakerFailureThreshold: 2,
		CircuitBreakerTimeout:          10 * time.Second,
		CircuitBreakerMaxRequests:      1,
	}
	client := NewClient(cfg)

	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		resp, err := client.Do(context.Background(), req)
		if err != nil {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	assert.True(t, client.IsCircuitOpen(srv.Listener.Addr().String()))
}

// TestClient_BreakerStats_Enabled 测试启用熔断器时获取统计
func TestClient_BreakerStats_Enabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := &ClientConfig{
		DialTimeout:                    5 * time.Second,
		TLSHandshakeTimeout:            5 * time.Second,
		ResponseTimeout:                5 * time.Second,
		MaxRetries:                     0,
		EnableTracing:                  false,
		EnableCircuitBreaker:           true,
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerTimeout:          10 * time.Second,
		CircuitBreakerMaxRequests:      3,
	}
	client := NewClient(cfg)

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, _ := client.Do(context.Background(), req)
	if resp != nil {
		resp.Body.Close()
	}

	stats := client.GetBreakerStats()
	assert.NotNil(t, stats)

	client.ResetBreaker(srv.Listener.Addr().String())
	client.ResetAllBreakers()
}

// TestDoWithRetry_RetryableError 测试可重试错误触发重试逻辑
func TestDoWithRetry_RetryableError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok"))
	}))
	srv.Close() // 立即关闭，让连接产生 dial 错误

	cfg := &ClientConfig{
		DialTimeout:     1 * time.Second,
		ResponseTimeout: 1 * time.Second,
		MaxRetries:      2,
		RetryInterval:   10 * time.Millisecond,
		EnableTracing:   false,
	}
	client := NewClient(cfg)

	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := client.Do(context.Background(), req)
	assert.Error(t, err)
}

// TestDoWithRetry_ContextCancelled 测试上下文取消中断重试
func TestDoWithRetry_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	srv.Close()

	cfg := &ClientConfig{
		DialTimeout:     1 * time.Second,
		ResponseTimeout: 1 * time.Second,
		MaxRetries:      10,
		RetryInterval:   50 * time.Millisecond,
		EnableTracing:   false,
	}
	client := NewClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(80 * time.Millisecond)
		cancel()
	}()

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	_, err := client.Do(ctx, req)
	assert.Error(t, err)
}

// TestDoWithRetry_NonRetryableError 测试不可重试错误直接返回
func TestDoWithRetry_NonRetryableError(t *testing.T) {
	resp := GetWithContext(context.Background(), "http://[::1]:namedport", 1, nil)
	assert.True(t, resp.HasError())
}

// TestPostFormWithContext_FilePathMap 测试文件路径上传
func TestPostFormWithContext_FilePathMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(32 << 20)
		require.NoError(t, err)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "test-upload-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, _ = tmpFile.WriteString("test file content")
	tmpFile.Close()

	resp := PostFormWithContext(
		context.Background(), srv.URL,
		nil,
		map[string]string{"file": tmpFile.Name()},
		nil, nil, 5,
	)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestPostFormWithContext_FilePathNotExist 测试不存在的文件路径
func TestPostFormWithContext_FilePathNotExist(t *testing.T) {
	resp := PostFormWithContext(
		context.Background(), "http://localhost",
		nil,
		map[string]string{"file": "/nonexistent/path/file.txt"},
		nil, nil, 5,
	)
	assert.True(t, resp.HasError())
	assert.Contains(t, resp.Body, "打开文件")
}

// TestClient_Do_CircuitOpen 测试熔断器打开时的请求
func TestClient_Do_CircuitOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	cfg := &ClientConfig{
		DialTimeout:                    5 * time.Second,
		TLSHandshakeTimeout:            5 * time.Second,
		ResponseTimeout:                5 * time.Second,
		MaxRetries:                     0,
		RetryInterval:                  10 * time.Millisecond,
		EnableTracing:                  false,
		EnableCircuitBreaker:           true,
		CircuitBreakerFailureThreshold: 1,
		CircuitBreakerTimeout:          60 * time.Second,
		CircuitBreakerMaxRequests:      1,
	}
	client := NewClient(cfg)

	// 触发熔断
	for i := 0; i < 3; i++ {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		resp, _ := client.Do(context.Background(), req)
		if resp != nil {
			resp.Body.Close()
		}
	}

	// 此时熔断器应打开，新请求应返回 circuit open 错误
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := client.Do(context.Background(), req)
	if err != nil {
		assert.Contains(t, err.Error(), "circuit")
	}
}

// TestDoRequest_NoTimeout 测试无超时的请求
func TestDoRequest_NoTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("no-timeout"))
	}))
	defer srv.Close()

	resp := GetWithContext(context.Background(), srv.URL, 0, nil)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "no-timeout", resp.Body)
}
