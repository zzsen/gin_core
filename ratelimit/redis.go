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

// Allow 使用滑动窗口算法检查请求是否被允许。
//
// 算法原理：
//  1. 以 Redis Sorted Set 存储请求记录，score 为请求时间戳（毫秒）
//  2. 每次请求先移除窗口（1 秒）外的旧记录
//  3. 统计窗口内的请求数，若未超过限制则写入新记录并放行
//
// 当 burst > ratePerSecond 时，使用 burst 作为窗口内的请求数上限，
// 从而允许短时间内的突发流量。
//
// 参数：
//   - ctx: 上下文，支持超时取消
//   - key: 限流键
//   - ratePerSecond: 每秒允许的请求数
//   - burst: 突发容量上限
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

// AllowTokenBucket 使用令牌桶算法检查请求是否被允许。
//
// 算法原理：
//  1. 以 Redis Hash 存储桶状态（当前令牌数 tokens 和上次更新时间 last_time）
//  2. 每次请求按 elapsed * rate 补充令牌，令牌数上限为 burst
//  3. 令牌充足（>= 1）时消耗一个令牌并放行，否则拒绝
//
// 与滑动窗口的区别：令牌桶天然支持突发流量——桶满时可瞬间消耗 burst 个令牌，
// 之后按 ratePerSecond 匀速恢复，适合对突发流量更宽容的场景。
//
// 参数：
//   - ctx: 上下文，支持超时取消
//   - key: 限流键（自动添加 "tb:" 前缀与滑动窗口的键隔离）
//   - ratePerSecond: 每秒令牌恢复速率
//   - burst: 令牌桶容量上限
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
