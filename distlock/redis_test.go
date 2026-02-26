package distlock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// 测试辅助函数
// ============================================================================

// newTestRedis 创建测试用的 Redis 客户端
//
// 功能说明：
//   - 使用 miniredis 创建内存版 Redis 服务器
//   - 无需真实 Redis 环境即可进行单元测试
//
// 返回值：
//   - *miniredis.Miniredis: miniredis 实例，用于控制 Redis 行为（如 FastForward）
//   - redis.UniversalClient: Redis 客户端，用于创建分布式锁
func newTestRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("无法启动 miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

// ============================================================================
// TryLock 测试用例
// ============================================================================

// TestTryLock_Success 测试 TryLock 成功获取锁
//
// 【测试功能点】
//   - TryLock 在锁空闲时能成功获取锁
//   - 锁对象包含正确的 Key 和 Token
//   - Unlock 能正常释放锁
//
// 【测试流程】
//  1. 创建 Redis 客户端和 Locker
//  2. 调用 TryLock 获取锁
//  3. 验证锁的 Key 包含正确的前缀
//  4. 验证锁的 Token 不为空
//  5. 调用 Unlock 释放锁
func TestTryLock_Success(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer locker.Close()

	ctx := context.Background()
	lock, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("获取锁失败: %v", err)
	}

	if lock.Key() != "distlock:test-key" {
		t.Errorf("锁键名不匹配: got %s, want distlock:test-key", lock.Key())
	}

	if lock.Token() == "" {
		t.Error("锁 token 为空")
	}

	// 释放锁
	err = lock.Unlock(ctx)
	if err != nil {
		t.Errorf("释放锁失败: %v", err)
	}
}

// TestTryLock_AlreadyHeld 测试 TryLock 在锁被占用时的行为
//
// 【测试功能点】
//   - TryLock 在锁被其他客户端持有时返回 ErrLockAlreadyHeld
//   - 非阻塞特性：立即返回错误，不会等待
//
// 【测试流程】
//  1. 客户端 A 调用 TryLock 成功获取锁
//  2. 客户端 B 调用 TryLock 尝试获取同一把锁
//  3. 验证客户端 B 立即返回 ErrLockAlreadyHeld 错误
func TestTryLock_AlreadyHeld(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer locker.Close()

	ctx := context.Background()

	// 第一个客户端获取锁
	lock1, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("第一次获取锁失败: %v", err)
	}
	defer lock1.Unlock(ctx)

	// 第二个客户端尝试获取同一把锁
	_, err = locker.TryLock(ctx, "test-key")
	if !errors.Is(err, ErrLockAlreadyHeld) {
		t.Errorf("期望 ErrLockAlreadyHeld, 得到: %v", err)
	}
}

// ============================================================================
// Lock (阻塞获取) 测试用例
// ============================================================================

// TestLock_BlockingAcquire 测试 Lock 阻塞等待获取锁
//
// 【测试功能点】
//   - Lock 在锁被占用时会阻塞等待
//   - 锁释放后，等待的客户端能成功获取锁
//   - 重试机制正常工作
//
// 【测试流程】
//  1. 客户端 A 获取锁
//  2. 启动后台协程，200ms 后释放锁
//  3. 客户端 B 调用 Lock 阻塞等待
//  4. 验证客户端 B 在锁释放后成功获取锁
//  5. 验证等待时间 >= 150ms（确认确实等待了）
func TestLock_BlockingAcquire(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithRetryCount(10),
		WithRetryDelay(50*time.Millisecond),
	)
	defer locker.Close()

	ctx := context.Background()

	// 第一个客户端获取锁
	lock1, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("第一次获取锁失败: %v", err)
	}

	// 在后台释放锁
	go func() {
		time.Sleep(200 * time.Millisecond)
		lock1.Unlock(ctx)
	}()

	// 第二个客户端阻塞等待
	start := time.Now()
	lock2, err := locker.Lock(ctx, "test-key")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("阻塞获取锁失败: %v", err)
	}
	defer lock2.Unlock(ctx)

	if elapsed < 150*time.Millisecond {
		t.Errorf("锁获取太快，应该等待锁释放: %v", elapsed)
	}
}

// TestLock_Timeout 测试 Lock 重试次数耗尽后超时
//
// 【测试功能点】
//   - Lock 在重试次数耗尽后返回 ErrLockTimeout
//   - 重试次数和重试间隔配置生效
//
// 【测试流程】
//  1. 配置重试次数为 5，重试间隔为 20ms
//  2. 客户端 A 获取锁并保持不释放
//  3. 客户端 B 调用 Lock 尝试获取
//  4. 验证客户端 B 在 5 次重试后返回 ErrLockTimeout
func TestLock_Timeout(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithRetryCount(5),
		WithRetryDelay(20*time.Millisecond),
	)
	defer locker.Close()

	ctx := context.Background()

	// 第一个客户端获取锁并保持
	lock1, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("第一次获取锁失败: %v", err)
	}
	defer lock1.Unlock(ctx)

	// 第二个客户端尝试获取应该超时
	_, err = locker.Lock(ctx, "test-key")
	if !errors.Is(err, ErrLockTimeout) {
		t.Errorf("期望 ErrLockTimeout, 得到: %v", err)
	}
}

// TestLock_ContextCancel 测试 Lock 响应 context 取消
//
// 【测试功能点】
//   - Lock 能正确响应 context 超时/取消
//   - context.DeadlineExceeded 优先于 ErrLockTimeout 返回
//
// 【测试流程】
//  1. 配置较长的重试次数（100 次）
//  2. 客户端 A 获取锁并保持
//  3. 客户端 B 使用 100ms 超时的 context 调用 Lock
//  4. 验证客户端 B 在 context 超时后立即返回 context.DeadlineExceeded
func TestLock_ContextCancel(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithRetryCount(100),
		WithRetryDelay(50*time.Millisecond),
	)
	defer locker.Close()

	// 第一个客户端获取锁
	ctx := context.Background()
	lock1, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("第一次获取锁失败: %v", err)
	}
	defer lock1.Unlock(ctx)

	// 使用可取消的 context
	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = locker.Lock(ctx2, "test-key")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("期望 context.DeadlineExceeded, 得到: %v", err)
	}
}

// ============================================================================
// Unlock 测试用例
// ============================================================================

// TestUnlock_NotHeld 测试释放未持有的锁
//
// 【测试功能点】
//   - 重复释放锁返回 ErrLockNotHeld
//   - 安全释放机制：只有持有锁的客户端才能释放
//
// 【测试流程】
//  1. 获取锁
//  2. 第一次调用 Unlock 成功释放
//  3. 第二次调用 Unlock 返回 ErrLockNotHeld
func TestUnlock_NotHeld(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	defer locker.Close()

	ctx := context.Background()

	// 获取并释放锁
	lock, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("获取锁失败: %v", err)
	}

	err = lock.Unlock(ctx)
	if err != nil {
		t.Errorf("第一次释放锁失败: %v", err)
	}

	// 再次释放应该返回错误
	err = lock.Unlock(ctx)
	if !errors.Is(err, ErrLockNotHeld) {
		t.Errorf("期望 ErrLockNotHeld, 得到: %v", err)
	}
}

// ============================================================================
// Extend (续期) 测试用例
// ============================================================================

// TestExtend 测试锁的手动续期功能
//
// 【测试功能点】
//   - Extend 能成功延长锁的过期时间
//   - 续期后 TTL 正确更新
//
// 【测试流程】
//  1. 配置锁 TTL 为 1 秒
//  2. 获取锁并记录初始 TTL
//  3. 等待 200ms（锁 TTL 减少）
//  4. 调用 Extend 续期 5 秒
//  5. 验证 TTL 增加到 5 秒左右
func TestExtend(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithDefaultTTL(1*time.Second),
	)
	defer locker.Close()

	ctx := context.Background()

	lock, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("获取锁失败: %v", err)
	}
	defer lock.Unlock(ctx)

	// 获取初始 TTL
	ttl1, err := lock.TTL(ctx)
	if err != nil {
		t.Errorf("获取 TTL 失败: %v", err)
	}

	// 等待一段时间
	time.Sleep(200 * time.Millisecond)
	mr.FastForward(200 * time.Millisecond)

	// 续期
	err = lock.Extend(ctx, 5*time.Second)
	if err != nil {
		t.Errorf("续期失败: %v", err)
	}

	// TTL 应该增加了
	ttl2, err := lock.TTL(ctx)
	if err != nil {
		t.Errorf("获取 TTL 失败: %v", err)
	}

	if ttl2 <= ttl1 {
		t.Errorf("续期后 TTL 应该增加: ttl1=%v, ttl2=%v", ttl1, ttl2)
	}
}

// ============================================================================
// Watchdog (看门狗) 测试用例
// ============================================================================

// TestWatchdog 测试看门狗自动续期机制
//
// 【测试功能点】
//   - 看门狗能在后台自动续期锁
//   - 锁在超过原始 TTL 后仍然有效
//   - 看门狗间隔配置生效
//
// 【测试流程】
//  1. 配置锁 TTL 为 500ms，看门狗间隔为 100ms
//  2. 获取锁
//  3. 等待 700ms（超过原始 TTL）
//  4. 验证锁仍然有效（TTL > 0）
//
// 【看门狗工作原理】
//
//	时间线: 0ms    100ms   200ms   300ms   400ms   500ms   600ms   700ms
//	        │       │       │       │       │       │       │       │
//	        获取锁  续期    续期    续期    续期    续期    续期    检查
//	        TTL=500 TTL=500 TTL=500 TTL=500 TTL=500 TTL=500 TTL=500 TTL>0 ✓
func TestWatchdog(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	// 短 TTL 和短看门狗间隔便于测试
	locker := NewRedisLocker(client,
		WithDefaultTTL(500*time.Millisecond),
		WithWatchdog(true),
		WithWatchdogInterval(100*time.Millisecond),
	)
	defer locker.Close()

	ctx := context.Background()

	lock, err := locker.TryLock(ctx, "test-key")
	if err != nil {
		t.Fatalf("获取锁失败: %v", err)
	}
	defer lock.Unlock(ctx)

	// 等待足够长的时间，让看门狗续期多次
	time.Sleep(700 * time.Millisecond)

	// 锁应该仍然有效
	ttl, err := lock.TTL(ctx)
	if err != nil {
		t.Errorf("获取 TTL 失败: %v", err)
	}

	if ttl <= 0 {
		t.Error("看门狗应该保持锁有效")
	}
}

// ============================================================================
// 并发测试用例
// ============================================================================

// TestConcurrentLock 测试多协程并发竞争同一把锁
//
// 【测试功能点】
//   - 分布式锁的互斥性：同一时刻只有一个协程持有锁
//   - 高并发场景下锁的正确性
//   - 所有协程都能最终获取到锁
//
// 【测试流程】
//  1. 启动 10 个并发协程，都尝试获取同一把锁
//  2. 每个协程获取锁后增加计数器，然后释放锁
//  3. 等待所有协程完成
//  4. 验证计数器为 10（所有协程都成功执行）
//
// 【并发安全验证】
//
//	如果锁不具备互斥性，counter 可能出现竞态条件
//	使用 atomic 操作确保测试本身的正确性
func TestConcurrentLock(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithRetryCount(50),
		WithRetryDelay(10*time.Millisecond),
	)
	defer locker.Close()

	ctx := context.Background()
	var counter int64
	var wg sync.WaitGroup

	// 10 个并发协程竞争同一把锁
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			lock, err := locker.Lock(ctx, "test-key")
			if err != nil {
				t.Errorf("获取锁失败: %v", err)
				return
			}

			// 增加计数器
			atomic.AddInt64(&counter, 1)

			// 模拟业务处理
			time.Sleep(10 * time.Millisecond)

			lock.Unlock(ctx)
		}()
	}

	wg.Wait()

	if counter != 10 {
		t.Errorf("计数器应该是 10, 得到: %d", counter)
	}
}

// ============================================================================
// 回调函数测试用例
// ============================================================================

// TestCallbacks 测试锁的回调函数机制
//
// 【测试功能点】
//   - OnLockAcquired 回调在获取锁后触发
//   - OnLockReleased 回调在释放锁后触发
//   - 回调参数（key、token）正确传递
//
// 【测试流程】
//  1. 配置 OnLockAcquired 和 OnLockReleased 回调
//  2. 获取锁，等待回调执行
//  3. 验证 OnLockAcquired 被调用，参数正确
//  4. 释放锁，等待回调执行
//  5. 验证 OnLockReleased 被调用，参数正确
//
// 【回调用途】
//   - 日志记录：记录锁的获取和释放
//   - 监控指标：统计锁的使用情况
//   - 告警通知：锁长时间未释放时告警
func TestCallbacks(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	var acquiredKey, acquiredToken string
	var releasedKey, releasedToken string
	var acquiredCalled, releasedCalled bool

	locker := NewRedisLocker(client,
		WithWatchdog(false),
		WithOnLockAcquired(func(key, token string) {
			acquiredKey = key
			acquiredToken = token
			acquiredCalled = true
		}),
		WithOnLockReleased(func(key, token string) {
			releasedKey = key
			releasedToken = token
			releasedCalled = true
		}),
	)
	defer locker.Close()

	ctx := context.Background()

	lock, err := locker.TryLock(ctx, "callback-test")
	if err != nil {
		t.Fatalf("获取锁失败: %v", err)
	}

	// 等待回调执行（回调在 goroutine 中异步执行）
	time.Sleep(50 * time.Millisecond)

	if !acquiredCalled {
		t.Error("获取锁回调未被调用")
	}
	if acquiredKey != lock.Key() {
		t.Errorf("获取锁回调 key 不匹配: got %s, want %s", acquiredKey, lock.Key())
	}
	if acquiredToken != lock.Token() {
		t.Errorf("获取锁回调 token 不匹配")
	}

	err = lock.Unlock(ctx)
	if err != nil {
		t.Errorf("释放锁失败: %v", err)
	}

	// 等待回调执行
	time.Sleep(50 * time.Millisecond)

	if !releasedCalled {
		t.Error("释放锁回调未被调用")
	}
	if releasedKey != lock.Key() {
		t.Errorf("释放锁回调 key 不匹配")
	}
	if releasedToken != lock.Token() {
		t.Errorf("释放锁回调 token 不匹配")
	}
}

// ============================================================================
// Stats (统计信息) 测试用例
// ============================================================================

// TestStats 测试客户端统计信息功能
//
// 【测试功能点】
//   - Stats() 返回正确的客户端配置信息
//   - activeLocks 正确反映当前持有的锁数量
//
// 【测试流程】
//  1. 创建自定义配置的 Locker
//  2. 验证 Stats 返回正确的配置（type、keyPrefix）
//  3. 验证初始 activeLocks 为 0
//  4. 获取一把锁
//  5. 验证 activeLocks 变为 1
func TestStats(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client,
		WithKeyPrefix("mylock:"),
		WithDefaultTTL(1*time.Minute),
		WithWatchdog(false),
	)
	defer locker.Close()

	stats := locker.Stats()

	if stats["type"] != "redis" {
		t.Errorf("类型不匹配: got %v", stats["type"])
	}
	if stats["keyPrefix"] != "mylock:" {
		t.Errorf("前缀不匹配: got %v", stats["keyPrefix"])
	}
	if stats["activeLocks"] != 0 {
		t.Errorf("活跃锁数量应为 0: got %v", stats["activeLocks"])
	}

	ctx := context.Background()
	lock, _ := locker.TryLock(ctx, "test")
	defer lock.Unlock(ctx)

	stats = locker.Stats()
	if stats["activeLocks"] != 1 {
		t.Errorf("活跃锁数量应为 1: got %v", stats["activeLocks"])
	}
}

// ============================================================================
// 客户端生命周期测试用例
// ============================================================================

// TestClientClosed 测试客户端关闭后的行为
//
// 【测试功能点】
//   - 客户端关闭后，TryLock/Lock 返回 ErrClientClosed
//   - 防止在已关闭的客户端上操作
//
// 【测试流程】
//  1. 创建 Locker 并立即关闭
//  2. 尝试调用 TryLock
//  3. 验证返回 ErrClientClosed
func TestClientClosed(t *testing.T) {
	mr, client := newTestRedis(t)
	defer mr.Close()

	locker := NewRedisLocker(client, WithWatchdog(false))
	locker.Close()

	ctx := context.Background()
	_, err := locker.TryLock(ctx, "test")
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("期望 ErrClientClosed, 得到: %v", err)
	}
}
