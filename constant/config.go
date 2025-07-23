package constant

// DefaultEnv 默认启动环境
// 当未明确指定运行环境时，系统将使用此默认环境
const DefaultEnv = "default"

// ProdEnv 生产环境标识
const ProdEnv = "prod"

// DefaultConfigDirPath 默认配置文件夹路径
const DefaultConfigDirPath = "./conf"

// DefaultConfigFileName 默认配置文件名称
const DefaultConfigFileName = "config.default.yml"

// CustomConfigFileNamePrefix 自定义配置文件名称前缀
// 环境特定配置文件的前缀，格式：config.{env}.yml
// 例如：config.dev.yml, config.prod.yml, config.test.yml
const CustomConfigFileNamePrefix = "config."

// CustomConfigFileNameSuffix 自定义配置文件名称后缀
// 配置文件的扩展名，使用YAML格式
// 支持YAML的所有特性，包括多文档、引用等
const CustomConfigFileNameSuffix = ".yml"

// DefaultDBSlowThreshold 数据库慢查询阈值（毫秒）
const DefaultDBSlowThreshold = 200

// DefaultPprofPort 默认性能分析端口
// pprof性能分析工具的默认监听端口
// 仅在非生产环境下启用，用于：
// - CPU性能分析
// - 内存使用分析
// - 协程状态监控
// - 阻塞分析等
// 访问地址：http://localhost:6060/debug/pprof/
const DefaultPprofPort = 6060

// 服务发现相关常量
// DefaultEtcdTimeout 默认Etcd连接超时时间（秒）
// Etcd客户端连接的默认超时时间
// 用于服务发现、配置中心等场景
// 如果网络环境较差，可适当增加此值
const DefaultEtcdTimeout = 5
