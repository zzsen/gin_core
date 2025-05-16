# 服务（Service）

## 一、概述
在 `gin_core` 框架中，`Service` 层主要用于编写业务逻辑，处理复杂的业务操作，为 `Controller` 层提供数据支持和业务处理结果。该层将业务逻辑与控制器分离，使代码结构更加清晰，便于维护和扩展。


## 二、目录结构
建议将 `Service` 相关代码统一存放在 `service` 目录下，根据不同的业务模块进行划分，例如：

```
└── service
    └── user
        └── user.go
```

## 三、编写 Service 示例
以下是一个简单的 `Service` 示例，用于处理用户添加业务：

```go
// service/user/user.go
package user

import (
    userEntity "demo/model/entity/user"
    "github.com/zzsen/gin_core/app"
)

func AddUser(user userEntity.User) error {
    return app.DB.Create(&user).Error
}
```
在上述示例中，`AddUser` 函数接收一个 `userEntity.User` 类型的参数，将其保存到数据库中，并返回可能出现的错误。

## 四、Service 层与其他层的交互
### 4.1 与 Controller 层的交互
`Controller` 层负责接收用户请求，调用 `Service` 层的方法处理业务逻辑，并返回处理结果给用户。示例如下：

```go
// controller/user/user.go
package user

import (
    "github.com/gin-gonic/gin"
    "demo/service/user"
    "demo/model/entity/user"
    "github.com/zzsen/gin_core/model/response"
)

func AddUserController(ctx *gin.Context) {
    var userReq user.User
    if err := ctx.ShouldBindJSON(&userReq); err != nil {
        response.FailWithMessage(ctx, "参数解析失败")
        return
    }

    if err := user.AddUser(userReq); err != nil {
        response.FailWithMessage(ctx, "添加用户失败")
        return
    }

    response.Ok(ctx)
}
```




### 4.2 与 Model 层的交互
`Service` 层通过 `Model` 层提供的数据模型和数据库操作方法，对数据进行增删改查等操作。例如，在上述 `AddUser` 函数中，使用 `app.DB.Create` 方法将用户数据保存到数据库中。


## 五、注意事项
* **业务逻辑封装**：将复杂的业务逻辑封装在 `Service` 层，避免 `Controller` 层代码过于臃肿。
* **错误处理**：在 `Service` 层中，对可能出现的错误进行适当的处理，并返回给 `Controller` 层，由 `Controller` 层统一返回给用户。
* **服务依赖**：`Service` 层可能依赖于其他服务，如数据库、Redis 等，在使用这些服务时，要确保其已经正确初始化。
* **事务处理**：对于涉及多个数据库操作的业务，要考虑使用事务来保证数据的一致性。