package core

import (
	"context"

	"github.com/zzsen/gin_core/core/lifecycle"
)

// RegisterAppHook 注册应用级生命周期钩子
// 支持完整的钩子配置（阶段、优先级、名称、执行函数）
func RegisterAppHook(hook lifecycle.AppHook) {
	lifecycle.RegisterAppHook(hook)
}

// OnBeforeInit 注册应用初始化前钩子
// 在 loadConfig 之后、initService 之前执行
func OnBeforeInit(fn func(ctx context.Context) error) {
	lifecycle.RegisterAppHook(lifecycle.AppHook{
		Phase: lifecycle.AppBeforeInit,
		Fn:    fn,
	})
}

// OnAfterInit 注册应用初始化后钩子
// 在所有服务初始化完成、HTTP 监听之前执行
func OnAfterInit(fn func(ctx context.Context) error) {
	lifecycle.RegisterAppHook(lifecycle.AppHook{
		Phase: lifecycle.AppAfterInit,
		Fn:    fn,
	})
}

// OnReady 注册服务就绪钩子
// 在 HTTP 服务开始监听后触发，适用于缓存预热等逻辑
func OnReady(fn func(ctx context.Context) error) {
	lifecycle.RegisterAppHook(lifecycle.AppHook{
		Phase: lifecycle.AppOnReady,
		Fn:    fn,
	})
}

// OnBeforeShutdown 注册应用关闭前钩子
// 在收到关闭信号后、CloseServices 之前执行
// 替代原 core.Start(exitfunctions ...) 的退出回调机制
func OnBeforeShutdown(fn func(ctx context.Context) error) {
	lifecycle.RegisterAppHook(lifecycle.AppHook{
		Phase: lifecycle.AppBeforeShutdown,
		Fn:    fn,
	})
}

// OnAfterShutdown 注册应用关闭后钩子
// 在所有服务关闭完成、进程退出前执行
func OnAfterShutdown(fn func(ctx context.Context) error) {
	lifecycle.RegisterAppHook(lifecycle.AppHook{
		Phase: lifecycle.AppAfterShutdown,
		Fn:    fn,
	})
}
