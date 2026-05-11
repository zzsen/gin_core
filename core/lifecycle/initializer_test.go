// Package lifecycle 并行初始化器测试
//
// ==================== 测试说明 ====================
// 本文件包含 ParallelInitializer 的单元测试。
//
// 测试覆盖内容：
// 1. 无服务时的 Init 行为
// 2. 单层无依赖并行初始化
// 3. 多层依赖串行/并行混合初始化
// 4. 初始化失败传播
// 5. 重试机制（RetryCount）
// 6. 超时机制（Timeout）
// 7. 逆序关闭（Close）
// 8. 关闭时依赖解析失败的回退逻辑
// 9. 全局便捷函数（SetInitConfig）
//
// 运行测试：go test -v ./core/lifecycle/... -run "TestInitializer|TestSetInitConfig"
// ==================================================
package lifecycle

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// buildRegistry 构建测试用注册中心并注册服务
func buildRegistry(services ...Service) *ServiceRegistry {
	r := NewServiceRegistry()
	for _, s := range services {
		_ = r.Register(s)
	}
	return r
}

// TestInitializer_NoServices 测试无服务场景
//
// 【功能点】没有需要初始化的服务时应正常返回
// 【测试流程】
// 1. 注册中心为空
// 2. Init 应返回 nil
func TestInitializer_NoServices(t *testing.T) {
	r := NewServiceRegistry()
	p := NewParallelInitializer(r, DefaultInitConfig)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
}

// TestInitializer_SingleLayer 测试单层并行初始化
//
// 【功能点】无依赖的多个服务应并行初始化
// 【测试流程】
// 1. 注册 3 个无依赖服务
// 2. Init 后所有服务状态为 Ready
func TestInitializer_SingleLayer(t *testing.T) {
	a := newMock("a", 1)
	b := newMock("b", 2)
	c := newMock("c", 3)
	r := buildRegistry(a, b, c)

	p := NewParallelInitializer(r, DefaultInitConfig)
	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)

	assert.Equal(t, StateReady, r.GetState("a"))
	assert.Equal(t, StateReady, r.GetState("b"))
	assert.Equal(t, StateReady, r.GetState("c"))
}

// TestInitializer_MultiLayer 测试多层依赖初始化
//
// 【功能点】有依赖关系的服务应按层级顺序初始化
// 【测试流程】
// 1. C 无依赖，B 依赖 C，A 依赖 B
// 2. 初始化顺序应为 C→B→A
func TestInitializer_MultiLayer(t *testing.T) {
	var mu sync.Mutex
	var order []string

	makeInitFn := func(name string) func(context.Context) error {
		return func(_ context.Context) error {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		}
	}

	a := newMock("A", 1, "B")
	a.initFn = makeInitFn("A")
	b := newMock("B", 1, "C")
	b.initFn = makeInitFn("B")
	c := newMock("C", 1)
	c.initFn = makeInitFn("C")

	r := buildRegistry(a, b, c)
	p := NewParallelInitializer(r, DefaultInitConfig)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.Equal(t, []string{"C", "B", "A"}, order)
}

// TestInitializer_InitFail 测试初始化失败传播
//
// 【功能点】某服务初始化失败时应返回错误
// 【测试流程】
// 1. 服务 a 的 Init 返回错误
// 2. Init 应返回包含错误信息的错误
func TestInitializer_InitFail(t *testing.T) {
	a := newMock("a", 1)
	a.initFn = func(_ context.Context) error { return errors.New("init boom") }

	r := buildRegistry(a)
	cfg := DefaultInitConfig
	cfg.RetryCount = 0
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "init boom")
}

// TestInitializer_Retry 测试重试机制
//
// 【功能点】初始化失败后应按 RetryCount 重试
// 【测试流程】
// 1. 服务前 2 次 Init 失败，第 3 次成功
// 2. RetryCount=2（共 3 次尝试），应成功
func TestInitializer_Retry(t *testing.T) {
	var count int32
	a := newMock("a", 1)
	a.initFn = func(_ context.Context) error {
		n := atomic.AddInt32(&count, 1)
		if n < 3 {
			return errors.New("not ready")
		}
		return nil
	}

	r := buildRegistry(a)
	cfg := InitConfig{
		MaxConcurrency: 1,
		Timeout:        5 * time.Second,
		RetryCount:     2,
		RetryInterval:  10 * time.Millisecond,
	}
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}

// TestInitializer_RetryExhausted 测试重试耗尽
//
// 【功能点】重试次数用完仍然失败时应返回错误
// 【测试流程】
// 1. 服务始终返回错误
// 2. RetryCount=1 时应尝试 2 次后失败
func TestInitializer_RetryExhausted(t *testing.T) {
	var count int32
	a := newMock("a", 1)
	a.initFn = func(_ context.Context) error {
		atomic.AddInt32(&count, 1)
		return errors.New("always fail")
	}

	r := buildRegistry(a)
	cfg := InitConfig{
		MaxConcurrency: 1,
		Timeout:        5 * time.Second,
		RetryCount:     1,
		RetryInterval:  10 * time.Millisecond,
	}
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已重试")
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

// TestInitializer_Timeout 测试超时机制
//
// 【功能点】单服务初始化超过 Timeout 应返回超时错误
// 【测试流程】
// 1. 服务 Init 中 sleep 200ms
// 2. Timeout 设为 50ms
// 3. 应返回超时错误
func TestInitializer_Timeout(t *testing.T) {
	a := newMock("a", 1)
	a.initFn = func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	r := buildRegistry(a)
	cfg := InitConfig{
		MaxConcurrency: 1,
		Timeout:        50 * time.Millisecond,
		RetryCount:     0,
	}
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "超时")
}

// TestInitializer_Close 测试逆序关闭
//
// 【功能点】Close 应按依赖逆序关闭服务（高层先关闭）
// 【测试流程】
// 1. A 依赖 B（层级：[B]→[A]）
// 2. 先初始化，再关闭
// 3. 关闭顺序应为 A→B
func TestInitializer_Close(t *testing.T) {
	var mu sync.Mutex
	var order []string

	makeCloseFn := func(name string) func(context.Context) error {
		return func(_ context.Context) error {
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		}
	}

	a := newMock("A", 1, "B")
	a.closeFn = makeCloseFn("A")
	b := newMock("B", 1)
	b.closeFn = makeCloseFn("B")

	r := buildRegistry(a, b)
	cfg := DefaultInitConfig
	cfg.Timeout = 5 * time.Second
	p := NewParallelInitializer(r, cfg)

	baseCfg := &config.BaseConfig{}
	err := p.Init(context.Background(), baseCfg)
	require.NoError(t, err)

	err = p.Close(context.Background(), baseCfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"A", "B"}, order)
}

// TestInitializer_Close_NoServices 测试无服务时关闭
//
// 【功能点】没有服务时 Close 应正常返回
func TestInitializer_Close_NoServices(t *testing.T) {
	r := NewServiceRegistry()
	p := NewParallelInitializer(r, DefaultInitConfig)

	err := p.Close(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
}

// TestInitializer_ShouldInitFilter 测试 ShouldInit 过滤
//
// 【功能点】ShouldInit=false 的服务不应被初始化
// 【测试流程】
// 1. 注册 2 个服务，1 个 ShouldInit=false
// 2. Init 后只有 ShouldInit=true 的服务变为 Ready
func TestInitializer_ShouldInitFilter(t *testing.T) {
	a := newMock("a", 1)

	b := newMock("b", 2)
	b.shouldInit = false

	r := buildRegistry(a, b)
	p := NewParallelInitializer(r, DefaultInitConfig)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.Equal(t, StateReady, r.GetState("a"))
	assert.Equal(t, StateUninitialized, r.GetState("b"))
}

// TestInitializer_Concurrency 测试并发限制
//
// 【功能点】MaxConcurrency 限制同时初始化的服务数
// 【测试流程】
// 1. 注册 4 个无依赖服务，每个 Init 持续 50ms
// 2. MaxConcurrency=2
// 3. 并发峰值不超过 2
func TestInitializer_Concurrency(t *testing.T) {
	var maxConcurrent int32
	var current int32

	makeSvc := func(name string) *mockService {
		s := newMock(name, 1)
		s.initFn = func(_ context.Context) error {
			n := atomic.AddInt32(&current, 1)
			for {
				old := atomic.LoadInt32(&maxConcurrent)
				if n <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, n) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&current, -1)
			return nil
		}
		return s
	}

	r := buildRegistry(makeSvc("a"), makeSvc("b"), makeSvc("c"), makeSvc("d"))
	cfg := InitConfig{
		MaxConcurrency: 2,
		Timeout:        5 * time.Second,
		RetryCount:     0,
	}
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.LessOrEqual(t, atomic.LoadInt32(&maxConcurrent), int32(2))
}

// TestInitializer_NoConcurrencyLimit 测试不限制并发
//
// 【功能点】MaxConcurrency=0 时不限制并发
func TestInitializer_NoConcurrencyLimit(t *testing.T) {
	var count int32
	makeSvc := func(name string) *mockService {
		s := newMock(name, 1)
		s.initFn = func(_ context.Context) error {
			atomic.AddInt32(&count, 1)
			time.Sleep(20 * time.Millisecond)
			return nil
		}
		return s
	}

	r := buildRegistry(makeSvc("a"), makeSvc("b"), makeSvc("c"))
	cfg := InitConfig{
		MaxConcurrency: 0,
		Timeout:        5 * time.Second,
		RetryCount:     0,
	}
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&count))
}

// TestInitializer_NoTimeout 测试无超时限制
//
// 【功能点】Timeout=0 时不设置超时
func TestInitializer_NoTimeout(t *testing.T) {
	a := newMock("a", 1)
	a.initFn = func(_ context.Context) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}
	r := buildRegistry(a)
	cfg := InitConfig{
		MaxConcurrency: 1,
		Timeout:        0,
		RetryCount:     0,
	}
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.Equal(t, StateReady, r.GetState("a"))
}

// TestSetInitConfig 测试全局初始化配置设置
//
// 【功能点】SetInitConfig 应更新全局配置
func TestSetInitConfig(t *testing.T) {
	original := globalInitConfig
	defer func() { globalInitConfig = original }()

	newCfg := InitConfig{
		MaxConcurrency: 8,
		Timeout:        60 * time.Second,
		RetryCount:     3,
		RetryInterval:  2 * time.Second,
	}
	SetInitConfig(newCfg)
	assert.Equal(t, newCfg, globalInitConfig)
}

// TestInitializer_SingleServiceLayer 测试单服务层直接初始化（跳过 errgroup）
//
// 【功能点】当层内只有 1 个服务时直接初始化，不使用 errgroup
// 【测试流程】
// 1. 注册 1 个服务
// 2. Init 后状态为 Ready
func TestInitializer_SingleServiceLayer(t *testing.T) {
	a := newMock("a", 1)
	r := buildRegistry(a)
	cfg := DefaultInitConfig
	cfg.Timeout = 5 * time.Second
	p := NewParallelInitializer(r, cfg)

	err := p.Init(context.Background(), &config.BaseConfig{})
	require.NoError(t, err)
	assert.Equal(t, StateReady, r.GetState("a"))
}
