# HTTP 客户端 (HTTP Client)

## 一、概述

`utils/http_client` 包提供高性能的 HTTP 客户端，内置以下能力：

- **连接池复用**：基于 `http.Transport` 的连接池管理
- **链路追踪**：自动集成 OpenTelemetry 链路追踪
- **请求重试**：网络超时和连接拒绝自动重试
- **熔断保护**：基于 `circuitbreaker` 包的熔断器，按主机名隔离

## 二、快速开始

### 2.1 使用便捷函数

推荐通过顶层便捷函数发送请求（自动使用全局单例客户端）：

```go
import (
    "context"
    "github.com/zzsen/gin_core/utils/http_client"
)

// GET 请求
resp := http_client.GetWithContext(ctx, "https://api.example.com/users", 10, nil)
if resp.HasError() {
    // 处理错误
}
fmt.Println(resp.Body)

// POST JSON
resp = http_client.PostJSONWithContext(ctx, "https://api.example.com/users",
    `{"name":"test"}`, 10, map[string]string{"X-Token": "abc"})

// PUT JSON
resp = http_client.PutJSONWithContext(ctx, "https://api.example.com/users/1",
    `{"name":"updated"}`, 10, nil)

// DELETE
resp = http_client.DeleteWithContext(ctx, "https://api.example.com/users/1", 10, nil)
```

### 2.2 表单提交与文件上传

调用链：[PostFormWithContext](../utils/http_client/http_client.go) → `multipart.Writer` → [doRequest](../utils/http_client/http_client.go) → [Client.Do](../utils/http_client/client.go)

```go
resp := http_client.PostFormWithContext(ctx,
    "https://api.example.com/upload",
    map[string]string{"field1": "value1"},     // 普通字段
    map[string]string{"file": "/path/to.pdf"}, // 文件路径
    nil,                                        // io.Reader 映射
    nil,                                        // 自定义 header
    30,                                         // 超时秒数
)
```

## 三、自定义客户端

### 3.1 配置项

通过 [ClientConfig](../utils/http_client/client.go) 自定义客户端行为：

```go
cfg := &http_client.ClientConfig{
    // 连接池
    MaxIdleConns:        200,
    MaxIdleConnsPerHost: 20,
    MaxConnsPerHost:     200,
    IdleConnTimeout:     90 * time.Second,

    // 超时
    DialTimeout:         10 * time.Second,
    TLSHandshakeTimeout: 5 * time.Second,
    ResponseTimeout:     15 * time.Second,

    // 重试
    MaxRetries:    3,
    RetryInterval: 200 * time.Millisecond,

    // 功能开关
    EnableTracing:        true,
    EnableCircuitBreaker: true,

    // 熔断器配置
    CircuitBreakerFailureThreshold: 5,
    CircuitBreakerTimeout:          30 * time.Second,
    CircuitBreakerMaxRequests:      3,
}

client := http_client.NewClient(cfg)
```

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `MaxIdleConns` | 100 | 最大空闲连接数 |
| `MaxIdleConnsPerHost` | 10 | 每主机最大空闲连接数 |
| `MaxConnsPerHost` | 100 | 每主机最大连接数 |
| `IdleConnTimeout` | 90s | 空闲连接超时 |
| `DialTimeout` | 30s | 连接建立超时 |
| `TLSHandshakeTimeout` | 10s | TLS 握手超时 |
| `ResponseTimeout` | 30s | 响应头超时 |
| `MaxRetries` | 3 | 最大重试次数 |
| `RetryInterval` | 100ms | 重试间隔 |
| `EnableTracing` | true | 是否启用链路追踪 |
| `EnableCircuitBreaker` | false | 是否启用熔断器 |
| `CircuitBreakerFailureThreshold` | 5 | 熔断触发阈值 |
| `CircuitBreakerTimeout` | 30s | 熔断超时 |
| `CircuitBreakerMaxRequests` | 3 | 半开状态最大请求数 |

### 3.2 使用自定义客户端

```go
client := http_client.NewClient(cfg)

req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.example.com/data", nil)
resp, err := client.Do(ctx, req)
```

## 四、响应处理

所有便捷函数返回 [ResponseWrapper](../utils/http_client/http_client.go)：

```go
type ResponseWrapper struct {
    StatusCode int         // HTTP 状态码
    Body       string      // 响应体
    Header     http.Header // 响应头
    Error      error       // 错误信息
}
```

| 方法 | 说明 |
|------|------|
| `IsSuccess()` | 状态码在 200-299 范围 |
| `HasError()` | 存在错误或状态码为 0 |

## 五、重试机制

调用链：[Client.Do](../utils/http_client/client.go) → [doWithRetry](../utils/http_client/client.go) → [isRetryableError](../utils/http_client/client.go)

可重试的错误类型：
- 网络超时错误（`net.Error.Timeout()`）
- 连接拒绝错误（`net.OpError` 且 `Op == "dial"`）

不可重试的错误（如 DNS 解析失败、TLS 证书错误）会立即返回。

## 六、熔断器

调用链：[Client.Do](../utils/http_client/client.go) → [doWithCircuitBreaker](../utils/http_client/client.go) → `circuitbreaker.Execute`

启用 `EnableCircuitBreaker` 后，按请求的 `Host` 自动创建独立熔断器：

- **5xx 响应**视为失败，累计触发熔断
- 熔断打开时直接返回 `circuitbreaker.ErrCircuitOpen`
- 5xx 响应仍会返回 `*http.Response`，让调用方决定后续处理

```go
// 查看熔断器状态
stats := client.GetBreakerStats()

// 重置指定主机的熔断器
client.ResetBreaker("api.example.com:8080")

// 检查熔断器是否打开
isOpen := client.IsCircuitOpen("api.example.com:8080")
```

## 七、全局客户端

调用链：[GetDefaultClient](../utils/http_client/client.go) → `sync.Once` → [NewClient](../utils/http_client/client.go)(nil)

所有便捷函数（`GetWithContext`、`PostJSONWithContext` 等）使用 [GetDefaultClient](../utils/http_client/client.go) 获取全局单例客户端，采用默认配置。

## 八、API 一览

| 函数 | 说明 |
|------|------|
| [GetWithContext](../utils/http_client/http_client.go) | GET 请求 |
| [PostJSONWithContext](../utils/http_client/http_client.go) | POST JSON 请求 |
| [PostParamsWithContext](../utils/http_client/http_client.go) | POST 参数请求 |
| [PutJSONWithContext](../utils/http_client/http_client.go) | PUT JSON 请求 |
| [DeleteWithContext](../utils/http_client/http_client.go) | DELETE 请求 |
| [PostFormWithContext](../utils/http_client/http_client.go) | POST 表单/文件上传 |
| [NewClient](../utils/http_client/client.go) | 创建自定义客户端 |
| [GetDefaultClient](../utils/http_client/client.go) | 获取全局默认客户端 |
