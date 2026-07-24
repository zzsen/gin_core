// Package distlock Etcd 分布式锁单元测试
//
// ==================== 测试说明 ====================
// 本文件包含 EtcdLocker 的单元测试。
// 由于 Etcd 需要外部服务，本文件中的集成测试需要可用的 Etcd 实例。
// 设置环境变量 ETCD_ENDPOINTS（如 "localhost:2379"）启用集成测试，
// 未设置时集成测试自动跳过。
//
// 测试覆盖内容：
// 1. NewEtcdLocker 构造验证
// 2. TryLock 基本流程
// 3. Lock 阻塞获取
// 4. LockWithRetry 重试逻辑
// 5. Unlock 释放
// 6. Close 后操作返回 ErrClientClosed
// 7. 锁竞争场景
// 8. Token/TTL/Stats 方法
//
// 运行测试：ETCD_ENDPOINTS=localhost:2379 go test -v ./distlock/... -run TestEtcd
// ==================================================
package distlock

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func getEtcdClient(t *testing.T) *clientv3.Client {
	endpoints := os.Getenv("ETCD_ENDPOINTS")
	if endpoints == "" {
		t.Skip("ETCD_ENDPOINTS not set, skipping etcd integration test")
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoints},
		DialTimeout: 5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// TestEtcdLocker_New 测试 EtcdLocker 创建
//
// 【功能点】验证 NewEtcdLocker 正确初始化
// 【测试流程】
// 1. 使用默认配置创建 EtcdLocker
// 2. 验证返回非 nil 且 session 存在
// 3. Close 正常执行
func TestEtcdLocker_New(t *testing.T) {
	client := getEtcdClient(t)

	locker, err := NewEtcdLocker(client,
		WithKeyPrefix("test:lock:"),
		WithDefaultTTL(10*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, locker)
	defer func() { _ = locker.Close() }()

	assert.NotNil(t, locker.session)
	assert.Equal(t, "test:lock:", locker.config.KeyPrefix)
}

// TestEtcdLocker_TryLock 测试 TryLock 基本流程
//
// 【功能点】TryLock 获取锁 → Token 非空 → Unlock → 再次 TryLock 成功
// 【测试流程】
// 1. TryLock 获取锁成功
// 2. 验证 Token 非空
// 3. Unlock 释放
// 4. 再次 TryLock 应成功
func TestEtcdLocker_TryLock(t *testing.T) {
	client := getEtcdClient(t)

	locker, err := NewEtcdLocker(client, WithKeyPrefix("test:trylock:"))
	require.NoError(t, err)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()

	lock, err := locker.TryLock(ctx, "my-key")
	require.NoError(t, err)
	require.NotNil(t, lock)

	assert.Equal(t, "my-key", lock.Key())
	assert.NotEmpty(t, lock.Token())

	err = lock.Unlock(ctx)
	assert.NoError(t, err)

	// 释放后再次获取应成功
	lock2, err := locker.TryLock(ctx, "my-key")
	require.NoError(t, err)
	require.NotNil(t, lock2)
	_ = lock2.Unlock(ctx)
}

// TestEtcdLocker_TryLock_AlreadyHeld 测试锁竞争
//
// 【功能点】两个 locker 争抢同一 key，后者返回 ErrLockAlreadyHeld
// 【测试流程】
// 1. locker1 获取锁
// 2. locker2 TryLock 同一 key 应返回 ErrLockAlreadyHeld
// 3. locker1 释放后 locker2 可以获取
func TestEtcdLocker_TryLock_AlreadyHeld(t *testing.T) {
	client := getEtcdClient(t)

	locker1, err := NewEtcdLocker(client, WithKeyPrefix("test:compete:"))
	require.NoError(t, err)
	defer func() { _ = locker1.Close() }()

	locker2, err := NewEtcdLocker(client, WithKeyPrefix("test:compete:"))
	require.NoError(t, err)
	defer func() { _ = locker2.Close() }()

	ctx := context.Background()

	lock1, err := locker1.TryLock(ctx, "shared")
	require.NoError(t, err)

	_, err = locker2.TryLock(ctx, "shared")
	assert.ErrorIs(t, err, ErrLockAlreadyHeld)

	_ = lock1.Unlock(ctx)

	lock2, err := locker2.TryLock(ctx, "shared")
	require.NoError(t, err)
	_ = lock2.Unlock(ctx)
}

// TestEtcdLocker_Lock_Blocking 测试阻塞获取
//
// 【功能点】Lock 阻塞直到锁可用
// 【测试流程】
// 1. locker1 获取锁
// 2. locker2 Lock 阻塞
// 3. goroutine 延迟释放 locker1 的锁
// 4. locker2 应在释放后成功获取
func TestEtcdLocker_Lock_Blocking(t *testing.T) {
	client := getEtcdClient(t)

	locker1, err := NewEtcdLocker(client, WithKeyPrefix("test:block:"))
	require.NoError(t, err)
	defer func() { _ = locker1.Close() }()

	locker2, err := NewEtcdLocker(client, WithKeyPrefix("test:block:"))
	require.NoError(t, err)
	defer func() { _ = locker2.Close() }()

	ctx := context.Background()

	lock1, err := locker1.TryLock(ctx, "blocking-key")
	require.NoError(t, err)

	go func() {
		time.Sleep(200 * time.Millisecond)
		_ = lock1.Unlock(ctx)
	}()

	ctxTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	lock2, err := locker2.Lock(ctxTimeout, "blocking-key")
	require.NoError(t, err)
	assert.NotNil(t, lock2)
	_ = lock2.Unlock(ctx)
}

// TestEtcdLocker_LockWithRetry 测试重试获取
//
// 【功能点】LockWithRetry 在重试次数内获取锁
func TestEtcdLocker_LockWithRetry(t *testing.T) {
	client := getEtcdClient(t)

	locker1, err := NewEtcdLocker(client, WithKeyPrefix("test:retry:"))
	require.NoError(t, err)
	defer func() { _ = locker1.Close() }()

	locker2, err := NewEtcdLocker(client, WithKeyPrefix("test:retry:"))
	require.NoError(t, err)
	defer func() { _ = locker2.Close() }()

	ctx := context.Background()

	lock1, err := locker1.TryLock(ctx, "retry-key")
	require.NoError(t, err)

	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = lock1.Unlock(ctx)
	}()

	lock2, err := locker2.LockWithRetry(ctx, "retry-key", 10, 50*time.Millisecond)
	require.NoError(t, err)
	assert.NotNil(t, lock2)
	_ = lock2.Unlock(ctx)
}

// TestEtcdLocker_TTL 测试 TTL 查询
//
// 【功能点】获取锁后 TTL > 0
func TestEtcdLocker_TTL(t *testing.T) {
	client := getEtcdClient(t)

	locker, err := NewEtcdLocker(client,
		WithKeyPrefix("test:ttl:"),
		WithDefaultTTL(15*time.Second),
	)
	require.NoError(t, err)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "ttl-key")
	require.NoError(t, err)
	defer func() { _ = lock.Unlock(ctx) }()

	ttl, err := lock.TTL(ctx)
	require.NoError(t, err)
	assert.True(t, ttl > 0, "TTL should be > 0, got %v", ttl)
}

// TestEtcdLocker_Close 测试关闭后操作
//
// 【功能点】Close 后 TryLock 返回 ErrClientClosed
func TestEtcdLocker_Close(t *testing.T) {
	client := getEtcdClient(t)

	locker, err := NewEtcdLocker(client, WithKeyPrefix("test:close:"))
	require.NoError(t, err)

	err = locker.Close()
	assert.NoError(t, err)

	ctx := context.Background()
	_, err = locker.TryLock(ctx, "any")
	assert.ErrorIs(t, err, ErrClientClosed)
}

// TestEtcdLocker_Stats 测试统计信息
//
// 【功能点】Stats 返回活跃锁数量
func TestEtcdLocker_Stats(t *testing.T) {
	client := getEtcdClient(t)

	locker, err := NewEtcdLocker(client, WithKeyPrefix("test:stats:"))
	require.NoError(t, err)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "stats-key")
	require.NoError(t, err)

	stats := locker.Stats()
	assert.Equal(t, "etcd", stats["type"])
	assert.Equal(t, 1, stats["active_locks"])

	_ = lock.Unlock(ctx)
}
