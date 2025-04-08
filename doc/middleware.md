# 中间件 (Middleware)

gin 的中间件, 和 controller 方法本质上是一样的, 都是 handlerFunc 类型的函数

## 编写中间件

### 写法

以超时控制为例, 了解中间件的写法

```go
package middleware

import (
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
)

func TimeoutHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

        // 执行后续方法
		c.Next()

		endTime := time.Now()
		tookTime := endTime.Sub(startTime)
        // 这里以记录日志为例, 实际上可以做很多事情
        if tookTime > 1 * time.Second {
            fmt.Println("请求耗时:", tookTime)
        }
	}
}
```

### 配置

一般来说, 中间也有自己的配置, 如, 超时中间件可以支持配置超时时长.

#### 方法参数

下面是一个超时中间件的例子, **支持将方法参数传入中间件**.

```go
package middleware

import (
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
)

func TimeoutHandler(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

        // 执行后续方法
		c.Next()

		endTime := time.Now()
		tookTime := endTime.Sub(startTime)
        // 这里以记录日志为例, 实际上可以做很多事情
        if tookTime > timeout * time.Second {
            fmt.Println("请求耗时:", tookTime)
        }
	}
}
```

#### 配置文件参数

下面是一个超时中间件的例子, **支持将配置文件的参数传入中间件**.

```go
package middleware

import (
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
)

func TimeoutHandler() gin.HandlerFunc {
    timeout := time.Duration(global.BaseConfig.Service.ApiTimeout) * time.Second
	return func(c *gin.Context) {
		startTime := time.Now()

        // 执行后续方法
		c.Next()

		endTime := time.Now()
		tookTime := endTime.Sub(startTime)
        // 这里以记录日志为例, 实际上可以做很多事情
        if tookTime > timeout * time.Second {
            fmt.Println("请求耗时:", tookTime)
        }
	}
}
```

## 使用中间件

中间件主要有以下使用方式:

1. 全局使用

2. 路由使用

### 全局使用

通过调用`core.RegisterMiddleware`方法, 将中间件加入到框架的中间件列表, 随后在配置文件中启用中间件即可.

1. 注册中间件到框架中间件列表

   ```go
   package core

   import (
       "time"

       "github.com/zzsen/gin_core/core"
       "github.com/zzsen/gin_core/global"
   )

   func initMiddleware() {
       if err := core.RegisterMiddleware("timeoutHandler", middleware.TimeoutHandler); err != nil {
           logger.Error(err.Error())
       }
   }
   // 然后在core.Start()启动服务前, 调用initMiddleware()方法即可
   // 也可将方法改为init方法, 在导入时使用
   ```

2. 配置中间件调用顺序
   在配置文件中, 可以配置中间件的调用顺序, 具体配置如下:

   ```yaml
   service: # http服务配置
   # ...
   middlewares: # 中间件, 注意: 顺序对应中间件调用顺序
     - "exceptionHandler" # 异常处理中间件
     - "traceLogHandler" # 请求日志中间件
     - "timeoutHandler" # 超时中间件
   # 上述配置中, 则会先调用异常处理中间件, 然后是请求日志中间件, 最后是超时中间件
   ```

### 路由使用

路由使用, 分为`路由组使用`和`单路由使用`.

1. 路由组使用

   ```go
   r := e.Group("customRouter2")
   r.Use(middleware.TimeoutHandler(2 * time.Second))
   ```

2. 单路由使用

   ```go
   r.GET("test", func(c *gin.Context) {
       c.JSON(200, middleware.TimeoutHandler(2 * time.Second), gin.H{
           "message": "success",
       })
   })
   ```
