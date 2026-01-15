// Package initialize 提供各种服务的初始化功能
// 本文件专门负责 OpenTelemetry 链路追踪的初始化和关闭
package initialize

import (
	"context"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/tracing"
)

// tracerShutdown 追踪器关闭函数
var tracerShutdown func(context.Context) error

// InitTracing 初始化 OpenTelemetry 链路追踪
// 该函数会：
// 1. 检查链路追踪配置是否存在
// 2. 初始化 OpenTelemetry TracerProvider
// 3. 配置全局追踪器和上下文传播器
func InitTracing() {
	if app.BaseConfig.Tracing == nil {
		logger.Info("[tracing] 未配置链路追踪，跳过初始化")
		return
	}

	if !app.BaseConfig.Tracing.Enabled {
		logger.Info("[tracing] 链路追踪已禁用")
		return
	}

	shutdown, err := tracing.InitTracer(app.BaseConfig.Tracing)
	if err != nil {
		logger.Error("[tracing] 初始化失败: %v", err)
		return
	}

	tracerShutdown = shutdown

	logger.Info("[tracing] 链路追踪已初始化, 服务名: %s, 导出器: %s, 端点: %s, 采样率: %.2f",
		app.BaseConfig.Tracing.ServiceName,
		app.BaseConfig.Tracing.ExporterType,
		app.BaseConfig.Tracing.Endpoint,
		app.BaseConfig.Tracing.SampleRate,
	)
}

// ShutdownTracing 关闭 OpenTelemetry 链路追踪
// 该函数会：
// 1. 检查追踪器是否已初始化
// 2. 使用超时上下文优雅关闭追踪器
// 3. 确保所有待发送的追踪数据被刷新
func ShutdownTracing() {
	if tracerShutdown == nil {
		return
	}

	// 创建带超时的上下文，确保关闭操作不会无限阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := tracerShutdown(ctx); err != nil {
		logger.Error("[tracing] 关闭失败: %v", err)
		return
	}

	logger.Info("[tracing] 链路追踪已关闭")
}

// IsTracingEnabled 返回链路追踪是否已启用
// 用于其他模块判断是否需要添加追踪功能
func IsTracingEnabled() bool {
	return tracing.IsEnabled()
}
