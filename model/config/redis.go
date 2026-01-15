// Package config 提供应用程序的配置结构定义
// 本文件定义了Redis缓存数据库的配置结构，支持单实例和集群模式
package config

// RedisInfo Redis配置信息
// 该结构体包含了连接Redis数据库所需的基本配置参数，支持单实例和集群两种部署模式
type RedisInfo struct {
	AliasName    string   `yaml:"aliasName"`    // 代表当前实例的名字，用于多Redis实例环境下的标识
	Addr         string   `yaml:"addr"`         // 服务器地址:端口，单实例模式下的Redis服务器地址
	ClusterAddrs []string `yaml:"clusterAddrs"` // 集群模式下的节点地址列表，支持多节点Redis Cluster
	UseCluster   bool     `yaml:"useCluster"`   // 是否使用集群模式，true为集群模式，false为单实例模式
	DB           int      `yaml:"db"`           // 单实例模式下redis的哪个数据库，Redis支持0-15共16个数据库
	Password     string   `yaml:"password"`     // 密码，用于Redis身份认证，支持空密码

	// 连接池配置
	PoolSize     int `yaml:"poolSize"`     // 连接池大小，默认10
	MinIdleConns int `yaml:"minIdleConns"` // 最小空闲连接数，默认5
}
