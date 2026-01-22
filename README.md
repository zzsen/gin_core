# gin_core

基于 Gin 封装的 Go Web 框架核心库，提供开箱即用的企业级功能，用于快速搭建高性能 Web 项目。

## 特性

- **服务管理**：依赖注入、服务生命周期管理、优雅关闭
- **数据库**：MySQL 连接池、读写分离、多数据库支持
- **缓存**：Redis 连接池、多实例支持
- **消息队列**：RabbitMQ 生产者/消费者、死信队列、批量发布
- **搜索引擎**：Elasticsearch 集成
- **配置中心**：Etcd 集成
- **日志系统**：结构化日志、自动切割、敏感信息脱敏
- **指标监控**：Prometheus 指标采集
- **链路追踪**：OpenTelemetry 分布式追踪
- **限流**：内存/Redis 限流器、多维度限流
- **熔断**：服务熔断保护、自动恢复
- **中间件**：异常处理、请求日志、超时控制、CORS 跨域

## 安装

### 新项目使用

```bash
# 1. 新建工程目录
mkdir [projectName] && cd [projectName]

# 2. 初始化 go.mod
go mod init [projectName]

# 3. 拉取 gin_core 依赖包
go get -u github.com/zzsen/gin_core
```

### 旧项目使用

```bash
go get -u github.com/zzsen/gin_core
```

## 快速开始

```go
package main

import (
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
    // 加载配置
    customConfig := &CustomConfig{}
    core.LoadConfig(customConfig)

    // 自定义路由
    opts := []gin.OptionFunc{
        func(e *gin.Engine) {
            e.GET("/hello", func(c *gin.Context) {
                response.OkWithData(c, "Hello, World!")
            })
        },
    }

    // 启动服务
    core.Start(opts, func() {
        // 退出前执行的清理操作
    })
}
```

## 文档

### 基础文档

| 文档 | 说明 |
|------|------|
| [目录结构](./doc/structure.md) | 项目目录结构说明 |
| [运行参数](./doc/args.md) | 命令行参数说明 |
| [运行环境](./doc/env.md) | 环境变量配置 |
| [配置](./doc/config.md) | 配置文件说明 |

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

## 许可证

MIT License
