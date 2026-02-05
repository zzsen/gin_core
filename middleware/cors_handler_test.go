// Package middleware CORS 中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含 CORS 跨域中间件的单元测试。
//
// 测试覆盖内容：
// 1. CORS 功能禁用时的行为
// 2. 允许所有来源（*）的配置
// 3. 特定来源白名单的配置
// 4. 通配符来源匹配（如 *.example.com）
// 5. 预检请求（OPTIONS）的处理
// 6. 允许携带凭证的配置
// 7. 自定义响应头的配置
// 8. isOriginAllowed 辅助函数测试
//
// 运行测试：go test -v ./middleware/... -run CORS
// ==================================================
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// ==================== 测试辅助函数 ====================

// setupCORSTestConfig 设置 CORS 测试配置
func setupCORSTestConfig(cfg config.CORSConfig) func() {
	originalConfig := app.BaseConfig
	app.BaseConfig = config.BaseConfig{
		CORS: cfg,
	}
	return func() {
		app.BaseConfig = originalConfig
	}
}

// createCORSTestRouter 创建 CORS 测试路由
func createCORSTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/api/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "created"})
	})
	return router
}

// ==================== CORSHandler 单元测试 ====================

// TestCORSHandler_Disabled 测试 CORS 功能禁用
//
// 【功能点】验证 enabled=false 时不设置任何 CORS 头
// 【测试流程】设置 Enabled=false，发送带 Origin 的请求，验证无 CORS 响应头
func TestCORSHandler_Disabled(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled: false,
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	// 验证无 CORS 头
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("禁用时不应设置 Access-Control-Allow-Origin 头")
	}
}

// TestCORSHandler_NoOrigin 测试无 Origin 头的请求
//
// 【功能点】验证无 Origin 头时不设置 CORS 响应头
// 【测试流程】发送无 Origin 头的请求，验证无 CORS 响应头
func TestCORSHandler_NoOrigin(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	// 不设置 Origin 头
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	// 验证无 CORS 头
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("无 Origin 时不应设置 Access-Control-Allow-Origin 头")
	}
}

// TestCORSHandler_AllowAllOrigins 测试允许所有来源
//
// 【功能点】验证 allowOrigins=["*"] 时返回 * 作为允许来源
// 【测试流程】配置 AllowOrigins=["*"]，验证返回 Access-Control-Allow-Origin: *
func TestCORSHandler_AllowAllOrigins(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://any-domain.com")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	// 验证返回 *
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("期望 Access-Control-Allow-Origin=*, 实际 %s", origin)
	}
}

// TestCORSHandler_SpecificOrigins 测试特定来源白名单
//
// 【功能点】验证只有白名单中的来源被允许
// 【测试流程】
//  1. 配置特定来源白名单
//  2. 测试白名单中的来源 - 应返回该来源
//  3. 测试非白名单来源 - 不应设置 CORS 头
func TestCORSHandler_SpecificOrigins(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"http://localhost:3000", "https://example.com"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	// 测试白名单中的来源
	t.Run("allowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		router.ServeHTTP(w, req)

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "http://localhost:3000" {
			t.Errorf("期望 Access-Control-Allow-Origin=http://localhost:3000, 实际 %s", origin)
		}

		// 应该有 Vary 头
		if vary := w.Header().Get("Vary"); vary != "Origin" {
			t.Errorf("期望 Vary=Origin, 实际 %s", vary)
		}
	})

	// 测试非白名单来源
	t.Run("disallowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Origin", "http://malicious.com")
		router.ServeHTTP(w, req)

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "" {
			t.Errorf("非白名单来源不应设置 Access-Control-Allow-Origin, 实际 %s", origin)
		}
	})
}

// TestCORSHandler_WildcardOrigin 测试通配符来源匹配
//
// 【功能点】验证 *.example.com 匹配所有 example.com 子域名
// 【测试流程】配置 *.example.com，测试各种子域名的匹配情况
func TestCORSHandler_WildcardOrigin(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*.example.com"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	tests := []struct {
		origin    string
		shouldSet bool
	}{
		{"https://api.example.com", true},
		{"https://www.example.com", true},
		{"https://sub.domain.example.com", true},
		{"https://example.org", false},
		{"https://notexample.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/test", nil)
			req.Header.Set("Origin", tt.origin)
			router.ServeHTTP(w, req)

			hasHeader := w.Header().Get("Access-Control-Allow-Origin") != ""
			if hasHeader != tt.shouldSet {
				t.Errorf("Origin %s: 期望设置CORS头=%v, 实际=%v", tt.origin, tt.shouldSet, hasHeader)
			}
		})
	}
}

// TestCORSHandler_PreflightRequest 测试预检请求（OPTIONS）
//
// 【功能点】验证 OPTIONS 请求返回 204 状态码并中止请求链
// 【测试流程】发送 OPTIONS 请求，验证返回 204 和正确的 CORS 头
func TestCORSHandler_PreflightRequest(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	router.ServeHTTP(w, req)

	// 预检请求应返回 204
	if w.Code != http.StatusNoContent {
		t.Errorf("预检请求期望状态码 204, 实际 %d", w.Code)
	}

	// 验证 CORS 头
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("期望 Access-Control-Allow-Origin=*, 实际 %s", origin)
	}

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, PUT, DELETE" {
		t.Errorf("期望 Access-Control-Allow-Methods=GET, POST, PUT, DELETE, 实际 %s", methods)
	}

	if headers := w.Header().Get("Access-Control-Allow-Headers"); headers != "Content-Type, Authorization" {
		t.Errorf("期望 Access-Control-Allow-Headers=Content-Type, Authorization, 实际 %s", headers)
	}
}

// TestCORSHandler_DefaultMethods 测试默认允许的方法
//
// 【功能点】验证未配置 AllowMethods 时使用默认值
// 【测试流程】不配置 AllowMethods，验证返回默认方法列表
func TestCORSHandler_DefaultMethods(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*"},
		// 不设置 AllowMethods
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if methods := w.Header().Get("Access-Control-Allow-Methods"); methods != "GET, POST, PUT, PATCH, DELETE, OPTIONS" {
		t.Errorf("期望默认方法列表, 实际 %s", methods)
	}
}

// TestCORSHandler_AllowCredentials 测试允许携带凭证
//
// 【功能点】验证 AllowCredentials=true 时设置相应头
// 【测试流程】配置 AllowCredentials=true，验证返回 Access-Control-Allow-Credentials: true
func TestCORSHandler_AllowCredentials(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:          true,
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowCredentials: true,
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(w, req)

	if cred := w.Header().Get("Access-Control-Allow-Credentials"); cred != "true" {
		t.Errorf("期望 Access-Control-Allow-Credentials=true, 实际 %s", cred)
	}
}

// TestCORSHandler_ExposeHeaders 测试暴露响应头
//
// 【功能点】验证 ExposeHeaders 配置正确返回
// 【测试流程】配置 ExposeHeaders，验证返回 Access-Control-Expose-Headers
func TestCORSHandler_ExposeHeaders(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:       true,
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Custom-Header", "X-Request-Id"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if expose := w.Header().Get("Access-Control-Expose-Headers"); expose != "X-Custom-Header, X-Request-Id" {
		t.Errorf("期望 Access-Control-Expose-Headers=X-Custom-Header, X-Request-Id, 实际 %s", expose)
	}
}

// TestCORSHandler_MaxAge 测试预检请求缓存时间
//
// 【功能点】验证 MaxAge 配置正确返回
// 【测试流程】配置 MaxAge，验证返回 Access-Control-Max-Age
func TestCORSHandler_MaxAge(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*"},
		MaxAge:       3600,
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "3600" {
		t.Errorf("期望 Access-Control-Max-Age=3600, 实际 %s", maxAge)
	}
}

// TestCORSHandler_DefaultMaxAge 测试默认 MaxAge
//
// 【功能点】验证未配置 MaxAge 时使用默认值 86400
// 【测试流程】不配置 MaxAge，验证返回默认值
func TestCORSHandler_DefaultMaxAge(t *testing.T) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*"},
		// 不设置 MaxAge
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "http://example.com")
	router.ServeHTTP(w, req)

	if maxAge := w.Header().Get("Access-Control-Max-Age"); maxAge != "86400" {
		t.Errorf("期望默认 Access-Control-Max-Age=86400, 实际 %s", maxAge)
	}
}

// ==================== isOriginAllowed 辅助函数测试 ====================

// TestIsOriginAllowed 测试来源检查函数
//
// 【功能点】验证 isOriginAllowed 函数的各种匹配场景
// 【测试流程】测试精确匹配、通配符匹配、空配置等场景
func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name         string
		origin       string
		allowOrigins []string
		expected     bool
	}{
		{"empty config allows all", "http://any.com", []string{}, true},
		{"wildcard allows all", "http://any.com", []string{"*"}, true},
		{"exact match", "http://example.com", []string{"http://example.com"}, true},
		{"exact no match", "http://other.com", []string{"http://example.com"}, false},
		{"wildcard subdomain match", "https://api.example.com", []string{"*.example.com"}, true},
		{"wildcard subdomain no match", "https://example.org", []string{"*.example.com"}, false},
		{"multiple origins first match", "http://a.com", []string{"http://a.com", "http://b.com"}, true},
		{"multiple origins second match", "http://b.com", []string{"http://a.com", "http://b.com"}, true},
		{"multiple origins no match", "http://c.com", []string{"http://a.com", "http://b.com"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOriginAllowed(tt.origin, tt.allowOrigins)
			if result != tt.expected {
				t.Errorf("isOriginAllowed(%s, %v) = %v, want %v", tt.origin, tt.allowOrigins, result, tt.expected)
			}
		})
	}
}

// ==================== 基准测试 ====================

// BenchmarkCORSHandler 基准测试 CORS 中间件性能
func BenchmarkCORSHandler(b *testing.B) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"http://localhost:3000", "https://example.com"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		router.ServeHTTP(w, req)
	}
}

// BenchmarkCORSHandler_Preflight 基准测试预检请求性能
func BenchmarkCORSHandler_Preflight(b *testing.B) {
	cleanup := setupCORSTestConfig(config.CORSConfig{
		Enabled:      true,
		AllowOrigins: []string{"*"},
	})
	defer cleanup()

	router := createCORSTestRouter(CORSHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("OPTIONS", "/api/test", nil)
		req.Header.Set("Origin", "http://example.com")
		router.ServeHTTP(w, req)
	}
}
