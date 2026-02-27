# 中间件 (Middleware)

## 一、概述
在 `gin_core` 框架中，中间件是处理请求的重要组成部分，它可以在请求到达控制器之前或之后执行一些额外的操作，如日志记录、错误处理、权限验证、超时控制等。中间件本质上是 `handlerFunc` 类型的函数，与控制器方法类似。


## 二、编写中间件

### 1. 写法

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

### 2. 配置

一般来说, 中间也有自己的配置, 如, 超时中间件可以支持配置超时时长.

#### 2.1 方法参数

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

#### 2.2 配置文件参数

下面是一个超时中间件的例子, **支持将配置文件的参数传入中间件**.

```go
package middleware

import (
	"time"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
)

func TimeoutHandler() gin.HandlerFunc {
    timeout := time.Duration(app.BaseConfig.Service.ApiTimeout) * time.Second
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

## 三、使用中间件

中间件主要有以下使用方式:

1. 全局使用

2. 路由使用

### 1. 全局使用

通过调用`core.RegisterMiddleware`方法, 将中间件加入到框架的中间件列表, 随后在配置文件中启用中间件即可.

1. 注册中间件到框架中间件列表

   ```go
   package middleware

   import (
       "github.com/zzsen/gin_core/core"
   )

   func init() {
       if err := core.RegisterMiddleware("timeoutHandler", TimeoutHandler); err != nil {
           panic(err)
       }
   }
   ```

   然后在 `main.go` 中通过 `import _ "my-project/middleware"` 引入，`init` 函数会自动注册中间件。

2. 配置中间件调用顺序
   在配置文件中, 可以配置中间件的调用顺序, 具体配置如下:

   ```yaml
   service: # http服务配置
   # ...
   middlewares: # 中间件, 注意: 顺序对应中间件调用顺序
     - "exceptionHandler" # 异常处理中间件
     - "traceIdHandler" # 请求id中间件
     - "traceLogHandler" # 请求日志中间件
     - "timeoutHandler" # 超时中间件
   # 上述配置中, 则会先调用异常处理中间件, 然后是请求日志中间件, 最后是超时中间件
   ```

### 2. 路由使用

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

## 四、内置中间件

本框架内置了以下中间件，通过配置 `service.middlewares` 启用：

| 名称 | 说明 |
|------|------|
| `prometheusHandler` | Prometheus 指标采集，统计请求计数、耗时分布和并发数 |
| `exceptionHandler` | 统一异常处理，捕获 panic 并返回标准错误响应 |
| `otelTraceHandler` | OpenTelemetry 链路追踪，支持 W3C Trace Context 标准 |
| `traceIdHandler` | 请求追踪 ID，优先从上游请求头（`X-Trace-ID`、`X-Request-ID`）读取，未传递时生成 UUID，并注入上下文和响应头 |
| `traceLogHandler` | 请求日志，记录请求方式、路由、状态码、耗时、IP 等信息 |
| `timeoutHandler` | 请求超时控制，基于 `service.apiTimeout` 配置 |
| `rateLimitHandler` | API 限流，支持内存 / Redis 存储和多维度限流策略 |
| `corsHandler` | CORS 跨域处理，支持预检请求（OPTIONS） |

这些中间件可以通过全局使用或路由使用的方式应用到项目中。

## 五、注意事项
* **中间件顺序**：在全局使用中间件时，配置文件中 middlewares 字段的顺序决定了中间件的调用顺序，需要根据业务需求合理安排。
* **中间件注册**：在使用 RegisterMiddleware 方法注册中间件时，确保中间件名称的唯一性，避免出现名称冲突。
* **性能影响**：中间件会在每个请求中执行，因此需要注意中间件的性能，避免在中间件中执行耗时操作。