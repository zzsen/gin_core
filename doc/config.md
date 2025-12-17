# 配置 (Config)

## 一、概述

gin_core 框架提供了强大、灵活且生产就绪的配置管理系统，支持现代应用开发中的各种配置需求：

### 🚀 **核心特性**
1. **多环境配置管理** - 支持开发、测试、生产等不同环境的配置隔离
2. **环境变量集成** - 与 Kubernetes、Docker 等容器化平台无缝集成
3. **配置安全加密** - 内置 AES 加密，保护敏感配置信息
4. **自定义配置扩展** - 基于 BaseConfig 灵活扩展项目特定配置
5. **外部配置支持** - 支持配置文件与代码分离部署
6. **热加载机制** - 支持配置变更的动态生效（结合 Etcd）

---

## 二、多环境配置

框架支持根据环境来加载配置，定义多个环境的配置文件，具体可见: [环境](./env.md)

### 2.1 配置文件结构
框架采用分层配置策略，通过环境标识符加载对应的配置文件，实现不同环境的配置隔离和管理。

#### 🗂️ **配置文件结构**
示例配置文件结构如下：
```bash
config
├── config.default.yml    # 🔧 基础配置（所有环境共享）
├── config.dev.yml        # 🛠️ 开发环境配置
├── config.test.yml       # 🧪 测试环境配置
├── config.prod.yml       # 🚀 生产环境配置
└── config.local.yml      # 💻 本地开发配置（通常不纳入版本控制）
```

#### 📋 **环境映射表**
| 环境标识 | 配置文件 | 用途 | Gin模式 |
|---------|---------|------|---------|
| `default` | config.default.yml | 基础配置 | debug |
| `dev` | config.dev.yml | 开发环境 | debug |
| `test` | config.test.yml | 单元测试 | debug |
| `prod` | config.prod.yml | 生产环境 | release |
| `local` | config.local.yml | 本地开发 | debug |

### 2.2 配置加载机制

#### 🔄 **加载流程**
```
开始
  ↓
检查命令行 --env 参数
  ↓
有值? → 是 → 使用命令行参数值
  ↓
  否
  ↓
检查 env 文件是否存在
  ↓
存在且有效? → 是 → 读取 env 文件首行
  ↓
  否
  ↓
使用默认值 "default"
  ↓
设置框架运行模式
  ↓
继续启动流程
```

#### ⚙️ **合并规则**
1. **基础配置**: 首先加载 `config.default.yml`
2. **环境覆盖**: 加载对应环境的配置文件，相同字段覆盖默认值
3. **深度合并**: 嵌套对象进行深度合并，而非完全替换
4. **类型保持**: 保持原有数据类型，防止类型转换错误

#### 💡 **配置示例**

```yml
# config.default.yml
service:
  port: 7777
````

```yml
# config.prod.yml
service:
  port: 7778
```

最终应用中的`service`的`port`为7778

---

## 三、环境变量集成

### 3.1 环境变量机制

框架支持通过 `{{变量名}}` 语法从系统环境变量中动态获取配置值，特别适用于容器化部署和CI/CD场景。

#### 🔧 **语法规则**
- **字符串替换**: `"{{ENV_VAR}}"` - 完整替换字符串
- **数值替换**: `{{PORT}}` - 直接替换数值（不需要引号）
- **嵌套替换**: `"jdbc:mysql://{{DB_HOST}}:{{DB_PORT}}/{{DB_NAME}}"` - 多变量组合

#### 🐳 **容器化部署示例**

**Docker Compose 配置**
```yaml
version: '3.8'
services:
  app:
    image: gin-core-app:latest
    environment:
      - ENV=prod
      - DB_HOST=mysql-server
      - DB_PORT=3306
      - DB_NAME=production_db
      - DB_USERNAME=app_user
      - DB_PASSWORD=secure_password
    depends_on:
      - mysql-server
      - redis-server
```

**Kubernetes ConfigMap + Secret**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  DB_HOST: "mysql-service"
  DB_PORT: "3306"
  DB_NAME: "production_db"
---
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
type: Opaque
data:
  DB_PASSWORD: "c2VjdXJlX3Bhc3N3b3Jk"  # base64编码
```

**应用配置文件**
```yaml
# config.prod.yml
db:
  host: "{{DB_HOST}}"
  port: {{DB_PORT}}
  dbName: "{{DB_NAME}}"
  username: "{{DB_USERNAME}}"
  password: "{{DB_PASSWORD}}"

```

#### 🔍 **环境变量最佳实践**

**1. 命名规范**
```bash
# 推荐的环境变量命名
APP_ENV=prod                    # 应用环境
DB_HOST=mysql.prod.com         # 数据库主机
DB_PORT=3306                   # 数据库端口  
DB_NAME=production_db          # 数据库名
```

**2. 敏感信息处理**
```bash
# 密码等敏感信息
export DB_PASSWORD="your_secure_password"
```


---

## 四、配置安全加密
考虑到不是所有项目都接入了k8s, 且环境变量配置稍显复杂, 故框架支持对配置中的参数进行加密存放。此时需要在运行项目时, 在命令行参数中加入`cipherKey`, 框架将会使用命令行中的cipherKey作为解密密钥, 对`CIPHER(xxx)`中的`xxx`进行解密, 并替换到配置中.
> 加密方式为aes

如：mysql的password为 `Hello World`, aes的密钥为 `UTabIUiHgDyh464+`
```yml
db:
  password: CIPHER(/t8wxJyz5nLKYDa7w8W3oQ==)
```

---

## 五、系统配置项详解

gin_core 框架提供了丰富的配置选项，涵盖了系统运行所需的各个方面。以下是对所有配置项的详细说明：

### 5.1 系统配置 (system)

系统功能开关配置，控制框架各个模块的启用状态：

```yaml
system:
  language: "zh"        # 系统语言设置，支持: zh(中文), en(英文)
  useMysql: true        # 是否启用MySQL数据库功能
  useRedis: true        # 是否启用Redis缓存功能
  useEs: true          # 是否启用Elasticsearch搜索引擎功能
  useRabbitMQ: true    # 是否启用RabbitMQ消息队列功能
  useSchedule: true    # 是否启用定时任务调度功能
  useEtcd: false       # 是否启用Etcd配置中心功能
```

### 5.2 HTTP服务配置 (service)

HTTP服务器相关配置，控制Web服务的运行参数：

```yaml
service:
  ip: "0.0.0.0"                    # 服务监听IP地址，0.0.0.0表示监听所有网卡
  port: 8055                       # 服务监听端口，建议使用8000-9000范围内的端口
  pprofPort: 6060                  # pprof性能分析工具端口，仅在非生产环境启用
  routePrefix: "routePrefix"       # 统一路由前缀，所有API路由都会添加此前缀
  sessionExpire: 3600              # 会话过期时间，单位：秒 (1小时)
  sessionPrefix: "gin_"            # Redis中会话缓存的键前缀
  apiTimeout: 1                    # 单个API请求超时时间，单位：秒
  readTimeout: 60                  # HTTP请求读取超时时间，单位：秒
  writeTimeout: 60                 # HTTP响应写入超时时间，单位：秒
  middlewares:                     # 中间件配置列表，注意：顺序对应中间件调用顺序
    - "exceptionHandler"           # 异常处理中间件，统一处理应用异常
    - "traceIdHandler"             # 请求追踪ID中间件，为每个请求生成唯一标识
    - "traceLogHandler"            # 请求日志中间件，记录请求详细信息
    - "timeoutHandler"             # 请求超时中间件，防止请求长时间阻塞
```

### 5.3 日志配置 (log)

日志系统配置，支持多级别日志和文件切割：

```yaml
log:
  filePath: "./log"                # 日志文件存储路径，相对于项目根目录
  maxAge: 30                       # 日志文件保存天数，超过此天数的日志文件会被自动删除
  rotationTime: 1                  # 日志文件按时间切割间隔，单位：小时，默认1小时切割一次
  rotationSize: 1024               # 日志文件按大小切割阈值，单位：KB，达到此大小会切割新文件
  printCaller: true                # 是否在日志中打印调用者信息（函数名和文件位置）
  loggers:                         # 分级别日志配置，支持不同级别使用不同的配置
    - level: "info"                # 日志级别：info级别日志配置
      fileName: "info"             # 日志文件名前缀
      rotationSize: 2048           # 此级别日志的切割大小，单位：KB
      rotationTime: 4              # 此级别日志的切割时间间隔，单位：小时
      maxAge: 7                    # 此级别日志的保存天数
    - level: "error"               # 日志级别：error级别日志配置
      fileName: "error"            # 错误日志文件名前缀
      filePath: "./log/error"      # 错误日志专用存储路径
      maxSize: 100                 # 最大文件大小，单位：MB
      maxAge: 30                   # 错误日志保存天数，通常保存更长时间
      rotationSize: 1024           # 错误日志切割大小，单位：KB
      rotationTime: 6              # 错误日志切割时间间隔，单位：小时
```

### 5.4 数据库配置 (db)

主数据库连接配置，支持连接池和GORM配置：

```yaml
db:
  host: "127.0.0.1"               # 数据库服务器地址
  port: 3306                      # 数据库端口，MySQL默认端口
  dbName: "dbName"                # 数据库名称，需要提前创建
  username: "username"            # 数据库用户名
  password: "password"            # 数据库密码，生产环境建议使用加密配置
  loc: "Local"                    # 时区设置，Local表示使用本地时区
  charset: "utf8mb4"              # 数据库字符集，utf8mb4支持完整的UTF-8字符, 默认: utf8mb4
  maxIdleConns: 100               # 连接池最大空闲连接数，建议根据并发量调整, 默认最小10
  maxOpenConns: 100               # 连接池最大打开连接数，建议根据数据库性能调整, 默认最小100
  connMaxIdleTime: 60             # 连接最大空闲时间，单位：秒，超时会被关闭, 默认最小60
  connMaxLifetime: 3600           # 连接最大生存时间，单位：秒 (1小时)
  logLevel: 3                     # GORM日志级别（1-关闭所有日志, 2-仅输出错误日志, 3-输出错误日志和慢查询, 4-输出错误日志和慢查询日志和所有sql）, 默认3
  ignoreRecordNotFoundError: true # 是否忽略"记录未找到"错误
  slowThreshold: 500              # 慢查询阈值，单位：毫秒，超过此时间的查询会被记录, 单位毫秒, 默认200毫秒
  migrate: ""                     # 数据库迁移模式：空-不迁移 create-重建表 update-更新表结构
  tablePrefix: ""                 # 表名前缀，所有表名都会自动添加此前缀，如设置为"t_"，则User表为t_user
  singularTable: true             # 是否使用单数表名，true时User表为user，false时User表为users
```

### 5.5 数据库读写分离配置 (dbResolvers)

支持多数据源和读写分离的数据库配置：

```yaml
dbResolvers:                      # 数据库读写分离和多数据源配置
  - sources:                      # 写库配置（主库）
      - host: "127.0.0.1"         # 主库地址
        port: 3306                # 主库端口
        dbName: "test"            # 主库数据库名
        username: "root"          # 主库用户名
        password: ""              # 主库密码
        migrate: ""               # 主库迁移设置
      - host: "127.0.0.1"         # 备用主库地址，支持多主库负载均衡
        port: 3306
        dbName: "test1"
        username: "root"
        password: ""
        migrate: ""
    replicas:                     # 读库配置（从库）
      - host: "127.0.0.1"         # 从库地址
        port: 3306                # 从库端口
        dbName: "test2"           # 从库数据库名
        username: "root"          # 从库用户名
        password: ""              # 从库密码
        migrate: ""               # 从库迁移设置（通常为空）
    tables:                       # 指定使用此配置的表名列表
      - "user"                    # 用户表使用读写分离
```

### 5.6 消息队列配置 (rabbitMQ)

RabbitMQ消息队列配置，支持多实例：

```yaml
rabbitMQ:                         # 主RabbitMQ连接配置
  host: "rabbitMqHost"            # RabbitMQ服务器地址
  port: 5672                      # RabbitMQ端口，默认5672
  username: "username"            # RabbitMQ用户名
  password: "password"            # RabbitMQ密码，建议使用加密配置

rabbitMQList:                     # 多RabbitMQ实例配置，支持连接多个消息队列服务
  - aliasName: "rabbitMQ1"        # 实例别名，用于在代码中引用
    host: "rabbitMqHost"          # 第一个RabbitMQ实例地址
    port: 5672
    username: "username"
    password: "password"
```

### 5.7 搜索引擎配置 (es)

Elasticsearch搜索引擎配置：

```yaml
es:                               # Elasticsearch配置
  addresses:                      # Elasticsearch集群地址列表
    - "10.23.17.83:9200"         # ES节点地址，支持多节点集群
  username: "elastic"             # ES用户名
  password: "Kingsoft@5688+&."   # ES密码，生产环境建议加密
```

### 5.8 配置中心 (etcd)

Etcd配置中心配置：

```yaml
etcd:                             # Etcd配置中心配置
  addresses:                      # Etcd集群地址列表
    - "http://esHost:9200"        # Etcd节点地址
  username: "elastic"             # Etcd用户名
  password: "esPassword"          # Etcd密码
  timeout: 5                      # 连接超时时间，单位：秒
```

### 5.9 缓存配置 (redis)

Redis缓存配置，支持多实例：

```yaml
redis:                            # 单Redis配置（主Redis实例）
  addr: "localhost:6379"          # Redis服务器地址和端口
  db: 0                           # Redis数据库编号，0-15
  password: "redis_password"      # Redis密码，如无密码可留空

redisList:                        # 多Redis配置（支持多个Redis实例）
  - aliasName: "redis1"           # Redis实例别名，用于在代码中引用
    addr: "redis1.example.com:6379" # 第一个Redis实例地址
    db: 1                          # 使用数据库1
    password: ""                   # 密码，如无密码可留空
  - aliasName: "redis2"           # 第二个Redis实例别名
    addr: "redis2.example.com:6379" # 第二个Redis实例地址
    db: 2                          # 使用数据库2
    password: ""
```

### 5.10 邮件配置 (smtp)

SMTP邮件发送配置：

```yaml
smtp:                             # SMTP邮件发送配置
  host: "smtp.example.com"        # SMTP服务器地址
  port: 587                       # SMTP端口，587为TLS端口，25为普通端口
  username: "your_email@example.com" # 发送邮箱用户名
  password: "your_email_password" # 邮箱密码或应用专用密码
  sender: "noreply@example.com"   # 发件人邮箱地址
```

### 5.11 应用配置 (secret)

应用密钥配置，支持加密存储：

```yaml
secret: CIPHER(/t8wxJyz5nLKYDa7w8W3oQ==) # 应用密钥，使用CIPHER()格式加密存储
```

---

## 六、自定义配置扩展

### 6.1 了解基础配置结构

框架提供了 `BaseConfig` 类型的配置，它包含了系统、服务、日志、数据库、消息队列等常见组件的配置信息。以下是 BaseConfig 的定义：

```golang
type BaseConfig struct {
    System       SystemInfo       `yaml:"system"`
    Service      ServiceInfo      `yaml:"service"`
    Log          LoggersConfig    `yaml:"log"`
    Db           *DbInfo          `yaml:"db"`
    Etcd         *EtcdInfo        `yaml:"etcd"`
    DbList       []DbInfo         `yaml:"dbList"`
    DbResolvers  DbResolvers      `yaml:"dbResolvers"`
    Redis        *RedisInfo       `yaml:"redis"`
    RedisList    []RedisInfo      `yaml:"redisList"`
    RabbitMQ     RabbitMQInfo     `yaml:"rabbitMQ"`
    RabbitMQList RabbitMqListInfo `yaml:"rabbitMQList"`
    Es           *EsInfo          `yaml:"es"`
    Smtp         SmtpInfo         `yaml:"smtp"`
}
```

### 6.2 自定义配置结构体
框架提供了 `BaseConfig` 类型的配置类型，项目可按需自行拓展。可以基于 `BaseConfig` 结构体创建自定义配置结构体，通过嵌入 `BaseConfig` 结构体并添加自定义字段来实现。

示例代码：
```golang
type CustomConfig struct {
    config.BaseConfig `yaml:",inline"`
    Secret            string `yaml:"secret"`
}
```

### 6.3 在配置文件中添加自定义配置项
在配置文件（如 `config.default.yml` 或特定环境的配置文件）中，你可以添加自定义配置项。以下是示例配置文件：
```yaml
# 系统配置
system: 
  language: "zh" # 语言

# 其他配置...

# 自定义配置项
secret: CIPHER(/t8wxJyz5nLKYDa7w8W3oQ==)
```

### 6.4 初始化自定义配置
在 `main.go` 文件中，你需要调用 `InitCustomConfig` 函数来初始化自定义配置。以下是示例代码：
```golang
func main() {
    // 初始化自定义配置
    core.InitCustomConfig(&CustomConfig{})

    // 其他代码...
    // 启动服务
    core.Start(execFunc)
}
```

### 6.5 使用自定义配置
在代码中，你可以通过 `app.Config` 来访问自定义配置。以下是一个示例：
```golang
func testFunc() {
  customConfig := app.Config.(*CustomConfig)
  // 使用自定义配置项
  secret := customConfig.Secret
  fmt.Println("Secret:", secret)
}
```
在这个示例中，我们通过类型断言将 `app.Config` 转换为 `*CustomConfig` 类型，然后访问 `Secret` 字段。

### 总结
通过以上步骤，你可以在 `gin_core` 框架中实现自定义配置。关键步骤包括创建自定义配置结构体、初始化自定义配置、在配置文件中添加自定义配置项以及在代码中使用自定义配置。这样可以让你根据项目的具体需求灵活地配置和使用框架。