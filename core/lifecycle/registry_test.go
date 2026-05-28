// Package lifecycle 服务注册中心测试
//
// ==================== 测试说明 ====================
// 本文件包含 ServiceRegistry 的单元测试。
//
// 测试覆盖内容：
// 1. 服务注册与重复注册检测
// 2. 服务查询（GetService、GetAllServices、GetServicesToInit）
// 3. 服务状态管理（GetState、SetState）
// 4. 服务级钩子注册与执行（RegisterHook、ExecuteHooks）
// 5. 应用级钩子注册与执行（RegisterAppHook、ExecuteAppHooks）
// 6. InitService 完整流程（含钩子和状态变更）
// 7. CloseService 完整流程
// 8. 全局便捷函数
//
// 运行测试：go test -v ./core/lifecycle/... -run TestRegistry
// ==================================================
package lifecycle

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
)

// TestRegistry_Register 测试服务注册
//
// 【功能点】验证服务注册和重复注册检测
// 【测试流程】
// 1. 注册一个服务，应成功
// 2. 再次注册同名服务，应返回错误
// 3. 注册不同名服务，应成功
func TestRegistry_Register(t *testing.T) {
	r := NewServiceRegistry()

	err := r.Register(newMock("svc-a", 1))
	require.NoError(t, err)

	err = r.Register(newMock("svc-a", 2))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已注册")

	err = r.Register(newMock("svc-b", 1))
	require.NoError(t, err)
}

// TestRegistry_GetService 测试服务查询
//
// 【功能点】验证按名称查询服务
// 【测试流程】
// 1. 注册服务后查询，应找到
// 2. 查询未注册的服务，应返回 false
func TestRegistry_GetService(t *testing.T) {
	r := NewServiceRegistry()
	_ = r.Register(newMock("svc-a", 1))

	svc, ok := r.GetService("svc-a")
	assert.True(t, ok)
	assert.Equal(t, "svc-a", svc.Name())

	_, ok = r.GetService("not-exist")
	assert.False(t, ok)
}

// TestRegistry_GetAllServices 测试获取全部服务
//
// 【功能点】验证返回所有已注册服务的副本
// 【测试流程】
// 1. 注册 2 个服务
// 2. GetAllServices 返回 2 个
func TestRegistry_GetAllServices(t *testing.T) {
	r := NewServiceRegistry()
	_ = r.Register(newMock("svc-a", 1))
	_ = r.Register(newMock("svc-b", 2))

	all := r.GetAllServices()
	assert.Len(t, all, 2)
	assert.Contains(t, all, "svc-a")
	assert.Contains(t, all, "svc-b")
}

// TestRegistry_GetServicesToInit 测试筛选需要初始化的服务
//
// 【功能点】ShouldInit 为 false 的服务应被过滤
// 【测试流程】
// 1. 注册 2 个服务，1 个 ShouldInit=true，1 个 false
// 2. GetServicesToInit 应只返回 1 个
func TestRegistry_GetServicesToInit(t *testing.T) {
	r := NewServiceRegistry()
	_ = r.Register(newMock("svc-a", 1)) // shouldInit=true (default in newMock)

	disabled := newMock("svc-b", 2)
	disabled.shouldInit = false
	_ = r.Register(disabled)

	services := r.GetServicesToInit(&config.BaseConfig{})
	assert.Len(t, services, 1)
	assert.Equal(t, "svc-a", services[0].Name())
}

// TestRegistry_State 测试服务状态管理
//
// 【功能点】验证状态的读写和默认值
// 【测试流程】
// 1. 注册后状态为 StateUninitialized
// 2. SetState 后 GetState 返回新值
// 3. 未注册服务的状态为 StateUninitialized
func TestRegistry_State(t *testing.T) {
	r := NewServiceRegistry()
	_ = r.Register(newMock("svc-a", 1))

	assert.Equal(t, StateUninitialized, r.GetState("svc-a"))

	r.SetState("svc-a", StateReady)
	assert.Equal(t, StateReady, r.GetState("svc-a"))

	assert.Equal(t, StateUninitialized, r.GetState("not-exist"))
}

// TestRegistry_ExecuteHooks 测试服务级钩子执行
//
// 【功能点】验证钩子按阶段筛选、按优先级排序执行
// 【测试流程】
// 1. 注册 2 个 BeforeInit 钩子（优先级 2 和 1）
// 2. 注册 1 个 AfterInit 钩子
// 3. ExecuteHooks(BeforeInit) 应按优先级 1→2 顺序执行
// 4. AfterInit 钩子不受影响
func TestRegistry_ExecuteHooks(t *testing.T) {
	r := NewServiceRegistry()
	var order []int

	r.RegisterHook("svc-a", Hook{
		Phase:    BeforeInit,
		Priority: 2,
		Fn:       func(_ context.Context, _ string) error { order = append(order, 2); return nil },
	})
	r.RegisterHook("svc-a", Hook{
		Phase:    BeforeInit,
		Priority: 1,
		Fn:       func(_ context.Context, _ string) error { order = append(order, 1); return nil },
	})
	r.RegisterHook("svc-a", Hook{
		Phase:    AfterInit,
		Priority: 1,
		Fn:       func(_ context.Context, _ string) error { order = append(order, 99); return nil },
	})

	err := r.ExecuteHooks(context.Background(), "svc-a", BeforeInit)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, order)
}

// TestRegistry_ExecuteHooks_Error 测试钩子执行失败
//
// 【功能点】钩子返回错误时应传播
// 【测试流程】
// 1. 注册一个返回错误的钩子
// 2. ExecuteHooks 应返回该错误
func TestRegistry_ExecuteHooks_Error(t *testing.T) {
	r := NewServiceRegistry()
	r.RegisterHook("svc-a", Hook{
		Phase: BeforeInit,
		Fn:    func(_ context.Context, _ string) error { return errors.New("hook failed") },
	})

	err := r.ExecuteHooks(context.Background(), "svc-a", BeforeInit)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook failed")
}

// TestRegistry_InitService 测试服务初始化完整流程
//
// 【功能点】验证 InitService 的状态流转和钩子执行
// 【测试流程】
// 1. 注册服务和 BeforeInit/AfterInit 钩子
// 2. InitService 后状态为 StateReady
// 3. 钩子按正确顺序执行
// 4. 重复初始化应直接返回（幂等）
func TestRegistry_InitService(t *testing.T) {
	r := NewServiceRegistry()
	var steps []string

	svc := newMock("svc-a", 1)
	svc.initFn = func(_ context.Context) error {
		steps = append(steps, "init")
		return nil
	}
	_ = r.Register(svc)

	r.RegisterHook("svc-a", Hook{
		Phase: BeforeInit,
		Fn:    func(_ context.Context, _ string) error { steps = append(steps, "before"); return nil },
	})
	r.RegisterHook("svc-a", Hook{
		Phase: AfterInit,
		Fn:    func(_ context.Context, _ string) error { steps = append(steps, "after"); return nil },
	})

	err := r.InitService(context.Background(), "svc-a")
	require.NoError(t, err)
	assert.Equal(t, StateReady, r.GetState("svc-a"))
	assert.Equal(t, []string{"before", "init", "after"}, steps)

	// 幂等：已初始化则跳过
	err = r.InitService(context.Background(), "svc-a")
	require.NoError(t, err)
}

// TestRegistry_InitService_NotRegistered 测试初始化未注册服务
//
// 【功能点】InitService 传入未注册名称应返回错误
func TestRegistry_InitService_NotRegistered(t *testing.T) {
	r := NewServiceRegistry()
	err := r.InitService(context.Background(), "not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "未注册")
}

// TestRegistry_InitService_Fail 测试初始化失败的状态变更
//
// 【功能点】Init 返回错误时状态应为 StateFailed
// 【测试流程】
// 1. 注册一个 Init 返回错误的服务
// 2. InitService 应返回错误
// 3. 状态变为 StateFailed
func TestRegistry_InitService_Fail(t *testing.T) {
	r := NewServiceRegistry()
	svc := newMock("svc-a", 1)
	svc.initFn = func(_ context.Context) error { return errors.New("init error") }
	_ = r.Register(svc)

	err := r.InitService(context.Background(), "svc-a")
	require.Error(t, err)
	assert.Equal(t, StateFailed, r.GetState("svc-a"))
}

// TestRegistry_InitService_BeforeHookFail 测试 BeforeInit 钩子失败
//
// 【功能点】BeforeInit 钩子失败时应中止初始化，状态为 StateFailed
func TestRegistry_InitService_BeforeHookFail(t *testing.T) {
	r := NewServiceRegistry()
	svc := newMock("svc-a", 1)
	initCalled := false
	svc.initFn = func(_ context.Context) error { initCalled = true; return nil }
	_ = r.Register(svc)

	r.RegisterHook("svc-a", Hook{
		Phase: BeforeInit,
		Fn:    func(_ context.Context, _ string) error { return errors.New("before hook failed") },
	})

	err := r.InitService(context.Background(), "svc-a")
	require.Error(t, err)
	assert.False(t, initCalled, "BeforeInit 失败后不应调用 Init")
	assert.Equal(t, StateFailed, r.GetState("svc-a"))
}

// TestRegistry_CloseService 测试服务关闭流程
//
// 【功能点】验证关闭流程的钩子和状态变更
// 【测试流程】
// 1. 初始化服务使其变为 Ready
// 2. CloseService 后状态为 StateClosed
// 3. BeforeClose/AfterClose 钩子执行
func TestRegistry_CloseService(t *testing.T) {
	r := NewServiceRegistry()
	var steps []string

	svc := newMock("svc-a", 1)
	svc.closeFn = func(_ context.Context) error {
		steps = append(steps, "close")
		return nil
	}
	_ = r.Register(svc)
	r.SetState("svc-a", StateReady)

	r.RegisterHook("svc-a", Hook{
		Phase: BeforeClose,
		Fn:    func(_ context.Context, _ string) error { steps = append(steps, "before-close"); return nil },
	})
	r.RegisterHook("svc-a", Hook{
		Phase: AfterClose,
		Fn:    func(_ context.Context, _ string) error { steps = append(steps, "after-close"); return nil },
	})

	err := r.CloseService(context.Background(), "svc-a")
	require.NoError(t, err)
	assert.Equal(t, StateClosed, r.GetState("svc-a"))
	assert.Equal(t, []string{"before-close", "close", "after-close"}, steps)
}

// TestRegistry_CloseService_NotReady 测试关闭未就绪服务
//
// 【功能点】未初始化或已关闭的服务不执行关闭
func TestRegistry_CloseService_NotReady(t *testing.T) {
	r := NewServiceRegistry()
	_ = r.Register(newMock("svc-a", 1))

	err := r.CloseService(context.Background(), "svc-a")
	require.NoError(t, err)

	err = r.CloseService(context.Background(), "not-exist")
	require.NoError(t, err)
}

// TestRegistry_AppHooks 测试应用级钩子
//
// 【功能点】验证应用级钩子的注册、阶段筛选、优先级排序
// 【测试流程】
// 1. 注册不同阶段和优先级的应用钩子
// 2. 执行 AppBeforeInit 阶段，验证只执行匹配的钩子
// 3. 验证优先级排序
func TestRegistry_AppHooks(t *testing.T) {
	r := NewServiceRegistry()
	var order []string

	r.RegisterAppHook(AppHook{
		Phase:    AppBeforeInit,
		Priority: 2,
		Name:     "hook-b",
		Fn:       func(_ context.Context) error { order = append(order, "b"); return nil },
	})
	r.RegisterAppHook(AppHook{
		Phase:    AppBeforeInit,
		Priority: 1,
		Name:     "hook-a",
		Fn:       func(_ context.Context) error { order = append(order, "a"); return nil },
	})
	r.RegisterAppHook(AppHook{
		Phase:    AppAfterInit,
		Priority: 1,
		Name:     "hook-after",
		Fn:       func(_ context.Context) error { order = append(order, "after"); return nil },
	})

	err := r.ExecuteAppHooks(context.Background(), AppBeforeInit)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, order)
}

// TestRegistry_AppHooks_Empty 测试空阶段钩子
//
// 【功能点】没有匹配钩子时应正常返回
func TestRegistry_AppHooks_Empty(t *testing.T) {
	r := NewServiceRegistry()
	err := r.ExecuteAppHooks(context.Background(), AppOnReady)
	require.NoError(t, err)
}

// TestRegistry_AppHooks_Error 测试应用钩子执行失败
//
// 【功能点】钩子返回错误时应传播
func TestRegistry_AppHooks_Error(t *testing.T) {
	r := NewServiceRegistry()
	r.RegisterAppHook(AppHook{
		Phase: AppBeforeInit,
		Fn:    func(_ context.Context) error { return errors.New("app hook failed") },
	})

	err := r.ExecuteAppHooks(context.Background(), AppBeforeInit)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app hook failed")
}

// TestRegistry_AppHooks_NoName 测试无名称钩子
//
// 【功能点】Name 为空时使用 phase.String() 作为默认名称，不影响执行
func TestRegistry_AppHooks_NoName(t *testing.T) {
	r := NewServiceRegistry()
	called := false
	r.RegisterAppHook(AppHook{
		Phase: AppOnReady,
		Fn:    func(_ context.Context) error { called = true; return nil },
	})

	err := r.ExecuteAppHooks(context.Background(), AppOnReady)
	require.NoError(t, err)
	assert.True(t, called)
}

// TestRegistry_InitService_Initializing 测试并发初始化检测
//
// 【功能点】状态为 StateInitializing 时再次 InitService 应返回错误
func TestRegistry_InitService_Initializing(t *testing.T) {
	r := NewServiceRegistry()
	_ = r.Register(newMock("svc-a", 1))
	r.SetState("svc-a", StateInitializing)

	err := r.InitService(context.Background(), "svc-a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "正在初始化中")
}

// TestRegistry_CloseService_BeforeCloseHookError_ContinuesClose 关闭前钩子失败仍继续关闭
//
// 【功能点】BeforeClose 钩子返回错误时仅记日志，Close 仍执行且最终 StateClosed
// 【测试流程】
// 1. 注册服务并置为 Ready，注册失败的 BeforeClose 钩子
// 2. CloseService 成功返回且 closeFn 被调用
func TestRegistry_CloseService_BeforeCloseHookError_ContinuesClose(t *testing.T) {
	r := NewServiceRegistry()
	var closed bool
	svc := newMock("svc-a", 1)
	svc.closeFn = func(context.Context) error {
		closed = true
		return nil
	}
	require.NoError(t, r.Register(svc))
	r.SetState("svc-a", StateReady)

	r.RegisterHook("svc-a", Hook{
		Phase: BeforeClose,
		Fn:    func(context.Context, string) error { return errors.New("before-close-hook") },
	})

	err := r.CloseService(context.Background(), "svc-a")
	require.NoError(t, err)
	assert.True(t, closed)
	assert.Equal(t, StateClosed, r.GetState("svc-a"))
}

// TestRegistry_CloseService_AfterCloseHookError_StillClosed AfterClose 钩子失败仍置为已关闭
//
// 【功能点】AfterClose 钩子失败仅记日志，服务状态仍为 StateClosed
// 【测试流程】
// 1. BeforeClose 成功、Close 成功、AfterClose 返回错误
// 2. CloseService 返回 nil，状态为 Closed
func TestRegistry_CloseService_AfterCloseHookError_StillClosed(t *testing.T) {
	r := NewServiceRegistry()
	svc := newMock("svc-a", 1)
	require.NoError(t, r.Register(svc))
	r.SetState("svc-a", StateReady)

	r.RegisterHook("svc-a", Hook{
		Phase: AfterClose,
		Fn:    func(context.Context, string) error { return errors.New("after-close-hook") },
	})

	err := r.CloseService(context.Background(), "svc-a")
	require.NoError(t, err)
	assert.Equal(t, StateClosed, r.GetState("svc-a"))
}

// TestRegistry_CloseService_CloseReturnsError 关闭实现返回错误
//
// 【功能点】service.Close 返回错误时 CloseService 返回该错误且不进入 AfterClose
// 【测试流程】
// 1. closeFn 返回错误
// 2. CloseService 返回错误，状态保持 Ready
func TestRegistry_CloseService_CloseReturnsError(t *testing.T) {
	r := NewServiceRegistry()
	var afterCalls int32
	svc := newMock("svc-a", 1)
	svc.closeFn = func(context.Context) error { return errors.New("close-failed") }
	require.NoError(t, r.Register(svc))
	r.SetState("svc-a", StateReady)

	r.RegisterHook("svc-a", Hook{
		Phase: AfterClose,
		Fn: func(context.Context, string) error {
			atomic.AddInt32(&afterCalls, 1)
			return nil
		},
	})

	err := r.CloseService(context.Background(), "svc-a")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close-failed")
	assert.Equal(t, StateReady, r.GetState("svc-a"))
	assert.Equal(t, int32(0), atomic.LoadInt32(&afterCalls))
}

// TestRegistry_CloseService_AlreadyClosed 重复关闭已关闭状态的服务
//
// 【功能点】状态非 Ready 时 CloseService 静默跳过，不调用 Close
// 【测试流程】
// 1. 将状态设为 StateClosed
// 2. CloseService 返回 nil 且 closeFn 不被调用
func TestRegistry_CloseService_AlreadyClosed(t *testing.T) {
	r := NewServiceRegistry()
	var closeCalls int32
	svc := newMock("svc-a", 1)
	svc.closeFn = func(context.Context) error {
		atomic.AddInt32(&closeCalls, 1)
		return nil
	}
	require.NoError(t, r.Register(svc))
	r.SetState("svc-a", StateClosed)

	err := r.CloseService(context.Background(), "svc-a")
	require.NoError(t, err)
	assert.Equal(t, int32(0), atomic.LoadInt32(&closeCalls))
}
