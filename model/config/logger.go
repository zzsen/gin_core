// Package config 提供应用程序的配置结构定义
// 本文件定义了日志系统的配置结构，支持多级别日志配置和日志轮转策略
package config

// LoggersConfig 日志系统全局配置
// 该结构体包含了日志系统的基础配置，如文件路径、轮转策略等
type LoggersConfig struct {
	FilePath     string         `yaml:"filePath"`     // 日志文件存储路径，所有日志文件的根目录
	MaxAge       int            `yaml:"maxAge"`       // 日志文件最大保存时间（天），超过时间的日志文件会被自动删除
	RotationTime int            `yaml:"rotationTime"` // 日志轮转时间间隔（分钟），定期创建新的日志文件
	RotationSize int            `yaml:"rotationSize"` // 日志轮转大小限制（KB），当日志文件达到指定大小时进行轮转
	Loggers      []LoggerConfig `yaml:"loggers"`      // 日志级别配置列表，支持为不同级别配置不同的输出策略
	PrintCaller  bool           `yaml:"printCaller"`  // 是否在日志中打印调用者信息（文件名和行号）
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
