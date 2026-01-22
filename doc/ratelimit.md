# 限流 (Rate Limiting)

## 概述

限流功能用于控制 API 请求频率，防止接口被恶意刷请求或因流量过大导致服务崩溃。本框架提供了灵活的限流配置，支持：

- **多种存储方式**：内存（单机）、Redis（分布式）
- **多种限流键**：IP、用户、全局
- **路径规则匹配**：支持精确匹配和通配符
- **自定义响应消息**：可配置限流提示信息

## 快速开始

### 1. 配置限流

在配置文件中添加限流配置：

```yaml
rateLimit:
  enabled: true
  defaultRate: 100      # 默认每秒请求数
  defaultBurst: 200     # 默认突发容量
  store: memory         # 存储类型: memory / redis
  cleanupInterval: 60   # 清理间隔（秒），仅内存模式有效
  message: "请求过于频繁，请稍后再试"
  rules:
    - path: "/api/login"
      method: "POST"
      rate: 5
      burst: 10
      keyType: "ip"
      message: "登录请求过于频繁"
```

### 2. 注册中间件

框架会自动注册限流中间件，无需手动操作。如需手动注册：

```go
import "github.com/zzsen/gin_core/middleware"

router := gin.Default()
router.Use(middleware.RateLimitHandler())
```

## 配置详解

### RateLimitConfig 配置结构

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `enabled` | bool | false | 是否启用限流 |
| `defaultRate` | int | 100 | 默认每秒允许的请求数 |
| `defaultBurst` | int | 200 | 默认突发容量（令牌桶大小） |
| `store` | string | "memory" | 存储类型：`memory` 或 `redis` |
| `cleanupInterval` | int | 60 | 清理间隔（秒），仅内存模式 |
| `message` | string | "请求过于频繁" | 默认限流提示消息 |
| `rules` | []RateLimitRule | [] | 限流规则列表 |

### RateLimitRule 规则结构

| 字段 | 类型 | 说明 |
|------|------|------|
| `path` | string | 路径匹配模式，支持 `*` 通配符 |
| `method` | string | HTTP 方法，为空表示匹配所有方法 |
| `rate` | int | 每秒允许的请求数 |
| `burst` | int | 突发容量 |
| `keyType` | string | 限流键类型：`ip` / `user` / `global` |
| `message` | string | 该规则的限流提示消息 |

## 限流键类型

### IP 限流 (keyType: "ip")

按客户端 IP 地址限流，每个 IP 有独立的请求配额。

```yaml
rules:
  - path: "/api/*"
    rate: 100
    burst: 200
    keyType: "ip"
```

**IP 获取优先级**：
1. `X-Forwarded-For` 头（代理场景）
2. `X-Real-IP` 头
3. `RemoteAddr`

### 用户限流 (keyType: "user")

按用户 ID 限流，需要在上下文中设置用户信息。如果未设置用户 ID，则回退到 IP 限流。

```yaml
rules:
  - path: "/api/orders"
    rate: 10
    burst: 20
    keyType: "user"
```

在处理器中设置用户 ID：

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := getUserFromToken(c)
        c.Set("userId", userID)
        c.Next()
    }
}
```

### 全局限流 (keyType: "global")

所有请求共享同一个配额，适用于保护后端服务整体负载。

```yaml
rules:
  - path: "/api/expensive-operation"
    rate: 10
    burst: 10
    keyType: "global"
```

## 路径匹配规则

### 精确匹配

```yaml
rules:
  - path: "/api/login"
    method: "POST"
    rate: 5
```

### 通配符匹配

使用 `*` 匹配任意路径：

```yaml
rules:
  - path: "/api/users/*"    # 匹配 /api/users/123, /api/users/profile 等
    rate: 50
  - path: "/api/*"          # 匹配所有 /api/ 开头的路径
    rate: 100
```

### 规则优先级

规则按定义顺序匹配，第一个匹配的规则生效。建议将更具体的规则放在前面：

```yaml
rules:
  - path: "/api/login"      # 精确匹配优先
    rate: 5
  - path: "/api/users/*"    # 特定路径通配符
    rate: 50
  - path: "/api/*"          # 通用通配符兜底
    rate: 100
```

## 存储方式

### 内存存储 (store: "memory")

适用于单机部署场景，使用令牌桶算法。

**优点**：
- 无外部依赖
- 低延迟

**缺点**：
- 仅单机有效
- 重启后数据丢失

```yaml
rateLimit:
  store: memory
  cleanupInterval: 60  # 每 60 秒清理过期条目
```

### Redis 存储 (store: "redis")

适用于分布式部署场景，使用滑动窗口算法。

**优点**：
- 跨实例共享限流状态
- 更精确的滑动窗口算法

**缺点**：
- 需要 Redis 依赖
- 网络延迟

```yaml
rateLimit:
  store: redis
```

> 注意：使用 Redis 存储时，需要确保 Redis 已配置并可连接。

## 响应格式

当请求被限流时，返回 HTTP 429 状态码：

```json
{
  "code": 429,
  "msg": "请求过于频繁，请稍后再试",
  "data": null
}
```

## 调用链

[RateLimitHandler()](../middleware/ratelimit_handler.go) 
→ [findMatchingRule()](../middleware/ratelimit_handler.go) 
→ [generateRateLimitKey()](../middleware/ratelimit_handler.go) 
→ [Limiter.Allow()](../ratelimit/limiter.go)

### 处理流程

1. **检查配置**：如果限流未启用，直接放行
2. **初始化限流器**：根据配置选择内存或 Redis 限流器
3. **匹配规则**：遍历规则列表，找到匹配的规则
4. **生成限流键**：根据 keyType 生成唯一键
5. **检查限流**：调用限流器判断是否允许
6. **响应处理**：允许则继续，拒绝则返回 429

## 示例配置

### 基础配置

```yaml
rateLimit:
  enabled: true
  defaultRate: 100
  defaultBurst: 200
  store: memory
```

### 多规则配置

```yaml
rateLimit:
  enabled: true
  defaultRate: 100
  defaultBurst: 200
  store: memory
  message: "系统繁忙，请稍后再试"
  rules:
    # 登录接口：每 IP 每分钟 5 次
    - path: "/api/login"
      method: "POST"
      rate: 5
      burst: 5
      keyType: "ip"
      message: "登录尝试过于频繁，请 1 分钟后再试"
    
    # 发送验证码：每 IP 每分钟 1 次
    - path: "/api/sms/send"
      method: "POST"
      rate: 1
      burst: 1
      keyType: "ip"
      message: "验证码发送过于频繁"
    
    # 用户订单接口：每用户每秒 10 次
    - path: "/api/orders/*"
      rate: 10
      burst: 20
      keyType: "user"
    
    # 全局高负载接口：整体每秒 100 次
    - path: "/api/report/generate"
      rate: 100
      burst: 100
      keyType: "global"
      message: "报表生成服务繁忙"
```

### 分布式配置

```yaml
rateLimit:
  enabled: true
  defaultRate: 1000
  defaultBurst: 2000
  store: redis
  message: "请求过于频繁"
  rules:
    - path: "/api/*"
      rate: 100
      burst: 200
      keyType: "ip"
```

## 最佳实践

1. **合理设置 burst**：burst 应大于等于 rate，允许短时间的突发流量
2. **优先使用 IP 限流**：对于匿名接口，IP 限流是最常用的方式
3. **敏感接口使用低限制**：登录、验证码等接口应设置较低的限制
4. **分布式环境使用 Redis**：确保多实例间限流状态一致
5. **监控限流指标**：记录被限流的请求，用于调整阈值

## 实现原理

### 令牌桶算法（Token Bucket）

内存限流器使用 `golang.org/x/time/rate` 实现令牌桶算法：

```
┌─────────────────────────────────────────┐
│              令牌桶 (Token Bucket)        │
│                                         │
│   ┌─────┐                               │
│   │ 令牌 │  ← 按 rate 速率持续补充         │
│   │ 生成 │                               │
│   │  器  │                               │
│   └──┬──┘                               │
│      ▼                                  │
│   ┌─────────────────┐                   │
│   │  ○ ○ ○ ○ ○ ○ ○  │ ← 令牌桶           │
│   │   容量 = burst   │   (最多存 burst 个) │
│   └────────┬────────┘                   │
│            ▼                            │
│   ┌─────────────────┐                   │
│   │     请求到达      │                   │
│   │   有令牌？取走    │                   │
│   │   无令牌？拒绝    │                   │
│   └─────────────────┘                   │
└─────────────────────────────────────────┘
```

**工作原理**：
1. 令牌以 `rate` 速率（每秒 rate 个）持续生成
2. 令牌存入桶中，桶容量为 `burst`
3. 每个请求消耗一个令牌
4. 桶满时新令牌被丢弃
5. 桶空时请求被拒绝

**核心代码**：

```go
// 创建令牌桶限流器
limiter := rate.NewLimiter(rate.Limit(ratePerSecond), burst)

// 检查是否允许
allowed := limiter.Allow()
```

### 滑动窗口算法（Sliding Window）

Redis 限流器使用 Lua 脚本实现滑动窗口算法：

```
时间窗口 (1秒)
├────────────────────────────────────────┤
│                                        │
│  ●  ●     ●  ●  ●     ●               │  ← 请求记录
│  ↑        ↑           ↑               │
│  t1       t2          now              │
│                                        │
│  窗口随时间滑动 →→→→→→→→→→            │
│                                        │
├────────────────────────────────────────┤
  now-1s                               now
```

**工作原理**：
1. 使用 Redis 有序集合（ZSET）存储请求时间戳
2. 每次请求时，移除窗口外的过期记录
3. 统计当前窗口内的请求数
4. 如果未超过限制，添加新请求记录

**Lua 脚本核心逻辑**：

```lua
-- 移除窗口外的请求
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- 统计窗口内请求数
local count = redis.call('ZCARD', key)

if count < limit then
    -- 添加新请求
    redis.call('ZADD', key, now, now .. '-' .. math.random())
    return 1  -- 允许
else
    return 0  -- 拒绝
end
```

### 两种算法对比

| 特性 | 令牌桶 | 滑动窗口 |
|------|--------|----------|
| 突发流量 | 支持（burst 控制） | 有限支持 |
| 精确度 | 较低（基于时间间隔） | 较高（精确到毫秒） |
| 内存占用 | 低（只存计数器） | 较高（存每个请求） |
| 分布式 | 需要额外实现 | Redis 原生支持 |
| 适用场景 | 单机、允许突发 | 分布式、精确限流 |

### 限流键生成

限流键决定了限流的维度：

```go
switch keyType {
case "ip":
    // 按 IP 限流：每个 IP 独立计数
    return "ip:" + clientIP + ":" + path
case "user":
    // 按用户限流：每个用户独立计数
    return "user:" + userID + ":" + path
case "global":
    // 全局限流：所有请求共享计数
    return "global:" + path
}
```

## 相关文档

- [熔断器](circuitbreaker.md)
- [中间件](middleware.md)
- [配置说明](config.md)
- [Redis 配置](service.md#redis)
