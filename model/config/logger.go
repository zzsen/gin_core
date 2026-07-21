// Package config 提供应用程序的配置结构定义
// 本文件定义了日志系统的配置结构，支持多级别日志配置和日志轮转策略
package config

// LoggersConfig 日志系统全局配置
// 该结构体包含了日志系统的基础配置，如文件路径、轮转策略等
type LoggersConfig struct {
	FilePath     string             `yaml:"filePath"`     // 日志文件存储路径，所有日志文件的根目录
	MaxAge       int                `yaml:"maxAge"`       // 日志文件最大保存时间（天），超过时间的日志文件会被自动删除
	RotationTime int                `yaml:"rotationTime"` // 日志轮转时间间隔（分钟），定期创建新的日志文件
	RotationSize int                `yaml:"rotationSize"` // 日志轮转大小限制（KB），当日志文件达到指定大小时进行轮转
	Loggers      []LoggerConfig     `yaml:"loggers"`      // 日志级别配置列表，支持为不同级别配置不同的输出策略
	PrintCaller  bool               `yaml:"printCaller"`  // 是否在日志中打印调用者信息（文件名和行号）
	Format       string             `yaml:"format"`       // 全局日志格式：text / json，空则 text
	Outputs      []LogOutputConfig  `yaml:"outputs"`      // 多输出配置：file / stdout / remote
	Resource     *LogResourceConfig `yaml:"resource"`     // 公共资源字段（service/env/host/version）
	FieldMap     map[string]string  `yaml:"fieldMap"`     // 可选字段名映射（源键 → 目标键）
}

// LoggerConfig 单个日志级别配置
// 该结构体定义了特定日志级别的配置参数，可以覆盖全局配置
type LoggerConfig struct {
	Level        string `yaml:"level"`        // 日志级别（trace、debug、info、warn、error、fatal、panic）
	FileName     string `yaml:"fileName"`     // 日志文件名，不包含路径和扩展名
	FilePath     string `yaml:"filePath"`     // 日志文件存储路径，覆盖全局配置
	MaxAge       int    `yaml:"maxAge"`       // 日志文件最大保存时间（天），覆盖全局配置
	RotationTime int    `yaml:"rotationTime"` // 日志轮转时间间隔（分钟），覆盖全局配置
	RotationSize int    `yaml:"rotationSize"` // 日志轮转大小限制（KB），覆盖全局配置
	Format       string `yaml:"format"`       // 覆盖全局 format，空则用全局
}

// LogOutputConfig 单个日志输出配置
type LogOutputConfig struct {
	Type     string              `yaml:"type"`     // file | stdout | remote
	Enabled  *bool               `yaml:"enabled"`  // nil = true
	MinLevel string              `yaml:"minLevel"` // 该输出最低级别，空表示不限制
	Remote   *RemoteOutputConfig `yaml:"remote"`   // type=remote 时的远程配置
}

// RemoteOutputConfig 远程输出配置
type RemoteOutputConfig struct {
	Driver          string             `yaml:"driver"`          // loki（后续可扩展 kafka/otlp）
	Format          string             `yaml:"format"`          // 远程专用 format，空则用全局
	QueueSize       int                `yaml:"queueSize"`       // 异步队列容量
	BatchSize       int                `yaml:"batchSize"`       // 批量发送条数
	FlushIntervalMs int                `yaml:"flushIntervalMs"` // 刷盘间隔（毫秒）
	Loki            *LokiOutputConfig  `yaml:"loki"`            // Loki Push 配置
	Retry           *RemoteRetryConfig `yaml:"retry"`           // 重试策略
}

// LokiOutputConfig Loki Push HTTP 配置
type LokiOutputConfig struct {
	URL           string   `yaml:"url"`           // Push URL，如 http://host:3100/loki/api/v1/push
	TimeoutMs     int      `yaml:"timeoutMs"`     // HTTP 超时（毫秒）
	Labels        []string `yaml:"labels"`        // 作为 Loki label 的字段名白名单
	BearerToken   string   `yaml:"bearerToken"`   // 可选 Bearer Token
	BasicUser     string   `yaml:"basicUser"`     // 可选 Basic Auth 用户名
	BasicPassword string   `yaml:"basicPassword"` // 可选 Basic Auth 密码
}

// RemoteRetryConfig 远程发送重试配置
type RemoteRetryConfig struct {
	MaxAttempts int `yaml:"maxAttempts"` // 最大尝试次数（含首次）
	BackoffMs   int `yaml:"backoffMs"`   // 重试间隔（毫秒）
}

// LogResourceConfig 公共资源维度字段
type LogResourceConfig struct {
	ServiceName string `yaml:"serviceName"`
	Env         string `yaml:"env"`
	Host        string `yaml:"host"`
	Version     string `yaml:"version"`
}

// OutputEnabled 判断输出是否启用。
//
// 【规则】Enabled 为 nil（YAML 未写）时视为 true，保持「配置了即启用」的默认语义。
func OutputEnabled(o LogOutputConfig) bool {
	if o.Enabled == nil {
		return true
	}
	return *o.Enabled
}

// ToDbLoggerConfig 转换为数据库日志配置
// 该方法会为数据库相关的日志配置添加"DB"后缀，用于区分不同类型的日志
// 返回：
//   - LoggersConfig: 修改后的日志配置，包含数据库日志的特定设置
func (loggersConfig LoggersConfig) ToDbLoggerConfig() LoggersConfig {
	// 数据库日志配置的后缀标识
	logSubfix := "DB"

	// 如果全局文件路径不为空，为其添加DB后缀
	if loggersConfig.FilePath != "" {
		loggersConfig.FilePath = loggersConfig.FilePath + logSubfix
	}

	// 遍历所有日志级别配置，为文件名和文件路径添加DB后缀
	for i := range loggersConfig.Loggers {
		if loggersConfig.Loggers[i].FileName != "" {
			loggersConfig.Loggers[i].FileName = loggersConfig.Loggers[i].FileName + logSubfix
		}
		if loggersConfig.Loggers[i].FilePath != "" {
			loggersConfig.Loggers[i].FilePath = loggersConfig.Loggers[i].FilePath + logSubfix
		}
	}
	return loggersConfig
}
