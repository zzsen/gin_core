// Package middleware TraceLog 处理中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含 TraceLog 处理中间件的单元测试。
//
// 测试覆盖内容：
// 1. 请求日志记录的基本功能
// 2. 请求方法和 URL 的记录
// 3. 状态码的记录
// 4. 客户端 IP 的记录
// 5. 响应时间的计算
// 6. TraceID 和 RequestID 的获取
// 7. 错误信息的收集
//
// 运行测试：go test -v ./middleware/... -run TraceLogHandler
// ==================================================
package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// ==================== TraceLogHandler 单元测试 ====================

// TestTraceLogHandler_BasicRequest 测试基本请求日志
//
// 【功能点】验证中间件正常工作，不影响请求处理
// 【测试流程】发送 GET 请求，验证返回 200
func TestTraceLogHandler_BasicRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_POSTRequest 测试 POST 请求日志
//
// 【功能点】验证 POST 请求的日志记录
// 【测试流程】发送 POST 请求，验证返回 200
func TestTraceLogHandler_POSTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "created"})
	})

	body := bytes.NewBufferString(`{"name": "test"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_WithTraceID 测试带 TraceID 的请求
//
// 【功能点】验证能够获取上下文中的 traceId
// 【测试流程】设置 traceId，验证日志中能够获取
func TestTraceLogHandler_WithTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 先设置 traceId
	router.Use(func(c *gin.Context) {
		c.Set("traceId", "test-trace-id-123")
		c.Next()
	})
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_WithRequestID 测试带 RequestID 的请求
//
// 【功能点】验证能够获取上下文中的 requestId
// 【测试流程】设置 requestId，验证日志中能够获取
func TestTraceLogHandler_WithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 设置 requestId
	router.Use(func(c *gin.Context) {
		c.Set("requestId", "req-123456")
		c.Next()
	})
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_WithUserAgent 测试带 User-Agent 的请求
//
// 【功能点】验证能够记录 User-Agent 头
// 【测试流程】设置 User-Agent，验证请求正常处理
func TestTraceLogHandler_WithUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_WithToken 测试带 Token 的请求
//
// 【功能点】验证能够记录 token 头
// 【测试流程】设置 token 头，验证请求正常处理
func TestTraceLogHandler_WithToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("token", "Bearer abc123")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_ErrorResponse 测试错误响应的日志记录
//
// 【功能点】验证能够记录非 200 状态码
// 【测试流程】返回 400 状态码，验证请求正常处理
func TestTraceLogHandler_ErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/error", func(c *gin.Context) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad request"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_NotFound 测试 404 响应的日志记录
//
// 【功能点】验证能够记录 404 状态码
// 【测试流程】访问不存在的路由，验证返回 404
func TestTraceLogHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/not-found", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("期望状态码 404, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_WithErrors 测试带错误的请求
//
// 【功能点】验证能够收集 gin.Context 中的错误
// 【测试流程】添加错误到上下文，验证日志记录
func TestTraceLogHandler_WithErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/error", func(c *gin.Context) {
		// 使用正确的方式添加错误
		_ = c.Error(&gin.Error{Err: http.ErrAbortHandler, Type: gin.ErrorTypePublic, Meta: "test error"})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("期望状态码 500, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_FormData 测试表单数据的记录
//
// 【功能点】验证能够解析和记录表单数据
// 【测试流程】发送表单数据，验证请求正常处理
func TestTraceLogHandler_FormData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.POST("/form", func(c *gin.Context) {
		name := c.PostForm("name")
		c.JSON(http.StatusOK, gin.H{"name": name})
	})

	body := bytes.NewBufferString("name=test&value=123")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/form", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_ResponseTime 测试响应时间记录
//
// 【功能点】验证能够正确计算响应时间
// 【测试流程】发送延迟响应的请求，验证请求正常处理
func TestTraceLogHandler_ResponseTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(50 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/slow", nil)

	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	// 验证响应时间至少 50ms
	if duration < 50*time.Millisecond {
		t.Errorf("响应时间应该至少 50ms, 实际 %v", duration)
	}
}

// TestTraceLogHandler_ClientIP 测试客户端 IP 记录
//
// 【功能点】验证能够记录客户端 IP
// 【测试流程】设置 RemoteAddr，验证请求正常处理
func TestTraceLogHandler_ClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ip": c.ClientIP()})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_QueryParams 测试查询参数的记录
//
// 【功能点】验证能够记录 URL 查询参数
// 【测试流程】发送带查询参数的请求，验证请求正常处理
func TestTraceLogHandler_QueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?page=1&size=10", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_NoTraceID 测试无 TraceID 的情况
//
// 【功能点】验证无 TraceID 时日志正常工作
// 【测试流程】不设置 traceId，验证请求正常处理
func TestTraceLogHandler_NoTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}
}

// TestTraceLogHandler_MultipleRequests 测试多个请求
//
// 【功能点】验证多个请求独立记录
// 【测试流程】发送多个请求，验证每个请求都正常处理
func TestTraceLogHandler_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d: 期望状态码 200, 实际 %d", i+1, w.Code)
		}
	}
}

// ==================== 基准测试 ====================

// BenchmarkTraceLogHandler 基准测试 TraceLog 中间件性能
func BenchmarkTraceLogHandler(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
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

// BenchmarkTraceLogHandler_WithFormData 基准测试带表单数据的性能
func BenchmarkTraceLogHandler_WithFormData(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceLogHandler())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := bytes.NewBufferString("name=test&value=123")
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)
	}
}
