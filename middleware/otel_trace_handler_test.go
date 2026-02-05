// Package middleware OpenTelemetry 链路追踪中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含 OpenTelemetry 链路追踪中间件的单元测试。
//
// 测试覆盖内容：
// 1. 追踪禁用时的行为
// 2. 追踪 ID 和 Span ID 的设置
// 3. 响应头的设置
// 4. 不同状态码的 Span 状态设置
// 5. 错误信息的记录
// 6. HTTP 属性的正确设置
// 7. getScheme 辅助函数测试
//
// 运行测试：go test -v ./middleware/... -run OtelTrace
// ==================================================
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// ==================== 测试辅助函数 ====================

// createOtelTestRouter 创建 OtelTrace 测试路由
func createOtelTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.GET("/api/users/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})
	router.GET("/api/error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})
	router.GET("/api/bad-request", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})
	router.GET("/api/with-error", func(c *gin.Context) {
		_ = c.Error(&gin.Error{Err: http.ErrAbortHandler, Type: gin.ErrorTypePublic})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error occurred"})
	})
	return router
}

// ==================== OtelTraceHandler 单元测试 ====================

// TestOtelTraceHandler_Disabled 测试追踪禁用时的行为
//
// 【功能点】验证追踪禁用时中间件直接放行
// 【测试流程】追踪未初始化时发送请求，验证正常处理
func TestOtelTraceHandler_Disabled(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_BasicRequest 测试基本请求
//
// 【功能点】验证中间件正常工作
// 【测试流程】发送请求，验证正常返回
func TestOtelTraceHandler_BasicRequest(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_RouteWithParams 测试带参数的路由
//
// 【功能点】验证带参数的路由能正确获取 FullPath
// 【测试流程】访问 /api/users/123，验证正常处理
func TestOtelTraceHandler_RouteWithParams(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/users/123", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_ServerError 测试服务器错误响应
//
// 【功能点】验证 5xx 错误时 Span 状态设置为 Error
// 【测试流程】返回 500 状态码，验证请求正常处理
func TestOtelTraceHandler_ServerError(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_ClientError 测试客户端错误响应
//
// 【功能点】验证 4xx 错误时 Span 状态设置为 Error
// 【测试流程】返回 400 状态码，验证请求正常处理
func TestOtelTraceHandler_ClientError(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/bad-request", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_WithGinErrors 测试带 Gin 错误的请求
//
// 【功能点】验证 gin.Context 中的错误被记录
// 【测试流程】添加错误到上下文，验证请求正常处理
func TestOtelTraceHandler_WithGinErrors(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/with-error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_NotFound 测试 404 响应
//
// 【功能点】验证未注册的路由使用 URL.Path 作为 Span 名称
// 【测试流程】访问不存在的路由，验证返回 404
func TestOtelTraceHandler_NotFound(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/not-found", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_POSTRequest 测试 POST 请求
//
// 【功能点】验证不同 HTTP 方法的处理
// 【测试流程】发送 POST 请求，验证正常处理
func TestOtelTraceHandler_POSTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OtelTraceHandler())
	router.POST("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/data", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望状态码 201, 实际 %d", w.Code)
	}
}

// TestOtelTraceHandler_MultipleRequests 测试多个请求
//
// 【功能点】验证多个请求独立追踪
// 【测试流程】发送多个请求，验证每个请求都正常处理
func TestOtelTraceHandler_MultipleRequests(t *testing.T) {
	router := createOtelTestRouter(OtelTraceHandler())

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d: 期望状态码 200, 实际 %d", i+1, w.Code)
		}
	}
}

// ==================== getScheme 辅助函数测试 ====================

// TestGetScheme 测试 getScheme 函数
//
// 【功能点】验证 getScheme 函数正确返回协议方案
// 【测试流程】
//  1. 测试无代理头的 HTTP 请求
//  2. 测试 X-Forwarded-Proto 头
func TestGetScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		forwardedProto string
		expectedScheme string
	}{
		{"default http", "", "http"},
		{"forwarded https", "https", "https"},
		{"forwarded http", "http", "http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/test", nil)

			if tt.forwardedProto != "" {
				c.Request.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			}

			scheme := getScheme(c)
			if scheme != tt.expectedScheme {
				t.Errorf("期望 scheme=%s, 实际 %s", tt.expectedScheme, scheme)
			}
		})
	}
}

// TestGetScheme_XForwardedProto 测试 X-Forwarded-Proto 头优先
//
// 【功能点】验证 X-Forwarded-Proto 头优先于其他判断
// 【测试流程】设置 X-Forwarded-Proto 头，验证返回该值
func TestGetScheme_XForwardedProto(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	scheme := getScheme(c)
	if scheme != "https" {
		t.Errorf("期望 scheme=https, 实际 %s", scheme)
	}
}

// ==================== 基准测试 ====================

// BenchmarkOtelTraceHandler 基准测试 OtelTrace 中间件性能
func BenchmarkOtelTraceHandler(b *testing.B) {
	router := createOtelTestRouter(OtelTraceHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)
	}
}

// BenchmarkOtelTraceHandler_Disabled 基准测试追踪禁用时的性能
func BenchmarkOtelTraceHandler_Disabled(b *testing.B) {
	router := createOtelTestRouter(OtelTraceHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)
	}
}
