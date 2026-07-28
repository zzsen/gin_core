// Package config 提供应用程序的配置结构定义
// 本文件定义 Etcd 客户端分层配置（非配置中心）
package config

// EtcdInfo Etcd 客户端配置信息（分层结构）
type EtcdInfo struct {
	Endpoints []string             `yaml:"endpoints"` // Etcd 集群节点地址列表
	Username  string               `yaml:"username"`  // 访问用户名
	Password  string               `yaml:"password"`  // 访问密码（勿写入日志）
	Required  *bool                `yaml:"required"`  // 连接失败是否阻断启动；nil/false 表示降级
	KeyPrefix string               `yaml:"keyPrefix"` // 非空时对 KV/Watcher/Lease 做 namespace 包装
	Dial      *EtcdDialConfig      `yaml:"dial"`      // 拨号与保活（秒）
	TLS       *EtcdTLSConfig       `yaml:"tls"`       // TLS 配置
	Health    *EtcdHealthConfig    `yaml:"health"`    // 健康检查策略
	Discovery *EtcdDiscoveryConfig `yaml:"discovery"` // 可选服务注册/发现（默认关闭）
}

// EtcdDiscoveryConfig Etcd 服务发现配置（opt-in）
type EtcdDiscoveryConfig struct {
	Enabled       bool              `yaml:"enabled"`       // 总开关，默认 false
	Register      *bool             `yaml:"register"`      // enabled 时是否自动注册本实例；nil 表示 true
	ServiceName   string            `yaml:"serviceName"`   // 服务名（enabled+register 时必填）
	Env           string            `yaml:"env"`           // 环境段；空则实现侧回退
	Prefix        string            `yaml:"prefix"`        // Key 前缀，叠在 etcd.keyPrefix 之上；默认 services/
	TTLSeconds    int               `yaml:"ttlSeconds"`    // Lease TTL（秒）
	InstanceID    string            `yaml:"instanceID"`    // 空则 {hostname}-{port}
	Weight        int               `yaml:"weight"`        // 负载权重，默认 1
	Meta          map[string]string `yaml:"meta"`          // 可选元数据
	AdvertiseIP   string            `yaml:"advertiseIP"`   // 空则取 service.ip
	AdvertisePort int               `yaml:"advertisePort"` // 0 则取 service.port
}

// IsRegister 是否自动注册；Register 为 nil 时默认 true
func (d *EtcdDiscoveryConfig) IsRegister() bool {
	if d == nil {
		return false
	}
	if d.Register == nil {
		return true
	}
	return *d.Register
}

// EffectivePrefix 返回 discovery Key 前缀；空则默认 services/
func (d *EtcdDiscoveryConfig) EffectivePrefix() string {
	if d == nil || d.Prefix == "" {
		return "services/"
	}
	return d.Prefix
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
