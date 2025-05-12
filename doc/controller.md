# 控制器 (Controller)

## 什么是 Controller
[前面章节](./router.md) 提到，我们通过 Router 将用户的请求基于 method 和 URL 分发到了对应的 Controller，那么 Controller 主要有什么职责呢？

简单地说，Controller 负责**解析用户的输入，处理后返回相应的结果**。例如：

* 在 RESTful 接口中，Controller 接受用户的参数，从数据库中查找内容返回给用户，或将用户的请求更新到数据库中。
* 在 HTML 页面请求中，Controller 根据用户访问不同的 URL，渲染不同的模板得到 HTML，后返回给用户。
* 在代理服务器中，Controller 将用户的请求转发到其他服务器，之后将那些服务器的处理结果返回给用户。

框架推荐的 Controller 层主要流程是：
1. 获取用户通过 HTTP 传递过来的请求参数。
2. 校验、组装参数。
3. 调用 Service 进行业务处理，必要时处理转换 Service 的返回结果，让它适应用户的需求。
4. 通过 HTTP 将结果响应给用户。


## 编写controller
### 1. 模型编写
在编写controller之前，先编写请求模型和数据库模型，用于接收http请求参数和数据库的数据交互。
请求模型统一放于`model/request`中，数据库模型统一放于`model/entity`中。

请求模型：
```golang
// model/request/user.go
package request

type User struct {
	Username string `binding:"required,gt=10"`
}
```

数据库模型：
```golang
// model/entity/user.go
package entity

type User struct {
	Id int `gorm:"primary_key;comment:主键;not null;" json:"id"`
	Username string `gorm:"size:500;comment:用户名;not null;" json:"username"`
	CreateTime time.Time `gorm:"column:create_time;comment:创建时间;not null;" json:"createTime"`
	UpdateTime time.Time `gorm:"column:update_time;comment:更新时间;not null;autoUpdateTime;" json:"updateTime"`
}
```

### 2.service编写
这里举个简单的例子，service相关详细内容，可参考[Service](./service.md)

```golang
// service/user/user.go
package user

import (
	userEntity "demo/model/entity/user"
)

func AddUser(user userEntity.User) error {
	return app.DB.Create(&user).Error
}
```

### 3. controller编写
框架建议所有controller都存放于`controller`目录下，当业务较复杂时，可使用多级目录存放。

```golang
// controller/user/user.go
package user

import (
	"demo/model/entity"
	"demo/model/request"
	userService "demo/service/user"
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)
func AddUser (ctx *gin.Context) {
	user := request.User{}
	// 数据校验
	if err := c.Bind(&user); err != nil {
		response.FailWithMessage(c, "参数错误")
		return
	}
	// 参数组装
	userEntity := entity.User{
		Username: user.Username
	}
	// service调用
	if err := userService.AddUser(userEntity); err != nil {
		response.FailWithMessage(c, "新建失败")
		return
	}
	// 设置响应内容和响应状态码
	response.OkWithMessage(c, "新建成功")
}
```

框架使用[validator](https://github.com/go-playground/validator)覆盖了gin自带的参数校验, 支持更多中类型的参数校验。接口响应也封装了一层，位于[Response](https://github.com/zzsen/gin_core/blob/master/model/response/response.go)。