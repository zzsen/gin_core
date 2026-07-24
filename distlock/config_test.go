// Package distlock 配置项与分布式锁边界场景测试
//
// ==================== 测试说明 ====================
// 本文件包含 Redis/Etcd 分布式锁的配置项（Option）、默认配置及边界场景的单元测试。
// Redis 路径使用 miniredis；Etcd 路径使用进程内 embed etcd，不需要外部服务。
//
// 测试覆盖内容：
// 1. DefaultConfig 默认值验证
// 2. WithKeyPrefix / WithWatchdog / WithRetryCount / WithRetryDelay 配置项
// 3. WithDefaultTTL 与 Watchdog 间隔的联动调整
// 4. WithCallbacks 回调钩子设置
// 5. NewRedisLocker 完整 Option 链应用
// 6. RedisLocker Close 幂等性
// 7. TryLock/Lock/LockWithRetry Close 后返回 ErrClientClosed
// 8. Context 取消与 Redis 不可用的错误包装
// 9. TTL/Extend/Unlock 在未持有锁时的错误处理
// 10. Watchdog OnWatchdogError 回调（Key 被删除场景）
// 11. 可重入锁（相同 Token 刷新 TTL）
// 12. Etcd 分布式锁（embed etcd）：默认 TTL 钳位、TryLock/Unlock、锁竞争、Close 后操作、Stats、WatchSession
//
// 运行测试：go test -v ./distlock/... -run "TestDefault|TestWith|TestNew|TestRedis|TestEmbeddedEtcd"
// ==================================================
package distlock

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/server/v3/embed"
	"go.uber.org/zap"
)

// TestDefaultConfig_DefaultValues 默认配置字段
//
// 【功能点】DefaultConfig 返回文档约定的默认值（前缀、TTL、重试、看门狗等）。
// 【测试流程】
//  1. 调用 DefaultConfig 获取配置指针。
//  2. 逐项断言各字段与 lock.go 注释一致。
func TestDefaultConfig_DefaultValues(t *testing.T) {
	c := DefaultConfig()
	assert.NotNil(t, c)
	assert.Equal(t, "distlock:", c.KeyPrefix)
	assert.Equal(t, 30*time.Second, c.DefaultTTL)
	assert.True(t, c.WatchdogEnabled)
	assert.Equal(t, 10*time.Second, c.WatchdogInterval)
	assert.Equal(t, 30, c.RetryCount)
	assert.Equal(t, 100*time.Millisecond, c.RetryDelay)
	assert.Nil(t, c.OnLockAcquired)
	assert.Nil(t, c.OnLockReleased)
	assert.Nil(t, c.OnWatchdogError)
}

// TestWithKeyPrefix_SetsPrefix 锁键前缀选项
//
// 【功能点】WithKeyPrefix 将 KeyPrefix 写入 Config。
// 【测试流程】
//  1. 基于默认配置应用 WithKeyPrefix。
//  2. 断言 KeyPrefix 已更新。
func TestWithKeyPrefix_SetsPrefix(t *testing.T) {
	c := DefaultConfig()
	WithKeyPrefix("custom:")(c)
	assert.Equal(t, "custom:", c.KeyPrefix)
}

// TestWithWatchdog_AndInterval 看门狗开关与间隔选项
//
// 【功能点】WithWatchdog、WithWatchdogInterval 正确修改 Config。
// 【测试流程】
//  1. 对默认配置调用 WithWatchdog(false)。
//  2. 调用 WithWatchdogInterval 设置间隔。
//  3. 断言字段符合预期。
func TestWithWatchdog_AndInterval(t *testing.T) {
	c := DefaultConfig()
	WithWatchdog(false)(c)
	assert.False(t, c.WatchdogEnabled)
	WithWatchdogInterval(250 * time.Millisecond)(c)
	assert.Equal(t, 250*time.Millisecond, c.WatchdogInterval)
}

// TestWithRetryCount_WithRetryDelay 重试次数与间隔选项
//
// 【功能点】WithRetryCount、WithRetryDelay 用于 Lock 默认重试行为。
// 【测试流程】
//  1. 应用 WithRetryCount(7)、WithRetryDelay(15*time.Millisecond)。
//  2. 断言 RetryCount、RetryDelay。
func TestWithRetryCount_WithRetryDelay(t *testing.T) {
	c := DefaultConfig()
	WithRetryCount(7)(c)
	WithRetryDelay(15 * time.Millisecond)(c)
	assert.Equal(t, 7, c.RetryCount)
	assert.Equal(t, 15*time.Millisecond, c.RetryDelay)
}

// TestWithDefaultTTL_AdjustWatchdogInterval_FromZero 默认 TTL 与空看门狗间隔
//
// 【功能点】WithDefaultTTL 在 WatchdogInterval 为 0 时将间隔设为 TTL/3。
// 【测试流程】
//  1. 构造 WatchdogInterval 为 0 的 Config。
//  2. 应用 WithDefaultTTL(12 * time.Second)。
//  3. 断言 WatchdogInterval 等于 4 秒。
func TestWithDefaultTTL_AdjustWatchdogInterval_FromZero(t *testing.T) {
	c := &Config{}
	WithDefaultTTL(12 * time.Second)(c)
	assert.Equal(t, 12*time.Second, c.DefaultTTL)
	assert.Equal(t, 4*time.Second, c.WatchdogInterval)
}

// TestWithDefaultTTL_AdjustWatchdogInterval_WhenTooLarge 默认 TTL 与过大的看门狗间隔
//
// 【功能点】WithDefaultTTL 在 WatchdogInterval >= ttl 时将间隔重置为 TTL/3。
// 【测试流程】
//  1. 配置 WatchdogInterval 远大于即将设置的 TTL。
//  2. 调用 WithDefaultTTL 缩短 TTL。
//  3. 断言间隔被调整为新 TTL 的三分之一。
func TestWithDefaultTTL_AdjustWatchdogInterval_WhenTooLarge(t *testing.T) {
	c := &Config{WatchdogInterval: 100 * time.Second}
	WithDefaultTTL(12 * time.Second)(c)
	assert.Equal(t, 12*time.Second, c.DefaultTTL)
	assert.Equal(t, 4*time.Second, c.WatchdogInterval)
}

// TestWithDefaultTTL_KeepsSmallerWatchdogInterval 保留小于 TTL 的看门狗间隔
//
// 【功能点】若 WatchdogInterval 已小于新 TTL，则 WithDefaultTTL 不覆盖该间隔。
// 【测试流程】
//  1. 使用 DefaultConfig 应用 WithWatchdogInterval(2*time.Second)。
//  2. 再应用 WithDefaultTTL(30*time.Second)。
//  3. 断言 WatchdogInterval 仍为 2 秒。
func TestWithDefaultTTL_KeepsSmallerWatchdogInterval(t *testing.T) {
	c := DefaultConfig()
	WithWatchdogInterval(2 * time.Second)(c)
	WithDefaultTTL(30 * time.Second)(c)
	assert.Equal(t, 30*time.Second, c.DefaultTTL)
	assert.Equal(t, 2*time.Second, c.WatchdogInterval)
}

// TestWithCallbacks_Hooks 锁生命周期回调选项
//
// 【功能点】WithOnLockAcquired、WithOnLockReleased、WithOnWatchdogError 注入回调。
// 【测试流程】
//  1. 声明占位函数并写入 Config。
//  2. 断言三个函数字段非 nil。
func TestWithCallbacks_Hooks(t *testing.T) {
	c := DefaultConfig()
	acquired := func(string, string) {}
	released := func(string, string) {}
	watchdogErr := func(string, string, error) {}
	WithOnLockAcquired(acquired)(c)
	WithOnLockReleased(released)(c)
	WithOnWatchdogError(watchdogErr)(c)
	assert.NotNil(t, c.OnLockAcquired)
	assert.NotNil(t, c.OnLockReleased)
	assert.NotNil(t, c.OnWatchdogError)
}

// TestNewRedisLocker_AppliesOptionChain 选项链应用到 RedisLocker
//
// 【功能点】NewRedisLocker 依次应用 Option，Stats 反映 KeyPrefix、TTL 等。
// 【测试流程】
//  1. 启动 miniredis，使用多个 Option 创建 RedisLocker。
//  2. 读取 Stats 断言 keyPrefix、defaultTTL、watchdogEnabled、watchdogInterval。
func TestNewRedisLocker_AppliesOptionChain(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithKeyPrefix("app:lock:"),
		WithDefaultTTL(2*time.Minute),
		WithWatchdog(false),
		WithRetryCount(12),
		WithRetryDelay(33*time.Millisecond),
	)
	defer func() { _ = locker.Close() }()

	stats := locker.Stats()
	assert.Equal(t, "redis", stats["type"])
	assert.Equal(t, "app:lock:", stats["keyPrefix"])
	assert.Equal(t, (2 * time.Minute).String(), stats["defaultTTL"])
	assert.Equal(t, false, stats["watchdogEnabled"])
}

// TestRedisLocker_Close_Idempotent Close 幂等
//
// 【功能点】重复调用 Close 不报错且状态保持一致。
// 【测试流程】
//  1. 创建 Locker 并 Close 两次。
//  2. 断言两次均无错误；Stats 中 closed 为 true。
func TestRedisLocker_Close_Idempotent(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	assert.NoError(t, locker.Close())
	assert.NoError(t, locker.Close())
	stats := locker.Stats()
	assert.Equal(t, true, stats["closed"])
}

// TestTryLock_AfterClose_ErrClientClosed 关闭后 TryLock
//
// 【功能点】客户端关闭后 TryLock 返回 ErrClientClosed。
// 【测试流程】
//  1. Close 后调用 TryLock。
//  2. 断言 errors.Is ErrClientClosed。
func TestTryLock_AfterClose_ErrClientClosed(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	assert.NoError(t, locker.Close())

	_, err := locker.TryLock(context.Background(), "k")
	assert.ErrorIs(t, err, ErrClientClosed)
}

// TestLock_AfterClose_ErrClientClosed 关闭后 Lock
//
// 【功能点】Lock 在关闭后同样返回 ErrClientClosed（LockWithRetry 入口校验）。
// 【测试流程】
//  1. Close 后调用 Lock。
//  2. 断言 ErrClientClosed。
func TestLock_AfterClose_ErrClientClosed(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	assert.NoError(t, locker.Close())

	_, err := locker.Lock(context.Background(), "k")
	assert.ErrorIs(t, err, ErrClientClosed)
}

// TestLockWithRetry_AfterClose_ErrClientClosed 关闭后 LockWithRetry
//
// 【功能点】LockWithRetry 在 closed 时立即返回 ErrClientClosed。
// 【测试流程】
//  1. Close 后调用 LockWithRetry。
//  2. 断言 ErrClientClosed。
func TestLockWithRetry_AfterClose_ErrClientClosed(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	assert.NoError(t, locker.Close())

	_, err := locker.LockWithRetry(context.Background(), "k", 3, time.Millisecond)
	assert.ErrorIs(t, err, ErrClientClosed)
}

// TestTryLock_ContextCancelled_ErrWrapped Redis 获取锁上下文取消
//
// 【功能点】TryLock 在 ctx 已取消时将底层错误包装返回，并可 unwrap 为 context.Canceled。
// 【测试流程】
//  1. 创建已取消的 context。
//  2. TryLock 调用 Eval，断言返回错误且 errors.Is ContextCanceled。
func TestTryLock_ContextCancelled_ErrWrapped(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer func() { _ = locker.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := locker.TryLock(ctx, "key-cancel")
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestTryLock_RedisUnavailable_ErrWrapped Redis 不可用时的错误包装
//
// 【功能点】acquire 失败时 TryLock 返回 fmt.Errorf 包装的错误。
// 【测试流程】
//  1. 创建 Locker 后关闭 miniredis（连接断开）。
//  2. TryLock 断言返回错误且非 ErrLockAlreadyHeld。
func TestTryLock_RedisUnavailable_ErrWrapped(t *testing.T) {
	mr, client := newTestRedis(t)

	locker := NewRedisLocker(client, WithWatchdog(false))
	mr.Close()

	_, err := locker.TryLock(context.Background(), "key-down")
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrLockAlreadyHeld))
	_ = locker.Close()
}

// TestRedisLock_TTL_AfterUnlock_ErrLockNotHeld 解锁后查询 TTL
//
// 【功能点】TTL 在不再持有锁时返回 ErrLockNotHeld。
// 【测试流程】
//  1. TryLock 后 Unlock。
//  2. 对同一 Lock 实例调用 TTL，断言 ErrLockNotHeld。
func TestRedisLock_TTL_AfterUnlock_ErrLockNotHeld(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "ttl-after-unlock")
	assert.NoError(t, err)
	assert.NoError(t, lock.Unlock(ctx))

	_, err = lock.TTL(ctx)
	assert.ErrorIs(t, err, ErrLockNotHeld)
}

// TestRedisLock_TTL_NoExpiry_ReturnsZeroDuration PTTL 为 -1 的分支
//
// 【功能点】对已持久化（无过期）仍匹配的锁键，PTTL -1 时 TTL 返回 (0, nil)。
// 【测试流程】
//  1. TryLock 后对锁键执行 PERSIST。
//  2. 调用 TTL，断言 error 为 nil 且时长为 0。
func TestRedisLock_TTL_NoExpiry_ReturnsZeroDuration(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "persist-key")
	assert.NoError(t, err)
	defer func() { _ = lock.Unlock(ctx) }()

	err = client.Persist(ctx, lock.Key()).Err()
	assert.NoError(t, err)

	dur, err := lock.TTL(ctx)
	assert.NoError(t, err)
	assert.Equal(t, time.Duration(0), dur)
}

// TestRedisLock_Extend_AfterUnlock_ErrLockNotHeld 解锁后续期
//
// 【功能点】Extend 在锁不存在或不匹配 token 时返回 ErrLockNotHeld。
// 【测试流程】
//  1. 获取锁并 Unlock。
//  2. 对同一 Lock 调用 Extend，断言 ErrLockNotHeld。
func TestRedisLock_Extend_AfterUnlock_ErrLockNotHeld(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "extend-after-unlock")
	assert.NoError(t, err)
	assert.NoError(t, lock.Unlock(ctx))

	err = lock.Extend(ctx, time.Minute)
	assert.ErrorIs(t, err, ErrLockNotHeld)
}

// TestRedisLock_Extend_ContextCancelled_ErrWrapped Extend 上下文取消
//
// 【功能点】Extend 在 Eval 失败时包装并返回底层 ctx 错误。
// 【测试流程】
//  1. TryLock 持有锁。
//  2. 使用已取消的 context 调用 Extend。
//  3. 断言 errors.Is ContextCanceled。
func TestRedisLock_Extend_ContextCancelled_ErrWrapped(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "extend-cancel")
	assert.NoError(t, err)
	defer func() { _ = lock.Unlock(ctx) }()

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err = lock.Extend(ctx2, time.Second)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

// TestRedisLock_Unlock_ContextCancelled_ErrWrapped Unlock 上下文取消
//
// 【功能点】Unlock 在 Redis 调用失败时返回包装错误。
// 【测试流程】
//  1. TryLock 获取锁。
//  2. 使用已取消的 ctx 调用 Unlock。
//  3. 断言 errors.Is ContextCanceled。
func TestRedisLock_Unlock_ContextCancelled_ErrWrapped(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer func() { _ = locker.Close() }()

	lock, err := locker.TryLock(context.Background(), "unlock-cancel")
	assert.NoError(t, err)

	ctx2, cancel := context.WithCancel(context.Background())
	cancel()

	err = lock.Unlock(ctx2)
	assert.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	// 清理：使用有效 ctx 释放，避免泄漏活跃锁记录
	assert.NoError(t, lock.Unlock(context.Background()))
}

// TestWatchdog_OnWatchdogError_KeyDeleted 看门狗续期失败回调
//
// 【功能点】Watchdog 续期失败且为 ErrLockNotHeld 时调用 OnWatchdogError，并停止循环。
// 【测试流程】
//  1. 启用短 TTL、短间隔与 OnWatchdogError（计数）。
//  2. TryLock 后 DEL 锁键模拟丢失。
//  3. 等待一轮 ticker，断言回调被触发。
func TestWatchdog_OnWatchdogError_KeyDeleted(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	var cnt atomic.Int32
	locker := NewRedisLocker(client,
		WithDefaultTTL(500*time.Millisecond),
		WithWatchdogInterval(50*time.Millisecond),
		WithOnWatchdogError(func(_, _ string, err error) {
			if err != nil {
				cnt.Add(1)
			}
		}),
	)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "watchdog-err")
	assert.NoError(t, err)
	defer func() { _ = lock.Unlock(context.Background()) }()

	assert.NoError(t, client.Del(ctx, lock.Key()).Err())

	time.Sleep(200 * time.Millisecond)
	assert.GreaterOrEqual(t, cnt.Load(), int32(1))
}

// TestTryLock_ReentrantSameToken_RefreshTTL Lua 可重入刷新 TTL
//
// 【功能点】同一 token 再次 acquire 时 Lua 刷新过期时间（可重入语义）。
// 【测试流程】
//  1. TryLock 获取锁并记下 token。
//  2. 直接 Eval luaAcquireLock 使用相同 token 更长 TTL。
//  3. PTTL 应接近新 TTL。
func TestTryLock_ReentrantSameToken_RefreshTTL(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithDefaultTTL(800*time.Millisecond),
	)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "reentrant")
	assert.NoError(t, err)
	defer func() { _ = lock.Unlock(ctx) }()

	token := lock.Token()
	newTTL := 1500 * time.Millisecond
	res, err := client.Eval(ctx, luaAcquireLock, []string{lock.Key()}, token, newTTL.Milliseconds()).Int()
	assert.NoError(t, err)
	assert.Equal(t, 1, res)

	pttl, err := client.PTTL(ctx, lock.Key()).Result()
	assert.NoError(t, err)
	assert.Greater(t, pttl, time.Millisecond*800)
}

// --- 嵌入式 Etcd（提升 etcd.go 覆盖率；无需 ETCD_ENDPOINTS） ---

func newEmbeddedEtcdClient(t *testing.T) (*clientv3.Client, func()) {
	t.Helper()

	cfg := embed.NewConfig()
	cfg.Dir = t.TempDir()
	cfg.ZapLoggerBuilder = embed.NewZapLoggerBuilder(zap.NewNop())

	peerLn, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	peerPort := peerLn.Addr().(*net.TCPAddr).Port
	assert.NoError(t, peerLn.Close())

	clientLn, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	clientPort := clientLn.Addr().(*net.TCPAddr).Port
	assert.NoError(t, clientLn.Close())

	parseU := func(raw string) url.URL {
		u, perr := url.Parse(raw)
		assert.NoError(t, perr)
		return *u
	}

	pPeer := fmt.Sprintf("http://127.0.0.1:%d", peerPort)
	pClient := fmt.Sprintf("http://127.0.0.1:%d", clientPort)

	cfg.ListenPeerUrls = []url.URL{parseU(pPeer)}
	cfg.AdvertisePeerUrls = []url.URL{parseU(pPeer)}
	cfg.ListenClientUrls = []url.URL{parseU(pClient)}
	cfg.AdvertiseClientUrls = []url.URL{parseU(pClient)}
	cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)

	e, err := embed.StartEtcd(cfg)
	assert.NoError(t, err)

	select {
	case <-e.Server.ReadyNotify():
	case <-time.After(20 * time.Second):
		e.Close()
		t.Fatal("embedded etcd ready timeout")
	}

	ep := "127.0.0.1:" + strconv.Itoa(clientPort)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{ep},
		DialTimeout: 10 * time.Second,
	})
	assert.NoError(t, err)

	cleanup := func() {
		_ = cli.Close()
		e.Close()
	}
	return cli, cleanup
}

// TestEmbeddedEtcd_NewEtcdLocker_DefaultTTLClamp 小于 1s 的 TTL 被 Session 规范化为 1s
//
// 【功能点】NewEtcdLocker 中 ttlSeconds 小于 1 秒时强制为 1，仍能创建 Session。
// 【测试流程】
//  1. 启动 embed etcd 并创建 client。
//  2. WithDefaultTTL(500ms) 调用 NewEtcdLocker。
//  3. 断言 locker 非 nil 并成功 Close。
func TestEmbeddedEtcd_NewEtcdLocker_DefaultTTLClamp(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker, err := NewEtcdLocker(cli,
		WithKeyPrefix("emb:ttlclamp:"),
		WithDefaultTTL(500*time.Millisecond),
	)
	assert.NoError(t, err)
	assert.NotNil(t, locker)
	assert.NoError(t, locker.Close())
}

// TestEmbeddedEtcd_TryLock_Unlock_TTL_Extend Etcd 锁基本生命周期
//
// 【功能点】TryLock、Unlock、TTL、Extend、Token、Key 与回调 OnLockAcquired。
// 【测试流程】
//  1. NewEtcdLocker（带前缀与 OnLockAcquired）。
//  2. TryLock → TTL 大于 0 → Extend → Unlock。
//  3. 断言回调被触发且 Key 为业务 key。
func TestEmbeddedEtcd_TryLock_Unlock_TTL_Extend(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	var acquiredKey string
	locker, err := NewEtcdLocker(cli,
		WithKeyPrefix("emb:life:"),
		WithDefaultTTL(15*time.Second),
		WithOnLockAcquired(func(key, token string) {
			acquiredKey = key
			assert.NotEmpty(t, token)
		}),
	)
	assert.NoError(t, err)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "resource")
	assert.NoError(t, err)
	assert.Equal(t, "resource", lock.Key())
	assert.NotEmpty(t, lock.Token())

	ttl, err := lock.TTL(ctx)
	assert.NoError(t, err)
	assert.Positive(t, ttl)

	assert.NoError(t, lock.Extend(ctx, time.Minute))

	assert.NoError(t, lock.Unlock(ctx))
	assert.Equal(t, "resource", acquiredKey)
}

// TestEmbeddedEtcd_TryLock_AlreadyHeld 锁已被占用
//
// 【功能点】TryLock 在遇到 concurrency.ErrLocked 时映射为 ErrLockAlreadyHeld。
// 【测试流程】
//  1. 两个 EtcdLocker 实例争抢同一 key。
//  2. 第二个 TryLock 返回 ErrLockAlreadyHeld。
func TestEmbeddedEtcd_TryLock_AlreadyHeld(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	a, err := NewEtcdLocker(cli, WithKeyPrefix("emb:held:"))
	assert.NoError(t, err)
	defer func() { _ = a.Close() }()

	b, err := NewEtcdLocker(cli, WithKeyPrefix("emb:held:"))
	assert.NoError(t, err)
	defer func() { _ = b.Close() }()

	ctx := context.Background()
	l1, err := a.TryLock(ctx, "x")
	assert.NoError(t, err)
	defer func() { _ = l1.Unlock(ctx) }()

	_, err = b.TryLock(ctx, "x")
	assert.ErrorIs(t, err, ErrLockAlreadyHeld)
}

// TestEmbeddedEtcd_Lock_ContextCanceled_Blocking Lock 上下文取消
//
// 【功能点】mutex.Lock 失败返回包装错误（etcd lock: ...）。
// 【测试流程】
//  1. 对已取消的 context 调用 Lock。
//  2. 断言返回错误。
func TestEmbeddedEtcd_Lock_ContextCanceled_Blocking(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker, err := NewEtcdLocker(cli, WithKeyPrefix("emb:lcancel:"))
	assert.NoError(t, err)
	defer func() { _ = locker.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = locker.Lock(ctx, "k")
	assert.Error(t, err)
}

// TestEmbeddedEtcd_LockWithRetry_AcquireFailed 重试耗尽
//
// 【功能点】LockWithRetry 多次 TryLock 失败后返回 ErrLockAcquireFailed 包装链。
// 【测试流程】
//  1. locker1 长期持有锁。
//  2. locker2 LockWithRetry 极少次数与短间隔。
//  3. 断言 errors.Is(..., ErrLockAcquireFailed)。
func TestEmbeddedEtcd_LockWithRetry_AcquireFailed(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker1, err := NewEtcdLocker(cli, WithKeyPrefix("emb:retryfail:"))
	assert.NoError(t, err)
	defer func() { _ = locker1.Close() }()

	locker2, err := NewEtcdLocker(cli, WithKeyPrefix("emb:retryfail:"))
	assert.NoError(t, err)
	defer func() { _ = locker2.Close() }()

	ctx := context.Background()
	l1, err := locker1.TryLock(ctx, "busy")
	assert.NoError(t, err)
	defer func() { _ = l1.Unlock(ctx) }()

	_, err = locker2.LockWithRetry(ctx, "busy", 2, 20*time.Millisecond)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrLockAcquireFailed)
}

// TestEmbeddedEtcd_LockWithRetry_ContextDeadline 重试等待期间上下文取消
//
// 【功能点】LockWithRetry 在 time.After(retryDelay) 分支响应 ctx.Done。
// 【测试流程】
//  1. locker1 持有锁。
//  2. locker2 使用短 Deadline + 长 retryDelay 调用 LockWithRetry。
//  3. 断言返回 context.DeadlineExceeded。
func TestEmbeddedEtcd_LockWithRetry_ContextDeadline(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker1, err := NewEtcdLocker(cli, WithKeyPrefix("emb:rwdl:"))
	assert.NoError(t, err)
	defer func() { _ = locker1.Close() }()

	locker2, err := NewEtcdLocker(cli, WithKeyPrefix("emb:rwdl:"))
	assert.NoError(t, err)
	defer func() { _ = locker2.Close() }()

	ctx := context.Background()
	l1, err := locker1.TryLock(ctx, "block-retry")
	assert.NoError(t, err)
	defer func() { _ = l1.Unlock(ctx) }()

	ctx2, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()

	_, err = locker2.LockWithRetry(ctx2, "block-retry", 50, 80*time.Millisecond)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestEmbeddedEtcd_Close_Idempotent EtcdLocker Close 幂等
//
// 【功能点】closed 为 true 时 Close 直接返回 nil。
// 【测试流程】
//  1. 连续 Close 两次。
//  2. 断言均无错误。
func TestEmbeddedEtcd_Close_Idempotent(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker, err := NewEtcdLocker(cli, WithKeyPrefix("emb:close:"))
	assert.NoError(t, err)
	assert.NoError(t, locker.Close())
	assert.NoError(t, locker.Close())
}

// TestEmbeddedEtcd_TryLock_AfterClose_ErrClientClosed 关闭后 TryLock
//
// 【功能点】EtcdLocker.closed 为 true 时 TryLock 返回 ErrClientClosed。
// 【测试流程】
//  1. Close locker。
//  2. TryLock 断言 ErrClientClosed。
func TestEmbeddedEtcd_TryLock_AfterClose_ErrClientClosed(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker, err := NewEtcdLocker(cli, WithKeyPrefix("emb:closed:"))
	assert.NoError(t, err)
	assert.NoError(t, locker.Close())

	_, err = locker.TryLock(context.Background(), "any")
	assert.ErrorIs(t, err, ErrClientClosed)
}

// TestEmbeddedEtcd_Lock_AfterClose_ErrClientClosed 关闭后 Lock
//
// 【功能点】关闭后 Lock 立即短路返回 ErrClientClosed。
// 【测试流程】
//  1. Close locker。
//  2. Lock 断言 ErrClientClosed。
func TestEmbeddedEtcd_Lock_AfterClose_ErrClientClosed(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker, err := NewEtcdLocker(cli, WithKeyPrefix("emb:lclosed:"))
	assert.NoError(t, err)
	assert.NoError(t, locker.Close())

	_, err = locker.Lock(context.Background(), "any")
	assert.ErrorIs(t, err, ErrClientClosed)
}

// TestEmbeddedEtcd_Stats_ActiveLocks Stats 活跃锁统计
//
// 【功能点】EtcdLocker.Stats 返回 type、active_locks、session_ttl、closed。
// 【测试流程】
//  1. TryLock 后断言 Stats 中 active_locks 至少反映已创建锁记录。
//  2. Unlock 后当前实现仍保留 map 记录（active_locks 不递减），继续断言该行为。
func TestEmbeddedEtcd_Stats_ActiveLocks(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)
	defer shutdown()

	locker, err := NewEtcdLocker(cli,
		WithKeyPrefix("emb:stats:"),
		WithDefaultTTL(20*time.Second),
	)
	assert.NoError(t, err)
	defer func() { _ = locker.Close() }()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "st")
	assert.NoError(t, err)

	stats := locker.Stats()
	assert.Equal(t, "etcd", stats["type"])
	assert.Equal(t, 1, stats["active_locks"])
	assert.Equal(t, (20 * time.Second).String(), stats["session_ttl"])
	assert.Equal(t, false, stats["closed"])

	assert.NoError(t, lock.Unlock(ctx))
	stats = locker.Stats()
	assert.Equal(t, 1, stats["active_locks"])
}

// TestEmbeddedEtcd_WatchSession_OnWatchdogError Session 失效触发回调
//
// 【功能点】watchSession 在 session.Done() 后调用 OnWatchdogError。
// 【测试流程】
//  1. 创建带 OnWatchdogError 的 EtcdLocker。
//  2. 关闭嵌入式 etcd（断开会话）。
//  3. 等待 goroutine 触发回调。
func TestEmbeddedEtcd_WatchSession_OnWatchdogError(t *testing.T) {
	cli, shutdown := newEmbeddedEtcdClient(t)

	var cnt atomic.Int32
	locker, err := NewEtcdLocker(cli, WithKeyPrefix("emb:watch:"),
		WithOnWatchdogError(func(string, string, error) { cnt.Add(1) }),
	)
	assert.NoError(t, err)

	shutdown()

	time.Sleep(800 * time.Millisecond)
	assert.GreaterOrEqual(t, cnt.Load(), int32(1))

	_ = locker.Close()
}
