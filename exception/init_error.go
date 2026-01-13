package exception

import (
	"fmt"
)

// InitError 初始化错误类型
// 用于初始化阶段的错误，提供更详细的错误上下文信息
// 初始化阶段的错误通常会导致服务无法启动，因此使用 panic 是合理的
// 但通过统一的错误类型，可以提供更好的错误追踪和诊断信息
type InitError struct {
	Service   string // 服务名称，如 "db", "redis", "es" 等
	Operation string // 操作名称，如 "初始化连接", "创建客户端" 等
	Config    string // 配置信息，如数据库别名、Redis别名等
	Err       error  // 底层错误
}

// Error 实现 error 接口
func (e *InitError) Error() string {
	if e.Config != "" {
		return fmt.Sprintf("[%s] %s失败 [%s]: %v", e.Service, e.Operation, e.Config, e.Err)
	}
	return fmt.Sprintf("[%s] %s失败: %v", e.Service, e.Operation, e.Err)
}

// Unwrap 返回底层错误，支持 errors.Unwrap
func (e *InitError) Unwrap() error {
	return e.Err
}

// NewInitError 创建初始化错误
// 参数：
//   - service: 服务名称
//   - operation: 操作名称
//   - err: 底层错误
//
// 返回：
//   - *InitError: 初始化错误实例
func NewInitError(service, operation string, err error) *InitError {
	return &InitError{
		Service:   service,
		Operation: operation,
		Err:       err,
	}
}

// NewInitErrorWithConfig 创建带配置信息的初始化错误
// 参数：
//   - service: 服务名称
//   - operation: 操作名称
//   - config: 配置信息（如别名等）
//   - err: 底层错误
//
// 返回：
//   - *InitError: 初始化错误实例
func NewInitErrorWithConfig(service, operation, config string, err error) *InitError {
	return &InitError{
		Service:   service,
		Operation: operation,
		Config:    config,
		Err:       err,
	}
}
