// Package ratelimit Redis 限流器 miniredis 单元测试
//
// ==================== 测试说明 ====================
// 本文件使用 miniredis 模拟 Redis，测试滑动窗口和令牌桶的完整逻辑，
// 不需要真实 Redis 连接。
//
// 测试覆盖内容：
// 1. 滑动窗口 Allow 基本限流
// 2. 滑动窗口 Allow burst > rate 场景
// 3. 令牌桶 AllowTokenBucket 基本限流
// 4. 令牌桶 AllowTokenBucket burst > rate 场景
// 5. 不同 key 独立限流
// 6. 并发安全性
//
// 运行测试：go test -v ./ratelimit/... -run "TestRedis_"
// ==================================================
package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMiniredis(t *testing.T) (*miniredis.Miniredis, goredis.UniversalClient) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { client.Close() })
	return mr, client
}

// TestRedis_Allow_Basic 滑动窗口基本限流
func TestRedis_Allow_Basic(t *testing.T) {
	mr, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	ctx := context.Background()

	ratePerSec := 5
	burst := 5

	for i := 0; i < ratePerSec; i++ {
		allowed, err := limiter.Allow(ctx, "key1", ratePerSec, burst)
		require.NoError(t, err)
		assert.True(t, allowed, "第 %d 次请求应被允许", i+1)
	}

	allowed, err := limiter.Allow(ctx, "key1", ratePerSec, burst)
	require.NoError(t, err)
	assert.False(t, allowed, "超过限制后应被拒绝")

	_ = mr
}

// TestRedis_Allow_BurstGreaterThanRate 滑动窗口 burst > rate
func TestRedis_Allow_BurstGreaterThanRate(t *testing.T) {
	_, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	ctx := context.Background()

	ratePerSec := 5
	burst := 10

	for i := 0; i < burst; i++ {
		allowed, err := limiter.Allow(ctx, "key-burst", ratePerSec, burst)
		require.NoError(t, err)
		assert.True(t, allowed, "第 %d 次请求应被允许（burst=10）", i+1)
	}

	allowed, err := limiter.Allow(ctx, "key-burst", ratePerSec, burst)
	require.NoError(t, err)
	assert.False(t, allowed)
}

// TestRedis_AllowTokenBucket_Basic 令牌桶基本限流
func TestRedis_AllowTokenBucket_Basic(t *testing.T) {
	_, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	ctx := context.Background()

	ratePerSec := 5
	burst := 5

	for i := 0; i < burst; i++ {
		allowed, err := limiter.AllowTokenBucket(ctx, "tb-key1", ratePerSec, burst)
		require.NoError(t, err)
		assert.True(t, allowed, "第 %d 次请求应被允许", i+1)
	}

	allowed, err := limiter.AllowTokenBucket(ctx, "tb-key1", ratePerSec, burst)
	require.NoError(t, err)
	assert.False(t, allowed, "令牌耗尽后应被拒绝")
}

// TestRedis_AllowTokenBucket_LargerBurst 令牌桶 burst 大于 rate
func TestRedis_AllowTokenBucket_LargerBurst(t *testing.T) {
	_, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	ctx := context.Background()

	ratePerSec := 5
	burst := 10

	for i := 0; i < burst; i++ {
		allowed, err := limiter.AllowTokenBucket(ctx, "tb-burst", ratePerSec, burst)
		require.NoError(t, err)
		assert.True(t, allowed, "第 %d 次请求应被允许（burst=10）", i+1)
	}

	allowed, err := limiter.AllowTokenBucket(ctx, "tb-burst", ratePerSec, burst)
	require.NoError(t, err)
	assert.False(t, allowed)
}

// TestRedis_DifferentKeys 不同 key 独立限流
func TestRedis_DifferentKeys(t *testing.T) {
	_, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		key := "indep-" + string(rune('A'+i))
		allowed, err := limiter.Allow(ctx, key, 1, 1)
		require.NoError(t, err)
		assert.True(t, allowed)
	}
}

// TestRedis_Concurrent 并发安全性
func TestRedis_Concurrent(t *testing.T) {
	_, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	ctx := context.Background()

	var wg sync.WaitGroup
	var allowedCount int32
	var errorCount int32

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, err := limiter.Allow(ctx, "concurrent", 100, 100)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				return
			}
			if allowed {
				atomic.AddInt32(&allowedCount, 1)
			}
		}()
	}

	wg.Wait()
	assert.Equal(t, int32(0), errorCount, "不应有错误")
	assert.Greater(t, allowedCount, int32(0), "至少应有一个请求被允许")
	t.Logf("并发测试: 50 请求, 允许 %d, 错误 %d", allowedCount, errorCount)
}

// TestRedis_Close 关闭限流器
func TestRedis_Close(t *testing.T) {
	_, client := setupMiniredis(t)
	limiter := NewRedisLimiter(client, "test:")
	assert.NoError(t, limiter.Close())
}
