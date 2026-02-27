// Package middleware TraceID 处理中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含 TraceID 处理中间件的单元测试。
//
// 测试覆盖内容：
// 1. TraceID 生成与设置
// 2. TraceID 存储到 gin.Context
// 3. TraceID 添加到响应头
// 4. 多个请求生成不同的 TraceID
// 5. UUID 格式验证
// 6. 从上游请求头 X-Trace-ID 读取 trace ID
// 7. 从上游请求头 X-Request-ID 读取 trace ID
// 8. 请求头优先级（X-Trace-ID > X-Request-ID）
// 9. 忽略空白请求头值
//
// 运行测试：go test -v ./middleware/... -run TraceIdHandler
// ==================================================
package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
)

// UUID 正则表达式
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ==================== TraceIdHandler 单元测试 ====================

// TestTraceIdHandler_GeneratesUUID 测试 TraceID 生成
//
// 【功能点】验证每个请求生成有效的 UUID 格式的 TraceID
// 【测试流程】发送请求，验证响应头中的 X-Trace-ID 是有效的 UUID
func TestTraceIdHandler_GeneratesUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	traceID := w.Header().Get("X-Trace-ID")
	if traceID == "" {
		t.Error("响应头中应该有 X-Trace-ID")
	}

	if !uuidRegex.MatchString(traceID) {
		t.Errorf("TraceID 应该是有效的 UUID 格式, 实际 %s", traceID)
	}
}

// TestTraceIdHandler_StoredInContext 测试 TraceID 存储到上下文
//
// 【功能点】验证 TraceID 被正确存储到 gin.Context
// 【测试流程】在处理器中获取 traceId，验证与响应头中的值一致
func TestTraceIdHandler_StoredInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var contextTraceID string

	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		// 从上下文获取 traceId
		if val, exists := c.Get("traceId"); exists {
			contextTraceID = val.(string)
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	headerTraceID := w.Header().Get("X-Trace-ID")

	if contextTraceID == "" {
		t.Error("上下文中应该有 traceId")
	}

	if contextTraceID != headerTraceID {
		t.Errorf("上下文中的 traceId (%s) 应该与响应头中的 (%s) 一致", contextTraceID, headerTraceID)
	}
}

// TestTraceIdHandler_DifferentRequestsDifferentIDs 测试不同请求生成不同的 TraceID
//
// 【功能点】验证每个请求生成唯一的 TraceID
// 【测试流程】发送多个请求，验证每个请求的 TraceID 都不同
func TestTraceIdHandler_DifferentRequestsDifferentIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	traceIDs := make(map[string]bool)
	requestCount := 10

	for i := 0; i < requestCount; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		traceID := w.Header().Get("X-Trace-ID")
		if traceIDs[traceID] {
			t.Errorf("TraceID %s 已经存在，应该生成唯一的 ID", traceID)
		}
		traceIDs[traceID] = true
	}

	if len(traceIDs) != requestCount {
		t.Errorf("应该生成 %d 个不同的 TraceID, 实际 %d", requestCount, len(traceIDs))
	}
}

// TestTraceIdHandler_ConcurrentRequests 测试并发请求
//
// 【功能点】验证并发请求时 TraceID 的唯一性
// 【测试流程】并发发送多个请求，验证每个请求的 TraceID 都不同
func TestTraceIdHandler_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	var wg sync.WaitGroup
	var mu sync.Mutex
	traceIDs := make(map[string]bool)
	requestCount := 100

	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			traceID := w.Header().Get("X-Trace-ID")

			mu.Lock()
			traceIDs[traceID] = true
			mu.Unlock()
		}()
	}

	wg.Wait()

	if len(traceIDs) != requestCount {
		t.Errorf("并发请求应该生成 %d 个不同的 TraceID, 实际 %d", requestCount, len(traceIDs))
	}
}

// TestTraceIdHandler_HeaderFormat 测试响应头格式
//
// 【功能点】验证响应头使用正确的名称 X-Trace-ID
// 【测试流程】发送请求，验证响应头名称正确
func TestTraceIdHandler_HeaderFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// 验证响应头存在
	if _, exists := w.Header()["X-Trace-Id"]; !exists {
		t.Error("响应头中应该有 X-Trace-ID (case-insensitive)")
	}
}

// TestTraceIdHandler_ContextKeyName 测试上下文键名
//
// 【功能点】验证使用正确的上下文键名 "traceId"
// 【测试流程】在处理器中验证键名
func TestTraceIdHandler_ContextKeyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	keyExists := false

	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		_, keyExists = c.Get("traceId")
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if !keyExists {
		t.Error("上下文中应该使用 'traceId' 作为键名")
	}
}

// TestTraceIdHandler_ChainedMiddlewares 测试与其他中间件链接
//
// 【功能点】验证 TraceID 在中间件链中正确传递
// 【测试流程】添加多个中间件，验证后续中间件能获取 TraceID
func TestTraceIdHandler_ChainedMiddlewares(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var traceIDInSecondMiddleware string

	router := gin.New()
	router.Use(TraceIdHandler())
	router.Use(func(c *gin.Context) {
		// 第二个中间件应该能获取到 traceId
		if val, exists := c.Get("traceId"); exists {
			traceIDInSecondMiddleware = val.(string)
		}
		c.Next()
	})
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if traceIDInSecondMiddleware == "" {
		t.Error("第二个中间件应该能获取到 traceId")
	}

	headerTraceID := w.Header().Get("X-Trace-ID")
	if traceIDInSecondMiddleware != headerTraceID {
		t.Errorf("中间件中的 traceId (%s) 应该与响应头中的 (%s) 一致",
			traceIDInSecondMiddleware, headerTraceID)
	}
}

// TestTraceIdHandler_POSTRequest 测试 POST 请求
//
// 【功能点】验证 POST 请求也能正确生成 TraceID
// 【测试流程】发送 POST 请求，验证响应头中有 TraceID
func TestTraceIdHandler_POSTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际 %d", w.Code)
	}

	traceID := w.Header().Get("X-Trace-ID")
	if traceID == "" {
		t.Error("POST 请求响应头中应该有 X-Trace-ID")
	}
}

// ==================== 上游请求头传播测试 ====================

// TestTraceIdHandler_PropagateFromXTraceID 测试从 X-Trace-ID 请求头读取 trace ID
//
// 【功能点】验证中间件优先使用上游传递的 X-Trace-ID
// 【测试流程】
// 1. 在请求中设置 X-Trace-ID 头
// 2. 验证响应头和上下文中使用的是上游传递的值
func TestTraceIdHandler_PropagateFromXTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamTraceID := "upstream-trace-id-12345"
	var contextTraceID string

	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		if val, exists := c.Get("traceId"); exists {
			contextTraceID = val.(string)
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", upstreamTraceID)
	router.ServeHTTP(w, req)

	if contextTraceID != upstreamTraceID {
		t.Errorf("上下文中的 traceId 应为上游传递的值 %s, 实际 %s", upstreamTraceID, contextTraceID)
	}

	headerTraceID := w.Header().Get("X-Trace-ID")
	if headerTraceID != upstreamTraceID {
		t.Errorf("响应头 X-Trace-ID 应为上游传递的值 %s, 实际 %s", upstreamTraceID, headerTraceID)
	}
}

// TestTraceIdHandler_PropagateFromXRequestID 测试从 X-Request-ID 请求头读取 trace ID
//
// 【功能点】验证中间件在没有 X-Trace-ID 时使用 X-Request-ID
// 【测试流程】
// 1. 在请求中仅设置 X-Request-ID 头
// 2. 验证响应头和上下文中使用的是 X-Request-ID 的值
func TestTraceIdHandler_PropagateFromXRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	upstreamRequestID := "request-id-abcdef"
	var contextTraceID string

	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		if val, exists := c.Get("traceId"); exists {
			contextTraceID = val.(string)
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", upstreamRequestID)
	router.ServeHTTP(w, req)

	if contextTraceID != upstreamRequestID {
		t.Errorf("上下文中的 traceId 应为 X-Request-ID 的值 %s, 实际 %s", upstreamRequestID, contextTraceID)
	}

	headerTraceID := w.Header().Get("X-Trace-ID")
	if headerTraceID != upstreamRequestID {
		t.Errorf("响应头 X-Trace-ID 应为 X-Request-ID 的值 %s, 实际 %s", upstreamRequestID, headerTraceID)
	}
}

// TestTraceIdHandler_HeaderPriority 测试请求头优先级
//
// 【功能点】验证 X-Trace-ID 优先级高于 X-Request-ID
// 【测试流程】
// 1. 同时设置 X-Trace-ID 和 X-Request-ID
// 2. 验证使用的是 X-Trace-ID 的值
func TestTraceIdHandler_HeaderPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	xTraceID := "x-trace-id-value"
	xRequestID := "x-request-id-value"
	var contextTraceID string

	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		if val, exists := c.Get("traceId"); exists {
			contextTraceID = val.(string)
		}
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", xTraceID)
	req.Header.Set("X-Request-ID", xRequestID)
	router.ServeHTTP(w, req)

	if contextTraceID != xTraceID {
		t.Errorf("当同时存在 X-Trace-ID 和 X-Request-ID 时应优先使用 X-Trace-ID, 期望 %s, 实际 %s", xTraceID, contextTraceID)
	}
}

// TestTraceIdHandler_IgnoreEmptyHeaders 测试忽略空白请求头
//
// 【功能点】验证空白的上游请求头不会被采用，仍然生成新的 UUID
// 【测试流程】
// 1. 设置空白值的 X-Trace-ID 和 X-Request-ID
// 2. 验证生成了新的 UUID 格式的 trace ID
func TestTraceIdHandler_IgnoreEmptyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Trace-ID", "   ")
	req.Header.Set("X-Request-ID", "")
	router.ServeHTTP(w, req)

	traceID := w.Header().Get("X-Trace-ID")
	if !uuidRegex.MatchString(traceID) {
		t.Errorf("空白请求头应被忽略并生成新的 UUID, 实际 %s", traceID)
	}
}

// TestTraceIdHandler_NoUpstreamHeader 测试无上游请求头时生成新 UUID
//
// 【功能点】验证没有上游请求头时仍然生成新的 UUID（向下兼容）
// 【测试流程】
// 1. 不设置任何追踪相关请求头
// 2. 验证生成了有效的 UUID
func TestTraceIdHandler_NoUpstreamHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	traceID := w.Header().Get("X-Trace-ID")
	if traceID == "" {
		t.Error("无上游请求头时应生成新的 TraceID")
	}

	if !uuidRegex.MatchString(traceID) {
		t.Errorf("生成的 TraceID 应为有效 UUID, 实际 %s", traceID)
	}
}

// ==================== 基准测试 ====================

// BenchmarkTraceIdHandler 基准测试 TraceID 中间件性能
func BenchmarkTraceIdHandler(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
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

// BenchmarkTraceIdHandler_UUIDGeneration 基准测试 UUID 生成
func BenchmarkTraceIdHandler_UUIDGeneration(b *testing.B) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceIdHandler())
	router.GET("/test", func(c *gin.Context) {
		// 只获取 traceId，不做其他操作
		c.Get("traceId")
		c.Status(http.StatusOK)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}
}
