// Package distlock 提供分布式锁功能
// 本文件实现基于 Redis 的分布式锁
package distlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// luaAcquireLock 获取锁的 Lua 脚本
// 使用 SET NX EX 原子操作，保证锁的互斥性
const luaAcquireLock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    -- 已持有锁，刷新过期时间（可重入）
    redis.call("PEXPIRE", KEYS[1], ARGV[2])
    return 1
elseif redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2]) then
    return 1
else
    return 0
end
`

// luaReleaseLock 释放锁的 Lua 脚本
// 只有持有锁的客户端才能释放，防止误删其他客户端的锁
const luaReleaseLock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
`

// luaExtendLock 续期锁的 Lua 脚本
// 只有持有锁的客户端才能续期
const luaExtendLock = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
`

// luaGetTTL 获取锁剩余时间的 Lua 脚本
// 只有持有锁的客户端才能查询
const luaGetTTL = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PTTL", KEYS[1])
else
    return -3
end
`

// RedisLocker 基于 Redis 的分布式锁客户端
type RedisLocker struct {
	client   redis.UniversalClient
	config   *Config
	mu       sync.Mutex
	closed   bool
	locks    map[string]*redisLock // 管理所有活跃的锁
	stopChan chan struct{}
}

// NewRedisLocker 创建 Redis 分布式锁客户端
// 参数：
//   - client: Redis 客户端
//   - opts: 配置选项
//
// 返回：
//   - *RedisLocker: 分布式锁客户端
func NewRedisLocker(client redis.UniversalClient, opts ...Option) *RedisLocker {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	return &RedisLocker{
		client:   client,
		config:   config,
		locks:    make(map[string]*redisLock),
		stopChan: make(chan struct{}),
	}
}

// TryLock 尝试获取锁（非阻塞）
// 如果锁被其他客户端持有，立即返回 ErrLockAlreadyHeld
//
// 参数：
//   - ctx: 上下文
//   - key: 锁的键名
//
// 返回：
//   - Lock: 锁对象，获取成功时返回
//   - error: 获取失败时返回错误
//
// 使用示例：
//
//	lock, err := locker.TryLock(ctx, "my-resource")
//	if err != nil {
//	    if errors.Is(err, distlock.ErrLockAlreadyHeld) {
//	        // 锁被其他客户端持有
//	    }
//	    return err
//	}
//	defer lock.Unlock(ctx)
//	// 执行业务逻辑
func (r *RedisLocker) TryLock(ctx context.Context, key string) (Lock, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrClientClosed
	}
	r.mu.Unlock()

	token := generateToken()
	fullKey := r.config.KeyPrefix + key

	ok, err := r.acquire(ctx, fullKey, token, r.config.DefaultTTL)
	if err != nil {
		return nil, fmt.Errorf("【分布式锁】获取锁失败: %w", err)
	}
	if !ok {
		return nil, ErrLockAlreadyHeld
	}

	lock := r.createLock(fullKey, token)
	return lock, nil
}

// Lock 获取锁（阻塞等待）
// 如果锁被其他客户端持有，会等待直到获取成功或超时
//
// 参数：
//   - ctx: 上下文，可通过 context.WithTimeout 设置超时
//   - key: 锁的键名
//
// 返回：
//   - Lock: 锁对象
//   - error: 获取失败或超时时返回错误
//
// 使用示例：
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	lock, err := locker.Lock(ctx, "my-resource")
//	if err != nil {
//	    return err
//	}
//	defer lock.Unlock(context.Background())
//	// 执行业务逻辑
func (r *RedisLocker) Lock(ctx context.Context, key string) (Lock, error) {
	return r.LockWithRetry(ctx, key, r.config.RetryCount, r.config.RetryDelay)
}

// LockWithRetry 获取锁（带重试配置）
// 参数：
//   - ctx: 上下文
//   - key: 锁的键名
//   - retryCount: 重试次数
//   - retryDelay: 重试间隔
//
// 返回：
//   - Lock: 锁对象
//   - error: 获取失败时返回错误
func (r *RedisLocker) LockWithRetry(ctx context.Context, key string, retryCount int, retryDelay time.Duration) (Lock, error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil, ErrClientClosed
	}
	r.mu.Unlock()

	token := generateToken()
	fullKey := r.config.KeyPrefix + key

	for i := 0; i <= retryCount; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ok, err := r.acquire(ctx, fullKey, token, r.config.DefaultTTL)
		if err != nil {
			return nil, fmt.Errorf("【分布式锁】获取锁失败: %w", err)
		}
		if ok {
			lock := r.createLock(fullKey, token)
			return lock, nil
		}

		// 等待后重试
		if i < retryCount {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay):
			}
		}
	}

	return nil, ErrLockTimeout
}

// acquire 尝试获取锁
func (r *RedisLocker) acquire(ctx context.Context, key, token string, ttl time.Duration) (bool, error) {
	result, err := r.client.Eval(ctx, luaAcquireLock, []string{key}, token, ttl.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// createLock 创建锁对象并启动看门狗
func (r *RedisLocker) createLock(key, token string) *redisLock {
	lock := &redisLock{
		locker: r,
		key:    key,
		token:  token,
	}

	r.mu.Lock()
	r.locks[key+":"+token] = lock
	r.mu.Unlock()

	// 启动看门狗
	if r.config.WatchdogEnabled {
		lock.startWatchdog()
	}

	// 触发回调
	if r.config.OnLockAcquired != nil {
		go r.config.OnLockAcquired(key, token)
	}

	return lock
}

// removeLock 移除锁记录
func (r *RedisLocker) removeLock(key, token string) {
	r.mu.Lock()
	delete(r.locks, key+":"+token)
	r.mu.Unlock()
}

// Close 关闭客户端
// 会停止所有看门狗协程，但不会自动释放锁
func (r *RedisLocker) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true
	close(r.stopChan)

	// 停止所有锁的看门狗
	for _, lock := range r.locks {
		lock.stopWatchdog()
	}

	return nil
}

// redisLock Redis 分布式锁
type redisLock struct {
	locker      *RedisLocker
	key         string
	token       string
	mu          sync.Mutex
	watchdogCtx context.Context
	watchdogFn  context.CancelFunc
}

// Key 获取锁的键名
func (l *redisLock) Key() string {
	return l.key
}

// Token 获取锁的唯一标识
func (l *redisLock) Token() string {
	return l.token
}

// TTL 获取锁的剩余生存时间
func (l *redisLock) TTL(ctx context.Context) (time.Duration, error) {
	result, err := l.locker.client.Eval(ctx, luaGetTTL, []string{l.key}, l.token).Int()
	if err != nil {
		return 0, err
	}

	switch result {
	case -3:
		return 0, ErrLockNotHeld
	case -2:
		return 0, ErrLockNotHeld // key 不存在
	case -1:
		return 0, nil // key 存在但没有过期时间
	default:
		return time.Duration(result) * time.Millisecond, nil
	}
}

// Unlock 释放锁
func (l *redisLock) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 停止看门狗
	l.stopWatchdog()

	// 释放锁
	result, err := l.locker.client.Eval(ctx, luaReleaseLock, []string{l.key}, l.token).Int()
	if err != nil {
		return fmt.Errorf("【分布式锁】释放锁失败: %w", err)
	}

	// 移除锁记录
	l.locker.removeLock(l.key, l.token)

	// 触发回调
	if l.locker.config.OnLockReleased != nil {
		go l.locker.config.OnLockReleased(l.key, l.token)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Extend 延长锁的过期时间
func (l *redisLock) Extend(ctx context.Context, ttl time.Duration) error {
	result, err := l.locker.client.Eval(ctx, luaExtendLock, []string{l.key}, l.token, ttl.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("【分布式锁】续期失败: %w", err)
	}

	if result == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// startWatchdog 启动看门狗
func (l *redisLock) startWatchdog() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.watchdogCtx != nil {
		return // 已经启动
	}

	ctx, cancel := context.WithCancel(context.Background())
	l.watchdogCtx = ctx
	l.watchdogFn = cancel

	go l.watchdogLoop(ctx)
}

// stopWatchdog 停止看门狗
func (l *redisLock) stopWatchdog() {
	if l.watchdogFn != nil {
		l.watchdogFn()
		l.watchdogFn = nil
		l.watchdogCtx = nil
	}
}

// watchdogLoop 看门狗循环
func (l *redisLock) watchdogLoop(ctx context.Context) {
	ticker := time.NewTicker(l.locker.config.WatchdogInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.locker.stopChan:
			return
		case <-ticker.C:
			err := l.Extend(ctx, l.locker.config.DefaultTTL)
			if err != nil {
				if l.locker.config.OnWatchdogError != nil {
					l.locker.config.OnWatchdogError(l.key, l.token, err)
				}
				// 如果续期失败（锁已丢失），停止看门狗
				if err == ErrLockNotHeld {
					return
				}
			}
		}
	}
}

// generateToken 生成唯一标识
func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Stats 获取客户端统计信息
func (r *RedisLocker) Stats() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()

	return map[string]interface{}{
		"type":             "redis",
		"keyPrefix":        r.config.KeyPrefix,
		"defaultTTL":       r.config.DefaultTTL.String(),
		"watchdogEnabled":  r.config.WatchdogEnabled,
		"watchdogInterval": r.config.WatchdogInterval.String(),
		"activeLocks":      len(r.locks),
		"closed":           r.closed,
	}
}
