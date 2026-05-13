# 异常体系 (Exception)

## 一、概述

`exception` 包定义了框架的统一异常处理体系。所有业务异常通过 `panic` 抛出，由框架的 `exceptionHandler` 中间件统一捕获并转换为标准 HTTP 响应。

### 核心设计

- **panic + recover 模式**：业务代码中通过 `panic(exception)` 抛出异常，中间件自动 recover 并返回响应
- **Handler 接口**：实现该接口的异常类型由框架识别并提取响应消息和状态码
- **未实现 Handler 的 panic**：被视为未知异常（HTTP 500）

## 二、Handler 接口

```go
type Handler interface {
    OnException(ctx *gin.Context) (msg string, code int)
}
```

所有自定义异常需实现 [Handler](../exception/index.go) 接口，`msg` 作为响应消息，`code` 作为业务状态码。

## 三、内置异常类型

### 3.1 CommonError - 通用业务异常

调用链：`panic` → [CommonError.OnException](../exception/common_error.go) → `ResponseExceptionCommon.GetCode()`

```go
import "github.com/zzsen/gin_core/exception"

// 抛出通用异常
panic(exception.NewCommonError("用户名已存在"))
```

| 字段 | 说明 |
|------|------|
| `msg` | 错误消息，会直接返回给前端 |

### 3.2 AuthFailed - 认证失败

调用链：`panic` → [AuthFailed.OnException](../exception/auth_failed.go) → `ResponseAuthFailed.GetCode()`

```go
// Token 无效或过期时
panic(exception.AuthFailed{})
```

返回认证失败的默认消息和状态码。

### 3.3 InvalidParam - 参数校验异常

调用链：`panic` → [InvalidParam.OnException](../exception/invalid_param.go) → `ResponseParamInvalid.GetCode()`

```go
// 手动构建
panic(exception.NewInvalidParam("用户名长度必须在 3-20 之间"))

// 从 validator 错误自动转换
err := validator.New().Struct(req)
if err != nil {
    panic(exception.NewInvalidParamFromValidator(err.(validator.ValidationErrors)))
}
```

[NewInvalidParamFromValidator](../exception/invalid_param.go) 会将 validator 的校验错误转为可读的中文消息，支持的校验标签：

| 标签 | 生成消息示例 |
|------|-------------|
| `required` | "Name 不能为空" |
| `min` | "Age 的值不能小于 18" |
| `max` | "Age 的值不能大于 100" |
| `len` | "Code 的长度必须为 6" |
| `email` | "Email 必须是有效的邮箱地址" |
| `url` | "Website 必须是有效的 URL 地址" |
| `numeric` | "Amount 必须是数字" |
| `alpha` | "Name 只能包含字母" |
| `alphanum` | "Code 只能包含字母和数字" |
| `gte` | "Score 的值必须大于或等于 0" |
| `lte` | "Score 的值必须小于或等于 100" |
| `gt` / `lt` | "Value 的值必须大于/小于 X" |
| `oneof` | "Status 的值必须是以下之一: active inactive" |
| 其他 | "Field 校验失败(标签: xxx, 值: yyy)" |

### 3.4 RpcError - RPC 调用异常

调用链：`panic` → [RpcError.OnException](../exception/rpc_error.go) → `logger.Error` → `debug.Stack()` → `ResponseExceptionRpc.GetCode()`

```go
panic(exception.NewRpcError("用户服务调用超时"))
```

会自动记录错误日志和完整调用栈。

### 3.5 InitError - 初始化错误

[InitError](../exception/init_error.go) 用于服务初始化阶段，提供结构化的错误上下文：

```go
panic(exception.NewInitError("db", "初始化连接", err))
panic(exception.NewInitErrorWithConfig("redis", "初始化连接", "cache-01", err))
```

| 字段 | 说明 |
|------|------|
| `Service` | 服务名称（如 db、redis、es） |
| `Operation` | 操作名称（如 初始化连接、创建客户端） |
| `Config` | 可选，配置标识（如别名） |
| `Err` | 底层错误，支持 `errors.Unwrap` |

输出格式：`[db] 初始化连接失败: <原始错误>` 或 `[redis] 初始化连接失败 [cache-01]: <原始错误>`

## 四、自定义异常

```go
type OrderNotFoundError struct {
    OrderID string
}

func (e OrderNotFoundError) Error() string {
    return fmt.Sprintf("订单 %s 不存在", e.OrderID)
}

func (e OrderNotFoundError) OnException(ctx *gin.Context) (string, int) {
    return e.Error(), 40400  // 自定义业务状态码
}

// 使用
panic(OrderNotFoundError{OrderID: "ORD-001"})
```

## 五、注意事项

1. **确保在 handler 链中调用**：`panic` 必须在有 `exceptionHandler` 中间件保护的请求处理链中使用
2. **不要暴露敏感信息**：`CommonError` 的消息会直接返回给前端，注意脱敏
3. **InitError 用于启动阶段**：初始化阶段的 `panic` 会导致服务终止，这是预期行为
