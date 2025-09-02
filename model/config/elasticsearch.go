// Package config 提供应用程序的配置结构定义
// 本文件定义了Elasticsearch搜索引擎的配置结构
package config

// EsInfo Elasticsearch配置信息
// 该结构体包含了连接Elasticsearch集群所需的基本配置参数
type EsInfo struct {
	Addresses []string `yaml:"addresses"` // Elasticsearch集群节点地址列表，支持多节点配置
	Username  string   `yaml:"username"`  // Elasticsearch访问用户名，用于身份认证
	Password  string   `yaml:"password"`  // Elasticsearch访问密码，用于身份认证
}
