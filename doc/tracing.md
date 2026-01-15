# 分布式链路追踪

本文档介绍 gin_core 框架集成的 OpenTelemetry 分布式链路追踪功能。

## 目录

- [功能概述](#功能概述)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [组件追踪](#组件追踪)
- [API 参考](#api-参考)
- [架构说明](#架构说明)
- [最佳实践](#最佳实践)

## 功能概述

### 支持的追踪能力

| 功能 | 说明 |
|------|------|
| HTTP 请求追踪 | 自动追踪所有入站 HTTP 请求 |
| 数据库追踪 | 追踪 MySQL 查询、插入、更新、删除操作 |
| Redis 追踪 | 追踪 Redis 命令执行 |
| HTTP 客户端追踪 | 追踪出站 HTTP 请求，支持跨服务传播 |
| 上下文传播 | 支持 W3C Trace Context 标准 |

### 支持的后端

- **OTLP**: OpenTelemetry Protocol（推荐）
- **Jaeger**: 通过 OTLP 接收器
- **Zipkin**: 通过 OTLP 接收器
- **Tempo**: Grafana Tempo
- **stdout**: 标准输出（调试用）

## 快速开始

### 1. 启用链路追踪

在配置文件中添加 `tracing` 配置：

```yaml
tracing:
  enabled: true
  serviceName: "my-service"
  exporterType: "otlp"
  endpoint: "localhost:4317"
  sampleRate: 1.0
  insecure: true
```

### 2. 使用 OpenTelemetry 中间件

确保在 `service.middlewares` 中添加 `otelTraceHandler`：

```yaml
service:
  middlewares:
    - "prometheusHandler"
    - "exceptionHandler"
    - "otelTraceHandler"  # OpenTelemetry 链路追踪
    - "traceLogHandler"
    - "timeoutHandler"
```

> **注意**: `otelTraceHandler` 会自动设置 `traceId` 到上下文中，可以替代 `traceIdHandler`。如果同时使用，`otelTraceHandler` 应放在 `traceIdHandler` 之后。

### 3. 启动 Jaeger（本地测试）

```bash
docker run -d --name jaeger \
  -e COLLECTOR_OTLP_ENABLED=true \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest
```

访问 http://localhost:16686 查看追踪数据。

## 配置说明

### 完整配置项

```yaml
tracing:
  # 是否启用链路追踪
  enabled: true
  
  # 服务名称，用于在追踪系统中标识当前服务
  serviceName: "my-service"
  
  # 导出器类型
  # - "otlp": OpenTelemetry Protocol（推荐）
  # - "stdout": 标准输出（仅调试）
  exporterType: "otlp"
  
  # 采集器端点地址
  # - OTLP gRPC: "localhost:4317"
  # - OTLP HTTP: "localhost:4318"
  endpoint: "localhost:4317"
  
  # 采样率（0.0 - 1.0）
  # - 1.0: 采样所有请求（开发环境）
  # - 0.1: 采样 10% 的请求（生产环境推荐）
  sampleRate: 1.0
  
  # 是否禁用 TLS
  insecure: true
  
  # 上下文传播格式
  # - "tracecontext": W3C Trace Context 标准（默认）
  # - "b3": Zipkin B3 格式
  propagatorType: "tracecontext"
  
  # 是否追踪数据库操作
  enableDBTracing: true
  
  # 是否追踪 Redis 操作
  enableRedisTracing: true
  
  # 是否追踪出站 HTTP 请求
  enableHTTPClientTracing: true
```

### 环境差异配置

**开发环境 (config.dev.yml)**:
```yaml
tracing:
  enabled: true
  sampleRate: 1.0  # 采样所有请求
  insecure: true
```

**生产环境 (config.prod.yml)**:
```yaml
tracing:
  enabled: true
  sampleRate: 0.1  # 采样 10% 请求
  insecure: false
  endpoint: "otel-collector.monitoring:4317"
```

## 组件追踪

### HTTP 请求追踪

自动追踪所有入站 HTTP 请求，记录以下信息：

- 请求方法、URL、路由
- 响应状态码
- 客户端 IP
- User-Agent
- 请求耗时

追踪数据示例：
```
Span: GET /api/users/:id
├── http.method: GET
├── http.url: /api/users/123
├── http.route: /api/users/:id
├── http.status_code: 200
├── net.peer.ip: 192.168.1.100
└── http.user_agent: Mozilla/5.0 ...
```

### 数据库追踪

自动追踪 GORM 数据库操作：

```
Span: db.query
├── db.system: mysql
├── db.name: mydb
├── db.table: users
├── db.statement: SELECT * FROM users WHERE id = ?
└── db.rows_affected: 1
```

### Redis 追踪

自动追踪 Redis 命令：

```
Span: redis.get
├── db.system: redis
├── redis.alias: default
├── db.redis.database_index: 0
└── db.statement: GET user:123
```

### HTTP 客户端追踪

使用 [tracing.NewTracingHTTPClient](#newtracingttpclient) 或 [tracing.WrapHTTPClient](#wraphttpclient) 追踪出站请求：

```go
import "github.com/zzsen/gin_core/tracing"

// 方式1：创建新的追踪 HTTP 客户端
client := tracing.NewTracingHTTPClient()

// 方式2：包装现有客户端
existingClient := &http.Client{Timeout: 30 * time.Second}
client := tracing.WrapHTTPClient(existingClient)

// 发送请求（自动传播追踪上下文）
resp, err := client.Do(req.WithContext(ctx))
```

## API 参考

### tracing 包

#### InitTracer

初始化 OpenTelemetry Tracer。

```go
func InitTracer(cfg *config.TracingConfig) (func(context.Context) error, error)
```

> **调用链**: [InitTracer](#inittracer) → createExporter → createResource → createSampler → createPropagator

#### StartSpan

开始一个新的 Span。

```go
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
```

**使用示例**:
```go
ctx, span := tracing.StartSpan(ctx, "my-operation",
    trace.WithSpanKind(trace.SpanKindInternal),
    trace.WithAttributes(
        attribute.String("key", "value"),
    ),
)
defer span.End()

// 执行操作...
```

#### GetTraceID

从 context 获取 TraceID。

```go
func GetTraceID(ctx context.Context) string
```

#### GetSpanID

从 context 获取 SpanID。

```go
func GetSpanID(ctx context.Context) string
```

#### SetSpanError

设置 Span 错误状态。

```go
func SetSpanError(span trace.Span, err error)
```

#### IsEnabled

返回链路追踪是否已启用。

```go
func IsEnabled() bool
```

### 中间件

#### OtelTraceHandler

OpenTelemetry HTTP 追踪中间件。

```go
func OtelTraceHandler() gin.HandlerFunc
```

> **调用链**: OtelTraceHandler → [tracing.StartSpan](#startspan) → [tracing.GetTraceID](#gettraceid)

### GORM 插件

#### NewGormTracingPlugin

创建 GORM 追踪插件。

```go
func NewGormTracingPlugin(dbName ...string) *GormTracingPlugin
```

**使用示例**:
```go
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
db.Use(tracing.NewGormTracingPlugin("mydb"))
```

### Redis Hook

#### NewRedisTracingHook

创建 Redis 追踪钩子。

```go
func NewRedisTracingHook(addr string, aliasName string, db int) *RedisTracingHook
```

**使用示例**:
```go
client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
client.AddHook(tracing.NewRedisTracingHook("localhost:6379", "default", 0))
```

### HTTP Transport

#### NewTracingHTTPClient

创建带追踪功能的 HTTP 客户端。

```go
func NewTracingHTTPClient() *http.Client
```

#### WrapHTTPClient

为现有的 HTTP 客户端添加追踪功能。

```go
func WrapHTTPClient(client *http.Client) *http.Client
```

## 架构说明

### 追踪流程

```
┌─────────────────────────────────────────────────────────────────┐
│                         HTTP Request                             │
│                  (携带 traceparent header)                       │
└─────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    OtelTraceHandler                              │
│              (提取/创建 Trace Context)                           │
└─────────────────────────────────────────────────────────────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          ▼                      ▼                      ▼
   ┌─────────────┐       ┌─────────────┐       ┌─────────────┐
   │   MySQL     │       │   Redis     │       │  HTTP Client│
   │  (gorm-otel)│       │ (redis-otel)│       │  (transport)│
   └─────────────┘       └─────────────┘       └─────────────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    OTLP Exporter                                 │
│         (导出到 Jaeger / Zipkin / Tempo / 等)                    │
└─────────────────────────────────────────────────────────────────┘
```

### Span 层级示例

```
[HTTP] GET /api/users/123          TraceID: abc123... SpanID: def456...
  ├── [MySQL] db.query             SpanID: ghi789...  duration: 5ms
  │     table: users
  │     sql: SELECT * FROM users WHERE id = ?
  ├── [Redis] redis.get            SpanID: jkl012...  duration: 1ms
  │     key: user:123:cache
  └── [HTTP Client] POST /notify   SpanID: mno345...  duration: 50ms
        (传播到下游服务)
```

## 最佳实践

### 1. 采样策略

- **开发环境**: `sampleRate: 1.0`（采样所有请求）
- **测试环境**: `sampleRate: 0.5`（采样 50%）
- **生产环境**: `sampleRate: 0.1`（采样 10%）

### 2. 服务命名

使用有意义的服务名称：

```yaml
# 好的命名
serviceName: "user-service"
serviceName: "order-api"
serviceName: "payment-gateway"

# 避免的命名
serviceName: "app"
serviceName: "service1"
```

### 3. 自定义 Span

在业务逻辑中添加自定义 Span：

```go
func ProcessOrder(ctx context.Context, orderID string) error {
    ctx, span := tracing.StartSpan(ctx, "process-order",
        trace.WithAttributes(
            attribute.String("order.id", orderID),
        ),
    )
    defer span.End()

    // 添加事件
    tracing.AddSpanEvent(span, "order.validated")

    // 处理订单...
    if err := doSomething(); err != nil {
        tracing.SetSpanError(span, err)
        return err
    }

    return nil
}
```

### 4. 敏感信息处理

SQL 语句会自动截断，但仍需注意：

- 避免在 Span 属性中记录密码、令牌等敏感信息
- 使用参数化查询，避免 SQL 中出现敏感数据

### 5. 性能考虑

- 生产环境使用较低的采样率
- 避免在高频操作中创建过多的 Span
- 使用批量导出器减少网络开销

## 故障排查

### 追踪数据未显示

1. 检查配置是否正确：
   ```yaml
   tracing:
     enabled: true  # 确保已启用
   ```

2. 检查采集器是否可达：
   ```bash
   telnet localhost 4317
   ```

3. 使用 stdout 导出器调试：
   ```yaml
   tracing:
     exporterType: "stdout"
   ```

### 数据库追踪不工作

确保在初始化数据库**之前**已初始化链路追踪：

```go
// 正确顺序
initialize.InitTracing()  // 先初始化追踪
initialize.InitDB()       // 再初始化数据库
```

框架默认按正确顺序初始化，通常无需手动处理。

### 跨服务追踪断裂

确保使用追踪 HTTP 客户端：

```go
// 错误：使用默认客户端，追踪上下文不会传播
resp, _ := http.Get("http://other-service/api")

// 正确：使用追踪客户端
client := tracing.NewTracingHTTPClient()
req, _ := http.NewRequestWithContext(ctx, "GET", "http://other-service/api", nil)
resp, _ := client.Do(req)
```
