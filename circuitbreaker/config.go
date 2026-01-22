package circuitbreaker

import "time"

// Config 熔断器配置
type Config struct {
	// Name 熔断器名称（通常是服务名）
	Name string

	// MaxRequests 半开状态下允许通过的最大请求数
	// 用于探测服务是否恢复
	// 默认值：3
	MaxRequests uint32

	// Interval 统计周期
	// 在此周期内统计失败次数，周期结束后重置计数器
	// 默认值：60秒
	Interval time.Duration

	// Timeout 熔断器打开后，等待多久进入半开状态
	// 默认值：30秒
	Timeout time.Duration

	// FailureThreshold 触发熔断的连续失败次数
	// 当连续失败次数达到此阈值时，熔断器打开
	// 默认值：5
	FailureThreshold uint32

	// FailureRatio 触发熔断的失败率（0.0-1.0）
	// 当请求数超过 MinRequests 且失败率达到此阈值时，熔断器打开
	// 默认值：0.5（50%）
	FailureRatio float64

	// MinRequests 计算失败率的最小请求数
	// 只有当请求数达到此值时，才会根据失败率判断是否熔断
	// 默认值：10
	MinRequests uint32

	// OnStateChange 状态变更回调函数
	// 当熔断器状态发生变化时调用
	OnStateChange func(name string, from, to State)
}

// DefaultConfig 返回默认配置
// 参数：
//   - name: 熔断器名称
//
// 返回：
//   - *Config: 默认配置
func DefaultConfig(name string) *Config {
	return &Config{
		Name:             name,
		MaxRequests:      3,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
		FailureRatio:     0.5,
		MinRequests:      10,
	}
}

// ConfigOption 配置选项函数类型
type ConfigOption func(*Config)

// WithMaxRequests 设置半开状态最大请求数
func WithMaxRequests(n uint32) ConfigOption {
	return func(c *Config) {
		c.MaxRequests = n
	}
}

// WithInterval 设置统计周期
func WithInterval(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.Interval = d
	}
}

// WithTimeout 设置熔断超时时间
func WithTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = d
	}
}

// WithFailureThreshold 设置连续失败阈值
func WithFailureThreshold(n uint32) ConfigOption {
	return func(c *Config) {
		c.FailureThreshold = n
	}
}

// WithFailureRatio 设置失败率阈值
func WithFailureRatio(ratio float64) ConfigOption {
	return func(c *Config) {
		c.FailureRatio = ratio
	}
}

// WithMinRequests 设置最小请求数
func WithMinRequests(n uint32) ConfigOption {
	return func(c *Config) {
		c.MinRequests = n
	}
}

// WithOnStateChange 设置状态变更回调
func WithOnStateChange(fn func(name string, from, to State)) ConfigOption {
	return func(c *Config) {
		c.OnStateChange = fn
	}
}

// NewConfig 使用选项模式创建配置
// 参数：
//   - name: 熔断器名称
//   - opts: 配置选项
//
// 返回：
//   - *Config: 配置实例
//
// 使用示例：
//
//	config := NewConfig("user-service",
//	    WithFailureThreshold(3),
//	    WithTimeout(10 * time.Second),
//	)
func NewConfig(name string, opts ...ConfigOption) *Config {
	config := DefaultConfig(name)
	for _, opt := range opts {
		opt(config)
	}
	return config
}
