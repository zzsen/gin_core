// Package ratelimit 提供限流功能
// 支持内存和 Redis 两种存储方式，适用于单机和分布式场景
package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter 限流器接口
// 定义限流器的基本行为
type Limiter interface {
	// Allow 检查是否允许请求
	// key: 限流键（如 IP、用户 ID、路径等）
	// ratePerSecond: 每秒允许的请求数
	// burst: 突发容量（令牌桶大小）
	// 返回: 是否允许请求，错误信息
	Allow(ctx context.Context, key string, ratePerSecond int, burst int) (bool, error)

	// Close 关闭限流器，释放资源
	Close() error
}

// MemoryLimiter 内存限流器
// 使用 golang.org/x/time/rate 实现令牌桶算法
// 适用于单机部署场景
type MemoryLimiter struct {
	limiters sync.Map      // 存储每个 key 的限流器
	mu       sync.Mutex    // 用于创建限流器时的互斥锁
	stopCh   chan struct{} // 停止清理协程的信号
	interval time.Duration // 清理间隔
}

// limiterEntry 限流器条目
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
	rate       int
	burst      int
}

// NewMemoryLimiter 创建内存限流器
// cleanupInterval: 清理过期条目的间隔时间
func NewMemoryLimiter(cleanupInterval time.Duration) *MemoryLimiter {
	ml := &MemoryLimiter{
		stopCh:   make(chan struct{}),
		interval: cleanupInterval,
	}

	// 启动清理协程
	go ml.cleanup()

	return ml
}

// Allow 检查是否允许请求
func (ml *MemoryLimiter) Allow(ctx context.Context, key string, ratePerSecond int, burst int) (bool, error) {
	// 获取或创建限流器
	entry := ml.getOrCreate(key, ratePerSecond, burst)

	// 更新最后访问时间
	entry.lastAccess = time.Now()

	// 检查是否允许
	return entry.limiter.Allow(), nil
}

// getOrCreate 获取或创建限流器
func (ml *MemoryLimiter) getOrCreate(key string, ratePerSecond int, burst int) *limiterEntry {
	// 先尝试读取
	if v, ok := ml.limiters.Load(key); ok {
		entry := v.(*limiterEntry)
		// 如果速率配置发生变化，重新创建限流器
		if entry.rate != ratePerSecond || entry.burst != burst {
			ml.mu.Lock()
			defer ml.mu.Unlock()
			// 双重检查
			if v, ok := ml.limiters.Load(key); ok {
				entry = v.(*limiterEntry)
				if entry.rate != ratePerSecond || entry.burst != burst {
					newEntry := &limiterEntry{
						limiter:    rate.NewLimiter(rate.Limit(ratePerSecond), burst),
						lastAccess: time.Now(),
						rate:       ratePerSecond,
						burst:      burst,
					}
					ml.limiters.Store(key, newEntry)
					return newEntry
				}
			}
		}
		return entry
	}

	// 创建新的限流器
	ml.mu.Lock()
	defer ml.mu.Unlock()

	// 双重检查
	if v, ok := ml.limiters.Load(key); ok {
		return v.(*limiterEntry)
	}

	entry := &limiterEntry{
		limiter:    rate.NewLimiter(rate.Limit(ratePerSecond), burst),
		lastAccess: time.Now(),
		rate:       ratePerSecond,
		burst:      burst,
	}
	ml.limiters.Store(key, entry)
	return entry
}

// cleanup 定期清理过期的限流器
func (ml *MemoryLimiter) cleanup() {
	ticker := time.NewTicker(ml.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ml.doCleanup()
		case <-ml.stopCh:
			return
		}
	}
}

// doCleanup 执行清理操作
func (ml *MemoryLimiter) doCleanup() {
	// 清理超过 10 分钟未访问的限流器
	expireTime := time.Now().Add(-10 * time.Minute)

	ml.limiters.Range(func(key, value interface{}) bool {
		entry := value.(*limiterEntry)
		if entry.lastAccess.Before(expireTime) {
			ml.limiters.Delete(key)
		}
		return true
	})
}

// Close 关闭限流器
func (ml *MemoryLimiter) Close() error {
	close(ml.stopCh)
	return nil
}

// Stats 获取当前限流器统计信息
func (ml *MemoryLimiter) Stats() map[string]interface{} {
	count := 0
	ml.limiters.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return map[string]interface{}{
		"type":     "memory",
		"count":    count,
		"interval": ml.interval.String(),
	}
}
