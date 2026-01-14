package lifecycle

import (
	"context"

	"github.com/zzsen/gin_core/model/config"
)

// Service 服务接口定义
// 所有需要在应用启动时初始化的服务都应该实现此接口
type Service interface {
	// Name 返回服务名称（唯一标识）
	Name() string

	// Priority 返回初始化优先级（数值越小越先初始化）
	// 用于在同一层级内排序
	Priority() int

	// Dependencies 返回依赖的服务名称列表
	// 依赖的服务会在当前服务之前初始化
	Dependencies() []string

	// ShouldInit 根据配置判断是否需要初始化
	ShouldInit(cfg *config.BaseConfig) bool

	// Init 执行初始化逻辑
	Init(ctx context.Context) error

	// Close 执行清理逻辑
	Close(ctx context.Context) error
}

// HealthChecker 健康检查接口（可选实现）
type HealthChecker interface {
	// HealthCheck 执行健康检查
	HealthCheck(ctx context.Context) error
}

// HookPhase 钩子执行阶段
type HookPhase int

const (
	BeforeInit  HookPhase = iota // 初始化之前
	AfterInit                    // 初始化之后
	BeforeClose                  // 关闭之前
	AfterClose                   // 关闭之后
)

// Hook 初始化钩子
type Hook struct {
	Phase    HookPhase                                           // 执行阶段
	Priority int                                                 // 执行优先级（数值越小越先执行）
	Fn       func(ctx context.Context, serviceName string) error // 钩子函数
}

// ServiceState 服务状态
type ServiceState int

const (
	StateUninitialized ServiceState = iota // 未初始化
	StateInitializing                      // 初始化中
	StateReady                             // 就绪
	StateFailed                            // 失败
	StateClosed                            // 已关闭
)

// String 返回状态的字符串表示
func (s ServiceState) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateInitializing:
		return "initializing"
	case StateReady:
		return "ready"
	case StateFailed:
		return "failed"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}
