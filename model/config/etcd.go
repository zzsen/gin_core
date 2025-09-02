// Package config 提供应用程序的配置结构定义
// 本文件定义了Etcd分布式键值存储的配置结构
package config

// EtcdInfo Etcd配置信息
// 该结构体包含了连接Etcd集群所需的基本配置参数
type EtcdInfo struct {
	Addresses []string `yaml:"addresses"` // Etcd集群节点地址列表，支持多节点配置
	Username  string   `yaml:"username"`  // Etcd访问用户名，用于身份认证
	Password  string   `yaml:"password"`  // Etcd访问密码，用于身份认证
	Timeout   *int     `yaml:"timeout"`   // 连接超时时间（秒），指针类型支持配置文件中不设置该字段
}
