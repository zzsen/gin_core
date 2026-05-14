// Package distlock 提供分布式锁功能
// 本文件实现基于 Etcd 的分布式锁，使用 Raft 共识保证强一致性
package distlock

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// EtcdLocker 基于 Etcd 的分布式锁客户端
// 使用 Etcd Lease + concurrency.Mutex 实现强一致性分布式锁
type EtcdLocker struct {
	client  *clientv3.Client
	config  *Config
	mu      sync.Mutex
	locks   map[string]*etcdLock
	closed  bool
	session *concurrency.Session
}

// NewEtcdLocker 创建 Etcd 分布式锁客户端
//
// 流程：
// 1. 应用配置选项
// 2. 创建 Etcd Session（内置 KeepAlive 看门狗）
// 3. 监听 Session 失效事件
func NewEtcdLocker(client *clientv3.Client, opts ...Option) (*EtcdLocker, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	ttlSeconds := int(config.DefaultTTL.Seconds())
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}

	session, err := concurrency.NewSession(client, concurrency.WithTTL(ttlSeconds))
	if err != nil {
		return nil, fmt.Errorf("create etcd session: %w", err)
	}

	locker := &EtcdLocker{
		client:  client,
		config:  config,
		locks:   make(map[string]*etcdLock),
		session: session,
	}

	go locker.watchSession()

	return locker, nil
}

// watchSession 监听 Session 失效（看门狗逻辑）
func (el *EtcdLocker) watchSession() {
	<-el.session.Done()
	if el.config.OnWatchdogError != nil {
		el.config.OnWatchdogError("", "", fmt.Errorf("etcd session expired"))
	}
}

// TryLock 尝试获取锁（非阻塞）
// 如果锁被其他客户端持有，立即返回 ErrLockAlreadyHeld
func (el *EtcdLocker) TryLock(ctx context.Context, key string) (Lock, error) {
	el.mu.Lock()
	if el.closed {
		el.mu.Unlock()
		return nil, ErrClientClosed
	}
	el.mu.Unlock()

	fullKey := el.config.KeyPrefix + key

	mutex := concurrency.NewMutex(el.session, fullKey)

	err := mutex.TryLock(ctx)
	if err != nil {
		if err == concurrency.ErrLocked {
			return nil, ErrLockAlreadyHeld
		}
		return nil, fmt.Errorf("etcd trylock: %w", err)
	}

	lock := &etcdLock{
		mutex:   mutex,
		session: el.session,
		client:  el.client,
		key:     key,
	}

	el.mu.Lock()
	el.locks[key] = lock
	el.mu.Unlock()

	if el.config.OnLockAcquired != nil {
		el.config.OnLockAcquired(key, lock.Token())
	}

	return lock, nil
}

// Lock 获取锁（阻塞等待）
// 如果锁被其他客户端持有，会等待直到获取成功或 ctx 取消
func (el *EtcdLocker) Lock(ctx context.Context, key string) (Lock, error) {
	el.mu.Lock()
	if el.closed {
		el.mu.Unlock()
		return nil, ErrClientClosed
	}
	el.mu.Unlock()

	fullKey := el.config.KeyPrefix + key

	mutex := concurrency.NewMutex(el.session, fullKey)

	if err := mutex.Lock(ctx); err != nil {
		return nil, fmt.Errorf("etcd lock: %w", err)
	}

	lock := &etcdLock{
		mutex:   mutex,
		session: el.session,
		client:  el.client,
		key:     key,
	}

	el.mu.Lock()
	el.locks[key] = lock
	el.mu.Unlock()

	if el.config.OnLockAcquired != nil {
		el.config.OnLockAcquired(key, lock.Token())
	}

	return lock, nil
}

// LockWithRetry 获取锁（带重试配置）
func (el *EtcdLocker) LockWithRetry(ctx context.Context, key string, retryCount int, retryDelay time.Duration) (Lock, error) {
	var lastErr error
	for i := 0; i <= retryCount; i++ {
		lock, err := el.TryLock(ctx, key)
		if err == nil {
			return lock, nil
		}
		lastErr = err
		if err == ErrClientClosed {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryDelay):
		}
	}
	return nil, fmt.Errorf("%w: %v", ErrLockAcquireFailed, lastErr)
}

// Close 关闭客户端，释放所有 Session
func (el *EtcdLocker) Close() error {
	el.mu.Lock()
	defer el.mu.Unlock()

	if el.closed {
		return nil
	}
	el.closed = true

	return el.session.Close()
}

// Stats 获取锁客户端统计信息
func (el *EtcdLocker) Stats() map[string]interface{} {
	el.mu.Lock()
	defer el.mu.Unlock()

	return map[string]interface{}{
		"type":        "etcd",
		"active_locks": len(el.locks),
		"session_ttl": el.config.DefaultTTL.String(),
		"closed":      el.closed,
	}
}

// etcdLock 实现 Lock 接口
type etcdLock struct {
	mutex   *concurrency.Mutex
	session *concurrency.Session
	client  *clientv3.Client
	key     string
}

// Key 获取锁的键名
func (l *etcdLock) Key() string {
	return l.key
}

// Token 获取锁的唯一标识（LeaseID 字符串）
func (l *etcdLock) Token() string {
	return strconv.FormatInt(int64(l.session.Lease()), 10)
}

// TTL 获取锁的剩余生存时间
func (l *etcdLock) TTL(ctx context.Context) (time.Duration, error) {
	resp, err := l.client.TimeToLive(ctx, l.session.Lease())
	if err != nil {
		return 0, fmt.Errorf("etcd ttl: %w", err)
	}
	if resp.TTL <= 0 {
		return 0, nil
	}
	return time.Duration(resp.TTL) * time.Second, nil
}

// Unlock 释放锁
func (l *etcdLock) Unlock(ctx context.Context) error {
	return l.mutex.Unlock(ctx)
}

// Extend 续期锁（通过 KeepAliveOnce 续期 Lease）
func (l *etcdLock) Extend(ctx context.Context, _ time.Duration) error {
	_, err := l.client.KeepAliveOnce(ctx, l.session.Lease())
	if err != nil {
		return fmt.Errorf("etcd extend: %w", err)
	}
	return nil
}
