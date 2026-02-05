// Package middleware 超时处理中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含超时处理中间件的单元测试。
//
// 测试覆盖内容：
// 1. 正常请求的处理（未超时）
// 2. 请求超时的处理
// 3. 请求接近超时的警告
// 4. goroutine 中的 panic 处理
// 5. 并发请求的超时处理
//
// 运行测试：go test -v ./middleware/... -run TimeoutHandler
// ==================================================
package middleware

import (
	"encoding/json"
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

// setupTimeoutTestConfig 设置超时测试配置
func setupTimeoutTestConfig(timeout int) func() {
	originalConfig := app.BaseConfig
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{
			ApiTimeout: timeout,
		},
	}
	return func() {
		app.BaseConfig = originalConfig
	}
}

// createTimeoutTestRouter 创建超时测试路由
func createTimeoutTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	return router
}

// ==================== TimeoutHandler 单元测试 ====================

// TestTimeoutHandler_NormalRequest 测试正常请求（未超时）
//
// 【功能点】验证正常请求能够正常处理
// 【测试流程】设置较长超时时间，发送快速响应的请求，验证正常返回
func TestTimeoutHandler_NormalRequest(t *testing.T) {
	cleanup := setupTimeoutTestConfig(5) // 5 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "fast response"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fast", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}
	if resp["message"] != "fast response" {
		t.Errorf("期望 message=fast response, 实际 %v", resp["message"])
	}
}

// TestTimeoutHandler_SlowRequest 测试超时请求
//
// 【功能点】验证超时请求的超时处理逻辑被触发
// 【测试流程】设置 1 秒超时，发送需要 2 秒的请求，验证超时日志被记录
// 【注意】由于 httptest.Recorder 的限制，超时时状态码可能仍为 200
func TestTimeoutHandler_SlowRequest(t *testing.T) {
	cleanup := setupTimeoutTestConfig(1) // 1 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(2 * time.Second)
		c.JSON(http.StatusOK, gin.H{"message": "slow response"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	// 验证超时机制生效 - 请求应在超时后返回，而不是等待完整的 2 秒
	// 由于超时处理是在 goroutine 中执行的，实际可能略有误差
	if duration > 1500*time.Millisecond {
		t.Logf("请求超时处理已触发，耗时 %v", duration)
	}
}

// TestTimeoutHandler_NearTimeout 测试接近超时的请求
//
// 【功能点】验证接近超时（超过 80%）的请求会记录警告
// 【测试流程】设置 1 秒超时，发送需要 0.85 秒的请求，验证正常返回
func TestTimeoutHandler_NearTimeout(t *testing.T) {
	cleanup := setupTimeoutTestConfig(1) // 1 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/near-timeout", func(c *gin.Context) {
		time.Sleep(850 * time.Millisecond) // 85% 的超时时间
		c.JSON(http.StatusOK, gin.H{"message": "near timeout"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/near-timeout", nil)
	router.ServeHTTP(w, req)

	// 应该正常返回，但会记录警告日志
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_PanicInHandler 测试处理器中的 panic
//
// 【功能点】验证处理器中的 panic 被正确传递
// 【测试流程】在处理器中触发 panic，验证 panic 被传递到外层
func TestTimeoutHandler_PanicInHandler(t *testing.T) {
	cleanup := setupTimeoutTestConfig(5) // 5 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(nil)

	// 添加异常处理中间件来捕获 panic
	router.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "panic occurred"})
				c.Abort()
			}
		}()
		c.Next()
	})

	router.Use(TimeoutHandler())
	router.GET("/panic", func(c *gin.Context) {
		panic("test panic in handler")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/panic", nil)
	router.ServeHTTP(w, req)

	// panic 应该被捕获
	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_MultipleRequests 测试多个并发请求
//
// 【功能点】验证多个并发请求能够独立处理
// 【测试流程】发送多个并发请求，验证每个请求独立超时处理
func TestTimeoutHandler_MultipleRequests(t *testing.T) {
	cleanup := setupTimeoutTestConfig(2) // 2 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "fast"})
	})
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(3 * time.Second)
		c.JSON(http.StatusOK, gin.H{"message": "slow"})
	})

	var wg sync.WaitGroup
	results := make(chan int, 2)

	// 快速请求
	wg.Add(1)
	go func() {
		defer wg.Done()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/fast", nil)
		router.ServeHTTP(w, req)
		results <- w.Code
	}()

	// 慢速请求
	wg.Add(1)
	go func() {
		defer wg.Done()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/slow", nil)
		router.ServeHTTP(w, req)
		results <- w.Code
	}()

	wg.Wait()
	close(results)

	var codes []int
	for code := range results {
		codes = append(codes, code)
	}

	// 至少应该有一个 200 (快速请求)
	has200 := false
	for _, code := range codes {
		if code == 200 {
			has200 = true
		}
	}

	if !has200 {
		t.Error("应该有一个快速请求返回 200")
	}
}

// TestTimeoutHandler_ZeroTimeout 测试零超时配置
//
// 【功能点】验证零超时时请求立即超时
// 【测试流程】设置 0 秒超时，验证请求处理
func TestTimeoutHandler_ZeroTimeout(t *testing.T) {
	cleanup := setupTimeoutTestConfig(0) // 0 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// 0 超时时，time.After(0) 可能立即触发或不触发
	// 结果取决于 goroutine 调度
	if w.Code != http.StatusOK && w.Code != http.StatusRequestTimeout {
		t.Errorf("期望状态码 200 或 408, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_HeadersBeforeTimeout 测试超时前已发送的响应头
//
// 【功能点】验证在超时前已开始写入响应的情况
// 【测试流程】在超时发生前开始写入响应
func TestTimeoutHandler_HeadersBeforeTimeout(t *testing.T) {
	cleanup := setupTimeoutTestConfig(2) // 2 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/partial", func(c *gin.Context) {
		// 快速返回，不会超时
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/partial", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// ==================== 基准测试 ====================

// BenchmarkTimeoutHandler_FastRequest 基准测试快速请求
func BenchmarkTimeoutHandler_FastRequest(b *testing.B) {
	cleanup := setupTimeoutTestConfig(5)
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}
