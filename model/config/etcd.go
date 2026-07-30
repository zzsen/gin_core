// Package config 提供应用程序的配置结构定义
// 本文件定义 Etcd 客户端分层配置（含可选配置中心开关；连接语义仍属客户端基建）
package config

// 默认配置中心 key 与白名单
const (
	DefaultConfigCenterPrefix     = "config/app.yml"
	defaultConfigCenterDebounceMs = 300
)

// defaultConfigCenterWhitelist 热更默认白名单
var defaultConfigCenterWhitelist = []string{"log.level", "rateLimit.*"}

// EtcdInfo Etcd 客户端配置信息（分层结构）
type EtcdInfo struct {
	Endpoints    []string                `yaml:"endpoints"`    // Etcd 集群节点地址列表
	Username     string                  `yaml:"username"`     // 访问用户名
	Password     string                  `yaml:"password"`     // 访问密码（勿写入日志）
	Required     *bool                   `yaml:"required"`     // 连接失败是否阻断启动；nil/false 表示降级
	KeyPrefix    string                  `yaml:"keyPrefix"`    // 非空时对 KV/Watcher/Lease 做 namespace 包装
	Dial         *EtcdDialConfig         `yaml:"dial"`         // 拨号与保活（秒）
	TLS          *EtcdTLSConfig          `yaml:"tls"`          // TLS 配置
	Health       *EtcdHealthConfig       `yaml:"health"`       // 健康检查策略
	Discovery    *EtcdDiscoveryConfig    `yaml:"discovery"`    // 可选服务注册/发现（默认关闭）
	ConfigCenter *EtcdConfigCenterConfig `yaml:"configCenter"` // 可选配置中心 overlay/热更（默认关闭）
}

// EtcdConfigCenterConfig Etcd 配置中心（opt-in overlay + Watch）
type EtcdConfigCenterConfig struct {
	Enabled    bool     `yaml:"enabled"`    // 总开关，默认 false
	Prefix     string   `yaml:"prefix"`     // Etcd key（整包 YAML）；空则默认 config/app.yml
	Required   *bool    `yaml:"required"`   // Get/解析失败是否阻断；nil/false 表示 warn 跳过
	DebounceMs int      `yaml:"debounceMs"` // Watch 防抖毫秒；<=0 默认 300
	Watch      *bool    `yaml:"watch"`      // 是否 Watch；nil 表示 enabled 时默认 true
	Whitelist  []string `yaml:"whitelist"`  // 热更白名单；空则用内置默认
}

// EtcdDiscoveryConfig Etcd 服务发现配置（opt-in）
type EtcdDiscoveryConfig struct {
	Enabled       bool                          `yaml:"enabled"`       // 总开关，默认 false
	Register      *bool                         `yaml:"register"`      // enabled 时是否自动注册本实例；nil 表示 true
	ServiceName   string                        `yaml:"serviceName"`   // 服务名（enabled+register 时必填）
	Env           string                        `yaml:"env"`           // 环境段；空则实现侧回退
	Prefix        string                        `yaml:"prefix"`        // Key 前缀，叠在 etcd.keyPrefix 之上；默认 services/
	TTLSeconds    int                           `yaml:"ttlSeconds"`    // Lease TTL（秒）
	InstanceID    string                        `yaml:"instanceID"`    // 空则 {hostname}-{port}
	Weight        int                           `yaml:"weight"`        // 负载权重，默认 1
	Meta          map[string]string             `yaml:"meta"`          // 可选元数据
	AdvertiseIP   string                        `yaml:"advertiseIP"`   // 空则取 service.ip
	AdvertisePort int                           `yaml:"advertisePort"` // 0 则取 service.port
	Keepalive     *EtcdDiscoveryKeepaliveConfig `yaml:"keepalive"`     // KeepAlive 重建
	Health        *EtcdDiscoveryHealthConfig    `yaml:"health"`        // 就绪联动摘除（默认关闭）
}

// EtcdDiscoveryKeepaliveConfig 注册租约续约与重建
type EtcdDiscoveryKeepaliveConfig struct {
	Rebuild           *bool `yaml:"rebuild"`           // 断流后是否重建；nil 表示 true
	MaxBackoffSeconds int   `yaml:"maxBackoffSeconds"` // 退避上限（秒）；<=0 默认 30
}

// EtcdDiscoveryHealthConfig 基于 ready 的两阶段健康摘除
type EtcdDiscoveryHealthConfig struct {
	Unlink           bool `yaml:"unlink"`           // 是否启用；默认 false
	IntervalSeconds  int  `yaml:"intervalSeconds"`  // 探测间隔；<=0 默认 5
	FailThreshold    int  `yaml:"failThreshold"`    // 连续失败进入/维持降级；<=0 默认 3
	SuccessThreshold int  `yaml:"successThreshold"` // 连续成功恢复；<=0 默认 2
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

// KeepaliveRebuild KeepAlive 断流后是否重建；nil/未配置默认 true
func (d *EtcdDiscoveryConfig) KeepaliveRebuild() bool {
	if d == nil || d.Keepalive == nil || d.Keepalive.Rebuild == nil {
		return true
	}
	return *d.Keepalive.Rebuild
}

// KeepaliveMaxBackoffSeconds 重建退避上限（秒）
func (d *EtcdDiscoveryConfig) KeepaliveMaxBackoffSeconds() int {
	if d == nil || d.Keepalive == nil || d.Keepalive.MaxBackoffSeconds <= 0 {
		return 30
	}
	return d.Keepalive.MaxBackoffSeconds
}

// HealthUnlink 是否启用 ready 联动摘除；默认 false
func (d *EtcdDiscoveryConfig) HealthUnlink() bool {
	if d == nil || d.Health == nil {
		return false
	}
	return d.Health.Unlink
}

// HealthIntervalSeconds 健康探测间隔（秒）
func (d *EtcdDiscoveryConfig) HealthIntervalSeconds() int {
	if d == nil || d.Health == nil || d.Health.IntervalSeconds <= 0 {
		return 5
	}
	return d.Health.IntervalSeconds
}

// HealthFailThreshold 连续失败阈值
func (d *EtcdDiscoveryConfig) HealthFailThreshold() int {
	if d == nil || d.Health == nil || d.Health.FailThreshold <= 0 {
		return 3
	}
	return d.Health.FailThreshold
}

// HealthSuccessThreshold 连续成功恢复阈值
func (d *EtcdDiscoveryConfig) HealthSuccessThreshold() int {
	if d == nil || d.Health == nil || d.Health.SuccessThreshold <= 0 {
		return 2
	}
	return d.Health.SuccessThreshold
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

// ConfigCenterEnabled 是否启用配置中心；未配置默认 false
func (e *EtcdInfo) ConfigCenterEnabled() bool {
	if e == nil || e.ConfigCenter == nil {
		return false
	}
	return e.ConfigCenter.Enabled
}

// ConfigCenterPrefix 返回 overlay key；空则默认 config/app.yml
func (e *EtcdInfo) ConfigCenterPrefix() string {
	if e == nil || e.ConfigCenter == nil || e.ConfigCenter.Prefix == "" {
		return DefaultConfigCenterPrefix
	}
	return e.ConfigCenter.Prefix
}

// ConfigCenterRequired overlay 失败是否阻断启动；nil/false 默认不阻断
func (e *EtcdInfo) ConfigCenterRequired() bool {
	if e == nil || e.ConfigCenter == nil || e.ConfigCenter.Required == nil {
		return false
	}
	return *e.ConfigCenter.Required
}

// ConfigCenterDebounceMs Watch 防抖毫秒；<=0 默认 300
func (e *EtcdInfo) ConfigCenterDebounceMs() int {
	if e == nil || e.ConfigCenter == nil || e.ConfigCenter.DebounceMs <= 0 {
		return defaultConfigCenterDebounceMs
	}
	return e.ConfigCenter.DebounceMs
}

// ConfigCenterWatch 是否启动 Watch；nil 时默认 true（即便未 enabled，查询语义为「若启用则 Watch」）
func (e *EtcdInfo) ConfigCenterWatch() bool {
	if e == nil || e.ConfigCenter == nil || e.ConfigCenter.Watch == nil {
		return true
	}
	return *e.ConfigCenter.Watch
}

// ConfigCenterWhitelist 热更白名单；空则返回内置默认副本
func (e *EtcdInfo) ConfigCenterWhitelist() []string {
	if e == nil || e.ConfigCenter == nil || len(e.ConfigCenter.Whitelist) == 0 {
		out := make([]string, len(defaultConfigCenterWhitelist))
		copy(out, defaultConfigCenterWhitelist)
		return out
	}
	out := make([]string, len(e.ConfigCenter.Whitelist))
	copy(out, e.ConfigCenter.Whitelist)
	return out
}
