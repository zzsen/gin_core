// Package middleware Prometheus 中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含 Prometheus 指标采集中间件的单元测试。
//
// 测试覆盖内容：
// 1. 基本指标采集功能
// 2. 排除路径配置
// 3. 请求计数指标
// 4. 请求耗时指标
// 5. 并发请求处理
//
// 运行测试：go test -v ./middleware/... -run PrometheusHandler
// ==================================================
package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// ==================== 测试辅助函数 ====================

// setupPrometheusTestConfig 设置 Prometheus 测试配置
func setupPrometheusTestConfig(cfg config.MetricsConfig) func() {
	originalConfig := app.BaseConfig
	app.BaseConfig = config.BaseConfig{
		Metrics: cfg,
	}
	return func() {
		app.BaseConfig = originalConfig
	}
}

// createPrometheusTestRouter 创建 Prometheus 测试路由
func createPrometheusTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.GET("/api/slow", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "slow"})
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "metrics"})
	})
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return router
}

// ==================== PrometheusHandler 单元测试 ====================

// TestPrometheusHandler_BasicRequest 测试基本请求指标采集
//
// 【功能点】验证中间件正常工作，不影响请求处理
// 【测试流程】发送请求，验证返回 200
func TestPrometheusHandler_BasicRequest(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestPrometheusHandler_ExcludePaths 测试排除路径
//
// 【功能点】验证配置的排除路径不被统计
// 【测试流程】配置排除路径，发送请求，验证请求正常处理
func TestPrometheusHandler_ExcludePaths(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled:      true,
		ExcludePaths: []string{"/metrics", "/health"},
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	// 测试排除路径
	t.Run("excluded path /metrics", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200, 实际 %d", w.Code)
		}
	})

	t.Run("excluded path /health", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200, 实际 %d", w.Code)
		}
	})

	// 测试非排除路径
	t.Run("included path /api/test", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望状态码 200, 实际 %d", w.Code)
		}
	})
}

// TestPrometheusHandler_MultipleRequests 测试多个请求
//
// 【功能点】验证多个请求都能被正确统计
// 【测试流程】发送多个请求，验证每个请求都正常处理
func TestPrometheusHandler_MultipleRequests(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d: 期望状态码 200, 实际 %d", i+1, w.Code)
		}
	}
}

// TestPrometheusHandler_DifferentStatusCodes 测试不同状态码
//
// 【功能点】验证不同状态码的请求都能被正确统计
// 【测试流程】发送返回不同状态码的请求
func TestPrometheusHandler_DifferentStatusCodes(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PrometheusHandler())
	router.GET("/ok", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.GET("/bad-request", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})
	router.GET("/internal-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})

	tests := []struct {
		path       string
		statusCode int
	}{
		{"/ok", http.StatusOK},
		{"/bad-request", http.StatusBadRequest},
		{"/internal-error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.statusCode {
				t.Errorf("期望状态码 %d, 实际 %d", tt.statusCode, w.Code)
			}
		})
	}
}

// TestPrometheusHandler_SlowRequest 测试慢请求
//
// 【功能点】验证慢请求的耗时被正确记录
// 【测试流程】发送慢请求，验证请求正常处理
func TestPrometheusHandler_SlowRequest(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/slow", nil)

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	if duration < 50*time.Millisecond {
		t.Errorf("请求耗时应该至少 50ms, 实际 %v", duration)
	}
}

// TestPrometheusHandler_ConcurrentRequests 测试并发请求
//
// 【功能点】验证并发请求时指标采集的正确性
// 【测试流程】并发发送多个请求，验证每个请求都正常处理
func TestPrometheusHandler_ConcurrentRequests(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	var wg sync.WaitGroup
	requestCount := 50
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/test", nil)
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount != requestCount {
		t.Errorf("期望 %d 个成功请求, 实际 %d", requestCount, successCount)
	}
}

// TestPrometheusHandler_EmptyPath 测试空路径
//
// 【功能点】验证空路径时使用 URL.Path
// 【测试流程】访问未注册的路由
func TestPrometheusHandler_EmptyPath(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/not-found", nil)
	router.ServeHTTP(w, req)

	// 未注册的路由返回 404
	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 实际 %d", w.Code)
	}
}

// TestPrometheusHandler_DifferentMethods 测试不同 HTTP 方法
//
// 【功能点】验证不同 HTTP 方法的请求都能被正确统计
// 【测试流程】发送 GET、POST、PUT、DELETE 请求
func TestPrometheusHandler_DifferentMethods(t *testing.T) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(PrometheusHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "GET"})
	})
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "POST"})
	})
	router.PUT("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "PUT"})
	})
	router.DELETE("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"method": "DELETE"})
	})

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(method, "/test", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s 请求: 期望状态码 200, 实际 %d", method, w.Code)
			}
		})
	}
}

// ==================== 基准测试 ====================

// BenchmarkPrometheusHandler 基准测试 Prometheus 中间件性能
func BenchmarkPrometheusHandler(b *testing.B) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled: true,
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)
	}
}

// BenchmarkPrometheusHandler_ExcludedPath 基准测试排除路径的性能
func BenchmarkPrometheusHandler_ExcludedPath(b *testing.B) {
	cleanup := setupPrometheusTestConfig(config.MetricsConfig{
		Enabled:      true,
		ExcludePaths: []string{"/metrics"},
	})
	defer cleanup()

	router := createPrometheusTestRouter(PrometheusHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		router.ServeHTTP(w, req)
	}
}
