package circuitbreaker

import (
	"context"
	"sync"
)

// Registry 熔断器注册中心
// 按服务名管理熔断器实例，支持全局单例模式
type Registry struct {
	breakers      sync.Map
	configFactory func(name string) *Config
	mu            sync.RWMutex
}

var (
	defaultRegistry *Registry
	registryOnce    sync.Once
)

// GetRegistry 获取全局注册中心
// 使用单例模式，线程安全
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		defaultRegistry = &Registry{
			configFactory: DefaultConfig,
		}
	})
	return defaultRegistry
}

// NewRegistry 创建新的注册中心
// 参数：
//   - configFactory: 配置工厂函数，用于为新服务创建配置
//
// 返回：
//   - *Registry: 注册中心实例
func NewRegistry(configFactory func(name string) *Config) *Registry {
	if configFactory == nil {
		configFactory = DefaultConfig
	}
	return &Registry{
		configFactory: configFactory,
	}
}

// SetConfigFactory 设置配置工厂函数
// 参数：
//   - fn: 配置工厂函数
func (r *Registry) SetConfigFactory(fn func(name string) *Config) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configFactory = fn
}

// Get 获取或创建熔断器
// 如果指定名称的熔断器已存在，则返回已有实例
// 否则使用配置工厂创建新实例
// 参数：
//   - name: 熔断器名称（通常是服务名）
//
// 返回：
//   - *CircuitBreaker: 熔断器实例
func (r *Registry) Get(name string) *CircuitBreaker {
	if cb, ok := r.breakers.Load(name); ok {
		return cb.(*CircuitBreaker)
	}

	r.mu.RLock()
	factory := r.configFactory
	r.mu.RUnlock()

	cb := New(factory(name))
	actual, _ := r.breakers.LoadOrStore(name, cb)
	return actual.(*CircuitBreaker)
}

// GetWithConfig 使用指定配置获取或创建熔断器
// 参数：
//   - config: 熔断器配置
//
// 返回：
//   - *CircuitBreaker: 熔断器实例
func (r *Registry) GetWithConfig(config *Config) *CircuitBreaker {
	if cb, ok := r.breakers.Load(config.Name); ok {
		return cb.(*CircuitBreaker)
	}

	cb := New(config)
	actual, _ := r.breakers.LoadOrStore(config.Name, cb)
	return actual.(*CircuitBreaker)
}

// Register 注册熔断器
// 参数：
//   - cb: 熔断器实例
func (r *Registry) Register(cb *CircuitBreaker) {
	r.breakers.Store(cb.Name(), cb)
}

// Remove 移除熔断器
// 参数：
//   - name: 熔断器名称
func (r *Registry) Remove(name string) {
	r.breakers.Delete(name)
}

// Reset 重置指定熔断器
// 参数：
//   - name: 熔断器名称
func (r *Registry) Reset(name string) {
	if cb, ok := r.breakers.Load(name); ok {
		cb.(*CircuitBreaker).Reset()
	}
}

// ResetAll 重置所有熔断器
func (r *Registry) ResetAll() {
	r.breakers.Range(func(key, value interface{}) bool {
		value.(*CircuitBreaker).Reset()
		return true
	})
}

// List 列出所有熔断器名称
// 返回：
//   - []string: 熔断器名称列表
func (r *Registry) List() []string {
	var names []string
	r.breakers.Range(func(key, value interface{}) bool {
		names = append(names, key.(string))
		return true
	})
	return names
}

// Stats 获取所有熔断器的状态统计
// 返回：
//   - map[string]BreakerStats: 熔断器名称到状态的映射
func (r *Registry) Stats() map[string]BreakerStats {
	stats := make(map[string]BreakerStats)
	r.breakers.Range(func(key, value interface{}) bool {
		cb := value.(*CircuitBreaker)
		stats[key.(string)] = BreakerStats{
			State:  cb.State(),
			Counts: cb.Counts(),
		}
		return true
	})
	return stats
}

// BreakerStats 熔断器状态统计
type BreakerStats struct {
	State  State
	Counts Counts
}

// ==================== 便捷函数 ====================

// Execute 使用全局注册中心执行受熔断器保护的函数
// 参数：
//   - ctx: 上下文
//   - serviceName: 服务名称
//   - fn: 要执行的函数
//
// 返回：
//   - error: 执行错误或熔断错误
//
// 使用示例：
//
//	err := circuitbreaker.Execute(ctx, "user-service", func() error {
//	    return callUserService()
//	})
//	if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
//	    // 熔断器打开，执行降级逻辑
//	    return getCachedData()
//	}
func Execute(ctx context.Context, serviceName string, fn func() error) error {
	return GetRegistry().Get(serviceName).Execute(ctx, fn)
}

// GetBreaker 从全局注册中心获取熔断器
// 参数：
//   - serviceName: 服务名称
//
// 返回：
//   - *CircuitBreaker: 熔断器实例
func GetBreaker(serviceName string) *CircuitBreaker {
	return GetRegistry().Get(serviceName)
}

// ResetBreaker 重置全局注册中心中的熔断器
// 参数：
//   - serviceName: 服务名称
func ResetBreaker(serviceName string) {
	GetRegistry().Reset(serviceName)
}
