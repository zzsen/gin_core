# gin_core
基于 gin 封装的核心库，包含 redis、logger、gorm，mysql 等基础库

## 文件目录
```bash
gin_core
├── go.mod                            # 依赖包管理文件
├── go.sum                            # 依赖包校验文件
├── conf                              # 配置文件
│   ├── config-default.yml            #   ├（供参考）默认配置文件
│   └── config-prod.yml               #   └（供参考）正式环境配置文件
├── constant                          # 常量
│   └── config.go                     #   └ 配置类常量
├── core                              # 核心文件
│   ├── cmdline.go                    #   ├ 命令行参数解析
│   ├── initConfig.go                 #   ├ 配置文件初始化
│   └── server.go                     #   └ 服务启动主方法
├── exception                         # 异常
│   ├── authFailed.go                 #   ├ 授权失败
│   ├── commonError.go                #   ├ 常规错误
│   ├── index.go                      #   ├ 普通失败
│   ├── invalidParam.go               #   ├ 参数校验不通过
│   └── rpcError.go                   #   └ rpc错误
├── global                            # 全局变量
│   └── global.go                     #   └ 全局变量，redisClient，mysqlClient等
├── initialize                        # 初始化
│   ├── mysql.go                      #   ├ 初始化mysql
│   └── redis.go                      #   └ 初始化redis
├── logger                            # 日志
├── main.go                           # （供参考）程序主入口
├── middleware                        # 中间件
│   ├── exceptionHandler.go           #   ├ 异常处理
│   ├── log.go                        #   ├ 日志
│   └── redisSession.go               #   └ session缓存
├── model                             # 模型
│   ├── config                        #   ├ 配置模型
│   │   ├── config.go                 #   │ ├ 配置模型
│   │   ├── mysql.go                  #   │ ├ 数据库配置模型
│   │   ├── mysqlResolver.go          #   │ ├ 数据库配置模型（读写分离，多库）
│   │   ├── redis.go                  #   │ ├ redis配置模型
│   │   ├── service.go                #   │ ├ 服务配置模型
│   │   ├── smtp.go                   #   │ ├ smtp配置模型
│   │   └── system.go                 #   │ └ 系统配置模型
│   ├── entity                        #   ├ 数据库模型
│   │   └── baseModel.go              #   │ └ 数据库基类模型
│   ├── request                       #   ├ 请求模型
│   │   ├── common.go                 #   │ ├ 常用请求模型（getById等）
│   │   └── page.go                   #   │ └ 分页请求模型
│   └── response                      #   └ 响应模型
│       ├── page.go                   #     ├ 分页响应模型
│       └── response.go               #     └ 响应模型
├── README.md                         # readme
├── request                           # 请求工具
│   └── index.go                      #   └ 参数检验
├── schedule                          # （供参考）定时器
├── sessionStore                      # session存储
└── utils                             # 工具类
    ├── email                         #   ├ 邮件工具类
    ├── encrpt                        #   ├ 加解密工具类（aes，rsa）
    ├── file                          #   ├ 文件工具类
    ├── ginContext                    #   ├ gin上下文工具类
    └── httpClient                    #   └ http请求工具类
```

## 框架使用

### 安装使用

1. 新建工程目录

   `mkdir [projectName] && cd [projectName]`

   > `projectName`替换为项目工程的名称

2. 初始化 go.mod

   `go mod init [projectName]`

   > `projectName`替换为项目工程的名称

3. 修改要加载仓库的拉取方式

   `git config --global url."git@github.com:".insteadof "https://github.com/"`

4. 修改 go 环境变量

   `go env -w GOPRIVATE=github.com`

5. 拉取`gin_core`依赖包

   `go get -u https://github.com/zzsen/gin_core`

### 程序主入口

在根路径新建`main.go`文件

```go
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/core"
	_ "github.com/zzsen/gin_core/middleware"
	"github.com/zzsen/gin_core/model/config"
)

// 继承config.Config，拓展自定义配置项：Secret
type CustomConfig struct {
    // 配置基类
	config.BaseConfig `yaml:",inline"`

    // 假设这里需要拓展自定义配置项为string类型的Secret
    Secret        string `yaml:"secret"`
}

// 退出服务前执行的方法，例如，异常上报等
func execFunc() {
	fmt.Println("server stop")
}

// 自定义路由
func getCustomRouter1() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("customRouter1")
		r.GET("test", func(c *gin.Context) {
			response.Ok(c)
		})
	}
}
func getCustomRouter2() func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("customRouter2")
		r.GET("test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "success",
			})
		})
	}
}

func main() {
    // 初始化配置
	customConfig := &CustomConfig{}
	core.LoadConfig(customConfig)

    // 初始化路由
	opts := []gin.OptionFunc{}
	opts = append(opts, getCustomRouter1())
	opts = append(opts, getCustomRouter2())

	//启动服务
	core.Start(opts, execFunc)
}

```

### 运行

`go run main.go` 或
`go run main.go --env={env} --conf={conf}`

|参数|说明| 默认值     |
|---|---|---------|
| env |运行环境, 建议: dev 和 prod, 运行环境会影响加载的配置文件，详见[配置文件](###配置文件) | default |
| conf |配置文件所在文件夹路径, 详见[配置文件](###配置文件)| ./conf |

### 配置文件

框架会根据启动程序时的命令行参数决定加载不同的配置文件, 默认情况下, 加载`./conf/config-default.yml`

> 配置说明详见[运行](###运行)

#### 示例

``` yml
# 自定义拓展配置项
secret: "I'm a secret"

# 框架定义配置项
service:                               # http服务配置项
  ip: "0.0.0.0"                          # 监听网卡地址
  port: 8056                             # 监听端口
  routePrefix: "/demo"                   # 路由路径前缀
  sessionValid: 1800                     # session有效时长
  sessionPrefix: "gin_"                  # session key前缀
  middlewares:                           # 中间件，注意: 顺序对应中间件调用顺序
    - "logHandler"
    - "exceptionHandler"
log:                                   # 日志配置项
  loggers:                             # logger数组，可以添加stdLogger(控制台打印)、fileLogger(文件打印)
    - type: "stdLogger"                  # 定义一个控制台打印logger，打印info以上级别日志
      level: "info"
    - type: "fileLogger"                 # 定义一个文件打印logger
      level: "info"                      # 打印info以上级别日志
      filePath: "./log/gin_core_demo.log" # 文件日志存储路径
      maxSize: 100                       # 单个文件最大值，单位mb
      maxAge: 1                          # 单个文件最大天数，超过该值，文件将被滚动存储
      maxBackups: 60                     # 最多保留多少个日志文件
      compress: false                    # 是否开启压缩
    - type: "fileLogger"
      level: "error"                     # 打印error以上级别日志
      filePath: "./log/gin_core_error.log"
      maxSize: 100
      maxAge: 1
      maxBackups: 60
      compress: false
db:                                    # 数据库连接配置
  host: "127.0.0.1"                      # 数据库ip
  port: 3306                             # 数据库端口
  dbName: "test"                         # 数据库名
  username: "root"                       # 连接用户
  password: ""                           # 连接密码(不要写在配置文件中提交到git)
  maxIdleConns: 10                       # 连接最长空闲时间
  maxOpenConns: 10                       # 最大并发连接数
  migrate: "update"                      #每次启动时更新数据库表的方式 update:增量更新表，create:删除所有表再重新建表 off:关闭自动更新
  enableLog: false                       # 是否开启日志
  slowThreshold: 100                     # 慢查询阈值
  tablePrefix: ""                        # 表名前缀
redis:                                 # redis连接配置
  addr: "localhost:6379"                 # redis地址
  db: 0                                  # redis库
  password: ""                           # redis密码
```

### 接口开发

#### controller 方法

在根路径下, 新建`controller`文件夹, `controller`文件夹中, 视情况看是否需要再细分不同的`controller`的`package`, 如果需要, 则新建对应`package`同名的文件夹, 然后新建对应`controller`, 如果不需要, 直接新建对应`controller`文件即可

```go
// controller/test/test.go
package test

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)

func Test(ctx *gin.Context) {
	response.Ok(ctx)
}
```

#### 路由

在根路径下, 新建`router`文件夹,`router`文件夹内, 视情况看是否需要再细分不同的`router`的`package`, 如果需要, 则新建对应`package`同名的文件夹, 然后新建对应`router`, 如果不需要, 直接新建对应`router`文件即可

```go
// router/test/test.go
package test

import (
	"github.com/gin-gonic/gin"

	// {当前项目module名}/controller/test
	"ginDemo/contorller/test"
)

func InitRouter(handlers ...gin.HandlerFunc) func(e *gin.Engine) {
	return func(e *gin.Engine) {
		r := e.Group("testRouter", handlers...)
		r.GET("test", test.Test)
	}
}

// router/router.go
package router

import (
	"github.com/gin-gonic/gin"

	// {当前项目module名}/router/test
	"ginDemo/router/test"
)

func InitRouter(routerFunc *([]gin.OptionFunc)) {
	*routerFunc = append(*routerFunc, test.InitRouter())
}
```

#### 中间件

在根路径下, 新建`middleware`文件夹

```go
// middleware/traceLogHandler.go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/logger"
)

func TraceLogHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.RequestURI
		logger.Info("receive a request：%s ", path)
	}
}

func RegisterHandler() {
	// 将TraceLogHandler方法以别名为TraceLogHandler注册到中间件列表
	core.RegisterMiddleware("TraceLogHandler", TraceLogHandler)
}

```

##### 中间件使用

1. 全局使用
   在`main.go`的 main 方法中, 调用`RegisterHandler`, 然后再`配置文件`的`middleware`中, 把中间件注册的别名添加进去.
   上述例子中, `TraceLogHandler`方法的别名为`TraceLogHandler`

    ``` yml
    config:
    # ...
    service: # http服务配置
        # ...
        middlewares: # 中间件，注意以下顺序对应中间件调用顺序
        - "middleware1"
        - "TraceLogHandler" # 上述例子的中间件
        - "middleware2"
    # ...
    ```

   > `RegisterHandler`也可改为放于`middleware`的 init 方法中, 在引包的时候自动调用, 注册中间件

2. 在 总路由router 中使用

   ```go
    // router/router.go
    package router

    import (
        // {当前项目module名}/router/test
        "ginDemo/router/test"
        // {当前项目module名}/middleware
        "ginDemo/middleware"
    )

    func InitRouter(routerFunc *([]gin.OptionFunc)) {
        // 以参数形式传递
        *routerFunc = append(*routerFunc, test.InitRouter(middleware.TraceLogHandler()))
    }
   ```

3. 在 子路由router 中使用

   ```go
    // router/test/test.go
    package test

    import (
        "github.com/gin-gonic/gin"

        // {当前项目module名}/controller/test
        "ginDemo/contorller/test"
    )

    func InitRouter(handlers ...gin.HandlerFunc) func(e *gin.Engine) {
        return func(e *gin.Engine) {
            // 注册在子路由的跟路由
            r := e.Group("testRouter", middleware.TraceLogHandler())
            // 注册在子路由的子路由
            r.GET("test", middleware.TraceLogHandler(), test.Test)
        }
    }
   ```

### 数据库模型

orm 使用的是`gorm`, 使用指南可见[gorm 官方文档](https://gorm.io/zh_CN/docs/)
<br>
在根路径中新建文件夹`model/entity`, 存放数据库模型文件.

```go
// model/entity/user.go
package model

type User struct {
	Id       int    `gorm:"primary_key;column:id" json:"id"`
	Uid      string `gorm:"not null; column:uid; comment:用户id; size:100" json:"uid"`
	Nickname string `gorm:"not null; column:nickname; comment:昵称; size:500" json:"nickname"`
}

```

### 定时器

在根路径中新建文件夹`schedule`, 存放定时器文件.

```go
// schedule/test.go
package schedule

import (
	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/logger"
)

func init() {
    StartCron()
}

func StartCron() {
	c := cron.New()
	//c.AddFunc("* * * * *", Print)
	c.AddFunc("@every 10s", Print)
	// 暂不启动定时任务
	c.Start()
}

func Print() {
	logger.Info("schedule run")
}
```

在`main.go`中, 引入`schedule`, 即可启动定时任务

> `import _ "ginDemo/schedule"`
