// Package config 提供应用程序的配置结构定义
// 本文件定义了限流相关的配置结构
package config

// RateLimitConfig 限流配置
// 用于控制 API 请求速率，防止服务过载
type RateLimitConfig struct {
	// Enabled 是否启用限流
	Enabled bool `yaml:"enabled"`
	// DefaultRate 默认每秒请求数
	DefaultRate int `yaml:"defaultRate"`
	// DefaultBurst 默认突发容量（令牌桶大小）
	DefaultBurst int `yaml:"defaultBurst"`
	// Store 存储类型: memory（单机）/ redis（分布式）
	Store string `yaml:"store"`
	// Rules 自定义限流规则列表
	Rules []RateLimitRule `yaml:"rules"`
	// Message 默认限流提示消息
	Message string `yaml:"message"`
	// CleanupInterval 内存限流器清理过期条目的间隔（秒），默认 60
	CleanupInterval int `yaml:"cleanupInterval"`
}

// RateLimitRule 限流规则
// 定义特定路径或接口的限流策略
type RateLimitRule struct {
	// Path 路径匹配，支持通配符（如 /api/*）
	Path string `yaml:"path"`
	// Method HTTP 方法，空表示所有方法
	Method string `yaml:"method"`
	// Rate 每秒请求数
	Rate int `yaml:"rate"`
	// Burst 突发容量
	Burst int `yaml:"burst"`
	// KeyType 限流维度: ip / user / global
	// - ip: 按客户端 IP 限流
	// - user: 按用户 ID 限流（需要认证）
	// - global: 全局限流（所有请求共享配额）
	KeyType string `yaml:"keyType"`
	// Message 自定义限流提示消息
	Message string `yaml:"message"`
}

// GetDefaultRate 获取默认速率，如果未配置则返回 100
func (c *RateLimitConfig) GetDefaultRate() int {
	if c.DefaultRate <= 0 {
		return 100
	}
	return c.DefaultRate
}

// GetDefaultBurst 获取默认突发容量，如果未配置则返回速率的 2 倍
func (c *RateLimitConfig) GetDefaultBurst() int {
	if c.DefaultBurst <= 0 {
		return c.GetDefaultRate() * 2
	}
	return c.DefaultBurst
}

// GetStore 获取存储类型，默认为 memory
func (c *RateLimitConfig) GetStore() string {
	if c.Store == "" {
		return "memory"
	}
	return c.Store
}

// GetMessage 获取默认限流消息
func (c *RateLimitConfig) GetMessage() string {
	if c.Message == "" {
		return "请求过于频繁，请稍后再试"
	}
	return c.Message
}

// GetCleanupInterval 获取清理间隔（秒）
func (c *RateLimitConfig) GetCleanupInterval() int {
	if c.CleanupInterval <= 0 {
		return 60
	}
	return c.CleanupInterval
}

// GetRate 获取规则速率，如果未配置则返回 0（使用默认值）
func (r *RateLimitRule) GetRate() int {
	if r.Rate <= 0 {
		return 0
	}
	return r.Rate
}

// GetBurst 获取规则突发容量，如果未配置则返回速率的 2 倍
func (r *RateLimitRule) GetBurst() int {
	if r.Burst <= 0 {
		rate := r.GetRate()
		if rate > 0 {
			return rate * 2
		}
		return 0
	}
	return r.Burst
}

// GetKeyType 获取限流维度，默认为 ip
func (r *RateLimitRule) GetKeyType() string {
	if r.KeyType == "" {
		return "ip"
	}
	return r.KeyType
}
