// Package core 应用生命周期钩子封装测试
//
// ==================== 测试说明 ====================
// 本文件验证 hooks.go 中对 lifecycle 应用级钩子的注册封装，
// 以及通过 lifecycle.ExecuteAppHooks 驱动的执行顺序与错误传播。
//
// 全局注册状态在各用例中通过 lifecycle.ResetGlobalStateForTest 清理，
// 请勿与本包内其他依赖全局注册表的用例并行执行（不使用 t.Parallel）。
//
// 运行测试：go test -short -count=1 ./core -run Hook
// ==================================================
package core

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzsen/gin_core/core/lifecycle"
)

func hookTestCleanup(t *testing.T) {
	t.Helper()
	lifecycle.ResetGlobalStateForTest()
	t.Cleanup(lifecycle.ResetGlobalStateForTest)
}

// TestRegisterAppHook 注册自定义应用钩子
//
// 【功能点】RegisterAppHook 将完整 AppHook 写入全局注册表并可被执行
// 【测试流程】
//  1. 清理全局状态后调用 RegisterAppHook 注册带名称与优先级的钩子
//  2. 调用 lifecycle.ExecuteAppHooks 匹配阶段执行
//  3. 断言钩子函数被调用且返回成功
func TestRegisterAppHook(t *testing.T) {
	hookTestCleanup(t)

	var ran bool
	RegisterAppHook(lifecycle.AppHook{
		Phase:    lifecycle.AppBeforeInit,
		Priority: 5,
		Name:     "named-hook",
		Fn: func(ctx context.Context) error {
			ran = true
			require.NotNil(t, ctx)
			return nil
		},
	})

	err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppBeforeInit)
	require.NoError(t, err)
	assert.True(t, ran)
}

// TestRegisterAppHook_PriorityOrder 应用钩子优先级排序
//
// 【功能点】同阶段多条钩子按 Priority 数值升序执行
// 【测试流程】
//  1. 注册两条 AppBeforeInit 钩子，优先级分别为 2 与 1
//  2. 执行该阶段钩子
//  3. 断言执行顺序为 1 先于 2
func TestRegisterAppHook_PriorityOrder(t *testing.T) {
	hookTestCleanup(t)

	var order []int
	RegisterAppHook(lifecycle.AppHook{
		Phase:    lifecycle.AppBeforeInit,
		Priority: 2,
		Name:     "second",
		Fn: func(ctx context.Context) error {
			order = append(order, 2)
			return nil
		},
	})
	RegisterAppHook(lifecycle.AppHook{
		Phase:    lifecycle.AppBeforeInit,
		Priority: 1,
		Name:     "first",
		Fn: func(ctx context.Context) error {
			order = append(order, 1)
			return nil
		},
	})

	err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppBeforeInit)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, order)
}

// TestRegisterAppHook_ErrorStopsChain 应用钩子错误中断
//
// 【功能点】任一应用钩子返回错误时 ExecuteAppHooks 立即失败
// 【测试流程】
//  1. 注册先成功后失败的两个同阶段钩子
//  2. 执行阶段
//  3. 断言返回错误且后续钩子未执行
func TestRegisterAppHook_ErrorStopsChain(t *testing.T) {
	hookTestCleanup(t)

	var secondRan bool
	RegisterAppHook(lifecycle.AppHook{
		Phase:    lifecycle.AppAfterInit,
		Priority: 1,
		Name:     "fail",
		Fn: func(ctx context.Context) error {
			return errors.New("hook failed")
		},
	})
	RegisterAppHook(lifecycle.AppHook{
		Phase:    lifecycle.AppAfterInit,
		Priority: 2,
		Name:     "after-fail",
		Fn: func(ctx context.Context) error {
			secondRan = true
			return nil
		},
	})

	err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppAfterInit)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hook failed")
	assert.False(t, secondRan)
}

// TestOnBeforeInit_OnAfterInit_ConvenienceWrappers 便捷注册函数阶段
//
// 【功能点】OnBeforeInit / OnAfterInit 等封装正确设置 Phase 并可被执行
// 【测试流程】
//  1. 分别注册各便捷函数对应阶段的钩子并记录命中
//  2. 对相应 AppHookPhase 调用 ExecuteAppHooks
//  3. 断言各阶段仅命中对应钩子
func TestOnBeforeInit_OnAfterInit_ConvenienceWrappers(t *testing.T) {
	hookTestCleanup(t)

	var hitBefore, hitAfter, hitReady, hitBeforeShutdown, hitAfterShutdown bool

	OnBeforeInit(func(ctx context.Context) error {
		hitBefore = true
		return nil
	})
	OnAfterInit(func(ctx context.Context) error {
		hitAfter = true
		return nil
	})
	OnReady(func(ctx context.Context) error {
		hitReady = true
		return nil
	})
	OnBeforeShutdown(func(ctx context.Context) error {
		hitBeforeShutdown = true
		return nil
	})
	OnAfterShutdown(func(ctx context.Context) error {
		hitAfterShutdown = true
		return nil
	})

	ctx := context.Background()
	require.NoError(t, lifecycle.ExecuteAppHooks(ctx, lifecycle.AppBeforeInit))
	assert.True(t, hitBefore)
	assert.False(t, hitAfter)

	require.NoError(t, lifecycle.ExecuteAppHooks(ctx, lifecycle.AppAfterInit))
	assert.True(t, hitAfter)

	require.NoError(t, lifecycle.ExecuteAppHooks(ctx, lifecycle.AppOnReady))
	assert.True(t, hitReady)

	require.NoError(t, lifecycle.ExecuteAppHooks(ctx, lifecycle.AppBeforeShutdown))
	assert.True(t, hitBeforeShutdown)

	require.NoError(t, lifecycle.ExecuteAppHooks(ctx, lifecycle.AppAfterShutdown))
	assert.True(t, hitAfterShutdown)
}

// TestExecuteAppHooks_NoHooksNoOp 无钩子阶段
//
// 【功能点】某阶段未注册任何钩子时执行应为成功空操作
// 【测试流程】
//  1. 清理全局状态且不注册钩子
//  2. 对 AppOnReady 调用 ExecuteAppHooks
//  3. 断言无错误
func TestExecuteAppHooks_NoHooksNoOp(t *testing.T) {
	hookTestCleanup(t)

	err := lifecycle.ExecuteAppHooks(context.Background(), lifecycle.AppOnReady)
	assert.NoError(t, err)
}
