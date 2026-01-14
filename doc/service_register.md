# 服务注册指南

本文档介绍如何使用框架的服务注册机制，以新增自定义 Redis 服务为例。

## 服务接口定义

所有服务需要实现 `core.Service` 接口：

```go
type Service interface {
    // Name 返回服务名称（唯一标识）
    Name() string

    // Priority 返回初始化优先级（数值越小越先初始化）
    Priority() int

    // Dependencies 返回依赖的服务名称列表
    Dependencies() []string

    // ShouldInit 根据配置判断是否需要初始化
    ShouldInit(cfg *config.BaseConfig) bool

    // Init 执行初始化逻辑
    Init(ctx context.Context) error

    // Close 执行清理逻辑
    Close(ctx context.Context) error
}
```

可选实现 `core.HealthChecker` 接口：

```go
type HealthChecker interface {
    HealthCheck(ctx context.Context) error
}
```

## 示例：自定义 Redis 服务

### 1. 定义服务结构体

```go
package service

import (
    "context"
    "fmt"

    "github.com/zzsen/gin_core/app"
    "github.com/zzsen/gin_core/model/config"
)

// MyRedisService 自定义Redis服务
type MyRedisService struct {
    // 可添加自定义字段
    prefix string
}

// NewMyRedisService 创建自定义Redis服务
func NewMyRedisService(prefix string) *MyRedisService {
    return &MyRedisService{prefix: prefix}
}
```

### 2. 实现 Service 接口

```go
// Name 返回服务名称（唯一标识）
func (s *MyRedisService) Name() string {
    return "my-redis"
}

// Priority 返回初始化优先级
// 数值越小越先初始化，同一层级内按此排序
func (s *MyRedisService) Priority() int {
    return 15 // 在 redis(10) 之后
}

// Dependencies 返回依赖的服务名称
// 依赖的服务会在当前服务之前完成初始化
func (s *MyRedisService) Dependencies() []string {
    return []string{"logger", "redis"} // 依赖日志和主Redis
}

// ShouldInit 根据配置判断是否需要初始化
func (s *MyRedisService) ShouldInit(cfg *config.BaseConfig) bool {
    // 可根据配置决定是否初始化
    return cfg.System.UseRedis
}

// Init 执行初始化逻辑
func (s *MyRedisService) Init(ctx context.Context) error {
    // 检查依赖
    if app.Redis == nil {
        return fmt.Errorf("主Redis未初始化")
    }

    // 执行初始化逻辑
    // 例如：预热缓存、创建连接池等
    fmt.Printf("[%s] 服务初始化完成，前缀: %s\n", s.Name(), s.prefix)
    return nil
}

// Close 执行清理逻辑
func (s *MyRedisService) Close(ctx context.Context) error {
    // 执行清理逻辑
    fmt.Printf("[%s] 服务已关闭\n", s.Name())
    return nil
}
```

### 3. 实现健康检查（可选）

```go
// HealthCheck 健康检查
func (s *MyRedisService) HealthCheck(ctx context.Context) error {
    if app.Redis == nil {
        return fmt.Errorf("redis未初始化")
    }
    return app.Redis.Ping(ctx).Err()
}
```

### 4. 注册服务

在 `main.go` 或初始化代码中注册：

```go
package main

import (
    "github.com/zzsen/gin_core/core"
    "your-project/service"
)

func main() {
    // 注册自定义服务（在 core.Run 之前）
    core.RegisterService(service.NewMyRedisService("cache:"))

    // 启动服务
    core.Run(
        core.WithRouters(router.Init),
        // ...
    )
}
```

## 使用初始化钩子

可以在服务初始化前后执行自定义逻辑：

```go
// 在 Redis 初始化后执行数据预热
core.RegisterServiceHook("redis", core.Hook{
    Phase:    core.AfterInit,
    Priority: 0, // 优先级，数值越小越先执行
    Fn: func(ctx context.Context, serviceName string) error {
        // 预热缓存数据
        fmt.Println("Redis初始化后执行数据预热")
        return nil
    },
})
```

### 钩子阶段

| 阶段 | 说明 |
|------|------|
| `core.BeforeInit` | 服务初始化之前 |
| `core.AfterInit` | 服务初始化之后 |
| `core.BeforeClose` | 服务关闭之前 |
| `core.AfterClose` | 服务关闭之后 |

## 查询服务状态

```go
state := core.GetServiceState("redis")
fmt.Println(state) // 输出: ready
```

### 服务状态

| 状态 | 说明 |
|------|------|
| `StateUninitialized` | 未初始化 |
| `StateInitializing` | 初始化中 |
| `StateReady` | 就绪 |
| `StateFailed` | 失败 |
| `StateClosed` | 已关闭 |

## 内置服务列表

| 服务名称 | 优先级 | 依赖 | 说明 |
|----------|--------|------|------|
| `logger` | 0 | 无 | 日志服务 |
| `redis` | 10 | logger | Redis缓存 |
| `mysql` | 10 | logger | MySQL数据库 |
| `elasticsearch` | 20 | logger | Elasticsearch搜索 |
| `etcd` | 20 | logger | Etcd配置中心 |
| `rabbitmq` | 30 | logger | RabbitMQ消息队列 |
| `schedule` | 100 | logger | 定时任务 |

## 初始化流程

```
┌─────────────────────────────────────────────────────────────┐
│                    服务初始化流程                            │
├─────────────────────────────────────────────────────────────┤
│  1. 收集所有已注册的服务                                      │
│  2. 根据 ShouldInit() 过滤需要初始化的服务                    │
│  3. 使用 Kahn 算法解析依赖关系，生成初始化层级                 │
│  4. 按层级顺序初始化，同层服务并行执行                         │
│  5. 每个服务初始化前后执行对应的钩子                           │
└─────────────────────────────────────────────────────────────┘
```

## 调用链

[`core.Run()`](../core/server.go) 
→ [`initService()`](../core/service.go) 
→ [`registerBuiltinServices()`](../core/service.go) 
→ [`lifecycle.InitAllServices()`](../core/lifecycle/initializer.go) 
→ [`ParallelInitializer.Init()`](../core/lifecycle/initializer.go) 
→ 各服务的 `Init()` 方法
