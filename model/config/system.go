package config

type SystemInfo struct {
	GcTime      int  `yaml:"gcTime"`      // gc时间, 预留, 暂时没用上
	UseRedis    bool `yaml:"useRedis"`    // 是否启用redis
	UseMysql    bool `yaml:"useMysql"`    // 是否启用mysql
	UseEs       bool `yaml:"useEs"`       // 是否启用elasticsearch
	UseEtcd     bool `yaml:"useEtcd"`     // 是否启用etcd
	UseRabbitMQ bool `yaml:"useRabbitMQ"` // 是否启用rabbitmq
	UseSchedule bool `yaml:"useSchedule"` // 是否启用定时任务
}
