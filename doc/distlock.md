# 分布式锁

分布式锁用于在分布式环境下保证资源的互斥访问，防止多个服务实例同时操作同一资源导致的数据不一致问题。

## 功能特性

- **基于 Redis 实现**：使用 SETNX + Lua 脚本保证原子性
- **看门狗机制**：自动续期防止业务执行时间过长导致锁过期
- **阻塞与非阻塞**：支持 `TryLock`（非阻塞）和 `Lock`（阻塞等待）
- **可重入支持**：同一客户端可重复获取同一把锁
- **安全释放**：只有持有锁的客户端才能释放锁
- **回调通知**：支持获取锁、释放锁、续期失败回调

## 快速开始

### 基本使用

```go
import (
    "context"
    "time"
    
    "gin_core/app"
    "gin_core/distlock"
)

func main() {
    // 创建分布式锁客户端
    locker := distlock.NewRedisLocker(app.Redis)
    defer locker.Close()
    
    ctx := context.Background()
    
    // 获取锁
    lock, err := locker.Lock(ctx, "my-resource")
    if err != nil {
        // 处理错误
        return
    }
    defer lock.Unlock(ctx) // 确保释放锁
    
    // 执行业务逻辑
    doSomething()
}
```

### 使用 TryLock（非阻塞）

```go
lock, err := locker.TryLock(ctx, "my-resource")
if err != nil {
    if errors.Is(err, distlock.ErrLockAlreadyHeld) {
        // 锁被其他客户端持有，执行降级逻辑
        return fallback()
    }
    return err
}
defer lock.Unlock(ctx)

// 执行业务逻辑
```

### 使用超时控制

```go
// 设置 5 秒超时
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

lock, err := locker.Lock(ctx, "my-resource")
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // 获取锁超时
        return errors.New("获取锁超时")
    }
    return err
}
defer lock.Unlock(context.Background()) // 使用新的 context 释放
```

## API 文档

### 创建客户端

#### [NewRedisLocker](#NewRedisLocker)

创建 Redis 分布式锁客户端。

```go
func NewRedisLocker(client redis.UniversalClient, opts ...Option) *RedisLocker
```

**参数：**
- `client`: Redis 客户端实例
- `opts`: 可选配置项

**配置选项：**

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `WithKeyPrefix(prefix)` | 锁键前缀 | `"distlock:"` |
| `WithDefaultTTL(ttl)` | 锁的默认过期时间 | `30s` |
| `WithWatchdog(enabled)` | 是否启用看门狗 | `true` |
| `WithWatchdogInterval(interval)` | 看门狗续期间隔 | `TTL/3` |
| `WithRetryCount(count)` | 默认重试次数 | `30` |
| `WithRetryDelay(delay)` | 默认重试间隔 | `100ms` |
| `WithOnLockAcquired(fn)` | 获取锁成功回调 | `nil` |
| `WithOnLockReleased(fn)` | 释放锁回调 | `nil` |
| `WithOnWatchdogError(fn)` | 看门狗错误回调 | `nil` |

**示例：**

```go
locker := distlock.NewRedisLocker(app.Redis,
    distlock.WithKeyPrefix("myapp:lock:"),
    distlock.WithDefaultTTL(1*time.Minute),
    distlock.WithWatchdog(true),
    distlock.WithRetryCount(50),
    distlock.WithRetryDelay(200*time.Millisecond),
)
```

### 获取锁

#### [TryLock](#TryLock)

尝试获取锁（非阻塞）。如果锁被其他客户端持有，立即返回错误。

```go
func (r *RedisLocker) TryLock(ctx context.Context, key string) (Lock, error)
```

**调用链：**
[TryLock](#TryLock) → [acquire](#acquire)（执行 Lua 脚本）→ [createLock](#createLock) → [startWatchdog](#startWatchdog)

**参数：**
- `ctx`: 上下文
- `key`: 锁的键名（会自动添加前缀）

**返回：**
- `Lock`: 锁对象
- `error`: 
  - `ErrLockAlreadyHeld`: 锁被其他客户端持有
  - `ErrClientClosed`: 客户端已关闭

---

#### [Lock](#Lock)

获取锁（阻塞等待）。如果锁被其他客户端持有，会重试直到获取成功或超时。

```go
func (r *RedisLocker) Lock(ctx context.Context, key string) (Lock, error)
```

**调用链：**
[Lock](#Lock) → [LockWithRetry](#LockWithRetry) → [acquire](#acquire)（循环重试）→ [createLock](#createLock) → [startWatchdog](#startWatchdog)

**参数：**
- `ctx`: 上下文，可通过 `context.WithTimeout` 设置超时
- `key`: 锁的键名

**返回：**
- `Lock`: 锁对象
- `error`:
  - `ErrLockTimeout`: 获取锁超时
  - `context.DeadlineExceeded`: context 超时
  - `context.Canceled`: context 被取消

---

#### [LockWithRetry](#LockWithRetry)

获取锁（带自定义重试配置）。

```go
func (r *RedisLocker) LockWithRetry(ctx context.Context, key string, retryCount int, retryDelay time.Duration) (Lock, error)
```

**参数：**
- `ctx`: 上下文
- `key`: 锁的键名
- `retryCount`: 重试次数
- `retryDelay`: 重试间隔

### 锁操作

#### [Unlock](#Unlock)

释放锁。只有持有锁的客户端才能释放。

```go
func (l *redisLock) Unlock(ctx context.Context) error
```

**调用链：**
[Unlock](#Unlock) → [stopWatchdog](#stopWatchdog) → 执行 `luaReleaseLock` → [removeLock](#removeLock) → 触发 `OnLockReleased` 回调

**返回：**
- `ErrLockNotHeld`: 未持有锁（已过期或被其他客户端持有）

---

#### [Extend](#Extend)

延长锁的过期时间。通常由看门狗自动调用，也可手动续期。

```go
func (l *redisLock) Extend(ctx context.Context, ttl time.Duration) error
```

---

#### [TTL](#TTL)

获取锁的剩余生存时间。

```go
func (l *redisLock) TTL(ctx context.Context) (time.Duration, error)
```

---

#### [Key](#Key) / [Token](#Token)

获取锁的键名和唯一标识。

```go
func (l *redisLock) Key() string
func (l *redisLock) Token() string
```

### 客户端管理

#### [Close](#Close)

关闭客户端。会停止所有看门狗协程，但不会自动释放锁。

```go
func (r *RedisLocker) Close() error
```

---

#### [Stats](#Stats)

获取客户端统计信息。

```go
func (r *RedisLocker) Stats() map[string]interface{}
```

**返回示例：**

```go
{
    "type":             "redis",
    "keyPrefix":        "distlock:",
    "defaultTTL":       "30s",
    "watchdogEnabled":  true,
    "watchdogInterval": "10s",
    "activeLocks":      2,
    "closed":           false,
}
```

## 看门狗机制

看门狗（Watchdog）用于自动续期锁，防止业务执行时间过长导致锁提前过期。

### 工作原理

```
┌─────────────────────────────────────────────────────────────────────┐
│                           时间线                                      │
├─────────────────────────────────────────────────────────────────────┤
│  0s        10s       20s       30s       40s       50s              │
│  │         │         │         │         │         │                │
│  ▼         ▼         ▼         ▼         ▼         ▼                │
│  获取锁    续期      续期      续期      续期      释放锁            │
│  TTL=30s   TTL=30s   TTL=30s   TTL=30s   TTL=30s                    │
│  ├─────────┴─────────┴─────────┴─────────┴─────────┤                │
│  │                  业务处理中                        │                │
│  └──────────────────────────────────────────────────┘                │
└─────────────────────────────────────────────────────────────────────┘
```

### 续期间隔

默认情况下，看门狗每 `TTL/3` 时间续期一次：

- **默认 TTL**：30 秒
- **续期间隔**：10 秒
- **续期时机**：在锁剩余 20 秒时续期，确保有足够的容错时间

### 禁用看门狗

某些场景下可能需要禁用看门狗：

```go
locker := distlock.NewRedisLocker(app.Redis,
    distlock.WithWatchdog(false),
    distlock.WithDefaultTTL(5*time.Minute), // 设置较长的 TTL
)
```

### 续期失败处理

```go
locker := distlock.NewRedisLocker(app.Redis,
    distlock.WithOnWatchdogError(func(key, token string, err error) {
        // 记录日志或告警
        logger.ErrorWithFields("分布式锁续期失败", logrus.Fields{
            "key":   key,
            "token": token,
            "error": err.Error(),
        })
        // 可能需要执行回滚逻辑
    }),
)
```

## 错误处理

### 预定义错误

| 错误 | 说明 |
|------|------|
| `ErrLockNotHeld` | 未持有锁 |
| `ErrLockAcquireFailed` | 获取锁失败 |
| `ErrLockTimeout` | 获取锁超时 |
| `ErrLockAlreadyHeld` | 锁已被其他客户端持有 |
| `ErrClientClosed` | 客户端已关闭 |

### 错误处理示例

```go
lock, err := locker.TryLock(ctx, "my-resource")
if err != nil {
    switch {
    case errors.Is(err, distlock.ErrLockAlreadyHeld):
        // 锁被占用，执行降级逻辑
        return fallback()
    case errors.Is(err, distlock.ErrClientClosed):
        // 客户端已关闭，需要重新初始化
        return errors.New("分布式锁客户端已关闭")
    default:
        // 其他错误（如 Redis 连接失败）
        return fmt.Errorf("获取锁失败: %w", err)
    }
}
```

## 使用场景

### 1. 防止重复提交

```go
func SubmitOrder(ctx context.Context, userID, orderID string) error {
    lockKey := fmt.Sprintf("order:submit:%s:%s", userID, orderID)
    
    lock, err := locker.TryLock(ctx, lockKey)
    if err != nil {
        if errors.Is(err, distlock.ErrLockAlreadyHeld) {
            return errors.New("订单正在处理中，请勿重复提交")
        }
        return err
    }
    defer lock.Unlock(ctx)
    
    // 处理订单
    return processOrder(orderID)
}
```

### 2. 定时任务防重

```go
func ScheduledTask(ctx context.Context) error {
    lock, err := locker.TryLock(ctx, "scheduled:daily-report")
    if err != nil {
        if errors.Is(err, distlock.ErrLockAlreadyHeld) {
            // 其他实例已在执行，跳过
            return nil
        }
        return err
    }
    defer lock.Unlock(ctx)
    
    // 执行定时任务
    return generateDailyReport()
}
```

### 3. 库存扣减

```go
func DeductStock(ctx context.Context, productID string, quantity int) error {
    lockKey := fmt.Sprintf("stock:%s", productID)
    
    // 使用超时控制
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    lock, err := locker.Lock(ctx, lockKey)
    if err != nil {
        return fmt.Errorf("获取库存锁失败: %w", err)
    }
    defer lock.Unlock(context.Background())
    
    // 检查库存
    stock, err := getStock(productID)
    if err != nil {
        return err
    }
    if stock < quantity {
        return errors.New("库存不足")
    }
    
    // 扣减库存
    return updateStock(productID, stock-quantity)
}
```

### 4. 分布式任务调度

```go
func ProcessTask(ctx context.Context, taskID string) error {
    lockKey := fmt.Sprintf("task:%s", taskID)
    
    // 长时间任务使用看门狗自动续期
    lock, err := locker.Lock(ctx, lockKey)
    if err != nil {
        return err
    }
    defer lock.Unlock(context.Background())
    
    // 执行长时间任务（看门狗会自动续期）
    return executeLongRunningTask(taskID)
}
```

## 实现原理

### Lua 脚本

#### 获取锁

```lua
if redis.call("GET", KEYS[1]) == ARGV[1] then
    -- 已持有锁，刷新过期时间（可重入）
    redis.call("PEXPIRE", KEYS[1], ARGV[2])
    return 1
elseif redis.call("SET", KEYS[1], ARGV[1], "NX", "PX", ARGV[2]) then
    return 1
else
    return 0
end
```

#### 释放锁

```lua
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
else
    return 0
end
```

#### 续期锁

```lua
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("PEXPIRE", KEYS[1], ARGV[2])
else
    return 0
end
```

### 安全性保证

1. **原子性**：使用 Lua 脚本保证获取/释放操作的原子性
2. **唯一标识**：每个锁使用随机生成的 token，防止误释放
3. **过期保护**：设置 TTL 防止死锁
4. **看门狗续期**：防止业务执行时间过长导致锁提前释放

## 最佳实践

1. **始终使用 defer 释放锁**
   ```go
   lock, err := locker.Lock(ctx, key)
   if err != nil {
       return err
   }
   defer lock.Unlock(context.Background()) // 使用 Background 确保能释放
   ```

2. **合理设置 TTL**
   - 太短：业务未完成锁就过期
   - 太长：异常情况下锁长时间不释放
   - 建议：预估业务执行时间的 2-3 倍，配合看门狗使用

3. **使用有意义的锁键名**
   ```go
   // Good
   "order:create:user123"
   "stock:deduct:product456"
   
   // Bad
   "lock1"
   "my-lock"
   ```

4. **超时控制**
   ```go
   ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
   defer cancel()
   lock, err := locker.Lock(ctx, key)
   ```

5. **监控和告警**
   ```go
   locker := distlock.NewRedisLocker(app.Redis,
       distlock.WithOnWatchdogError(func(key, token string, err error) {
           metrics.IncrCounter("distlock_watchdog_error_total")
           alerting.Send("分布式锁续期失败", key)
       }),
   )
   ```

## 配置参考

### 生产环境推荐配置

```go
locker := distlock.NewRedisLocker(app.Redis,
    distlock.WithKeyPrefix("prod:lock:"),
    distlock.WithDefaultTTL(30*time.Second),
    distlock.WithWatchdog(true),
    distlock.WithWatchdogInterval(10*time.Second),
    distlock.WithRetryCount(50),
    distlock.WithRetryDelay(100*time.Millisecond),
    distlock.WithOnWatchdogError(func(key, token string, err error) {
        logger.ErrorWithFields("分布式锁续期失败", logrus.Fields{
            "key":   key,
            "error": err.Error(),
        })
    }),
)
```

### 短期锁配置（如防重复提交）

```go
locker := distlock.NewRedisLocker(app.Redis,
    distlock.WithDefaultTTL(5*time.Second),
    distlock.WithWatchdog(false), // 短期锁不需要看门狗
    distlock.WithRetryCount(0),   // 不重试，快速失败
)
```

### 长期任务锁配置

```go
locker := distlock.NewRedisLocker(app.Redis,
    distlock.WithDefaultTTL(5*time.Minute),
    distlock.WithWatchdog(true),
    distlock.WithWatchdogInterval(1*time.Minute),
)
```
