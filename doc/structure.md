# 目录结构

## 项目目录结构

```bash
gin_project
├── go.mod                            # 依赖包管理文件
├── go.sum                            # 依赖包校验文件
├── main.go                           # （供参考）程序主入口
├── conf                              # 配置文件
│   ├── config-default.yml            #   ├（供参考）默认配置文件
│   └── config-prod.yml               #   └（供参考）正式环境配置文件
├── constant                          # 常量（可选）
├── custom_config                     # 存放自定义配置类（可选）
├── middleware                        # 中间件（可选）
├── router                            # 路由规则（可选）
├── controller                        # 控制器（可选）
├── service                           # 业务逻辑层（可选）
├── schedule                          # 定时任务（可选）
├── model                             # 模型（可选）
│   ├── config                        #   ├ 配置模型（可选）
│   ├── entity                        #   ├ 数据库模型（可选）
│   ├── request                       #   ├ 请求模型（可选）
│   └── response                      #   └ 响应模型（可选）
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
│   ├── config-default.yml            #   ├（供参考）默认配置文件
│   └── config-prod.yml               #   └（供参考）正式环境配置文件
├── constant                          # 常量
│   └── config.go                     #   └ 配置类常量
├── core                              # 核心文件
│   ├── cmdline.go                    #   ├ 命令行参数解析
│   ├── config.go                     #   ├ 配置文件初始化
│   ├── engine.go                     #   ├ 路由初始化
│   ├── middleware.go                 #   ├ 配置默认中间件(异常处理, 请求日志, 超时处理等)
│   ├── service.go                    #   ├ 服务初始化入口
│   ├── validator.go                  #   ├ 参数校验（使用github.com/go-playground/validator/v10覆盖gin的参数校验）
│   ├── server.go                     #   ├ 服务启动主方法
│   ├── lifecycle                     #   ├ 服务生命周期管理
│   │   ├── interface.go              #   │ ├ 服务接口定义
│   │   ├── registry.go               #   │ ├ 服务注册中心
│   │   ├── resolver.go               #   │ ├ 依赖解析器
│   │   ├── initializer.go            #   │ ├ 并行初始化器
│   │   └── bootstrap.go              #   │ └ 消息队列/定时任务配置
│   └── services                      #   └ 内置服务实现
│       ├── logger_service.go         #     ├ 日志服务
│       ├── tracing_service.go        #     ├ 链路追踪服务
│       ├── redis_service.go          #     ├ Redis服务
│       ├── mysql_service.go          #     ├ MySQL服务
│       ├── elasticsearch_service.go  #     ├ Elasticsearch服务
│       ├── rabbitmq_service.go       #     ├ RabbitMQ服务
│       ├── etcd_service.go           #     ├ Etcd服务
│       └── schedule_service.go       #     └ 定时任务服务
├── exception                         # 异常
│   ├── auth_failed.go                #   ├ 授权失败
│   ├── common_error.go               #   ├ 常规错误
│   ├── index.go                      #   ├ 普通失败
│   ├── init_error.go                 #   ├ 初始化错误（结构化错误类型）
│   ├── invalid_param.go              #   ├ 参数校验不通过
│   └── rpc_error.go                  #   └ rpc错误
├── app                               # 全局应用
│   ├── app.go                        #   ├ 全局变量定义（DB, Redis, ES, Etcd等）
│   ├── db.go                         #   ├ 数据库工具方法
│   ├── redis.go                      #   ├ Redis工具方法
│   ├── mq.go                         #   ├ 消息队列工具方法（带重试机制）
│   └── pool_stats.go                 #   └ 连接池统计和健康检查
├── metrics                           # Prometheus 指标监控
│   ├── metrics.go                    #   ├ 指标定义（HTTP、连接池指标）
│   └── collector.go                  #   └ 指标收集器
├── tracing                           # OpenTelemetry 链路追踪
│   ├── tracing.go                    #   ├ 追踪核心初始化
│   ├── gorm_plugin.go                #   ├ GORM 数据库追踪插件
│   ├── redis_hook.go                 #   ├ Redis 追踪钩子
│   └── http_transport.go             #   └ HTTP 客户端追踪传输层
├── initialize                        # 初始化
│   ├── elasticsearch.go              #   ├ 初始化es
│   ├── etcd.go                       #   ├ 初始化etcd
│   ├── mysql_base.go                 #   ├ 初始化mysql基类, 供其他mysql初始化使用
│   ├── mysql_resolver_test.go        #   ├ (测试用例)初始化db读写分离
│   ├── mysql_resolver.go             #   ├ 初始化db读写分离
│   ├── mysql.go                      #   ├ 初始化mysql
│   ├── rabbitmq_consumer.go          #   ├ 初始化消息队列消费者
│   ├── redis.go                      #   ├ 初始化redis
│   └── tracing.go                    #   └ 初始化链路追踪
├── logger                            # 日志
├── main.go                           # （供参考）程序主入口
├── middleware                        # 中间件
│   ├── exception_handler.go          #   ├ 异常处理
│   ├── otel_trace_handler.go         #   ├ OpenTelemetry 链路追踪
│   ├── prometheus_handler.go         #   ├ Prometheus 指标采集
│   ├── timeout_handler.go            #   ├ 超时处理
│   ├── trace_id_handler.go           #   ├ 请求追踪ID
│   └── trace_log_handler.go          #   └ 请求日志
├── model                             # 模型
│   ├── config                        #   ├ 配置模型
│   │   ├── config.go                 #   │ ├ 配置模型
│   │   ├── elasticsearch.go          #   │ ├ es配置模型
│   │   ├── logger.go                 #   │ ├ 日志配置模型
│   │   ├── metrics.go                #   │ ├ 指标监控配置模型
│   │   ├── tracing.go                #   │ ├ 链路追踪配置模型
│   │   ├── etcd.go                   #   │ ├ etcd配置模型
│   │   ├── mysql.go                  #   │ ├ 数据库配置模型
│   │   ├── mysql_resolver.go         #   │ ├ 数据库配置模型（读写分离, 多库）
│   │   ├── rabbitmq.go               #   │ ├ 消息队列配置模型
│   │   ├── redis.go                  #   │ ├ redis配置模型
│   │   ├── schedule.go               #   │ ├ 定时任务配置模型
│   │   ├── service.go                #   │ ├ 服务配置模型
│   │   ├── smtp.go                   #   │ ├ smtp配置模型
│   │   └── system.go                 #   │ └ 系统配置模型
│   ├── entity                        #   ├ 数据库模型
│   │   └── base_model.go             #   │ └ 数据库基类模型
│   ├── request                       #   ├ 请求模型
│   │   ├── common.go                 #   │ ├ 常用请求模型（getById等）
│   │   └── page.go                   #   │ └ 分页请求模型
│   └── response                      #   └ 响应模型
│       ├── page.go                   #     ├ 分页响应模型
│       └── response.go               #     └ 响应模型
├── doc                               # 文档
│   ├── tracing.md                    #   ├ 链路追踪文档
│   └── ...                           #   └ 其他文档
├── README.md                         # readme
├── request                           # 请求工具
│   └── index.go                      #   └ 参数检验
└── utils                             # 工具类
    ├── email                         #   ├ 邮件工具类
    ├── encrpt                        #   ├ 加解密工具类（aes, rsa）
    ├── file                          #   ├ 文件工具类
    ├── gin_context                   #   ├ gin上下文工具类
    └── http_client                   #   └ http请求工具类
        ├── client.go                 #     ├ 高性能HTTP客户端（连接池、重试）
        └── http_client.go            #     └ HTTP请求方法封装
```
