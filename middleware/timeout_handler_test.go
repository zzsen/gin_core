// Package middleware 超时处理中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含超时处理中间件的单元测试。
//
// 测试覆盖内容：
// 1. 正常请求的处理（未超时）
// 2. 协作式超时处理（context-aware 处理器）
// 3. 非协作式超时处理（处理器忽略 context）
// 4. 请求接近超时的警告
// 5. 处理器中的 panic 传播
// 6. 并发请求的独立超时处理
// 7. 零超时配置跳过超时控制
// 8. 超时上下文传播验证
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

// TestTimeoutHandler_CooperativeTimeout 测试协作式超时（context-aware 处理器）
//
// 【功能点】验证 context-aware 的处理器在超时时能被正确中断，并返回 408 响应
// 【测试流程】
// 1. 设置 1 秒超时
// 2. 处理器通过 select 监听 ctx.Done()，实现协作式超时
// 3. 验证请求在超时后返回 408，且耗时接近超时时间
func TestTimeoutHandler_CooperativeTimeout(t *testing.T) {
	cleanup := setupTimeoutTestConfig(1) // 1 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/slow", func(c *gin.Context) {
		select {
		case <-time.After(3 * time.Second):
			c.JSON(http.StatusOK, gin.H{"message": "slow response"})
		case <-c.Request.Context().Done():
			return
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	if duration > 1500*time.Millisecond {
		t.Errorf("协作式超时应在 ~1s 内返回，实际耗时 %v", duration)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}
	if resp["msg"] != "Request timed out" {
		t.Errorf("期望超时响应消息，实际 %v", resp["msg"])
	}
}

// TestTimeoutHandler_NonCooperativeTimeout 测试非协作式超时（处理器忽略 context）
//
// 【功能点】验证不检查 context 的处理器在超时后仍能正常执行完毕，
// 由于响应已写入，中间件仅记录超时日志而不覆盖响应
// 【测试流程】
// 1. 设置 1 秒超时
// 2. 处理器使用 time.Sleep 阻塞（不监听 context）
// 3. 验证处理器的响应被保留（200），但总耗时超过超时时间
func TestTimeoutHandler_NonCooperativeTimeout(t *testing.T) {
	cleanup := setupTimeoutTestConfig(1) // 1 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(1500 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "slow response"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	if duration < 1*time.Second {
		t.Errorf("非协作式处理器应阻塞至完成，实际耗时 %v", duration)
	}

	if w.Code != http.StatusOK {
		t.Errorf("处理器已写入响应，期望状态码 200, 实际 %d", w.Code)
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

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_PanicInHandler 测试处理器中的 panic
//
// 【功能点】验证处理器中的 panic 沿中间件链自然传播，由上层 recover 中间件捕获
// 【测试流程】
// 1. 在 TimeoutHandler 之前注册 recover 中间件
// 2. 在处理器中触发 panic
// 3. 验证 panic 被上层中间件正确捕获并返回 500
func TestTimeoutHandler_PanicInHandler(t *testing.T) {
	cleanup := setupTimeoutTestConfig(5) // 5 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(nil)

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

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_MultipleRequests 测试多个并发请求
//
// 【功能点】验证多个并发请求能够独立处理，互不影响
// 【测试流程】
// 1. 同时发送快速请求和 context-aware 的慢速请求
// 2. 验证快速请求返回 200
// 3. 验证慢速请求因超时返回 408（通过检查响应内容）
func TestTimeoutHandler_MultipleRequests(t *testing.T) {
	cleanup := setupTimeoutTestConfig(2) // 2 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/fast", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "fast"})
	})
	router.GET("/slow", func(c *gin.Context) {
		select {
		case <-time.After(5 * time.Second):
			c.JSON(http.StatusOK, gin.H{"message": "slow"})
		case <-c.Request.Context().Done():
			return
		}
	})

	type result struct {
		path string
		code int
		msg  string
	}

	var wg sync.WaitGroup
	results := make(chan result, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/fast", nil)
		router.ServeHTTP(w, req)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		msg, _ := resp["message"].(string)
		if msg == "" {
			msg, _ = resp["msg"].(string)
		}
		results <- result{path: "/fast", code: w.Code, msg: msg}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/slow", nil)
		router.ServeHTTP(w, req)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		msg, _ := resp["message"].(string)
		if msg == "" {
			msg, _ = resp["msg"].(string)
		}
		results <- result{path: "/slow", code: w.Code, msg: msg}
	}()

	wg.Wait()
	close(results)

	for r := range results {
		switch r.path {
		case "/fast":
			if r.code != http.StatusOK {
				t.Errorf("快速请求期望状态码 200, 实际 %d", r.code)
			}
			if r.msg != "fast" {
				t.Errorf("快速请求期望 message=fast, 实际 %s", r.msg)
			}
		case "/slow":
			if r.msg != "Request timed out" {
				t.Errorf("慢速请求期望超时消息，实际 %s", r.msg)
			}
		}
	}
}

// TestTimeoutHandler_ZeroTimeout 测试零超时配置
//
// 【功能点】验证零超时时跳过超时控制，请求正常处理
// 【测试流程】设置 0 秒超时，验证请求直接通过，返回 200
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

	if w.Code != http.StatusOK {
		t.Errorf("零超时应跳过超时控制，期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_HeadersBeforeTimeout 测试超时前已发送的响应头
//
// 【功能点】验证在超时前已开始写入响应的情况
// 【测试流程】在超时发生前开始写入响应，验证正常返回
func TestTimeoutHandler_HeadersBeforeTimeout(t *testing.T) {
	cleanup := setupTimeoutTestConfig(2) // 2 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/partial", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/partial", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTimeoutHandler_ContextPropagation 测试超时上下文传播
//
// 【功能点】验证 context.WithTimeout 设置的截止时间能正确传播到处理器
// 【测试流程】
// 1. 设置 2 秒超时
// 2. 在处理器中检查 context 是否包含截止时间
// 3. 验证截止时间在合理范围内
func TestTimeoutHandler_ContextPropagation(t *testing.T) {
	cleanup := setupTimeoutTestConfig(2) // 2 秒超时
	defer cleanup()

	router := createTimeoutTestRouter(TimeoutHandler())
	router.GET("/ctx", func(c *gin.Context) {
		deadline, ok := c.Request.Context().Deadline()
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no deadline"})
			return
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > 2*time.Second {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid deadline"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "context has deadline"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ctx", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}
	if resp["message"] != "context has deadline" {
		t.Errorf("期望 context 包含 deadline, 实际响应 %v", resp)
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
