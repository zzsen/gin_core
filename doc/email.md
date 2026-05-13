# 邮件工具 (Email)

## 一、概述

`utils/email` 包提供基于 SMTP 协议的 HTML 邮件发送功能，支持两种 TLS 模式：

| 方法 | 适用端口 | 说明 |
|------|----------|------|
| [SendHtmlByTLS](../utils/email/email.go) | 587 | STARTTLS 模式，先建立明文连接再升级为 TLS |
| [SendHtml](../utils/email/email.go) | 465 | 直接 TLS 模式，建立连接即加密 |

同时提供自定义的 SMTP 认证实现，支持 PLAIN 和 CRAM-MD5 两种认证方式。

## 二、配置

```go
import "github.com/zzsen/gin_core/utils/email"

conf := email.SmtpConfig{
    Sender:             "noreply@example.com",  // 发件人地址
    Username:           "smtp_user",             // SMTP 认证用户名
    Password:           "smtp_password",         // SMTP 认证密码
    Host:               "smtp.example.com",      // SMTP 服务器地址
    Port:               465,                     // SMTP 端口
    InsecureSkipVerify: false,                   // 是否跳过 TLS 证书验证
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `Sender` | `string` | 发件人邮箱地址 |
| `Username` | `string` | SMTP 认证用户名 |
| `Password` | `string` | SMTP 认证密码 |
| `Host` | `string` | SMTP 服务器地址 |
| `Port` | `int` | SMTP 端口号（465 或 587） |
| `InsecureSkipVerify` | `bool` | 是否跳过 TLS 证书校验，仅在自签名证书场景下设为 `true` |

## 三、发送邮件

### 3.1 通过 STARTTLS 发送（587 端口）

调用链：[SendHtmlByTLS](../utils/email/email.go) → `email.NewEmail` → `em.Send` (STARTTLS)

```go
err := email.SendHtmlByTLS(conf, "receiver@example.com", "邮件主题", "<h1>邮件内容</h1>")
```

### 3.2 通过直接 TLS 发送（465 端口）

调用链：[SendHtml](../utils/email/email.go) → `tls.Dial` → `smtp.NewClient` → `client.Auth` → `client.Mail` → `client.Rcpt` → `client.Data` → `client.Quit`

```go
err := email.SendHtml(conf, "receiver@example.com", "邮件主题", "<h1>邮件内容</h1>")
```

**执行流程：**
1. 建立 TLS 连接
2. 创建 SMTP 客户端并认证
3. 设置发件人、收件人
4. 构造邮件内容并写入
5. 发送 QUIT 关闭会话

## 四、认证方式

### 4.1 PLAIN 认证

[plainAuth](../utils/email/auth.go) 是标准库 `smtp.PlainAuth` 的自定义版本，**移除了强制 TLS 检查**，允许在非 TLS 连接上发送凭据。

> ⚠️ 安全风险：在非 TLS 连接上使用 PLAIN 认证，用户名和密码将以明文传输。仅应在已确认网络链路安全的环境（如 VPN、内网）中使用。

### 4.2 CRAM-MD5 认证

[CRAMMD5Auth](../utils/email/auth.go) 实现 RFC 2195 定义的 CRAM-MD5 认证机制，使用挑战-响应方式，密码不会以明文传输。

```go
auth := email.CRAMMD5Auth("username", "secret")
```

## 五、注意事项

1. **端口选择**：587 端口使用 `SendHtmlByTLS`，465 端口使用 `SendHtml`
2. **证书验证**：生产环境建议 `InsecureSkipVerify` 设为 `false`
3. **HTML 内容**：`text` 参数支持 HTML 格式，邮件以 HTML 形式渲染
