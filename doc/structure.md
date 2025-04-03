# 目录结构

## 项目目录结构

```bash
gin_project
├── go.mod                            # 依赖包管理文件
├── go.sum                            # 依赖包校验文件
├── main.go                           # （供参考）程序主入口
├── conf                              # 配置文件
│   ├── config-default.yml            #   ├（供参考）默认配置文件
│   └── config-prod.yml               #   └（供参考）正式环境配置文件
├── constant                          # 常量（可选）
├── custom_config                     # 存放自定义配置类（可选）
├── middleware                        # 中间件（可选）
├── router                            # 路由规则（可选）
├── controller                        # 控制器（可选）
├── service                           # 业务逻辑层（可选）
├── schedule                          # 定时任务（可选）
├── model                             # 模型（可选）
│   ├── config                        #   ├ 配置模型（可选）
│   ├── entity                        #   ├ 数据库模型（可选）
│   ├── request                       #   ├ 请求模型（可选）
│   └── response                      #   └ 响应模型（可选）
└── utils                             # 工具类（可选）
```

如上，由框架约定的目录：

- `middleware` 用于编写中间件，具体参见 [Middleware](./middleware.md)。
- `router` 用于配置 URL 路由规则，具体参见 [Router](./router.md)。
- `controller` 用于解析用户的输入，处理后返回相应的结果，具体参见 [Controller](./controller.md)。
- `service` 用于编写业务逻辑层，具体参见 [Service](./service.md)。
- `schedule` 用于编写定时任务，具体参见 [Schedule](./schedule.md)。
- `model` 用于放置领域模型，如配置文件、数据库模型等，具体参见 [Model](./model.md)。

## 框架目录结构

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
│   ├── init_middleware.go            #   ├ 配置默认中间件(异常处理, 请求日志, 超时处理等)
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
│   ├── etcd.go                       #   ├ 初始化etcd
│   ├── mysql_base.go                 #   ├ 初始化mysql基类, 供其他mysql初始化使用
│   ├── mysql_resolver_test.go        #   ├ (测试用例)初始化db读写分离
│   ├── mysql_resolver.go             #   ├ 初始化db读写分离
│   ├── mysql.go                      #   ├ 初始化mysql
│   ├── rabbitmq_consumer.go          #   ├ 初始化消息队列消费者
│   └── redis.go                      #   └ 初始化redis
├── logger                            # 日志
├── main.go                           # （供参考）程序主入口
├── middleware                        # 中间件
│   ├── exception_handler.go          #   ├ 异常处理
│   ├── timeout_handler.go            #   ├ 超时处理
│   └── trace_log_handler.go          #   ├ 请求日志
├── model                             # 模型
│   ├── config                        #   ├ 配置模型
│   │   ├── config.go                 #   │ ├ 配置模型
│   │   ├── elasticsearch.go          #   │ ├ es配置模型
│   │   ├── logger.go                 #   │ ├ 日志配置模型
│   │   ├── etcd.go                   #   │ ├ etcd配置模型
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
