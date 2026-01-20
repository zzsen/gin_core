// Package config 提供应用程序的配置结构定义
// 本文件定义了基础配置结构，包含系统、服务、日志、数据库、缓存、消息队列等各个组件的配置信息
package config

// BaseConfig 应用程序基础配置结构
// 该结构体包含了应用程序运行所需的所有配置信息，支持YAML格式的配置文件
type BaseConfig struct {
	System       SystemInfo       `yaml:"system"`       // 系统基础配置，控制各组件是否启用
	Service      ServiceInfo      `yaml:"service"`      // 服务配置，包含端口、超时时间等
	Log          LoggersConfig    `yaml:"log"`          // 日志配置，包含文件路径、轮转策略等
	Metrics      MetricsConfig    `yaml:"metrics"`      // Prometheus 指标监控配置
	Tracing      *TracingConfig   `yaml:"tracing"`      // OpenTelemetry 链路追踪配置
	RateLimit    RateLimitConfig  `yaml:"rateLimit"`    // 限流配置，用于控制API请求速率
	Db           *DbInfo          `yaml:"db"`           // 单数据库配置，指向单个数据库实例
	Etcd         *EtcdInfo        `yaml:"etcd"`         // Etcd配置，用于服务发现和配置管理
	DbList       []DbInfo         `yaml:"dbList"`       // 多数据库列表配置，支持分库分表
	DbResolvers  DbResolvers      `yaml:"dbResolvers"`  // 数据库解析器配置，支持读写分离
	Redis        *RedisInfo       `yaml:"redis"`        // 单Redis配置，指向单个Redis实例
	RedisList    []RedisInfo      `yaml:"redisList"`    // 多Redis列表配置，支持多实例部署
	RabbitMQ     RabbitMQInfo     `yaml:"rabbitMQ"`     // RabbitMQ配置，用于消息队列
	RabbitMQList RabbitMqListInfo `yaml:"rabbitMQList"` // RabbitMQ列表配置，支持多实例部署
	Es           *EsInfo          `yaml:"es"`           // Elasticsearch配置，用于搜索引擎
	Smtp         SmtpInfo         `yaml:"smtp"`         // SMTP配置，用于邮件发送
}
