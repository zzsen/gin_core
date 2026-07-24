// Package config 提供应用程序的配置结构定义
// 本文件定义 Etcd 客户端分层配置（非配置中心）
package config

// EtcdInfo Etcd 客户端配置信息（分层结构）
type EtcdInfo struct {
	Endpoints []string          `yaml:"endpoints"` // Etcd 集群节点地址列表
	Username  string            `yaml:"username"`  // 访问用户名
	Password  string            `yaml:"password"`  // 访问密码（勿写入日志）
	Required  *bool             `yaml:"required"`  // 连接失败是否阻断启动；nil/false 表示降级
	KeyPrefix string            `yaml:"keyPrefix"` // 非空时对 KV/Watcher/Lease 做 namespace 包装
	Dial      *EtcdDialConfig   `yaml:"dial"`      // 拨号与保活（秒）
	TLS       *EtcdTLSConfig    `yaml:"tls"`       // TLS 配置
	Health    *EtcdHealthConfig `yaml:"health"`    // 健康检查策略
}

// EtcdDialConfig Etcd 拨号相关配置（时间单位：秒）
type EtcdDialConfig struct {
	Timeout             *int  `yaml:"timeout"`             // 拨号超时（秒），未设或 0 使用默认
	KeepAliveTime       *int  `yaml:"keepAliveTime"`       // KeepAlive 时间（秒）
	KeepAliveTimeout    *int  `yaml:"keepAliveTimeout"`    // KeepAlive 超时（秒）
	AutoSyncInterval    *int  `yaml:"autoSyncInterval"`    // 自动同步间隔（秒），0 表示关闭
	PermitWithoutStream *bool `yaml:"permitWithoutStream"` // 允许无 stream 时 KeepAlive
}

// EtcdTLSConfig Etcd TLS 配置
type EtcdTLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CAFile             string `yaml:"caFile"`
	CertFile           string `yaml:"certFile"`
	KeyFile            string `yaml:"keyFile"`
	InsecureSkipVerify bool   `yaml:"insecureSkipVerify"`
}

// EtcdHealthConfig Etcd 健康检查配置
type EtcdHealthConfig struct {
	Strategy string `yaml:"strategy"` // any | all；默认 any
}

// IsRequired 连接失败是否阻断启动；Required 为 nil 时默认 false
func (e *EtcdInfo) IsRequired() bool {
	if e == nil || e.Required == nil {
		return false
	}
	return *e.Required
}

// HealthStrategy 返回探活策略；空或 Health 为 nil 时默认 any
func (e *EtcdInfo) HealthStrategy() string {
	if e == nil || e.Health == nil || e.Health.Strategy == "" {
		return "any"
	}
	return e.Health.Strategy
}
