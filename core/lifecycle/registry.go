package lifecycle

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// ServiceRegistry 服务注册中心
// 管理所有服务的注册、初始化和关闭
type ServiceRegistry struct {
	services map[string]Service      // 已注册的服务
	hooks    map[string][]Hook       // 服务钩子
	states   map[string]ServiceState // 服务状态
	mu       sync.RWMutex            // 读写锁
}

// 全局服务注册中心实例
var globalRegistry = NewServiceRegistry()

// NewServiceRegistry 创建新的服务注册中心
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]Service),
		hooks:    make(map[string][]Hook),
		states:   make(map[string]ServiceState),
	}
}

// Register 注册服务
// 参数：
//   - service: 要注册的服务
//
// 返回：
//   - error: 如果服务名称已存在则返回错误
func (r *ServiceRegistry) Register(service Service) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := service.Name()
	if _, exists := r.services[name]; exists {
		return fmt.Errorf("服务 '%s' 已注册", name)
	}

	r.services[name] = service
	r.states[name] = StateUninitialized
	return nil
}

// RegisterHook 注册钩子
// 参数：
//   - serviceName: 服务名称
//   - hook: 钩子配置
func (r *ServiceRegistry) RegisterHook(serviceName string, hook Hook) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks[serviceName] = append(r.hooks[serviceName], hook)
}

// GetService 获取服务
func (r *ServiceRegistry) GetService(name string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	return service, exists
}

// GetState 获取服务状态
func (r *ServiceRegistry) GetState(name string) ServiceState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	state, exists := r.states[name]
	if !exists {
		return StateUninitialized
	}
	return state
}

// SetState 设置服务状态
func (r *ServiceRegistry) SetState(name string, state ServiceState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.states[name] = state
}

// GetAllServices 获取所有服务
func (r *ServiceRegistry) GetAllServices() map[string]Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Service)
	for k, v := range r.services {
		result[k] = v
	}
	return result
}

// GetServicesToInit 获取需要初始化的服务列表
func (r *ServiceRegistry) GetServicesToInit(cfg *config.BaseConfig) []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Service
	for _, service := range r.services {
		if service.ShouldInit(cfg) {
			result = append(result, service)
		}
	}
	return result
}

// ExecuteHooks 执行指定阶段的钩子
func (r *ServiceRegistry) ExecuteHooks(ctx context.Context, serviceName string, phase HookPhase) error {
	r.mu.RLock()
	hooks := r.hooks[serviceName]
	r.mu.RUnlock()

	// 按优先级排序
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].Priority < hooks[j].Priority
	})

	// 执行匹配阶段的钩子
	for _, hook := range hooks {
		if hook.Phase == phase {
			if err := hook.Fn(ctx, serviceName); err != nil {
				return fmt.Errorf("执行钩子失败 [%s, phase=%d]: %w", serviceName, phase, err)
			}
		}
	}
	return nil
}

// InitService 初始化单个服务
func (r *ServiceRegistry) InitService(ctx context.Context, name string) error {
	service, exists := r.GetService(name)
	if !exists {
		return fmt.Errorf("服务 '%s' 未注册", name)
	}

	// 检查状态
	state := r.GetState(name)
	if state == StateReady {
		return nil // 已初始化
	}
	if state == StateInitializing {
		return fmt.Errorf("服务 '%s' 正在初始化中", name)
	}

	// 设置为初始化中
	r.SetState(name, StateInitializing)

	// 执行初始化前钩子
	if err := r.ExecuteHooks(ctx, name, BeforeInit); err != nil {
		r.SetState(name, StateFailed)
		return err
	}

	// 执行初始化
	logger.Info("[服务初始化] 正在初始化服务: %s", name)
	if err := service.Init(ctx); err != nil {
		r.SetState(name, StateFailed)
		logger.Error("[服务初始化] 服务 %s 初始化失败: %v", name, err)
		return err
	}

	// 执行初始化后钩子
	if err := r.ExecuteHooks(ctx, name, AfterInit); err != nil {
		r.SetState(name, StateFailed)
		return err
	}

	// 设置为就绪
	r.SetState(name, StateReady)
	logger.Info("[服务初始化] 服务 %s 初始化成功", name)
	return nil
}

// CloseService 关闭单个服务
func (r *ServiceRegistry) CloseService(ctx context.Context, name string) error {
	service, exists := r.GetService(name)
	if !exists {
		return nil
	}

	state := r.GetState(name)
	if state != StateReady {
		return nil // 未初始化或已关闭
	}

	// 执行关闭前钩子
	if err := r.ExecuteHooks(ctx, name, BeforeClose); err != nil {
		logger.Error("[服务关闭] 执行关闭前钩子失败 [%s]: %v", name, err)
	}

	// 执行关闭
	logger.Info("[服务关闭] 正在关闭服务: %s", name)
	if err := service.Close(ctx); err != nil {
		logger.Error("[服务关闭] 服务 %s 关闭失败: %v", name, err)
		return err
	}

	// 执行关闭后钩子
	if err := r.ExecuteHooks(ctx, name, AfterClose); err != nil {
		logger.Error("[服务关闭] 执行关闭后钩子失败 [%s]: %v", name, err)
	}

	r.SetState(name, StateClosed)
	logger.Info("[服务关闭] 服务 %s 已关闭", name)
	return nil
}

// --- 全局函数（便捷方法）---

// RegisterService 注册服务到全局注册中心
func RegisterService(service Service) error {
	return globalRegistry.Register(service)
}

// RegisterServiceHook 注册钩子到全局注册中心
func RegisterServiceHook(serviceName string, hook Hook) {
	globalRegistry.RegisterHook(serviceName, hook)
}

// GetServiceState 获取服务状态
func GetServiceState(name string) ServiceState {
	return globalRegistry.GetState(name)
}

// GetGlobalRegistry 获取全局注册中心
func GetGlobalRegistry() *ServiceRegistry {
	return globalRegistry
}
