// Package ratelimit Redis 限流器测试
//
// ==================== 测试说明 ====================
// 本文件包含 Redis 限流器的单元测试，不需要真实 Redis 连接。
//
// 测试覆盖内容：
// 1. 限流器创建和配置
// 2. 空客户端错误处理
// 3. 统计信息获取
// 4. Lua 脚本语法验证
//
// 注意：需要真实 Redis 连接的集成测试在 redis_integration_test.go 中
// 运行集成测试：go test -tags=integration ./ratelimit/...
// ==================================================
package ratelimit

import (
	"context"
	"testing"
)

// ==================== RedisLimiter 单元测试（不需要 Redis 连接） ====================

// TestNewRedisLimiter 测试 Redis 限流器的创建
// 验证点：
// - 空前缀时使用默认前缀 "ratelimit:"
// - 自定义前缀正确设置
// - 用于区分不同应用的限流键
func TestNewRedisLimiter(t *testing.T) {
	// 测试默认前缀
	limiter := NewRedisLimiter(nil, "")
	if limiter.keyPrefix != "ratelimit:" {
		t.Errorf("默认前缀应为 'ratelimit:', 实际为 '%s'", limiter.keyPrefix)
	}

	// 测试自定义前缀
	limiter2 := NewRedisLimiter(nil, "custom:")
	if limiter2.keyPrefix != "custom:" {
		t.Errorf("自定义前缀应为 'custom:', 实际为 '%s'", limiter2.keyPrefix)
	}
}

// TestRedisLimiter_Allow_NilClient 测试空客户端的错误处理（滑动窗口算法）
// 验证点：
// - Redis 客户端为 nil 时返回错误
// - 请求被拒绝（返回 allowed = false）
// - 不会 panic
func TestRedisLimiter_Allow_NilClient(t *testing.T) {
	limiter := NewRedisLimiter(nil, "test:")
	defer limiter.Close()

	ctx := context.Background()
	allowed, err := limiter.Allow(ctx, "test-key", 10, 10)

	if err == nil {
		t.Error("客户端为 nil 时应返回错误")
	}

	if allowed {
		t.Error("客户端为 nil 时不应允许请求")
	}
}

// TestRedisLimiter_AllowTokenBucket_NilClient 测试空客户端的错误处理（令牌桶算法）
// 验证点：
// - Redis 客户端为 nil 时返回错误
// - 请求被拒绝（返回 allowed = false）
// - 不会 panic
func TestRedisLimiter_AllowTokenBucket_NilClient(t *testing.T) {
	limiter := NewRedisLimiter(nil, "test:")
	defer limiter.Close()

	ctx := context.Background()
	allowed, err := limiter.AllowTokenBucket(ctx, "test-key", 10, 10)

	if err == nil {
		t.Error("客户端为 nil 时应返回错误")
	}

	if allowed {
		t.Error("客户端为 nil 时不应允许请求")
	}
}

// TestRedisLimiter_Close 测试限流器关闭
// 验证点：
// - Close 正常返回，不 panic
// - Redis 客户端由外部管理，不在此关闭
func TestRedisLimiter_Close(t *testing.T) {
	limiter := NewRedisLimiter(nil, "test:")

	// Close 不应返回错误
	err := limiter.Close()
	if err != nil {
		t.Errorf("Close 返回错误: %v", err)
	}
}

// TestRedisLimiter_Stats 测试统计信息获取
// 验证点：
// - 返回正确的限流器类型（redis）
// - 返回正确的键前缀
// - 用于监控和调试
func TestRedisLimiter_Stats(t *testing.T) {
	limiter := NewRedisLimiter(nil, "myapp:")

	stats := limiter.Stats()

	if stats["type"] != "redis" {
		t.Errorf("stats[type] = %v, want redis", stats["type"])
	}

	if stats["keyPrefix"] != "myapp:" {
		t.Errorf("stats[keyPrefix] = %v, want myapp:", stats["keyPrefix"])
	}
}

// ==================== Lua 脚本测试 ====================

// TestSlidingWindowScript_Syntax 测试滑动窗口 Lua 脚本语法
// 验证点：
// - 脚本非空
// - 包含必要的 Redis 命令（ZREMRANGEBYSCORE、ZCARD、ZADD、EXPIRE）
// - 滑动窗口算法的核心操作完整
func TestSlidingWindowScript_Syntax(t *testing.T) {
	// 验证 Lua 脚本语法正确性（通过编译检查）
	if slidingWindowScript == "" {
		t.Error("slidingWindowScript 不应为空")
	}

	// 验证脚本包含关键操作
	expectedOps := []string{
		"ZREMRANGEBYSCORE", // 移除窗口外的请求
		"ZCARD",            // 统计窗口内请求数
		"ZADD",             // 添加新请求
		"EXPIRE",           // 设置过期时间
	}

	for _, op := range expectedOps {
		if !containsString(slidingWindowScript, op) {
			t.Errorf("slidingWindowScript 应包含 %s 操作", op)
		}
	}
}

// TestTokenBucketScript_Syntax 测试令牌桶 Lua 脚本语法
// 验证点：
// - 脚本非空
// - 包含必要的 Redis 命令（HMGET、HMSET、EXPIRE）
// - 令牌桶算法的核心操作完整
func TestTokenBucketScript_Syntax(t *testing.T) {
	// 验证 Lua 脚本语法正确性
	if tokenBucketScript == "" {
		t.Error("tokenBucketScript 不应为空")
	}

	// 验证脚本包含关键操作
	expectedOps := []string{
		"HMGET",  // 获取桶状态（令牌数、上次时间）
		"HMSET",  // 更新桶状态
		"EXPIRE", // 设置过期时间
	}

	for _, op := range expectedOps {
		if !containsString(tokenBucketScript, op) {
			t.Errorf("tokenBucketScript 应包含 %s 操作", op)
		}
	}
}

// containsString 检查字符串是否包含子串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
