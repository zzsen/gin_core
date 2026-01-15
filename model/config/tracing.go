// Package config 提供应用程序的配置结构定义
// 本文件定义了链路追踪相关的配置结构，支持 OpenTelemetry 标准的分布式追踪
package config

// TracingConfig 链路追踪配置结构
// 支持集成 OpenTelemetry，可导出到 Jaeger、OTLP、Zipkin 等后端
type TracingConfig struct {
	// Enabled 是否启用链路追踪
	// 设置为 true 时启用完整的分布式追踪功能
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	// ServiceName 服务名称
	// 用于在追踪系统中标识当前服务，建议使用有意义的名称如 "user-service"
	ServiceName string `mapstructure:"serviceName" yaml:"serviceName"`

	// ExporterType 导出器类型
	// 支持的类型：
	//   - "otlp": OpenTelemetry Protocol，推荐使用，支持 gRPC 传输
	//   - "jaeger": Jaeger 原生协议
	//   - "stdout": 标准输出，仅用于调试
	ExporterType string `mapstructure:"exporterType" yaml:"exporterType"`

	// Endpoint 采集器端点地址
	// 根据 ExporterType 不同，格式也不同：
	//   - otlp: "localhost:4317" (gRPC) 或 "localhost:4318" (HTTP)
	//   - jaeger: "http://localhost:14268/api/traces"
	Endpoint string `mapstructure:"endpoint" yaml:"endpoint"`

	// SampleRate 采样率
	// 范围 0.0 到 1.0，表示采样的比例
	//   - 1.0: 采样所有请求（开发环境推荐）
	//   - 0.1: 采样 10% 的请求（生产环境推荐）
	//   - 0.0: 不采样任何请求
	SampleRate float64 `mapstructure:"sampleRate" yaml:"sampleRate"`

	// Insecure 是否禁用 TLS
	// 设置为 true 时使用不安全连接，仅建议在开发环境使用
	Insecure bool `mapstructure:"insecure" yaml:"insecure"`

	// PropagatorType 上下文传播器类型
	// 支持的类型：
	//   - "tracecontext": W3C Trace Context 标准（默认，推荐）
	//   - "b3": Zipkin B3 格式
	//   - "jaeger": Jaeger 原生格式
	PropagatorType string `mapstructure:"propagatorType" yaml:"propagatorType"`

	// EnableDBTracing 是否启用数据库追踪
	// 设置为 true 时会追踪所有数据库操作（查询、插入、更新、删除）
	EnableDBTracing bool `mapstructure:"enableDBTracing" yaml:"enableDBTracing"`

	// EnableRedisTracing 是否启用 Redis 追踪
	// 设置为 true 时会追踪所有 Redis 操作
	EnableRedisTracing bool `mapstructure:"enableRedisTracing" yaml:"enableRedisTracing"`

	// EnableHTTPClientTracing 是否启用 HTTP 客户端追踪
	// 设置为 true 时会追踪所有出站 HTTP 请求
	EnableHTTPClientTracing bool `mapstructure:"enableHTTPClientTracing" yaml:"enableHTTPClientTracing"`
}
