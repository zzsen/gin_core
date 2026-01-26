// Package ratelimit 限流器测试
//
// ==================== 测试说明 ====================
// 本文件包含内存限流器的单元测试，不需要外部依赖（如 Redis）。
//
// 测试覆盖内容：
// 1. 限流器创建和初始化
// 2. 基础限流功能（令牌桶算法）
// 3. 不同 key 独立限流
// 4. 令牌恢复机制
// 5. 并发安全性
// 6. 速率动态变更
// 7. 统计信息
// 8. 资源清理
//
// 运行测试：go test -v ./ratelimit/...
// ==================================================
package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== MemoryLimiter 单元测试 ====================

// TestNewMemoryLimiter 测试内存限流器的创建
//
// 【功能点】验证内存限流器的创建和初始化
// 【测试流程】创建限流器，验证实例非空、stopCh 已初始化、interval 正确设置
func TestNewMemoryLimiter(t *testing.T) {
	limiter := NewMemoryLimiter(time.Second)
	defer limiter.Close()

	if limiter == nil {
		t.Fatal("NewMemoryLimiter 应返回非空实例")
	}

	if limiter.stopCh == nil {
		t.Error("stopCh 应被初始化")
	}

	if limiter.interval != time.Second {
		t.Errorf("interval = %v, want %v", limiter.interval, time.Second)
	}
}

// TestMemoryLimiter_Allow_Basic 测试基础限流功能
//
// 【功能点】验证令牌桶基础限流：初始 burst 个令牌，用完后拒绝
// 【测试流程】发送 burst+1 个请求，验证前 burst 个成功，最后 1 个被拒绝
func TestMemoryLimiter_Allow_Basic(t *testing.T) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	key := "test-key"
	rate := 10
	burst := 10

	// 前 10 次请求应该都允许（burst = 10）
	for i := 0; i < burst; i++ {
		allowed, err := limiter.Allow(ctx, key, rate, burst)
		if err != nil {
			t.Fatalf("Allow 返回错误: %v", err)
		}
		if !allowed {
			t.Errorf("第 %d 次请求应该被允许", i+1)
		}
	}

	// 第 11 次请求应该被拒绝（令牌用完）
	allowed, err := limiter.Allow(ctx, key, rate, burst)
	if err != nil {
		t.Fatalf("Allow 返回错误: %v", err)
	}
	if allowed {
		t.Error("超过 burst 限制后应该被拒绝")
	}
}

// TestMemoryLimiter_Allow_DifferentKeys 测试不同 key 的独立限流
//
// 【功能点】验证每个 key 有独立的令牌桶，互不影响
// 【测试流程】对 10 个不同 key 各发送 1 次请求，验证全部成功
func TestMemoryLimiter_Allow_DifferentKeys(t *testing.T) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	rate := 5
	burst := 5

	// 不同的 key 应该有独立的限流计数
	for i := 0; i < 10; i++ {
		key := "key-" + string(rune('A'+i))
		allowed, err := limiter.Allow(ctx, key, rate, burst)
		if err != nil {
			t.Fatalf("Allow 返回错误: %v", err)
		}
		if !allowed {
			t.Errorf("key %s 的第一次请求应该被允许", key)
		}
	}
}

// TestMemoryLimiter_Allow_RateRecovery 测试令牌恢复机制
//
// 【功能点】验证令牌用完后等待一段时间可恢复
// 【测试流程】消耗所有令牌，等待 150ms，验证有新令牌可用
func TestMemoryLimiter_Allow_RateRecovery(t *testing.T) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	key := "recovery-test"
	rate := 100 // 每秒 100 个令牌
	burst := 10

	// 消耗所有令牌
	for i := 0; i < burst; i++ {
		limiter.Allow(ctx, key, rate, burst)
	}

	// 验证令牌已用完
	allowed, _ := limiter.Allow(ctx, key, rate, burst)
	if allowed {
		t.Error("令牌应该已用完")
	}

	// 等待令牌恢复（rate=100/s，150ms 应恢复约 15 个令牌）
	time.Sleep(150 * time.Millisecond)

	// 应该有新的令牌可用
	allowed, err := limiter.Allow(ctx, key, rate, burst)
	if err != nil {
		t.Fatalf("Allow 返回错误: %v", err)
	}
	if !allowed {
		t.Error("等待后应该有新令牌可用")
	}
}

// TestMemoryLimiter_Allow_Concurrent 测试并发安全性
//
// 【功能点】验证多协程并发访问同一 key 不会 panic 且限流正确
// 【测试流程】50 个协程各发 10 次请求，验证总允许数不超过 burst
func TestMemoryLimiter_Allow_Concurrent(t *testing.T) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	key := "concurrent-test"
	rate := 1000
	burst := 100

	var wg sync.WaitGroup
	var allowedCount int32
	numGoroutines := 50
	requestsPerGoroutine := 10

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

// TestMemoryLimiter_RateChange 测试速率动态变更
//
// 【功能点】验证速率配置变更后限流器重新创建，新配置立即生效
// 【测试流程】先用低速率消耗令牌，再切换高速率，验证请求被允许
func TestMemoryLimiter_RateChange(t *testing.T) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	key := "rate-change-test"

	// 使用低速率，消耗完令牌
	rate1 := 5
	burst1 := 5
	for i := 0; i < burst1; i++ {
		limiter.Allow(ctx, key, rate1, burst1)
	}

	// 切换到高速率，应该重新创建限流器，新令牌桶有更多容量
	rate2 := 100
	burst2 := 100
	allowed, err := limiter.Allow(ctx, key, rate2, burst2)
	if err != nil {
		t.Fatalf("Allow 返回错误: %v", err)
	}
	if !allowed {
		t.Error("更改速率后应该允许请求")
	}
}

// TestMemoryLimiter_Stats 测试统计信息获取
//
// 【功能点】验证 Stats 返回正确的限流器类型和活跃数量
// 【测试流程】创建 5 个不同 key 的限流器，验证 stats 返回 type=memory, count=5
func TestMemoryLimiter_Stats(t *testing.T) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()

	// 创建 5 个不同 key 的限流器
	for i := 0; i < 5; i++ {
		key := "stats-test-" + string(rune('A'+i))
		limiter.Allow(ctx, key, 10, 10)
	}

	stats := limiter.Stats()

	if stats["type"] != "memory" {
		t.Errorf("stats[type] = %v, want memory", stats["type"])
	}

	count := stats["count"].(int)
	if count != 5 {
		t.Errorf("stats[count] = %v, want 5", count)
	}
}

// TestMemoryLimiter_Close 测试限流器关闭
//
// 【功能点】验证 Close 正常返回不 panic，停止清理协程
// 【测试流程】创建限流器后调用 Close，验证无错误返回
func TestMemoryLimiter_Close(t *testing.T) {
	limiter := NewMemoryLimiter(time.Millisecond * 100)

	// 正常关闭不应 panic
	err := limiter.Close()
	if err != nil {
		t.Errorf("Close 返回错误: %v", err)
	}
}

// TestMemoryLimiter_Cleanup 测试过期条目清理机制
//
// 【功能点】验证清理协程正常运行不崩溃
// 【测试流程】创建限流器后等待清理协程执行，验证无异常
func TestMemoryLimiter_Cleanup(t *testing.T) {
	// 使用较短的清理间隔进行测试
	limiter := NewMemoryLimiter(time.Millisecond * 50)
	defer limiter.Close()

	ctx := context.Background()

	// 创建一些限流器
	for i := 0; i < 3; i++ {
		key := "cleanup-test-" + string(rune('A'+i))
		limiter.Allow(ctx, key, 10, 10)
	}

	// 验证限流器已创建
	stats := limiter.Stats()
	if stats["count"].(int) != 3 {
		t.Errorf("初始 count = %v, want 3", stats["count"])
	}

	// 注意：doCleanup 只会清理超过 10 分钟未访问的条目
	// 在单元测试中我们不等待那么长时间
	// 这里只验证清理协程不会崩溃
	time.Sleep(time.Millisecond * 100)
}

// ==================== 基准测试 ====================
// 用于测试限流器的性能表现

// BenchmarkMemoryLimiter_Allow 基准测试单 key 串行访问性能
// 测试场景：同一个 key 的连续请求处理速度
func BenchmarkMemoryLimiter_Allow(b *testing.B) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	key := "bench-key"
	rate := 10000
	burst := 10000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, key, rate, burst)
	}
}

// BenchmarkMemoryLimiter_Allow_Parallel 基准测试并行访问性能
// 测试场景：多 goroutine 同时访问同一个 key 的性能
func BenchmarkMemoryLimiter_Allow_Parallel(b *testing.B) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	rate := 100000
	burst := 100000

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		key := "bench-parallel-key"
		for pb.Next() {
			limiter.Allow(ctx, key, rate, burst)
		}
	})
}

// BenchmarkMemoryLimiter_Allow_DifferentKeys 基准测试多 key 访问性能
// 测试场景：不同 key 的请求处理速度（模拟多 IP 场景）
func BenchmarkMemoryLimiter_Allow_DifferentKeys(b *testing.B) {
	limiter := NewMemoryLimiter(time.Minute)
	defer limiter.Close()

	ctx := context.Background()
	rate := 10000
	burst := 10000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench-key-" + string(rune(i%100))
		limiter.Allow(ctx, key, rate, burst)
	}
}
