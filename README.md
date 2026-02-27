# gin_core

基于 [Gin](https://github.com/gin-gonic/gin) 封装的 Go Web 框架核心库，提供开箱即用的企业级功能，用于快速搭建高性能 Web 项目。

## 特性

| 分类 | 功能 | 说明 |
|------|------|------|
| **服务管理** | 生命周期管理 | 依赖注入、并行初始化、优雅关闭 |
| **数据库** | MySQL | 连接池、读写分离、多数据库、自动迁移 |
| **缓存** | Redis | 连接池、多实例、集群模式 |
| **消息队列** | RabbitMQ | 生产者 / 消费者、死信队列、发布确认、批量发送 |
| **搜索引擎** | Elasticsearch | Typed Client 集成 |
| **配置中心** | Etcd | 服务发现、分布式配置 |
| **日志** | Logrus | 结构化日志、按级别分文件、自动切割、敏感信息脱敏 |
| **监控** | Prometheus | HTTP 指标采集、连接池指标、自定义 Collector |
| **链路追踪** | OpenTelemetry | DB / Redis / HTTP 自动埋点、W3C Trace Context |
| **限流** | 令牌桶 | 内存 / Redis 存储、按 IP / 用户 / 全局、路径规则匹配 |
| **熔断** | 熔断器 | 自动熔断与恢复、半开探测 |
| **安全** | 加密配置 | AES 加密敏感配置、环境变量注入 |
| **中间件** | 八大内置中间件 | 异常处理、请求日志、超时控制、CORS 跨域等 |
| **工具** | 常用工具包 | HTTP 客户端、邮件发送、AES / RSA 加解密、分布式锁 |

## 安装

```bash
go get -u github.com/zzsen/gin_core
```

## 快速开始

### 1. 创建项目

```bash
mkdir my-project && cd my-project
go mod init my-project
go get -u github.com/zzsen/gin_core
```

### 2. 编写入口文件

```go
package main

import (
    "fmt"

    "github.com/gin-gonic/gin"
    "github.com/zzsen/gin_core/core"
    "github.com/zzsen/gin_core/model/config"
    "github.com/zzsen/gin_core/model/response"
)

// 自定义配置（继承 BaseConfig）
type CustomConfig struct {
    config.BaseConfig `yaml:",inline"`
    Secret            string `yaml:"secret"`
}

func main() {
    // 1. 初始化自定义配置
    core.InitCustomConfig(&CustomConfig{})

    // 2. 注册路由
    core.AddOptionFunc(func(e *gin.Engine) {
        e.GET("/hello", func(c *gin.Context) {
            response.OkWithData(c, "Hello, World!")
        })
    })

    // 3. 启动服务（可传入退出时执行的回调函数）
    core.Start(func() {
        fmt.Println("server stopped")
    })
}
```

### 3. 添加配置文件

在项目根目录创建 `conf/config.default.yml`：

```yaml
system:
  useMysql: false
  useRedis: false
  useEs: false
  useRabbitMQ: false
  useSchedule: false

service:
  ip: "0.0.0.0"
  port: 8080
  middlewares:
    - "exceptionHandler"
    - "traceIdHandler"
    - "traceLogHandler"
```

### 4. 运行

```bash
go run main.go
```

访问 `http://localhost:8080/hello` 即可看到响应。

## 启动流程

```
main()
  → core.InitCustomConfig()          # 设置自定义配置
  → core.AddOptionFunc()             # 注册路由
  → core.AddMessageQueueConsumer()   # 注册 MQ 消费者（可选）
  → core.AddSchedule()               # 注册定时任务（可选）
  → core.Start()
       → overrideValidator()         # 自定义验证器
       → loadConfig()                # 加载配置文件
       → initMiddleware()            # 注册中间件
       → initService()               # 初始化服务（DB、Redis 等）
       → initEngine()                # 创建 Gin 引擎、注册路由
       → server.ListenAndServe()     # 启动 HTTP 服务

优雅关闭：SIGINT/SIGTERM → 执行退出回调 → 关闭服务连接 → server.Shutdown()
```

## 核心 API 速查

| API | 说明 |
|-----|------|
| `core.InitCustomConfig(&cfg)` | 设置自定义配置结构体 |
| `core.AddOptionFunc(fn)` | 注册路由配置函数 |
| `core.AddMessageQueueConsumer(mq)` | 注册 MQ 消费者 |
| `core.AddMessageQueueProducer(mq)` | 注册 MQ 生产者 |
| `core.AddSchedule(schedule)` | 注册定时任务 |
| `core.RegisterMiddleware(name, fn)` | 注册自定义中间件 |
| `core.RegisterService(svc)` | 注册自定义服务 |
| `core.Start(exitFns...)` | 启动服务器 |

| 全局变量 (app 包) | 说明 |
|-----|------|
| `app.DB` | 默认 MySQL 连接 |
| `app.DBResolver` | 读写分离 MySQL 连接 |
| `app.GetDbByName(name)` | 按别名获取数据库连接 |
| `app.Redis` | 默认 Redis 连接 |
| `app.GetRedisByName(name)` | 按别名获取 Redis 连接 |
| `app.ES` | Elasticsearch 客户端 |
| `app.Etcd` | Etcd 客户端 |
| `app.SendRabbitMqMsg(...)` | 发送 MQ 消息 |
| `app.SendRabbitMqMsgWithConfirm(...)` | 发送 MQ 消息（带确认） |
| `app.SendRabbitMqMsgBatch(...)` | 批量发送 MQ 消息 |
| `app.BaseConfig` | 框架基础配置 |

## 内置中间件

通过 `service.middlewares` 配置启用，顺序即调用顺序：

| 名称 | 说明 |
|------|------|
| `prometheusHandler` | Prometheus 指标采集（请求计数、耗时分布、并发数） |
| `exceptionHandler` | 统一异常处理，捕获 panic 并返回标准错误响应 |
| `otelTraceHandler` | OpenTelemetry 链路追踪（W3C Trace Context） |
| `traceIdHandler` | 请求追踪 ID（优先从上游请求头读取，未传递时生成 UUID） |
| `traceLogHandler` | 请求日志（记录请求 / 响应详情） |
| `timeoutHandler` | 请求超时控制（基于 `service.apiTimeout` 配置） |
| `rateLimitHandler` | API 限流（内存 / Redis，支持多维度限流） |
| `corsHandler` | CORS 跨域处理 |

## 内置健康检查

框架自动注册以下端点，无需额外配置：

| 端点 | 说明 |
|------|------|
| `GET /healthy` | 存活检查（Liveness） |
| `GET /healthy/ready` | 就绪检查（Readiness），检查所有依赖服务 |
| `GET /healthy/stats` | 连接池统计信息 |
| `GET /metrics` | Prometheus 指标端点（需启用 `metrics.enabled`） |

## 文档

### 基础文档

| 文档 | 说明 |
|------|------|
| [目录结构](./doc/structure.md) | 项目目录结构说明 |
| [运行参数](./doc/args.md) | 命令行参数说明（`--env`、`--config`、`--cipherKey`） |
| [运行环境](./doc/env.md) | 环境变量配置 |
| [配置](./doc/config.md) | 配置文件说明（多环境、加密、环境变量替换） |

### 核心功能

| 文档 | 说明 |
|------|------|
| [日志](./doc/logger.md) | 日志系统配置和使用 |
| [中间件](./doc/middleware.md) | 内置中间件和自定义中间件 |
| [路由](./doc/router.md) | 路由配置和分组 |
| [控制器](./doc/controller.md) | 控制器编写规范 |
| [服务](./doc/service.md) | 业务逻辑层编写 |
| [服务注册](./doc/service_register.md) | 服务注册和依赖管理 |
| [定时任务](./doc/schedule.md) | 定时任务配置 |

### 高级功能

| 文档 | 说明 |
|------|------|
| [指标监控](./doc/metrics.md) | Prometheus 指标采集 |
| [链路追踪](./doc/tracing.md) | OpenTelemetry 分布式追踪 |
| [限流](./doc/ratelimit.md) | API 限流配置和使用 |
| [熔断器](./doc/circuitbreaker.md) | 服务熔断保护 |
| [死信队列](./doc/dead_letter_queue.md) | RabbitMQ 死信队列 |
| [分布式锁](./doc/distlock.md) | Redis 分布式锁 |

## 许可证

MIT License
