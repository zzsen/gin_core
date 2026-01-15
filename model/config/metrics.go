// Package config 提供应用程序的配置结构定义
package config

// MetricsConfig Prometheus 指标监控配置
type MetricsConfig struct {
	Enabled      bool     `yaml:"enabled"`      // 是否启用指标监控
	Path         string   `yaml:"path"`         // 指标端点路径，默认 /metrics
	ExcludePaths []string `yaml:"excludePaths"` // 不统计的路径列表
}
