# 请求校验 (Validation)

## 一、概述

`request` 包提供基于 [go-playground/validator](https://github.com/go-playground/validator) v10 的请求参数校验功能。校验失败时自动抛出 `InvalidParam` 异常，由框架统一捕获并返回标准错误响应。

## 二、快速开始

调用链：[request.Validate](../request/index.go) → `validator.New().Struct` → 校验失败时 `panic(exception.NewInvalidParam(...))`

```go
import "github.com/zzsen/gin_core/request"

type CreateUserRequest struct {
    Name  string `json:"name"  validate:"required,min=3,max=20"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age"   validate:"gte=0,lte=150"`
}

func CreateUserHandler(c *gin.Context) {
    var req CreateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        panic(exception.NewInvalidParam(err.Error()))
    }

    // 执行参数校验（失败会自动 panic InvalidParam）
    request.Validate(req)

    // 业务逻辑...
}
```

## 三、校验规则

### 3.1 常用 validate 标签

| 标签 | 示例 | 说明 |
|------|------|------|
| `required` | `validate:"required"` | 必填 |
| `min` | `validate:"min=3"` | 最小值（数值）或最小长度（字符串） |
| `max` | `validate:"max=100"` | 最大值或最大长度 |
| `len` | `validate:"len=6"` | 精确长度 |
| `email` | `validate:"email"` | 有效邮箱格式 |
| `url` | `validate:"url"` | 有效 URL 格式 |
| `numeric` | `validate:"numeric"` | 数字字符串 |
| `alpha` | `validate:"alpha"` | 仅字母 |
| `alphanum` | `validate:"alphanum"` | 字母和数字 |
| `gte` | `validate:"gte=0"` | 大于或等于 |
| `lte` | `validate:"lte=100"` | 小于或等于 |
| `gt` / `lt` | `validate:"gt=0"` | 大于 / 小于 |
| `oneof` | `validate:"oneof=active inactive"` | 值为指定列表之一 |

### 3.2 组合校验

多个规则用逗号分隔（AND 关系）：

```go
type Pagination struct {
    Page     int `validate:"required,gte=1"`
    PageSize int `validate:"required,gte=1,lte=100"`
}
```

### 3.3 嵌套结构体

支持嵌套结构体校验，使用 `dive` 标签校验切片/Map 元素：

```go
type BatchRequest struct {
    Items []ItemRequest `validate:"required,min=1,dive"`
}

type ItemRequest struct {
    ID   string `validate:"required"`
    Name string `validate:"required,min=1"`
}
```

## 四、错误消息格式化

校验失败时，[formatValidationErrors](../exception/invalid_param.go) 会自动生成中文错误消息：

```
【参数校验不通过】; Name不能为空; Email必须是有效的邮箱地址
```

支持嵌套路径展示，如 `Items[0].ID不能为空`。

## 五、手动使用 validator

如果需要更细粒度的控制，可直接使用 validator 并手动转换异常：

```go
import (
    "github.com/go-playground/validator/v10"
    "github.com/zzsen/gin_core/exception"
)

validate := validator.New()
err := validate.Struct(req)
if err != nil {
    valErrs := err.(validator.ValidationErrors)
    panic(exception.NewInvalidParamFromValidator(valErrs))
}
```

## 六、注意事项

1. [request.Validate](../request/index.go) 内部会 `panic`，必须在有 `exceptionHandler` 中间件保护的 handler 链中调用
2. `validate` 标签定义在结构体字段上，与 `json` / `form` 标签并列
3. 更多校验规则请参考 [validator 官方文档](https://pkg.go.dev/github.com/go-playground/validator/v10)
