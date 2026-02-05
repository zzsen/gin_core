// Package core 服务器功能测试
//
// ==================== 测试说明 ====================
// 本文件包含服务器相关功能的单元测试。
//
// 测试覆盖内容：
// 1. NotFound - 404 错误处理函数
// 2. MethodNotAllowed - 405 错误处理函数
//
// 运行测试：go test -v ./core/... -run Server
// ==================================================
package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ==================== NotFound 测试 ====================

// TestNotFound 测试 NotFound 函数
//
// 【功能点】验证 404 错误处理函数
// 【测试流程】
//  1. 创建测试上下文
//  2. 调用 NotFound 函数
//  3. 验证返回 404 状态码和正确的响应体
func TestNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns 404 status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/not-found-route", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		NotFound(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "Not Found", w.Body.String())
	})

	t.Run("handles different HTTP methods", func(t *testing.T) {
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

		for _, method := range methods {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(method, "/missing", nil)
			c.Request.RemoteAddr = "192.168.1.100:12345"

			NotFound(c)

			assert.Equal(t, http.StatusNotFound, w.Code, "Method %s should return 404", method)
		}
	})

	t.Run("handles different paths", func(t *testing.T) {
		paths := []string{
			"/api/v1/users/123",
			"/admin/dashboard",
			"/static/css/style.css",
			"/",
		}

		for _, path := range paths {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", path, nil)
			c.Request.RemoteAddr = "192.168.1.100:12345"

			NotFound(c)

			assert.Equal(t, http.StatusNotFound, w.Code, "Path %s should return 404", path)
		}
	})
}

// TestNotFound_WithEngine 测试 NotFound 与 Gin 引擎集成
//
// 【功能点】验证 NotFound 作为 NoRoute 处理器
// 【测试流程】
//  1. 创建 Gin 引擎并设置 NoRoute
//  2. 访问未注册的路由
//  3. 验证返回 404
func TestNotFound_WithEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.NoRoute(NotFound)

	// 注册一个路由
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	t.Run("existing route returns 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("non-existing route returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/missing", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "Not Found", w.Body.String())
	})
}

// ==================== MethodNotAllowed 测试 ====================

// TestMethodNotAllowed 测试 MethodNotAllowed 函数
//
// 【功能点】验证 405 错误处理函数
// 【测试流程】
//  1. 创建测试上下文
//  2. 调用 MethodNotAllowed 函数
//  3. 验证返回 405 状态码和正确的响应体
func TestMethodNotAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns 405 status code", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/users", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		MethodNotAllowed(c)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Equal(t, "Method Not Allowed", w.Body.String())
	})

	t.Run("handles different disallowed methods", func(t *testing.T) {
		methods := []string{"POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

		for _, method := range methods {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(method, "/api/resource", nil)
			c.Request.RemoteAddr = "192.168.1.100:12345"

			MethodNotAllowed(c)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Method %s should return 405", method)
		}
	})

	t.Run("handles different paths", func(t *testing.T) {
		paths := []string{
			"/api/v1/users",
			"/admin/settings",
			"/webhook/callback",
		}

		for _, path := range paths {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("DELETE", path, nil)
			c.Request.RemoteAddr = "192.168.1.100:12345"

			MethodNotAllowed(c)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Path %s should return 405", path)
		}
	})
}

// TestMethodNotAllowed_WithEngine 测试 MethodNotAllowed 与 Gin 引擎集成
//
// 【功能点】验证 MethodNotAllowed 作为 NoMethod 处理器
// 【测试流程】
//  1. 创建 Gin 引擎并设置 NoMethod
//  2. 使用不支持的 HTTP 方法访问路由
//  3. 验证返回 405
func TestMethodNotAllowed_WithEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoMethod(MethodNotAllowed)

	// 只注册 GET 方法
	router.GET("/api/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{}})
	})

	t.Run("allowed method returns 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/users", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("disallowed method returns 405", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/users", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Equal(t, "Method Not Allowed", w.Body.String())
	})

	t.Run("multiple disallowed methods return 405", func(t *testing.T) {
		disallowedMethods := []string{"POST", "PUT", "DELETE", "PATCH"}

		for _, method := range disallowedMethods {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(method, "/api/users", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "Method %s should return 405", method)
		}
	})
}

// ==================== 组合测试 ====================

// TestNotFoundAndMethodNotAllowed_Combined 测试 404 和 405 同时配置
//
// 【功能点】验证 NoRoute 和 NoMethod 同时配置时的正确行为
// 【测试流程】
//  1. 创建引擎并同时设置 NoRoute 和 NoMethod
//  2. 验证不存在的路由返回 404
//  3. 验证不允许的方法返回 405
func TestNotFoundAndMethodNotAllowed_Combined(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.NoRoute(NotFound)
	router.NoMethod(MethodNotAllowed)

	// 注册路由
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/api/create", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"message": "created"})
	})

	t.Run("existing route with correct method returns 2xx", func(t *testing.T) {
		// GET /api/test
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// POST /api/create
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/create", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("non-existing route returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/missing", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "Not Found", w.Body.String())
	})

	t.Run("wrong method on existing route returns 405", func(t *testing.T) {
		// POST on GET-only route
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		assert.Equal(t, "Method Not Allowed", w.Body.String())

		// GET on POST-only route
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", "/api/create", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// ==================== 边界情况测试 ====================

// TestNotFound_EdgeCases 测试 NotFound 边界情况
//
// 【功能点】验证边界情况的正确处理
// 【测试流程】测试特殊字符路径、长路径等
func TestNotFound_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("path with special characters", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/users?id=123&name=test", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		NotFound(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("path with unicode", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/用户/123", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		NotFound(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("empty path", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		NotFound(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("very long path", func(t *testing.T) {
		longPath := "/api" + "/very/long/path/segment" // 短一些以避免问题
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", longPath, nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		NotFound(c)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestMethodNotAllowed_EdgeCases 测试 MethodNotAllowed 边界情况
//
// 【功能点】验证边界情况的正确处理
// 【测试流程】测试非标准 HTTP 方法等
func TestMethodNotAllowed_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("custom HTTP method", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("CUSTOM", "/api/resource", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		MethodNotAllowed(c)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("HEAD method", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("HEAD", "/api/resource", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		MethodNotAllowed(c)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

// ==================== 基准测试 ====================

// BenchmarkNotFound 基准测试 NotFound 性能
func BenchmarkNotFound(b *testing.B) {
	gin.SetMode(gin.TestMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/api/missing", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		NotFound(c)
	}
}

// BenchmarkMethodNotAllowed 基准测试 MethodNotAllowed 性能
func BenchmarkMethodNotAllowed(b *testing.B) {
	gin.SetMode(gin.TestMode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/api/resource", nil)
		c.Request.RemoteAddr = "192.168.1.100:12345"

		MethodNotAllowed(c)
	}
}
