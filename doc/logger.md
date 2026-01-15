# 日志模块

`logger` 模块提供统一的日志记录功能，基于 [logrus](https://github.com/sirupsen/logrus) 实现，支持日志轮转、多级别配置、结构化日志和敏感信息脱敏。

## 目录

- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [基础日志函数](#基础日志函数)
- [结构化日志](#结构化日志)
- [敏感信息脱敏](#敏感信息脱敏)
- [调用者信息](#调用者信息)
- [日志轮转](#日志轮转)
- [最佳实践](#最佳实践)

## 快速开始

### 基础使用

```go
import "github.com/zzsen/gin_core/logger"

// 记录不同级别的日志
logger.Debug("这是一条调试日志")
logger.Info("用户登录成功")
logger.Warn("连接池使用率过高: %d%%", 85)
logger.Error("数据库连接失败: %v", err)
logger.Trace("请求详情: %s", requestBody)
```

### 结构化日志

```go
// 带字段的日志（推荐用于记录请求上下文）
logger.InfoWithFields(map[string]any{
    "userId":    12345,
    "traceId":   "abc-123",
    "action":    "login",
}, "用户操作")
```

## 配置说明

在 `config.yaml` 中配置日志：

```yaml
loggers:
  filePath: "./log/"        # 日志文件存储路径
  maxAge: 30                # 日志保存天数
  rotationTime: 60          # 轮转时间间隔（分钟）
  rotationSize: 1024        # 轮转大小限制（KB）
  printCaller: true         # 是否打印调用者信息
  loggers:                  # 各级别单独配置（可选）
    - level: "error"
      fileName: "error"     # 错误日志单独文件
      maxAge: 90            # 错误日志保留更久
    - level: "info"
      fileName: "info"
```

### 配置参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `filePath` | string | `./log/` | 日志文件存储路径 |
| `maxAge` | int | 30 | 日志文件最大保存时间（天） |
| `rotationTime` | int | 60 | 日志轮转时间间隔（分钟） |
| `rotationSize` | int | 1024 | 日志轮转大小限制（KB） |
| `printCaller` | bool | false | 是否打印调用者信息（文件、行号、函数名） |
| `loggers` | array | - | 各日志级别单独配置 |

### 日志级别

支持以下日志级别（从低到高）：

| 级别 | 说明 | 使用场景 |
|------|------|----------|
| `trace` | 最详细的追踪信息 | 请求详情、调试追踪 |
| `debug` | 调试信息 | 开发调试 |
| `info` | 一般信息 | 业务操作记录 |
| `warn` | 警告信息 | 潜在问题提醒 |
| `error` | 错误信息 | 错误记录 |
| `fatal` | 致命错误 | 程序无法继续运行 |
| `panic` | 恐慌错误 | 程序崩溃 |

## 基础日志函数

### 简单日志

```go
// 不带格式化
logger.Info("服务启动成功")

// 带格式化参数
logger.Info("服务启动成功，端口: %d", 8080)
logger.Error("处理请求失败: %v", err)
logger.Warn("缓存命中率低: %.2f%%", hitRate*100)
logger.Debug("请求参数: %+v", params)
logger.Trace("完整请求体: %s", body)
```

### 带请求ID的日志

```go
// 用于关联同一请求的日志
logger.Add(requestId, "处理订单", nil)           // 成功时记录 Info
logger.Add(requestId, "处理订单失败", err)        // 失败时记录 Error
```

## 结构化日志

使用 `*WithFields` 系列函数记录结构化日志，便于日志分析和检索：

```go
// Info 级别
logger.InfoWithFields(map[string]any{
    "userId":    12345,
    "orderId":   "ORD-001",
    "amount":    99.99,
    "traceId":   traceId,
}, "订单创建成功")

// Error 级别
logger.ErrorWithFields(map[string]any{
    "userId":    12345,
    "error":     err.Error(),
    "stackInfo": string(debug.Stack()),
}, "订单创建失败")

// Warn 级别
logger.WarnWithFields(map[string]any{
    "poolUsage": 85,
    "threshold": 80,
}, "连接池使用率过高")

// Debug 级别
logger.DebugWithFields(map[string]any{
    "sql":      sql,
    "duration": duration,
}, "SQL执行")

// Trace 级别（请求追踪）
logger.TraceWithFields(map[string]any{
    "traceId":      traceId,
    "requestId":    requestId,
    "statusCode":   200,
    "responseTime": "15ms",
    "clientIp":     clientIP,
    "reqMethod":    "POST",
    "reqUri":       "/api/orders",
}, "请求日志")
```

## 敏感信息脱敏

日志模块内置敏感信息自动脱敏功能，**无需手动处理**。

### 自动脱敏的敏感字段

以下关键词（不区分大小写）会被自动检测并脱敏：

- `password`, `pwd`, `passwd`
- `token`, `accessToken`, `refreshToken`
- `secret`, `apiKey`, `api_key`
- `authorization`, `auth`
- `credential`, `private`

### 字段脱敏示例

```go
// 敏感字段会自动脱敏
logger.InfoWithFields(map[string]any{
    "username": "john",
    "password": "mysecret123",    // 输出: pa****23
    "token":    "abc123xyz789",   // 输出: ab****89
}, "用户登录")
```

### 消息内容脱敏

日志消息中的敏感信息也会自动脱敏：

```go
// 输入
logger.Info("用户登录 password=abc123456 token=xyz987654321")

// 输出（自动脱敏）
// 用户登录 password=ab****56 token=xy****21
```

支持的消息格式：

| 格式 | 示例 | 脱敏后 |
|------|------|--------|
| key=value | `password=secret123` | `password=se****23` |
| key: value | `token: abcdefgh` | `token: ab****gh` |
| JSON | `"password": "secret"` | `"password": "****"` |

### 手动脱敏

如需手动脱敏，可使用以下函数：

```go
// 脱敏单个值
masked := logger.MaskValue("mysecretpassword")
// 输出: my****rd

// 脱敏字段映射
fields := logger.SanitizeFields(map[string]any{
    "username": "john",
    "password": "secret123",
})

// 脱敏消息内容
msg := logger.SanitizeMessage("login with password=secret123")
```

## 调用者信息

启用 `printCaller: true` 后，日志会包含调用位置信息：

```yaml
loggers:
  printCaller: true
```

输出示例：

```
time="2024-01-15 10:30:00" level=info msg="用户登录成功" file="D:/project/service/user.go:45" func="main.(*UserService).Login"
```

## 日志轮转

### 轮转策略

日志文件按以下规则自动轮转：

1. **时间轮转**：根据 `rotationTime` 定期创建新文件
2. **大小轮转**：文件达到 `rotationSize` 限制时轮转
3. **自动清理**：超过 `maxAge` 天的日志自动删除

### 文件命名规则

根据轮转时间间隔，文件命名模式不同：

| 轮转时间 | 文件命名模式 | 示例 |
|----------|--------------|------|
| ≤ 60分钟 | `{level}.{YYYYMMDDHHmm}.log` | `info.202401151030.log` |
| 1-24小时 | `{level}.{YYYYMMDDHH}.log` | `info.2024011510.log` |
| ≥ 24小时 | `{level}.{YYYYMMDD}.log` | `info.20240115.log` |

### 日志目录结构

```
log/
├── trace.202401151030.log
├── debug.202401151030.log
├── info.202401151030.log
├── warn.202401151030.log
├── error.202401151030.log
└── ...
```

## 最佳实践

### 1. 统一使用封装函数

```go
// ✅ 推荐：使用封装函数
logger.Info("用户登录成功")
logger.ErrorWithFields(fields, "处理失败")

// ❌ 不推荐：直接使用 Logger 实例
logger.Logger.Info("用户登录成功")
logger.Logger.WithFields(logrus.Fields{...}).Error("处理失败")
```

### 2. 使用结构化日志记录上下文

```go
// ✅ 推荐：结构化日志便于检索
logger.InfoWithFields(map[string]any{
    "userId":  userId,
    "traceId": traceId,
    "action":  "createOrder",
}, "订单创建成功")

// ❌ 不推荐：信息混在消息中难以检索
logger.Info("用户 %d 创建订单成功，traceId: %s", userId, traceId)
```

### 3. 错误日志包含堆栈信息

```go
import "runtime/debug"

logger.ErrorWithFields(map[string]any{
    "error":     err.Error(),
    "stackInfo": string(debug.Stack()),
}, "处理请求失败")
```

### 4. 无需担心敏感信息泄露

```go
// 敏感信息会自动脱敏，无需手动处理
logger.InfoWithFields(map[string]any{
    "username": username,
    "password": password,  // 自动脱敏
    "token":    token,     // 自动脱敏
}, "认证请求")
```

### 5. 请求追踪使用 Trace 级别

```go
// 请求日志使用 Trace 级别，便于控制输出
logger.TraceWithFields(map[string]any{
    "traceId":      traceId,
    "statusCode":   statusCode,
    "responseTime": responseTime,
}, "请求日志")
```
