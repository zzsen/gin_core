// Package middleware 限流中间件测试
//
// ==================== 测试说明 ====================
// 本文件包含限流中间件的单元测试，不需要外部依赖（使用内存限流器）。
//
// 测试覆盖内容：
// 1. 限流功能禁用时的行为
// 2. 基础限流功能（默认规则）
// 3. 不同 IP 独立限流
// 4. 路径规则匹配（精确匹配、通配符）
// 5. 全局限流键类型
// 6. 代理场景下的 IP 获取（X-Forwarded-For、X-Real-IP）
// 7. 辅助函数测试（findMatchingRule、generateRateLimitKey）
// 8. 性能基准测试
//
// 运行测试：go test -v ./middleware/... -run RateLimit
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

// setupRateLimitTestConfig 设置测试配置
// 备份原始配置，设置测试配置，返回清理函数
func setupRateLimitTestConfig(cfg config.RateLimitConfig) func() {
	originalConfig := app.BaseConfig
	app.BaseConfig = config.BaseConfig{
		RateLimit: cfg,
	}
	return func() {
		app.BaseConfig = originalConfig
	}
}

// createTestRouter 创建测试路由
// 包含三个测试端点：/api/test、/api/login、/api/public
func createTestRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	if middleware != nil {
		router.Use(middleware)
	}
	router.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	})
	router.POST("/api/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "login"})
	})
	router.GET("/api/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public"})
	})
	return router
}

// ==================== RateLimitHandler 单元测试 ====================

// TestRateLimitHandler_Disabled 测试限流功能禁用时的行为
//
// 【功能点】验证 enabled=false 时所有请求都放行
// 【测试流程】设置 Enabled=false，发送 100 个请求，验证全部返回 200
func TestRateLimitHandler_Disabled(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled: false,
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// 发送多个请求，都应该成功
	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}
}

// TestRateLimitHandler_BasicRateLimit 测试基础限流功能
//
// 【功能点】验证请求数超过 burst 后返回 HTTP 429
// 【测试流程】
//  1. 设置 DefaultBurst=5
//  2. 发送 6 个请求
//  3. 验证前 5 个成功，第 6 个返回 429
func TestRateLimitHandler_BasicRateLimit(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  5,
		DefaultBurst: 5,
		Store:        "memory",
		Message:      "请求过于频繁",
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// 前 5 次请求应该成功
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}

	// 第 6 次请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("第 6 次请求应返回 429, 实际返回 %d", w.Code)
	}
}

// TestRateLimitHandler_DifferentIPs 测试不同 IP 的独立限流
//
// 【功能点】验证每个 IP 有独立的限流计数
// 【测试流程】使用 3 个不同 IP 各发送 3 个请求，验证全部成功（共 9 个）
func TestRateLimitHandler_DifferentIPs(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  3,
		DefaultBurst: 3,
		Store:        "memory",
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// 不同 IP 应该有独立的限流
	ips := []string{"192.168.1.1:1234", "192.168.1.2:1234", "192.168.1.3:1234"}

	for _, ip := range ips {
		for i := 0; i < 3; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = ip
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("IP %s 的第 %d 次请求应返回 200, 实际返回 %d", ip, i+1, w.Code)
			}
		}
	}
}

// TestRateLimitHandler_PathRule 测试路径规则精确匹配
//
// 【功能点】验证特定路径使用特定规则，其他路径使用默认规则
// 【测试流程】
//  1. 配置 /api/login 限制为 2 次
//  2. 验证 /api/test 使用默认规则（允许 10 次）
//  3. 验证 /api/login 第 3 次返回 429
func TestRateLimitHandler_PathRule(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  100,
		DefaultBurst: 100,
		Store:        "memory",
		Rules: []config.RateLimitRule{
			{
				Path:    "/api/login",
				Method:  "POST",
				Rate:    2,
				Burst:   2,
				KeyType: "ip",
				Message: "登录请求过于频繁",
			},
		},
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// /api/test 使用默认规则，100 次应该都成功
	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("/api/test 请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}

	// /api/login 使用特定规则，只允许 2 次
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/login", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("/api/login 请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}

	// 第 3 次登录请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/login", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("/api/login 第 3 次请求应返回 429, 实际返回 %d", w.Code)
	}
}

// TestRateLimitHandler_GlobalKeyType 测试全局限流键类型
//
// 【功能点】验证 keyType="global" 时所有请求共享同一个配额
// 【测试流程】配置全局限制为 3，使用 3 个不同 IP 发送请求，验证总共最多 3 次成功
func TestRateLimitHandler_GlobalKeyType(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  100,
		DefaultBurst: 100,
		Store:        "memory",
		Rules: []config.RateLimitRule{
			{
				Path:    "/api/public",
				Rate:    3,
				Burst:   3,
				KeyType: "global",
			},
		},
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// 不同 IP 共享全局限流
	ips := []string{"192.168.1.1:1234", "192.168.1.2:1234", "192.168.1.3:1234"}

	successCount := 0
	for _, ip := range ips {
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/public", nil)
			req.RemoteAddr = ip
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			}
		}
	}

	// 全局限制为 3，所以最多只有 3 次成功
	if successCount > 3 {
		t.Errorf("全局限流应最多允许 3 次请求, 实际允许 %d 次", successCount)
	}
}

// TestRateLimitHandler_WildcardPath 测试路径通配符匹配
//
// 【功能点】验证 path="/api/*" 匹配所有 /api/ 开头的路径
// 【测试流程】配置通配符规则限制为 2，验证第 3 次请求返回 429
func TestRateLimitHandler_WildcardPath(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  100,
		DefaultBurst: 100,
		Store:        "memory",
		Rules: []config.RateLimitRule{
			{
				Path:    "/api/*",
				Rate:    2,
				Burst:   2,
				KeyType: "ip",
			},
		},
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// /api/test 应匹配通配符规则
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}

	// 第 3 次应被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("第 3 次请求应返回 429, 实际返回 %d", w.Code)
	}
}

// TestRateLimitHandler_XForwardedFor 测试 X-Forwarded-For 头的 IP 获取
//
// 【功能点】验证代理场景下从 X-Forwarded-For 头获取真实 IP 进行限流
// 【测试流程】设置 X-Forwarded-For 头发送请求，验证按该 IP 限流
func TestRateLimitHandler_XForwardedFor(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  3,
		DefaultBurst: 3,
		Store:        "memory",
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// 使用 X-Forwarded-For 头
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Forwarded-For", "10.0.0.100")
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}

	// 第 4 次应被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.100")
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("第 4 次请求应返回 429, 实际返回 %d", w.Code)
	}
}

// TestRateLimitHandler_XRealIP 测试 X-Real-IP 头的 IP 获取
//
// 【功能点】验证从 X-Real-IP 头获取真实 IP 进行限流
// 【测试流程】设置 X-Real-IP 头发送请求，验证按该 IP 限流
func TestRateLimitHandler_XRealIP(t *testing.T) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  3,
		DefaultBurst: 3,
		Store:        "memory",
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	// 使用 X-Real-IP 头
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Real-IP", "10.0.0.200")
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d 应返回 200, 实际返回 %d", i+1, w.Code)
		}
	}

	// 第 4 次应被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.200")
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("第 4 次请求应返回 429, 实际返回 %d", w.Code)
	}
}

// ==================== 辅助函数测试 ====================

// TestFindMatchingRule 测试规则匹配函数
//
// 【功能点】验证规则匹配逻辑（精确匹配优先、方法过滤、通配符）
// 【测试流程】
//  1. 测试精确路径匹配
//  2. 测试 HTTP 方法过滤
//  3. 测试通配符匹配
//  4. 测试无匹配返回 nil
func TestFindMatchingRule(t *testing.T) {
	rules := []config.RateLimitRule{
		{Path: "/api/login", Method: "POST", Rate: 5},
		{Path: "/api/users/*", Rate: 10},
		{Path: "/api/*", Rate: 100},
	}

	tests := []struct {
		name         string
		method       string
		path         string
		expectedRate int
		shouldMatch  bool
	}{
		{"精确匹配 POST", "POST", "/api/login", 5, true},
		{"精确匹配 GET 不匹配方法", "GET", "/api/login", 0, false},
		{"通配符匹配 users", "GET", "/api/users/123", 10, true},
		{"通配符匹配 api", "GET", "/api/test", 100, true},
		{"无匹配", "GET", "/other/path", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := findMatchingRule(tt.method, tt.path, rules)
			if tt.shouldMatch {
				if rule == nil {
					t.Error("应该匹配规则")
				} else if rule.Rate != tt.expectedRate {
					t.Errorf("Rate = %d, want %d", rule.Rate, tt.expectedRate)
				}
			} else {
				if rule != nil {
					t.Error("不应该匹配规则")
				}
			}
		})
	}
}

// TestGenerateRateLimitKey 测试限流键生成函数
//
// 【功能点】验证不同 keyType 生成正确格式的限流键
// 【测试流程】
//  1. 测试 keyType="ip" → "ip:{IP}:{path}"
//  2. 测试 keyType="user" → "user:{userID}:{path}"
//  3. 测试 keyType="global" → "global:{path}"
//  4. 测试默认类型降级为 IP
func TestGenerateRateLimitKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		path        string
		remoteAddr  string
		userID      interface{}
		keyType     string
		expectedKey string
	}{
		{"IP 类型", "/api/test", "192.168.1.1:12345", nil, "ip", "ip:192.168.1.1:/api/test"},
		{"用户类型", "/api/test", "192.168.1.1:12345", "user123", "user", "user:user123:/api/test"},
		{"用户类型无用户", "/api/test", "192.168.1.1:12345", nil, "user", "ip:192.168.1.1:/api/test"},
		{"全局类型", "/api/test", "192.168.1.1:12345", nil, "global", "global:/api/test"},
		{"默认类型", "/api/test", "192.168.1.1:12345", nil, "", "ip:192.168.1.1:/api/test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试上下文
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", tt.path, nil)
			c.Request.RemoteAddr = tt.remoteAddr

			// 设置用户 ID
			if tt.userID != nil {
				c.Set("userID", tt.userID)
			}

			key := generateRateLimitKey(c, tt.keyType, tt.path)
			if key != tt.expectedKey {
				t.Errorf("key = %s, want %s", key, tt.expectedKey)
			}
		})
	}
}

// ==================== 基准测试 ====================
// 用于测试限流中间件的性能表现

// BenchmarkRateLimitHandler 基准测试无规则时的性能
// 测试场景：使用默认规则处理请求的速度
func BenchmarkRateLimitHandler(b *testing.B) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  10000,
		DefaultBurst: 10000,
		Store:        "memory",
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)
	}
}

// BenchmarkRateLimitHandler_WithRules 基准测试有规则时的性能
// 测试场景：存在多条规则时的规则匹配和限流检查速度
func BenchmarkRateLimitHandler_WithRules(b *testing.B) {
	cleanup := setupRateLimitTestConfig(config.RateLimitConfig{
		Enabled:      true,
		DefaultRate:  10000,
		DefaultBurst: 10000,
		Store:        "memory",
		Rules: []config.RateLimitRule{
			{Path: "/api/login", Method: "POST", Rate: 100, Burst: 100},
			{Path: "/api/users/*", Rate: 1000, Burst: 1000},
			{Path: "/api/*", Rate: 5000, Burst: 5000},
		},
	})
	defer cleanup()

	router := createTestRouter(RateLimitHandler())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		router.ServeHTTP(w, req)
	}
}
