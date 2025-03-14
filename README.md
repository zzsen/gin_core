# gin_core
基于 gin 封装的核心库, 包含 redis、logger、gorm, mysql, es, rabbitmq 等基础库

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
│   ├── init_config.go                #   ├ 配置文件初始化
│   ├── init_service.go               #   ├ 服务初始化（mysql, es, 消息队列, 定时任务等）
│   └── server.go                     #   └ 服务启动主方法
├── exception                         # 异常
│   ├── auth_failed.go                #   ├ 授权失败
│   ├── common_error.go               #   ├ 常规错误
│   ├── index.go                      #   ├ 普通失败
│   ├── invalid_param.go              #   ├ 参数校验不通过
│   └── rpc_error.go                  #   └ rpc错误
├── global                            # 全局变量
│   └── global.go                     #   └ 全局变量, redisClient, mysqlClient等
├── initialize                        # 初始化
│   ├── elasticsearch.go              #   ├ 初始化es
│   ├── mysql_resolver_test.go        #   ├ (测试用例)初始化db读写分离
│   ├── mysql_resolver.go             #   ├ 初始化db读写分离
│   ├── mysql.go                      #   ├ 初始化mysql
│   ├── rabbitmq.go                   #   ├ 初始化消息队列
│   └── redis.go                      #   └ 初始化redis
├── logger                            # 日志
├── main.go                           # （供参考）程序主入口
├── middleware                        # 中间件
│   ├── exception_handler.go          #   ├ 异常处理
│   └── log.go                        #   ├ 日志
├── model                             # 模型
│   ├── config                        #   ├ 配置模型
│   │   ├── config.go                 #   │ ├ 配置模型
│   │   ├── elasticsearch.go          #   │ ├ es配置模型
│   │   ├── logger.go                 #   │ ├ 日志配置模型
│   │   ├── mysql.go                  #   │ ├ 数据库配置模型
│   │   ├── mysql_resolver.go         #   │ ├ 数据库配置模型（读写分离, 多库）
│   │   ├── rabbitmq.go               #   │ ├ 消息队列配置模型
│   │   ├── redis.go                  #   │ ├ redis配置模型
│   │   ├── schedule.go               #   │ ├ 定时任务配置模型
│   │   ├── service.go                #   │ ├ 服务配置模型
│   │   ├── smtp.go                   #   │ ├ smtp配置模型
│   │   └── system.go                 #   │ └ 系统配置模型
│   ├── entity                        #   ├ 数据库模型
│   │   └── base_model.go             #   │ └ 数据库基类模型
│   ├── request                       #   ├ 请求模型
│   │   ├── common.go                 #   │ ├ 常用请求模型（getById等）
│   │   └── page.go                   #   │ └ 分页请求模型
│   └── response                      #   └ 响应模型
│       ├── page.go                   #     ├ 分页响应模型
│       └── response.go               #     └ 响应模型
├── README.md                         # readme
├── request                           # 请求工具
│   └── index.go                      #   └ 参数检验
└── utils                             # 工具类
    ├── email                         #   ├ 邮件工具类
    ├── encrpt                        #   ├ 加解密工具类（aes, rsa）
    ├── file                          #   ├ 文件工具类
    ├── gin_context                   #   ├ gin上下文工具类
    └── http_client                   #   └ http请求工具类
```

## 框架使用

### 安装使用

1. 新建工程目录

   `mkdir [projectName] && cd [projectName]`

   > `projectName`替换为项目工程的名称

2. 初始化 go.mod

   `go mod init [projectName]`

   > `projectName`替换为项目工程的名称
 
3. 拉取`gin_core`依赖包

   `go get -u github.com/zzsen/gin_core`

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

// 继承config.Config, 拓展自定义配置项：Secret
type CustomConfig struct {
    // 配置基类
	config.BaseConfig `yaml:",inline"`

    // 假设这里需要拓展自定义配置项为string类型的Secret
    Secret        string `yaml:"secret"`
}

// 退出服务前执行的方法, 例如, 异常上报等
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
`go run main.go --env {env} --conf {conf} --cipherKey {cipherKey}`

|参数|说明| 默认值     |
|---|---|---------|
| env |运行环境, 建议: dev 和 prod, 运行环境会影响加载的配置文件, 详见[配置文件](###配置文件) | default |
| conf |配置文件所在文件夹路径, 详见[配置文件](###配置文件)| ./conf |
| cipherKey |解密key, 当配置文件中含加密内容时使用, 解密失败时不阻断服务启动| 空字符串 |

### 配置文件

框架会根据启动程序时的命令行参数决定加载不同的配置文件, 配置文件名格式：`config.{env}.yml`, 默认情况下, 加载`./conf/config-default.yml`

> 配置说明详见[运行](###运行)

#### 数据库配置
1. 单库使用
``` yml
system:
  useMysql: true # 启用mysql
db:
  host: "" # 数据库地址
  port: 3306 # 数据库端口
  dbName: "" # 数据库名
  username: "" # 数据库账号
  password: "" # 数据库密码
```

操作数据库进行读写时, 使用`global.DB`即可

2. 多库使用
``` yml
system:
  useMysql: true # 启用mysql
dbList:
  - aliasName: "db1" # 数据库别名
    host: "" # 数据库地址
    port: 3306 # 数据库端口
    dbName: "" # 数据库名
    username: "" # 数据库账号
    password: "" # 数据库密码
  - aliasName: "db2" # 数据库别名
    host: "" # 数据库地址
    port: 3306 # 数据库端口
    dbName: "" # 数据库名
    username: "" # 数据库账号
    password: "" # 数据库密码
```

操作数据库进行读写时, 使用`global.DBList[别名]`或`global.GetDbByName(别名)`即可

3. 读写分离
``` yml
system:
  useMysql: true # 启用mysql

dbResolvers:
  - sources:  # 支持多个
      - host: "" # 数据库地址
        port: 3306 # 数据库端口
        dbName: "" # 数据库名
        username: "" # 数据库账号
        password: "" # 数据库密码
        migrate: ""
    replicas:  # 支持多个
      - host: "" # 数据库地址
        port: 3306 # 数据库端口
        dbName: "" # 数据库名
        username: "" # 数据库账号
        password: "" # 数据库密码
        migrate: ""
    tables:  # 使用该库表
      - "user"
```

操作数据库进行读写时, 使用`global.DBResolver`即可
> 更多内容可见 [DBResolver](https://gorm.io/zh_CN/docs/dbresolver.html)


#### redis配置
1. 单redis配置
``` yml
system:
  useRedis: true # 启用redis
redis:
  addr: ""
  db: 1
  password: ""
```
操作redis进行读写时, 使用`global.Redis`即可

2. 多redis配置
``` yml
system:
  useRedis: true # 启用redis
redisList:
  - aliasName: "redis1" # 别名
    addr: ""
    db: 1
    password: ""
  - aliasName: "redis2" # 别名
    addr: ""
    db: 1
    password: ""
```
操作redis进行读写时, 使用`global.RedisList[别名]`或`global.GetRedisByName(别名)`即可

#### 日志
能力：
1. 日志切割 (根据大小/时间)
2. 设置日志最大保留时间
3. 设置不同级别的日志路径
```go
  // 相关依赖库
	"github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
```

日志配置示例:
```yml
log: # 全局配置
  filePath: "./log" # 日志文件路径
  maxAge: 30 # 日志文件保存天数, 默认 30 天
  rotationTime: 1 # 日志文件切割时间, 单位: 分钟, 默认60分钟
  rotationSize: 1 # 日志文件切割大小, 单位: KB, 默认 1024KB, 即1MB
  printCaller: true # 是否打印函数名和文件信息
  loggers: # 具体level的log的配置
    - level: "info"
      fileName: "info"
      rotationSize: 2
      RotationTime: 4
      maxAge: 2
    - level: "error"
      fileName: "error"
      FilePath: "./log/error"
      maxSize: 100
      maxAge: 3
      rotationSize: 3
      RotationTime: 6
```
|配置|说明|默认值|
|--|--|--|
|level|日志级别|-|
|filePath|日志文件存放路径|./log|
|fileName|日志文件名, 无需“.log”文件后缀|app|
|maxAge|日志文件最大保留时间, 单位: 天|30天|
|rotationTime|日志文件切割时间, 单位: 分钟|60分钟|
|rotationSize|日志文件切割大小, 单位: KB|1024KB, 即1MB|
|printCaller|是否打印函数名和文件信息|false|

配置默认填充默认值, 当具体配置值不为空且合法时, 则填充对应自定义值
```go
	// 设置日志文件路径
	filePath := defaultFilePath
	if globalConfig.FilePath != "" {
		filePath = globalConfig.FilePath
	}
	if loggerConfig.FilePath != "" {
		filePath = loggerConfig.FilePath
	}
```

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
  middlewares:                           # 中间件, 注意: 顺序对应中间件调用顺序
    - "logHandler"
    - "exceptionHandler"
log:                                   # 日志配置项
  filePath: "./log"                      # 日志文件路径, 默认 ./log
  maxAge: 30                             # 日志文件保存天数, 默认 30 天
  rotationTime: 1                        # 日志文件切割时间, 单位: 分钟, 默认60分钟
  rotationSize: 1                        # 日志文件切割大小, 单位: KB, 默认 1024KB, 即1MB
  printCaller: true                      # 是否打印函数名和文件信息
  loggers:                               # 具体不同级别的日志配置
    - level: "info"
      fileName: "info"
      FilePath: "./log/info"      
      maxAge: 2
      rotationSize: 2
      RotationTime: 4
    - level: "error"
      fileName: "error"
      FilePath: "./log/error"
      maxSize: 100
      maxAge: 3
      rotationSize: 3
      RotationTime: 6
db:                                    # 数据库连接配置
  host: "127.0.0.1"                      # 数据库ip
  port: 3306                             # 数据库端口
  dbName: "test"                         # 数据库名
  username: "root"                       # 连接用户
  password: ""                           # 连接密码(不要写在配置文件中提交到git)
  maxIdleConns: 100                      # 最大空闲连接数, 默认最小10
  maxOpenConns: 100                      # 最大连接数, 默认最小100
  connMaxIdleTime: 60                    # 最大空闲时间, 单位: 秒, 默认最小60
  connMaxLifetime: 3600                  # 最大连接存活时间, 单位: 秒, 默认最小60
  logLevel: 3                            # 日志级别（1-关闭所有日志, 2-仅输出错误日志, 3-输出错误日志和慢查询, 4-输出错误日志和慢查询日志和所有sql）, 默认3
  slowThreshold: 100                     # 慢查询阈值, 单位: 毫秒,默认200
  charset: "utf8mb4"                     # 数据库编码, 默认: utf8mb4
  loc: "Local"                           # 时区, 默认: Local
  tablePrefix: ""                        # 表名前缀
  migrate: "update"                      #每次启动时更新数据库表的方式 update:增量更新表, create:删除所有表再重新建表 off:关闭自动更新
dbResolvers:                           # 多库连接配置
  - sources:                             # 写库
      - host: "127.0.0.1"                  # 数据库地址
        port: 3306                         # 数据库端口
        dbName: "test"                     # 数据库名
        username: "root"                   # 数据库账号
        password: ""                       # 数据库密码
    replicas:                            # 读库
      - host: "127.0.0.1"                  # 数据库地址
        port: 3306                         # 数据库端口
        dbName: "test1"                    # 数据库名
        username: "root"                   # 数据库账号
        password: ""                       # 数据库密码
    tables:                              # 该库对应的表
      - "user"
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
        middlewares: # 中间件, 注意以下顺序对应中间件调用顺序
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

orm 使用的是`gorm`, 使用指南可见 [gorm 官方文档](https://gorm.io/zh_CN/docs/), 多数据库支持可见 [DBResolver](https://gorm.io/zh_CN/docs/dbresolver.html)

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
	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

func init() {
  // 将方法添加到定时任务中
	core.AddSchedule(config.ScheduleInfo{
		Cron: "@every 10s",
		Cmd:  Print,
	})
}

// 需要定时执行的方法
func Print() {
	logger.Info("schedule run")
}
```

在`main.go`中, 引入`schedule`, 即可启动定时任务

> `import _ "ginDemo/schedule"`

### 消息队列
1. 配置
```yml
# 单mq使用
rabbitMQ:
  host: "rabbitMqHost"
  port: 5672
  username: "username"
  password: "password"
# 多mq使用
rabbitMQList:
  - aliasName: "rabbitMQ1" # 别名
    host: "rabbitMqHost"
    port: 5672
    username: "username"
    password: "password"
  - aliasName: "rabbitMQ2" # 别名
    host: "rabbitMqHost"
    port: 5672
    username: "username"
    password: "password"
```

2. 消费者定义
在根路径中新建文件夹`mq`, 存放消息队列方法的文件
```go
// mq/test.go
package mq

import (
	"fmt"
	"github.com/zzsen/gin_core/core"
)

func init() {
  // 添加消息队列方法
	core.AddMessageQueueConsumer(config.MessageQueue{
		QueueName:    "QueueName",
		ExchangeName: "ExchangeName",
		ExchangeType: "fanout",
		RoutingKey:   "RoutingKey",
		Fun:          mqFunc,
	})

  // 上述示例为单mq使用, 当有多个mq时, 可添加MQName参数, 设置为对应的aliasName, 如：
	// core.AddMessageQueueConsumer(config.MessageQueue{
	// 	QueueName:    "QueueName",
	// 	ExchangeName: "ExchangeName",
	// 	ExchangeType: "fanout",
	// 	RoutingKey:   "RoutingKey",
  //  MQName:       "rabbitMQ1", // 对应rabbitMQList中的aliasName字段, 会根据aliasName在配置中获取连接串
	// 	Fun:          mqFunc,
	// })
}

// 处理消息的方法
func mqFunc(message string) error {
	fmt.Println("message", message)
	return nil
}
```

在`main.go`中, 引入`mq`, 即可启动定时任务
> `import _ "ginDemo/mq"`

发送消息只需要调用global中封装的方法即可
```go
// 单mq使用
global.SendRabbitMqMsg("QueueName", "ExchangeName", "fanout", "RoutingKey", "message")
// 多mq使用, 可同时往多个mq发送消息
global.SendRabbitMqMsg("QueueName", "ExchangeName", "fanout", "RoutingKey", "message", "mq1", "mq2")
```
