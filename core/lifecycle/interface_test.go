// Package lifecycle 接口定义测试
//
// ==================== 测试说明 ====================
// 本文件包含 lifecycle 包中枚举类型 String() 方法的单元测试。
//
// 测试覆盖内容：
// 1. AppHookPhase 所有阶段的字符串表示
// 2. ServiceState 所有状态的字符串表示
// 3. 未知值的 fallback 行为
//
// 运行测试：go test -v ./core/lifecycle/... -run String
// ==================================================
package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppHookPhase_String 测试应用级钩子阶段字符串表示
//
// 【功能点】验证所有 AppHookPhase 枚举值的 String() 输出
// 【测试流程】
// 1. 遍历所有已知阶段，断言字符串输出
// 2. 测试未知值返回 "unknown"
func TestAppHookPhase_String(t *testing.T) {
	tests := []struct {
		phase    AppHookPhase
		expected string
	}{
		{AppBeforeInit, "AppBeforeInit"},
		{AppAfterInit, "AppAfterInit"},
		{AppOnReady, "AppOnReady"},
		{AppBeforeShutdown, "AppBeforeShutdown"},
		{AppAfterShutdown, "AppAfterShutdown"},
		{AppOnInitFailed, "AppOnInitFailed"},
		{AppHookPhase(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.phase.String())
		})
	}
}

// TestServiceState_String 测试服务状态字符串表示
//
// 【功能点】验证所有 ServiceState 枚举值的 String() 输出
// 【测试流程】
// 1. 遍历所有已知状态，断言字符串输出
// 2. 测试未知值返回 "unknown"
func TestServiceState_String(t *testing.T) {
	tests := []struct {
		state    ServiceState
		expected string
	}{
		{StateUninitialized, "uninitialized"},
		{StateInitializing, "initializing"},
		{StateReady, "ready"},
		{StateFailed, "failed"},
		{StateClosed, "closed"},
		{ServiceState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}
