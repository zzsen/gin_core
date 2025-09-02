// Package config 提供应用程序的配置结构定义
// 本文件定义了系统级别的配置结构，用于控制各个功能组件是否启用
package config

// SystemInfo 系统级别配置信息
// 该结构体包含了控制应用程序各个功能组件是否启用的开关配置
type SystemInfo struct {
	GcTime      int  `yaml:"gcTime"`      // GC时间，预留配置项，暂时没用上，用于未来垃圾回收调优
	UseRedis    bool `yaml:"useRedis"`    // 是否启用Redis缓存服务，控制Redis相关功能的可用性
	UseMysql    bool `yaml:"useMysql"`    // 是否启用MySQL数据库服务，控制数据库相关功能的可用性
	UseEs       bool `yaml:"useEs"`       // 是否启用Elasticsearch搜索引擎，控制搜索相关功能的可用性
	UseEtcd     bool `yaml:"useEtcd"`     // 是否启用Etcd分布式键值存储，控制服务发现和配置管理功能
	UseRabbitMQ bool `yaml:"useRabbitMQ"` // 是否启用RabbitMQ消息队列，控制异步消息处理功能
	UseSchedule bool `yaml:"useSchedule"` // 是否启用定时任务功能，控制定时任务调度器的可用性
}
