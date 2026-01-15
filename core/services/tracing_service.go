// Package services 提供框架内置服务的实现
// 本文件实现了 OpenTelemetry 链路追踪服务
package services

import (
	"context"

	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/model/config"
)

// TracingService 链路追踪服务
// 实现了 core.Service 接口，用于管理 OpenTelemetry 追踪器的生命周期
type TracingService struct{}

// Name 返回服务名称
func (s *TracingService) Name() string { return "tracing" }

// Priority 返回初始化优先级
// 追踪服务需要在日志之后、其他服务之前初始化
// 这样可以确保其他服务的初始化过程也能被追踪
func (s *TracingService) Priority() int { return 5 }

// Dependencies 返回依赖
// 追踪服务依赖日志服务，用于记录初始化状态
func (s *TracingService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 判断是否需要初始化
// 只有在配置了链路追踪且启用时才初始化
func (s *TracingService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.Tracing != nil && cfg.Tracing.Enabled
}

// Init 初始化链路追踪
func (s *TracingService) Init(ctx context.Context) error {
	initialize.InitTracing()
	return nil
}

// Close 关闭链路追踪
// 确保所有待发送的追踪数据被刷新到采集器
func (s *TracingService) Close(ctx context.Context) error {
	initialize.ShutdownTracing()
	return nil
}
