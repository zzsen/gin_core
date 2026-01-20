//go:build integration

// Package ratelimit Redis 限流器集成测试
//
// ==================== 集成测试说明 ====================
// 本文件需要真实的 Redis 连接才能运行。
// 运行命令：go test -tags=integration ./ratelimit/...
//
// 测试覆盖内容：
// 1. 滑动窗口算法的实际限流效果
// 2. 令牌桶算法的实际限流效果
// 3. 不同 key 独立限流
// 4. 窗口滑动/令牌恢复机制
// 5. 并发安全性（Redis 原子操作）
// 6. 性能基准测试
//
// 配置说明：
// - 使用 DB 15 避免影响其他数据
// - 测试结束后自动清理测试键
// - 如果无法连接 Redis，测试将失败（而非跳过）
// ==================================================
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// 硬编码的 Redis 测试配置
const (
	integrationTestRedisHost     = "localhost"
	integrationTestRedisPort     = 6379
	integrationTestRedisPassword = ""
	integrationTestRedisDB       = 15 // 使用独立的数据库避免影响其他数据
)

// getIntegrationTestRedisClient 获取测试用的 Redis 客户端
func getIntegrationTestRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", integrationTestRedisHost, integrationTestRedisPort),
		Password: integrationTestRedisPassword,
		DB:       integrationTestRedisDB,
	})
}

// requireRedis 确保 Redis 可连接，否则测试失败
func requireRedis(t *testing.T) *redis.Client {
	client := getIntegrationTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		t.Fatalf("无法连接 Redis (%s:%d): %v", integrationTestRedisHost, integrationTestRedisPort, err)
	}

	return client
}

// cleanupTestKeys 清理测试用的 Redis 键
func cleanupTestKeys(client *redis.Client, prefix string) {
	ctx := context.Background()
	keys, _ := client.Keys(ctx, prefix+"*").Result()
	if len(keys) > 0 {
		client.Del(ctx, keys...)
	}
}

// ==================== RedisLimiter 集成测试 ====================

// TestRedisLimiter_Allow_Integration 测试滑动窗口算法的基础限流功能
// 验证点：
// - 窗口内请求数不超过限制
// - 超过限制后请求被拒绝
// - Lua 脚本正确执行
func TestRedisLimiter_Allow_Integration(t *testing.T) {
	client := requireRedis(t)
	defer client.Close()

	prefix := "test:ratelimit:allow:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx := context.Background()
	key := "basic-test"
	rate := 5
	burst := 5

	// 前 5 次请求应该允许
	for i := 0; i < burst; i++ {
		allowed, err := limiter.Allow(ctx, key, rate, burst)
		if err != nil {
			t.Fatalf("Allow 返回错误: %v", err)
		}
		if !allowed {
			t.Errorf("第 %d 次请求应该被允许", i+1)
		}
	}

	// 第 6 次请求应该被拒绝
	allowed, err := limiter.Allow(ctx, key, rate, burst)
	if err != nil {
		t.Fatalf("Allow 返回错误: %v", err)
	}
	if allowed {
		t.Error("超过限制后应该被拒绝")
	}
}

// TestRedisLimiter_Allow_DifferentKeys_Integration 测试不同 key 的独立限流
// 验证点：
// - 每个 key 有独立的滑动窗口
// - 一个 key 的限流不影响其他 key
// - 适用于分布式场景下的多租户限流
func TestRedisLimiter_Allow_DifferentKeys_Integration(t *testing.T) {
	client := requireRedis(t)
	defer client.Close()

	prefix := "test:ratelimit:diffkeys:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx := context.Background()
	rate := 3
	burst := 3

	// 不同的 key 应该有独立的限流
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		allowed, err := limiter.Allow(ctx, key, rate, burst)
		if err != nil {
			t.Fatalf("Allow 返回错误: %v", err)
		}
		if !allowed {
			t.Errorf("key %s 的第一次请求应该被允许", key)
		}
	}
}

// TestRedisLimiter_Allow_WindowSliding_Integration 测试滑动窗口的滑动机制
// 验证点：
// - 窗口内请求用完后被限流
// - 等待窗口滑动后配额恢复
// - 使用 Redis 有序集合实现精确的滑动窗口
func TestRedisLimiter_Allow_WindowSliding_Integration(t *testing.T) {
	client := requireRedis(t)
	defer client.Close()

	prefix := "test:ratelimit:sliding:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx := context.Background()
	key := "sliding-test"
	rate := 5
	burst := 5

	// 消耗所有配额
	for i := 0; i < burst; i++ {
		limiter.Allow(ctx, key, rate, burst)
	}

	// 验证已被限流
	allowed, _ := limiter.Allow(ctx, key, rate, burst)
	if allowed {
		t.Error("应该被限流")
	}

	// 等待窗口滑动（1 秒窗口 + 100ms 余量）
	time.Sleep(1100 * time.Millisecond)

	// 应该有新的配额
	allowed, err := limiter.Allow(ctx, key, rate, burst)
	if err != nil {
		t.Fatalf("Allow 返回错误: %v", err)
	}
	if !allowed {
		t.Error("窗口滑动后应该允许请求")
	}
}

// TestRedisLimiter_Allow_Concurrent_Integration 测试并发安全性
// 验证点：
// - 多个客户端同时访问同一个 key 不会超限
// - Redis Lua 脚本保证原子性
// - 适用于分布式多实例场景
func TestRedisLimiter_Allow_Concurrent_Integration(t *testing.T) {
	client := requireRedis(t)
	defer client.Close()

	prefix := "test:ratelimit:concurrent:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx := context.Background()
	key := "concurrent-test"
	rate := 100
	burst := 100

	var wg sync.WaitGroup
	var allowedCount int32
	numGoroutines := 50
	requestsPerGoroutine := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				allowed, err := limiter.Allow(ctx, key, rate, burst)
				if err != nil {
					t.Errorf("Allow 返回错误: %v", err)
					return
				}
				if allowed {
					atomic.AddInt32(&allowedCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	// 允许的请求数不应超过 burst
	if allowedCount > int32(burst) {
		t.Errorf("允许的请求数 %d 超过 burst %d", allowedCount, burst)
	}

	t.Logf("并发测试: 总请求 %d, 允许 %d", numGoroutines*requestsPerGoroutine, allowedCount)
}

// TestRedisLimiter_AllowTokenBucket_Integration 测试令牌桶算法的基础限流功能
// 验证点：
// - 初始令牌数为 burst
// - 令牌用完后请求被拒绝
// - 使用 Redis Hash 存储桶状态
func TestRedisLimiter_AllowTokenBucket_Integration(t *testing.T) {
	client := requireRedis(t)
	defer client.Close()

	prefix := "test:ratelimit:tokenbucket:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx := context.Background()
	key := "tb-test"
	rate := 10
	burst := 10

	// 前 10 次请求应该允许
	for i := 0; i < burst; i++ {
		allowed, err := limiter.AllowTokenBucket(ctx, key, rate, burst)
		if err != nil {
			t.Fatalf("AllowTokenBucket 返回错误: %v", err)
		}
		if !allowed {
			t.Errorf("第 %d 次请求应该被允许", i+1)
		}
	}

	// 令牌用完后应该被拒绝
	allowed, err := limiter.AllowTokenBucket(ctx, key, rate, burst)
	if err != nil {
		t.Fatalf("AllowTokenBucket 返回错误: %v", err)
	}
	if allowed {
		t.Error("令牌用完后应该被拒绝")
	}
}

// TestRedisLimiter_AllowTokenBucket_Refill_Integration 测试令牌桶的令牌补充机制
// 验证点：
// - 令牌用完后，按 rate 速率补充
// - 等待后有新令牌可用
// - 令牌数不超过 burst 上限
func TestRedisLimiter_AllowTokenBucket_Refill_Integration(t *testing.T) {
	client := requireRedis(t)
	defer client.Close()

	prefix := "test:ratelimit:tbrefill:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx := context.Background()
	key := "tb-refill-test"
	rate := 100 // 每秒 100 个令牌
	burst := 5

	// 消耗所有令牌
	for i := 0; i < burst; i++ {
		limiter.AllowTokenBucket(ctx, key, rate, burst)
	}

	// 验证令牌已用完
	allowed, _ := limiter.AllowTokenBucket(ctx, key, rate, burst)
	if allowed {
		t.Error("令牌应该已用完")
	}

	// 等待令牌补充（rate=100/s，100ms 应补充约 10 个令牌）
	time.Sleep(100 * time.Millisecond)

	// 应该有新的令牌
	allowed, err := limiter.AllowTokenBucket(ctx, key, rate, burst)
	if err != nil {
		t.Fatalf("AllowTokenBucket 返回错误: %v", err)
	}
	if !allowed {
		t.Error("令牌补充后应该允许请求")
	}
}

// ==================== 基准测试 ====================
// 用于测试 Redis 限流器在真实环境下的性能表现

// BenchmarkRedisLimiter_Allow_Integration 基准测试滑动窗口算法性能
// 测试场景：单 key 连续请求的处理速度（包含网络延迟）
func BenchmarkRedisLimiter_Allow_Integration(b *testing.B) {
	client := getIntegrationTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		client.Close()
		b.Fatalf("无法连接 Redis: %v", err)
	}
	cancel()
	defer client.Close()

	prefix := "bench:ratelimit:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx = context.Background()
	key := "bench-key"
	rate := 100000
	burst := 100000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, key, rate, burst)
	}
}

// BenchmarkRedisLimiter_AllowTokenBucket_Integration 基准测试令牌桶算法性能
// 测试场景：单 key 连续请求的处理速度（包含网络延迟）
func BenchmarkRedisLimiter_AllowTokenBucket_Integration(b *testing.B) {
	client := getIntegrationTestRedisClient()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := client.Ping(ctx).Err(); err != nil {
		cancel()
		client.Close()
		b.Fatalf("无法连接 Redis: %v", err)
	}
	cancel()
	defer client.Close()

	prefix := "bench:ratelimit:tb:"
	defer cleanupTestKeys(client, prefix)

	limiter := NewRedisLimiter(client, prefix)
	defer limiter.Close()

	ctx = context.Background()
	key := "bench-tb-key"
	rate := 100000
	burst := 100000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.AllowTokenBucket(ctx, key, rate, burst)
	}
}
