package lifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
	"golang.org/x/sync/errgroup"
)

// InitConfig 初始化配置
type InitConfig struct {
	MaxConcurrency int           // 最大并发数（0表示不限制）
	Timeout        time.Duration // 单个服务初始化超时（0表示不限制）
	RetryCount     int           // 失败重试次数
	RetryInterval  time.Duration // 重试间隔
}

// DefaultInitConfig 默认初始化配置
var DefaultInitConfig = InitConfig{
	MaxConcurrency: 4,
	Timeout:        30 * time.Second,
	RetryCount:     0,
	RetryInterval:  time.Second,
}

// 全局初始化配置
var globalInitConfig = DefaultInitConfig

// SetInitConfig 设置全局初始化配置
func SetInitConfig(cfg InitConfig) {
	globalInitConfig = cfg
}

// ParallelInitializer 并行初始化器
type ParallelInitializer struct {
	registry *ServiceRegistry
	config   InitConfig
}

// NewParallelInitializer 创建并行初始化器
func NewParallelInitializer(registry *ServiceRegistry, cfg InitConfig) *ParallelInitializer {
	return &ParallelInitializer{
		registry: registry,
		config:   cfg,
	}
}

// Init 执行并行初始化
// 参数：
//   - ctx: 上下文
//   - baseConfig: 基础配置，用于判断哪些服务需要初始化
func (p *ParallelInitializer) Init(ctx context.Context, baseConfig *config.BaseConfig) error {
	// 1. 获取需要初始化的服务
	services := p.registry.GetServicesToInit(baseConfig)
	if len(services) == 0 {
		logger.Info("[并行初始化] 没有需要初始化的服务")
		return nil
	}

	// 构建服务映射（只包含需要初始化的服务）
	serviceMap := make(map[string]Service)
	for _, s := range services {
		serviceMap[s.Name()] = s
	}

	// 2. 解析依赖关系
	resolver := NewDependencyResolver(serviceMap)

	// 验证依赖
	missing := resolver.ValidateDependencies()
	if len(missing) > 0 {
		// 只警告，不阻止初始化（依赖可能被禁用）
		for svc, deps := range missing {
			logger.Warn("[并行初始化] 服务 %s 的依赖未找到: %v", svc, deps)
		}
	}

	// 获取分层初始化顺序
	layers, err := resolver.Resolve()
	if err != nil {
		return fmt.Errorf("解析依赖关系失败: %w", err)
	}

	logger.Info("[并行初始化] 开始初始化 %d 个服务，共 %d 层", len(serviceMap), len(layers))

	// 3. 逐层初始化
	for i, layer := range layers {
		logger.Info("[并行初始化] 正在初始化第 %d 层，包含 %d 个服务: %v", i+1, len(layer), layer)

		if err := p.initLayer(ctx, layer); err != nil {
			return fmt.Errorf("第 %d 层初始化失败: %w", i+1, err)
		}

		logger.Info("[并行初始化] 第 %d 层初始化完成", i+1)
	}

	logger.Info("[并行初始化] 所有服务初始化完成")
	return nil
}

// initLayer 并行初始化同一层级的服务
func (p *ParallelInitializer) initLayer(ctx context.Context, serviceNames []string) error {
	if len(serviceNames) == 0 {
		return nil
	}

	// 如果只有一个服务，直接初始化
	if len(serviceNames) == 1 {
		return p.initServiceWithRetry(ctx, serviceNames[0])
	}

	// 使用 errgroup 并行执行
	g, ctx := errgroup.WithContext(ctx)

	// 限制并发数
	var sem chan struct{}
	if p.config.MaxConcurrency > 0 {
		sem = make(chan struct{}, p.config.MaxConcurrency)
	}

	for _, name := range serviceNames {
		name := name // 捕获变量
		g.Go(func() error {
			// 获取信号量
			if sem != nil {
				sem <- struct{}{}
				defer func() { <-sem }()
			}

			return p.initServiceWithRetry(ctx, name)
		})
	}

	return g.Wait()
}

// initServiceWithRetry 初始化单个服务，支持重试
func (p *ParallelInitializer) initServiceWithRetry(ctx context.Context, name string) error {
	var lastErr error

	for attempt := 0; attempt <= p.config.RetryCount; attempt++ {
		// 重试时等待
		if attempt > 0 {
			logger.Info("[并行初始化] 重试初始化服务 %s，第 %d 次重试", name, attempt)
			time.Sleep(p.config.RetryInterval)
		}

		// 带超时的初始化
		err := p.initServiceWithTimeout(ctx, name)
		if err == nil {
			return nil
		}

		lastErr = err
		logger.Error("[并行初始化] 服务 %s 初始化失败: %v", name, err)
	}

	return fmt.Errorf("服务 %s 初始化失败（已重试 %d 次）: %w", name, p.config.RetryCount, lastErr)
}

// initServiceWithTimeout 带超时的服务初始化
func (p *ParallelInitializer) initServiceWithTimeout(ctx context.Context, name string) error {
	// 如果配置了超时
	if p.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.Timeout)
		defer cancel()
	}

	// 使用 channel 等待初始化完成
	done := make(chan error, 1)
	go func() {
		done <- p.registry.InitService(ctx, name)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("服务 %s 初始化超时", name)
	}
}

// Close 按逆序关闭所有服务
func (p *ParallelInitializer) Close(ctx context.Context, baseConfig *config.BaseConfig) error {
	// 获取需要关闭的服务
	services := p.registry.GetServicesToInit(baseConfig)
	if len(services) == 0 {
		return nil
	}

	// 构建服务映射
	serviceMap := make(map[string]Service)
	for _, s := range services {
		serviceMap[s.Name()] = s
	}

	// 解析依赖关系获取层级
	resolver := NewDependencyResolver(serviceMap)
	layers, err := resolver.Resolve()
	if err != nil {
		// 如果解析失败，按注册顺序关闭
		logger.Warn("[并行初始化] 解析依赖关系失败，按默认顺序关闭: %v", err)
		for _, service := range services {
			_ = p.registry.CloseService(ctx, service.Name())
		}
		return nil
	}

	logger.Info("[服务关闭] 开始关闭服务，共 %d 层", len(layers))

	// 逆序关闭
	for i := len(layers) - 1; i >= 0; i-- {
		layer := layers[i]
		logger.Info("[服务关闭] 正在关闭第 %d 层: %v", i+1, layer)

		// 层内可以并行关闭
		g, ctx := errgroup.WithContext(ctx)
		for _, name := range layer {
			name := name
			g.Go(func() error {
				return p.registry.CloseService(ctx, name)
			})
		}

		if err := g.Wait(); err != nil {
			logger.Error("[服务关闭] 第 %d 层关闭时出错: %v", i+1, err)
		}
	}

	logger.Info("[服务关闭] 所有服务已关闭")
	return nil
}

// --- 全局便捷函数 ---

// InitAllServices 初始化所有已注册的服务
func InitAllServices(ctx context.Context, baseConfig *config.BaseConfig) error {
	initializer := NewParallelInitializer(globalRegistry, globalInitConfig)
	return initializer.Init(ctx, baseConfig)
}

// CloseAllServices 关闭所有已注册的服务
func CloseAllServices(ctx context.Context, baseConfig *config.BaseConfig) error {
	initializer := NewParallelInitializer(globalRegistry, globalInitConfig)
	return initializer.Close(ctx, baseConfig)
}
