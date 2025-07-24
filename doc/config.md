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

## 五、自定义配置扩展

### 5.1 了解基础配置结构

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

### 5.2 自定义配置结构体
框架提供了 `BaseConfig` 类型的配置类型，项目可按需自行拓展。可以基于 `BaseConfig` 结构体创建自定义配置结构体，通过嵌入 `BaseConfig` 结构体并添加自定义字段来实现。

示例代码：
```golang
type CustomConfig struct {
    config.BaseConfig `yaml:",inline"`
    Secret            string `yaml:"secret"`
}
```

### 5.3 在配置文件中添加自定义配置项
在配置文件（如 `config.default.yml` 或特定环境的配置文件）中，你可以添加自定义配置项。以下是示例配置文件：
```yaml
# 系统配置
system: 
  language: "zh" # 语言

# 其他配置...

# 自定义配置项
secret: CIPHER(/t8wxJyz5nLKYDa7w8W3oQ==)
```

### 5.4 初始化自定义配置
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

### 5.5 使用自定义配置
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