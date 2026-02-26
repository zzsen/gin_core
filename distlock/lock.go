// Package distlock 提供分布式锁功能
// 用于在分布式环境下保证资源的互斥访问
package distlock

import (
	"context"
	"errors"
	"time"
)

// 预定义错误
var (
	// ErrLockNotHeld 当前未持有锁
	ErrLockNotHeld = errors.New("lock not held")

	// ErrLockAcquireFailed 获取锁失败
	ErrLockAcquireFailed = errors.New("failed to acquire lock")

	// ErrLockTimeout 获取锁超时
	ErrLockTimeout = errors.New("lock acquire timeout")

	// ErrLockAlreadyHeld 锁已被持有（用于 TryLock）
	ErrLockAlreadyHeld = errors.New("lock already held by another client")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("lock client is closed")
)

// Lock 分布式锁接口
type Lock interface {
	// Key 获取锁的键名
	Key() string

	// Token 获取锁的唯一标识（用于安全释放）
	Token() string

	// TTL 获取锁的剩余生存时间
	TTL(ctx context.Context) (time.Duration, error)

	// Unlock 释放锁
	// 只有持有锁的客户端才能释放锁
	Unlock(ctx context.Context) error

	// Extend 延长锁的过期时间
	// 用于手动续期（通常由看门狗自动处理）
	Extend(ctx context.Context, ttl time.Duration) error
}

// Locker 分布式锁客户端接口
type Locker interface {
	// TryLock 尝试获取锁（非阻塞）
	// 如果锁被其他客户端持有，立即返回 ErrLockAlreadyHeld
	TryLock(ctx context.Context, key string) (Lock, error)

	// Lock 获取锁（阻塞等待）
	// 如果锁被其他客户端持有，会等待直到获取成功或超时
	Lock(ctx context.Context, key string) (Lock, error)

	// LockWithRetry 获取锁（带重试配置）
	LockWithRetry(ctx context.Context, key string, retryCount int, retryDelay time.Duration) (Lock, error)

	// Close 关闭客户端
	// 会停止所有看门狗协程
	Close() error
}

// Config 分布式锁配置
type Config struct {
	// KeyPrefix 锁键前缀
	// 默认值: "distlock:"
	KeyPrefix string

	// DefaultTTL 锁的默认过期时间
	// 默认值: 30s
	DefaultTTL time.Duration

	// WatchdogEnabled 是否启用看门狗（自动续期）
	// 默认值: true
	WatchdogEnabled bool

	// WatchdogInterval 看门狗续期间隔
	// 默认值: DefaultTTL / 3
	WatchdogInterval time.Duration

	// RetryCount 默认重试次数（用于 Lock 方法）
	// 默认值: 30
	RetryCount int

	// RetryDelay 默认重试间隔
	// 默认值: 100ms
	RetryDelay time.Duration

	// OnLockAcquired 获取锁成功回调
	OnLockAcquired func(key, token string)

	// OnLockReleased 释放锁回调
	OnLockReleased func(key, token string)

	// OnWatchdogError 看门狗续期失败回调
	OnWatchdogError func(key, token string, err error)
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		KeyPrefix:        "distlock:",
		DefaultTTL:       30 * time.Second,
		WatchdogEnabled:  true,
		WatchdogInterval: 10 * time.Second, // DefaultTTL / 3
		RetryCount:       30,
		RetryDelay:       100 * time.Millisecond,
	}
}

// Option 配置选项函数
type Option func(*Config)

// WithKeyPrefix 设置锁键前缀
func WithKeyPrefix(prefix string) Option {
	return func(c *Config) {
		c.KeyPrefix = prefix
	}
}

// WithDefaultTTL 设置默认过期时间
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Config) {
		c.DefaultTTL = ttl
		// 自动调整看门狗间隔为 TTL 的 1/3
		if c.WatchdogInterval == 0 || c.WatchdogInterval >= ttl {
			c.WatchdogInterval = ttl / 3
		}
	}
}

// WithWatchdog 设置是否启用看门狗
func WithWatchdog(enabled bool) Option {
	return func(c *Config) {
		c.WatchdogEnabled = enabled
	}
}

// WithWatchdogInterval 设置看门狗续期间隔
func WithWatchdogInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.WatchdogInterval = interval
	}
}

// WithRetryCount 设置默认重试次数
func WithRetryCount(count int) Option {
	return func(c *Config) {
		c.RetryCount = count
	}
}

// WithRetryDelay 设置默认重试间隔
func WithRetryDelay(delay time.Duration) Option {
	return func(c *Config) {
		c.RetryDelay = delay
	}
}

// WithOnLockAcquired 设置获取锁成功回调
func WithOnLockAcquired(fn func(key, token string)) Option {
	return func(c *Config) {
		c.OnLockAcquired = fn
	}
}

// WithOnLockReleased 设置释放锁回调
func WithOnLockReleased(fn func(key, token string)) Option {
	return func(c *Config) {
		c.OnLockReleased = fn
	}
}

// WithOnWatchdogError 设置看门狗错误回调
func WithOnWatchdogError(fn func(key, token string, err error)) Option {
	return func(c *Config) {
		c.OnWatchdogError = fn
	}
}
