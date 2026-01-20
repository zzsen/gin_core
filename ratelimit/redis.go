// Package ratelimit 提供限流功能
// 本文件实现基于 Redis 的分布式限流器
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisLimiter Redis 限流器
// 使用滑动窗口算法实现分布式限流
// 适用于集群部署场景
type RedisLimiter struct {
	client    redis.UniversalClient
	keyPrefix string
}

// NewRedisLimiter 创建 Redis 限流器
// client: Redis 客户端
// keyPrefix: 限流键前缀，用于区分不同应用
func NewRedisLimiter(client redis.UniversalClient, keyPrefix string) *RedisLimiter {
	if keyPrefix == "" {
		keyPrefix = "ratelimit:"
	}
	return &RedisLimiter{
		client:    client,
		keyPrefix: keyPrefix,
	}
}

// slidingWindowScript 滑动窗口限流 Lua 脚本
// 使用 Redis 的有序集合实现滑动窗口
const slidingWindowScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])

-- 移除窗口外的请求记录
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- 获取当前窗口内的请求数
local count = redis.call('ZCARD', key)

if count < limit then
    -- 添加当前请求
    redis.call('ZADD', key, now, now .. '-' .. math.random())
    -- 设置过期时间
    redis.call('EXPIRE', key, math.ceil(window / 1000))
    return 1
else
    return 0
end
`

// Allow 检查是否允许请求（滑动窗口算法）
func (rl *RedisLimiter) Allow(ctx context.Context, key string, ratePerSecond int, burst int) (bool, error) {
	if rl.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}

	fullKey := rl.keyPrefix + key
	now := time.Now().UnixMilli()
	window := int64(1000) // 1 秒窗口（毫秒）
	limit := int64(ratePerSecond)

	// 如果 burst 大于 rate，使用 burst 作为限制
	if burst > ratePerSecond {
		limit = int64(burst)
	}

	result, err := rl.client.Eval(ctx, slidingWindowScript, []string{fullKey}, now, window, limit).Int()
	if err != nil {
		return false, fmt.Errorf("redis eval error: %w", err)
	}

	return result == 1, nil
}

// tokenBucketScript 令牌桶限流 Lua 脚本
// 另一种实现方式，支持突发流量
const tokenBucketScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])

-- 获取当前桶状态
local bucket = redis.call('HMGET', key, 'tokens', 'last_time')
local tokens = tonumber(bucket[1])
local last_time = tonumber(bucket[2])

-- 初始化
if tokens == nil then
    tokens = burst
    last_time = now
end

-- 计算新增的令牌数
local elapsed = (now - last_time) / 1000
local new_tokens = elapsed * rate
tokens = math.min(burst, tokens + new_tokens)

-- 尝试获取令牌
if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'tokens', tokens, 'last_time', now)
    redis.call('EXPIRE', key, math.ceil(burst / rate) + 1)
    return 1
else
    redis.call('HMSET', key, 'tokens', tokens, 'last_time', now)
    redis.call('EXPIRE', key, math.ceil(burst / rate) + 1)
    return 0
end
`

// AllowTokenBucket 检查是否允许请求（令牌桶算法）
// 这是另一种实现方式，支持更好的突发流量处理
func (rl *RedisLimiter) AllowTokenBucket(ctx context.Context, key string, ratePerSecond int, burst int) (bool, error) {
	if rl.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}

	fullKey := rl.keyPrefix + "tb:" + key
	now := time.Now().UnixMilli()

	result, err := rl.client.Eval(ctx, tokenBucketScript, []string{fullKey}, now, ratePerSecond, burst).Int()
	if err != nil {
		return false, fmt.Errorf("redis eval error: %w", err)
	}

	return result == 1, nil
}

// Close 关闭限流器
func (rl *RedisLimiter) Close() error {
	// Redis 客户端由外部管理，这里不关闭
	return nil
}

// Stats 获取限流器统计信息
func (rl *RedisLimiter) Stats() map[string]interface{} {
	return map[string]interface{}{
		"type":      "redis",
		"keyPrefix": rl.keyPrefix,
	}
}
