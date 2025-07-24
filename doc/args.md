# 运行参数 (Args)

在使用 gin_core 框架启动项目时，可以通过命令行参数来指定一些关键配置信息。以下是对各个运行参数的详细说明：

## 一、参数列表

| 参数 | 说明 | 默认值 | 必填 | 示例值 |
|------|------|--------|------|--------|
| `env` | 运行环境标识，影响配置文件加载和框架行为 | `default` | ❌ | `dev`, `prod`, `test` |
| `config` | 配置文件所在文件夹路径 | `./conf` | ❌ | `./config`, `/etc/app/conf` |
| `cipherKey` | 配置文件解密密钥，用于解密敏感配置信息 | 空字符串 | ❌ | `mySecretKey123` |

### 参数详细说明

#### env (运行环境)
- **作用**: 确定当前应用的运行环境，影响配置文件选择和框架行为
- **常用值**: 
  - `dev` - 开发环境
  - `test` - 测试环境
  - `prod` - 生产环境
- **影响范围**: 
  - 配置文件加载(`config.{env}.yml`)
  - Gin框架模式(生产环境自动切换Release模式)
  - pprof性能分析工具启用状态

#### config (配置文件路径)
- **作用**: 指定配置文件所在的目录路径
- **支持格式**: 相对路径(`./conf`)或绝对路径(`/etc/app/conf`)
- **注意事项**: 确保路径存在且程序有读取权限

#### cipherKey (解密密钥)
- **作用**: 解密配置文件中`CIPHER()`格式的加密内容
- **安全特性**: 解密失败不会阻断服务启动，仅记录警告日志
- **使用场景**: 保护数据库密码、API密钥等敏感配置信息

## 二、环境参数获取优先级

### 获取顺序说明

框架按照以下优先级顺序确定运行环境：

1. **命令行参数 (最高优先级)**
   ```bash
   # 直接指定环境
   go run main.go --env prod
   ```

2. **env 文件读取 (次优先级)**
   
   如果命令行未指定 env 参数，框架会读取项目根目录下的 `env` 文件：
   
   **env 文件内容示例：**
   ```
   dev
   ```
   
   **读取规则：**
   - 读取文件首行内容
   - 使用正则表达式 `[a-zA-Z0-9_]+` 匹配有效环境名
   - 忽略注释和空行

3. **默认值 (最低优先级)**
   
   如果以上方式都无法获取有效环境值，使用 `default` 作为默认环境。

### 环境确定流程

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

## 三、配置文件加载机制

### 文件命名规范

配置文件必须遵循以下命名规范：

```
配置目录/
├── config.default.yml    # 默认配置文件 (必需)
├── config.dev.yml        # 开发环境配置
├── config.test.yml       # 测试环境配置
├── config.staging.yml    # 预发布环境配置
└── config.prod.yml       # 生产环境配置
```

### 加载规则详解

1. **基础配置加载**
   - 首先加载 `config.default.yml` 文件
   - 如果默认配置文件不存在，程序正常启动但可能缺少必要配置
   - 默认配置提供所有配置项的基础值

2. **环境配置覆盖**
   - 当环境不是 `default` 时，加载对应的 `config.{env}.yml` 文件
   - 如果环境配置文件不存在，程序会退出并报错
   - 环境配置中的项目会覆盖默认配置中的相同项
   - 未在环境配置中定义的项保持默认值

3. **配置合并示例**

   **config.default.yml:**
   ```yaml
   service:
     port: 8080
     debug: true
     timeout: 30
   database:
     host: localhost
     port: 3306
   ```

   **config.prod.yml:**
   ```yaml
   service:
     debug: false    # 覆盖默认值
   database:
     host: prod-db.example.com    # 覆盖默认值
     # port 和 timeout 保持默认值
   ```

   **最终生产环境配置：**
   ```yaml
   service:
     port: 8080      # 来自默认配置
     debug: false    # 来自生产环境配置
     timeout: 30     # 来自默认配置
   database:
     host: prod-db.example.com    # 来自生产环境配置
     port: 3306      # 来自默认配置
   ```

### 启动示例

```bash
# 开发环境启动
go run main.go --env dev --config ./custom_conf

# 框架加载顺序：
# 1. ./custom_conf/config.default.yml
# 2. ./custom_conf/config.dev.yml
```

## 四、配置文件加密功能

### 加密内容格式

在配置文件中，敏感信息可以使用 `CIPHER()` 格式进行加密：

```yaml
database:
  host: localhost
  username: myuser
  password: CIPHER(encrypted_password_string)    # 加密的密码

redis:
  password: CIPHER(encrypted_redis_password)     # 加密的Redis密码

api:
  secret_key: CIPHER(encrypted_api_secret)       # 加密的API密钥
```

### 解密机制

1. **自动识别**: 框架启动时自动扫描配置文件中的 `CIPHER()` 标记
2. **密钥解密**: 使用 `cipherKey` 参数提供的密钥进行AES ECB解密
3. **内容替换**: 解密成功后将加密内容替换为明文
4. **错误容错**: 解密失败时记录警告日志，但不阻断服务启动

### 使用示例

```bash
# 生产环境启动，包含加密配置
go run main.go --env prod --config ./conf --cipherKey mySecretKey123

# 推荐：使用环境变量传递密钥
export CIPHER_KEY="mySecretKey123"
go run main.go --env prod --config ./conf --cipherKey $CIPHER_KEY
```

### 安全建议

- **密钥管理**: 不要将密钥硬编码在脚本中，使用环境变量或密钥管理系统
- **权限控制**: 确保配置文件和密钥只有必要的用户可以访问
- **密钥轮换**: 定期更换加密密钥，提高安全性
- **日志保护**: 密钥不会出现在应用日志中

## 五、最佳实践和注意事项

### 环境配置最佳实践

1. **环境标识规范**
   - 使用简短有意义的环境名: `dev`, `test`, `prod`
   - 避免使用特殊字符，只使用字母、数字和下划线
   - 保持环境名称的一致性

2. **配置文件组织**
   - 将配置文件统一放在专门的目录中
   - 为每个环境创建独立的配置文件
   - 在默认配置中提供完整的配置模板

3. **敏感信息处理**
   - 对数据库密码、API密钥等敏感信息进行加密
   - 使用环境变量传递解密密钥
   - 不要在版本控制中提交包含明文密码的配置文件

### 常见问题解决

1. **配置文件找不到**
   ```
   错误: 配置文件目录下不存在自定义配置文件
   解决: 检查文件路径和文件名是否正确
   ```

2. **解密失败**
   ```
   警告: 配置中含加密内容，但解密失败
   解决: 验证 cipherKey 参数是否正确
   ```

3. **权限问题**
   ```
   错误: 无法读取配置文件
   解决: 检查文件权限，确保程序有读取权限
   ```

通过合理使用这些运行参数，可以灵活地配置项目的运行环境和加载不同的配置文件，以满足不同场景下的开发和部署需求。