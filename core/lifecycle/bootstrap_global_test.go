// Package lifecycle bootstrap 与全局注册代理测试
//
// ==================== 测试说明 ====================
// 本文件包含 bootstrap 列表 API 和全局注册代理函数的单元测试。
// 使用 lifecycle.ResetGlobalStateForTest 清理全局状态，请勿与本包内其他用例并行执行。
//
// 测试覆盖内容：
// 1. AddMessageQueueConsumer / Producer / Schedule 列表 API 一致性
// 2. 全局代理函数（RegisterService、GetService、GetAllServices 等）
// 3. ExecuteAppHooks 错误传播
// 4. InitServices / CloseServices 完整流程（含钩子和状态变更）
// 5. InitAllServices / CloseAllServices 全量初始化与关闭
// 6. InitServices 空注册表行为
// 7. InitServices 使用 AppBaseConfig
//
// 运行测试：go test -short -count=1 ./core/lifecycle/... -run "TestBootstrap|TestGlobal"
// ==================================================
package lifecycle

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// TestBootstrap_MessageQueueAndScheduleLists 验证 bootstrap 列表 API
//
// 【功能点】AddMessageQueueConsumer / Producer / Schedule 与对应 Get 列表一致
// 【测试流程】
// 1. ResetGlobalStateForTest 清空全局累积列表
// 2. 添加 MQ 与定时任务配置
// 3. 断言 Get 列表长度与内容
func TestBootstrap_MessageQueueAndScheduleLists(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	consumer := &config.MessageQueue{
		MQName: "m1", QueueName: "q1", ExchangeName: "ex",
		ExchangeType: "topic", RoutingKey: "rk",
		Fun: func(string) error { return nil },
	}
	AddMessageQueueConsumer(consumer)

	producer := &config.MessageQueue{
		MQName: "p1", QueueName: "q2", ExchangeName: "ex2",
		ExchangeType: "direct", RoutingKey: "",
	}
	AddMessageQueueProducer(producer)

	AddSchedule(config.ScheduleInfo{
		Name: "daily",
		Cron: "@every 1h",
		Cmd:  func() {},
	})

	cl := GetMessageQueueConsumerList()
	pl := GetMessageQueueProducerList()
	sl := GetScheduleList()

	require.Len(t, cl, 1)
	require.Len(t, pl, 1)
	require.Len(t, sl, 1)
	assert.Same(t, consumer, cl[0])
	assert.Same(t, producer, pl[0])
	assert.Equal(t, "@every 1h", sl[0].Cron)
	assert.Equal(t, "daily", sl[0].Name)
}

// TestGlobal_RegistryProxies 验证 registry.go 全局函数代理到 globalRegistry
//
// 【功能点】RegisterService / Hook / AppHook、ExecuteAppHooks、GetServiceState、GetGlobalRegistry
// 【测试流程】
// 1. 清空全局状态后注册服务与应用钩子
// 2. 调用全局 ExecuteAppHooks、校验 GetGlobalRegistry 可见同一服务
// 3. 重复注册同名服务应失败
func TestGlobal_RegistryProxies(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	svc := newMock("global-svc", 1)
	require.NoError(t, RegisterService(svc))

	reg := GetGlobalRegistry()
	require.NotNil(t, reg)
	got, ok := reg.GetService("global-svc")
	require.True(t, ok)
	assert.Equal(t, "global-svc", got.Name())

	RegisterServiceHook("global-svc", Hook{
		Phase: BeforeInit,
		Fn:    func(context.Context, string) error { return nil },
	})

	var ran bool
	RegisterAppHook(AppHook{
		Phase:    AppBeforeInit,
		Priority: 1,
		Name:     "t-hook",
		Fn: func(context.Context) error {
			ran = true
			return nil
		},
	})
	require.NoError(t, ExecuteAppHooks(context.Background(), AppBeforeInit))
	assert.True(t, ran)

	assert.Equal(t, StateUninitialized, GetServiceState("global-svc"))

	err := RegisterService(newMock("global-svc", 2))
	require.Error(t, err)
}

// TestGlobal_ExecuteAppHooks_Error 验证全局 ExecuteAppHooks 错误传递
//
// 【功能点】全局 ExecuteAppHooks 在钩子失败时返回错误
// 【测试流程】
// 1. 注册返回错误的应用钩子
// 2. 调用 ExecuteAppHooks 应返回错误
func TestGlobal_ExecuteAppHooks_Error(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	RegisterAppHook(AppHook{
		Phase: AppAfterInit,
		Fn:    func(context.Context) error { return errors.New("app-hook-fail") },
	})
	err := ExecuteAppHooks(context.Background(), AppAfterInit)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "app-hook-fail")
}

// TestBootstrap_InitServices_CloseServices 验证 InitServices/CloseServices 走全局注册中心
//
// 【功能点】InitServices、CloseServices 能初始化并关闭已注册的 mock 服务
// 【测试流程】
// 1. Reset 全局状态并缩短并行初始化超时配置
// 2. RegisterService 后 InitServices，再 CloseServices
// 3. 断言服务状态 Ready → Closed
func TestBootstrap_InitServices_CloseServices(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	origInit := globalInitConfig
	defer func() { globalInitConfig = origInit }()
	globalInitConfig = InitConfig{
		MaxConcurrency: 1,
		Timeout:        5 * time.Second,
		RetryCount:     0,
		RetryInterval:  10 * time.Millisecond,
	}

	var initCalls, closeCalls int32
	svc := newMock("boot-svc", 1)
	svc.initFn = func(context.Context) error {
		atomic.AddInt32(&initCalls, 1)
		return nil
	}
	svc.closeFn = func(context.Context) error {
		atomic.AddInt32(&closeCalls, 1)
		return nil
	}
	require.NoError(t, RegisterService(svc))

	ctx := context.Background()
	require.NoError(t, InitServices(ctx))
	assert.Equal(t, int32(1), atomic.LoadInt32(&initCalls))
	assert.Equal(t, StateReady, GetServiceState("boot-svc"))

	require.NoError(t, CloseServices(ctx))
	assert.Equal(t, int32(1), atomic.LoadInt32(&closeCalls))
	assert.Equal(t, StateClosed, GetServiceState("boot-svc"))
}

// TestBootstrap_InitAllServices_CloseAllServices 验证 InitAllServices/CloseAllServices
//
// 【功能点】与 InitServices/CloseServices 等价路径显式覆盖全局并行初始化入口
// 【测试流程】
// 1. 注册单个服务并调用 InitAllServices(ctx, cfg)
// 2. 调用 CloseAllServices
func TestBootstrap_InitAllServices_CloseAllServices(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	origInit := globalInitConfig
	defer func() { globalInitConfig = origInit }()
	globalInitConfig = InitConfig{
		MaxConcurrency: 1,
		Timeout:        5 * time.Second,
		RetryCount:     0,
	}

	require.NoError(t, RegisterService(newMock("all-svc", 1)))
	ctx := context.Background()
	cfg := &config.BaseConfig{}

	require.NoError(t, InitAllServices(ctx, cfg))
	assert.Equal(t, StateReady, GetServiceState("all-svc"))

	require.NoError(t, CloseAllServices(ctx, cfg))
	assert.Equal(t, StateClosed, GetServiceState("all-svc"))
}

// TestBootstrap_InitServices_EmptyRegistry 无服务时 InitServices 成功
//
// 【功能点】全局注册中心为空时 InitServices 返回 nil
// 【测试流程】
// 1. Reset 后不注册任何服务
// 2. 调用 InitServices
func TestBootstrap_InitServices_EmptyRegistry(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	require.NoError(t, InitServices(context.Background()))
}

// TestBootstrap_InitServices_UsesAppBaseConfig 验证 InitServices 使用 app.BaseConfig
//
// 【功能点】InitServices(ctx) 等价于 InitAllServices(ctx, &app.BaseConfig)
// 【测试流程】
// 1. 保存并恢复 app.BaseConfig
// 2. 注册 ShouldInit 依赖 BaseConfig 的服务并断言 InitServices 行为一致
func TestBootstrap_InitServices_UsesAppBaseConfig(t *testing.T) {
	ResetGlobalStateForTest()
	defer ResetGlobalStateForTest()

	origAppCfg := app.BaseConfig
	defer func() { app.BaseConfig = origAppCfg }()
	app.BaseConfig = config.BaseConfig{}

	origInit := globalInitConfig
	defer func() { globalInitConfig = origInit }()
	globalInitConfig = InitConfig{MaxConcurrency: 1, Timeout: 5 * time.Second, RetryCount: 0}

	disabled := newMock("skip-svc", 1)
	disabled.shouldInit = false
	require.NoError(t, RegisterService(disabled))
	require.NoError(t, RegisterService(newMock("run-svc", 2)))

	require.NoError(t, InitServices(context.Background()))
	assert.Equal(t, StateUninitialized, GetServiceState("skip-svc"))
	assert.Equal(t, StateReady, GetServiceState("run-svc"))
}
