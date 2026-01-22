# 熔断器 (Circuit Breaker)

## 概述

熔断器用于保护系统免受下游服务故障的影响，防止级联故障。当下游服务异常时，熔断器会自动"断开"，快速失败而不是等待超时，从而保护系统资源。

**核心特性**：
- **自动熔断**：连续失败或失败率达到阈值时自动触发
- **自动恢复**：超时后自动探测服务是否恢复
- **状态回调**：状态变更时触发回调，便于监控告警
- **注册中心**：统一管理多个服务的熔断器

## 快速开始

### 1. 基础使用

```go
import "github.com/zzsen/gin_core/circuitbreaker"

// 使用全局注册中心（推荐）
err := circuitbreaker.Execute(ctx, "user-service", func() error {
    return callUserService()
})

if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
    // 熔断器打开，执行降级逻辑
    return getCachedData()
}
```

### 2. 自定义配置

```go
import "github.com/zzsen/gin_core/circuitbreaker"

// 创建自定义配置
config := circuitbreaker.NewConfig("payment-service",
    circuitbreaker.WithFailureThreshold(3),      // 连续失败 3 次触发熔断
    circuitbreaker.WithTimeout(10*time.Second),  // 10 秒后尝试恢复
    circuitbreaker.WithMaxRequests(2),           // 半开状态最多 2 个探测请求
)

// 创建熔断器
cb := circuitbreaker.New(config)

// 执行受保护的调用
err := cb.Execute(ctx, func() error {
    return callPaymentService()
})
```

### 3. 使用注册中心

```go
// 创建注册中心（带配置工厂）
registry := circuitbreaker.NewRegistry(func(name string) *circuitbreaker.Config {
    return circuitbreaker.NewConfig(name,
        circuitbreaker.WithFailureThreshold(5),
        circuitbreaker.WithTimeout(30*time.Second),
    )
})

// 获取熔断器（自动创建）
cb := registry.Get("order-service")
err := cb.Execute(ctx, func() error {
    return callOrderService()
})

// 查看所有熔断器状态
stats := registry.Stats()
for name, stat := range stats {
    fmt.Printf("%s: state=%s, requests=%d\n", 
        name, stat.State, stat.Counts.Requests)
}
```

## 配置详解

### Config 配置结构

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Name` | string | "default" | 熔断器名称（通常是服务名） |
| `MaxRequests` | uint32 | 3 | 半开状态下允许的最大探测请求数 |
| `Interval` | Duration | 60s | 统计周期，周期结束后重置计数器 |
| `Timeout` | Duration | 30s | 熔断器打开后，等待多久进入半开状态 |
| `FailureThreshold` | uint32 | 5 | 触发熔断的连续失败次数 |
| `FailureRatio` | float64 | 0.5 | 触发熔断的失败率（0.0-1.0） |
| `MinRequests` | uint32 | 10 | 计算失败率的最小请求数 |
| `OnStateChange` | func | nil | 状态变更回调函数 |

### 配置选项函数

```go
// 设置半开状态最大请求数
circuitbreaker.WithMaxRequests(5)

// 设置统计周期
circuitbreaker.WithInterval(30 * time.Second)

// 设置熔断超时时间
circuitbreaker.WithTimeout(15 * time.Second)

// 设置连续失败阈值
circuitbreaker.WithFailureThreshold(3)

// 设置失败率阈值
circuitbreaker.WithFailureRatio(0.6)

// 设置最小请求数
circuitbreaker.WithMinRequests(20)

// 设置状态变更回调
circuitbreaker.WithOnStateChange(func(name string, from, to circuitbreaker.State) {
    log.Printf("熔断器 %s: %s -> %s", name, from, to)
})
```

## 熔断器状态

### 三种状态

| 状态 | 说明 | 行为 |
|------|------|------|
| **Closed** | 关闭（正常） | 所有请求正常通过，统计成功/失败 |
| **Open** | 打开（熔断中） | 所有请求直接失败，返回 `ErrCircuitOpen` |
| **HalfOpen** | 半开（探测中） | 允许部分请求通过，用于探测服务是否恢复 |

### 状态转换

```
                    ┌─────────────────────────────────────────┐
                    │                                         │
                    │  ┌─────────┐                            │
                    │  │ Closed  │ ← 正常运行                  │
                    │  └────┬────┘                            │
                    │       │                                 │
                    │       │ 连续失败 >= FailureThreshold     │
                    │       │ 或 失败率 >= FailureRatio        │
                    │       ▼                                 │
                    │  ┌─────────┐                            │
            ┌───────┼─→│  Open   │ ← 熔断中（快速失败）         │
            │       │  └────┬────┘                            │
            │       │       │                                 │
            │       │       │ 等待 Timeout 时间                │
            │       │       ▼                                 │
            │       │  ┌─────────┐                            │
            │       │  │HalfOpen │ ← 探测中                    │
            │       │  └────┬────┘                            │
            │       │       │                                 │
            │       │       ├─── 探测成功 ───→ 回到 Closed     │
            │       │       │                                 │
            └───────┼───────┴─── 探测失败 ───→ 回到 Open      │
                    │                                         │
                    └─────────────────────────────────────────┘
```

### 状态判断

```go
cb := circuitbreaker.GetBreaker("user-service")

switch cb.State() {
case circuitbreaker.StateClosed:
    fmt.Println("服务正常")
case circuitbreaker.StateOpen:
    fmt.Println("服务熔断中")
case circuitbreaker.StateHalfOpen:
    fmt.Println("服务探测中")
}
```

## 错误处理

### 预定义错误

```go
var (
    // 熔断器处于打开状态
    ErrCircuitOpen = errors.New("circuit breaker is open")

    // 半开状态下请求过多
    ErrTooManyRequests = errors.New("too many requests in half-open state")
)
```

### 错误处理示例

```go
err := circuitbreaker.Execute(ctx, "user-service", func() error {
    return callUserService()
})

switch {
case err == nil:
    // 调用成功
    return result, nil

case errors.Is(err, circuitbreaker.ErrCircuitOpen):
    // 熔断器打开，执行降级
    logger.Warn("user-service 熔断中，使用缓存")
    return getCachedData()

case errors.Is(err, circuitbreaker.ErrTooManyRequests):
    // 半开状态请求过多，稍后重试
    logger.Warn("user-service 探测中，请稍后")
    return nil, errors.New("服务暂时不可用")

default:
    // 业务错误
    return nil, err
}
```

## 统计与监控

### 获取统计信息

```go
cb := circuitbreaker.GetBreaker("user-service")
counts := cb.Counts()

fmt.Printf("总请求: %d\n", counts.Requests)
fmt.Printf("成功: %d\n", counts.TotalSuccesses)
fmt.Printf("失败: %d\n", counts.TotalFailures)
fmt.Printf("连续成功: %d\n", counts.ConsecutiveSuccesses)
fmt.Printf("连续失败: %d\n", counts.ConsecutiveFailures)
```

### 状态变更监控

```go
config := circuitbreaker.NewConfig("user-service",
    circuitbreaker.WithOnStateChange(func(name string, from, to circuitbreaker.State) {
        // 记录日志
        logger.Warnf("熔断器状态变更: %s %s -> %s", name, from, to)
        
        // 发送告警
        if to == circuitbreaker.StateOpen {
            alerting.Send(fmt.Sprintf("服务 %s 触发熔断", name))
        }
        
        // 上报指标
        metrics.SetGauge("circuit_breaker_state", float64(to), 
            "service", name)
    }),
)
```

### 注册中心统计

```go
registry := circuitbreaker.GetRegistry()
stats := registry.Stats()

for name, stat := range stats {
    fmt.Printf("服务: %s\n", name)
    fmt.Printf("  状态: %s\n", stat.State)
    fmt.Printf("  请求: %d\n", stat.Counts.Requests)
    fmt.Printf("  失败: %d\n", stat.Counts.TotalFailures)
}
```

## 手动控制

### 手动重置

```go
// 重置单个熔断器
cb := circuitbreaker.GetBreaker("user-service")
cb.Reset()

// 重置所有熔断器
circuitbreaker.GetRegistry().ResetAll()
```

### 注册自定义熔断器

```go
registry := circuitbreaker.GetRegistry()

// 使用自定义配置注册
config := circuitbreaker.NewConfig("special-service",
    circuitbreaker.WithFailureThreshold(1),  // 一次失败就熔断
    circuitbreaker.WithTimeout(5*time.Second),
)
cb := circuitbreaker.New(config)
registry.Register(cb)
```

## 使用场景

### HTTP 客户端调用（推荐：内置熔断）

框架的 HTTP 客户端已内置熔断器支持，只需启用即可：

```go
import "github.com/zzsen/gin_core/utils/http_client"

// 创建带熔断器的 HTTP 客户端
config := http_client.DefaultClientConfig()
config.EnableCircuitBreaker = true                    // 启用熔断器
config.CircuitBreakerFailureThreshold = 5             // 连续失败 5 次触发熔断
config.CircuitBreakerTimeout = 30 * time.Second       // 30 秒后尝试恢复

client := http_client.NewClient(config)

// 正常使用，熔断器自动按目标 Host 管理
resp, err := client.Do(ctx, req)
if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
    // 熔断器打开，执行降级
    return getCachedData()
}

// 查看熔断器状态
stats := client.GetBreakerStats()
for host, stat := range stats {
    fmt.Printf("%s: state=%s\n", host, stat.State)
}

// 手动重置熔断器
client.ResetBreaker("api.example.com")
```

### HTTP 客户端调用（手动包装）

```go
func CallExternalAPI(ctx context.Context, url string) ([]byte, error) {
    var result []byte
    
    err := circuitbreaker.Execute(ctx, "external-api", func() error {
        resp, err := http.Get(url)
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode >= 500 {
            return fmt.Errorf("server error: %d", resp.StatusCode)
        }
        
        result, err = io.ReadAll(resp.Body)
        return err
    })
    
    if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
        return getCachedResponse(url)
    }
    
    return result, err
}
```

### 数据库操作

```go
func QueryDatabase(ctx context.Context, query string) (*Result, error) {
    var result *Result
    
    err := circuitbreaker.Execute(ctx, "database", func() error {
        var err error
        result, err = db.Query(query)
        return err
    })
    
    return result, err
}
```

### gRPC 调用

```go
func CallGRPCService(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    var resp *pb.Response
    
    err := circuitbreaker.Execute(ctx, "grpc-service", func() error {
        var err error
        resp, err = client.Call(ctx, req)
        return err
    })
    
    return resp, err
}
```

## 实现原理

### 状态机模型

熔断器本质是一个有限状态机，核心数据结构：

```go
type CircuitBreaker struct {
    name          string        // 熔断器名称
    config        *Config       // 配置
    mu            sync.Mutex    // 互斥锁
    state         State         // 当前状态
    counts        Counts        // 请求统计
    expiry        time.Time     // 状态过期时间
    halfOpenCount uint32        // 半开状态请求数
}
```

### 熔断触发条件

熔断器在以下两种情况下触发熔断：

**1. 连续失败次数达到阈值**

```go
if counts.ConsecutiveFailures >= config.FailureThreshold {
    // 触发熔断
    setState(StateOpen)
}
```

**2. 失败率达到阈值**

```go
if counts.Requests >= config.MinRequests {
    ratio := float64(counts.TotalFailures) / float64(counts.Requests)
    if ratio >= config.FailureRatio {
        // 触发熔断
        setState(StateOpen)
    }
}
```

### 请求执行流程

```go
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
    // 1. 检查上下文是否取消
    if ctx.Err() != nil {
        return ctx.Err()
    }
    
    // 2. 请求前检查（是否允许通过）
    if err := cb.beforeRequest(); err != nil {
        return err  // ErrCircuitOpen 或 ErrTooManyRequests
    }
    
    // 3. 执行业务逻辑
    err := fn()
    
    // 4. 记录结果（更新统计，可能触发状态转换）
    cb.afterRequest(err == nil)
    
    return err
}
```

### 状态转换逻辑

```go
// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() error {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    state := cb.currentState(time.Now())
    
    switch state {
    case StateClosed:
        return nil  // 正常通过
    case StateOpen:
        return ErrCircuitOpen  // 直接拒绝
    case StateHalfOpen:
        if cb.halfOpenCount >= cb.config.MaxRequests {
            return ErrTooManyRequests  // 探测请求已满
        }
        cb.halfOpenCount++
        return nil  // 允许探测
    }
    return nil
}

// afterRequest 请求后处理
func (cb *CircuitBreaker) afterRequest(success bool) {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    
    if success {
        // 成功：更新计数，半开状态下可能关闭熔断器
        cb.onSuccess(state)
    } else {
        // 失败：更新计数，可能打开熔断器
        cb.onFailure(state)
    }
}
```

### 时间窗口重置

统计周期结束后自动重置计数器：

```go
func (cb *CircuitBreaker) currentState(now time.Time) State {
    switch cb.state {
    case StateClosed:
        // 检查统计周期是否过期
        if cb.expiry.Before(now) {
            cb.reset(now)  // 重置计数器
        }
    case StateOpen:
        // 检查是否可以进入半开状态
        if cb.expiry.Before(now) {
            cb.setState(StateHalfOpen, now)
        }
    }
    return cb.state
}
```

### 并发安全

使用互斥锁保护所有状态访问和修改：

```go
func (cb *CircuitBreaker) State() State {
    cb.mu.Lock()
    defer cb.mu.Unlock()
    return cb.currentState(time.Now())
}
```

## 调用链

[Execute()](../circuitbreaker/registry.go) 
→ [Registry.Get()](../circuitbreaker/registry.go) 
→ [CircuitBreaker.Execute()](../circuitbreaker/breaker.go) 
→ [beforeRequest()](../circuitbreaker/breaker.go) 
→ 业务函数 
→ [afterRequest()](../circuitbreaker/breaker.go)

## 最佳实践

1. **合理设置阈值**：根据服务特点设置合适的失败阈值和超时时间
2. **区分服务**：为不同服务创建独立的熔断器，避免相互影响
3. **监控告警**：使用 `OnStateChange` 回调实现状态变更告警
4. **降级策略**：熔断时提供合理的降级方案（缓存、默认值等）
5. **定期重置**：在维护窗口期可以手动重置熔断器

## 相关文档

- [限流](ratelimit.md)
- [中间件](middleware.md)
- [配置说明](config.md)
