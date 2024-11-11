package config

type SystemInfo struct {
	GcTime   int  `yaml:"gcTime"`   // gc时间, 预留, 暂时没用上
	UseRedis bool `yaml:"useRedis"` // 是否启用redis
	UseMysql bool `yaml:"useMysql"` // 是否启用mysql
}
