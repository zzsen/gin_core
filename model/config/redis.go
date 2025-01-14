package config

type RedisInfo struct {
	AliasName    string   `yaml:"aliasName"`    // 代表当前实例的名字
	Addr         string   `yaml:"addr"`         // 服务器地址:端口
	ClusterAddrs []string `yaml:"clusterAddrs"` // 集群模式下的节点地址列表
	UseCluster   bool     `yaml:"useCluster"`   // 是否使用集群模式
	DB           int      `yaml:"db"`           // 单实例模式下redis的哪个数据库
	Password     string   `yaml:"password"`     // 密码
}
