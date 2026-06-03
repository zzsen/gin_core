// Package core 服务注册与全局辅助 API 封装测试
//
// ==================== 测试说明 ====================
// 本文件覆盖 service.go 中对 lifecycle 的全局封装（服务注册、钩子、状态、
// 初始化配置、消息队列与定时任务累积器），以及 validator.go 中自定义验证器与 binding 集成。
//
// 使用 lifecycle.ResetGlobalStateForTest 避免污染全局注册表；顺序执行，不使用 t.Parallel。
//
// 运行测试：go test -short -count=1 ./core -run 'Service|Validator|Kind'
// ==================================================
package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/model/config"
)

func svcWrapperCleanup(t *testing.T) {
	t.Helper()
	lifecycle.ResetGlobalStateForTest()
	t.Cleanup(lifecycle.ResetGlobalStateForTest)
}

type stubService struct {
	name       string
	priority   int
	shouldInit bool
}

func (s *stubService) Name() string                           { return s.name }
func (s *stubService) Priority() int                          { return s.priority }
func (s *stubService) Dependencies() []string                 { return nil }
func (s *stubService) ShouldInit(cfg *config.BaseConfig) bool { return s.shouldInit }
func (s *stubService) Init(ctx context.Context) error         { return nil }
func (s *stubService) Close(ctx context.Context) error        { return nil }

// TestRegisterService_AndDuplicate 服务注册与重复注册
//
// 【功能点】RegisterService 首次成功、同名再次注册返回错误
// 【测试流程】
//  1. 清理全局状态并注册 stubService
//  2. 再次注册同名实例
//  3. 断言第二次返回错误且文案包含「已注册」
func TestRegisterService_AndDuplicate(t *testing.T) {
	svcWrapperCleanup(t)

	svc := &stubService{name: "unit-stub", priority: 10, shouldInit: false}
	require.NoError(t, RegisterService(svc))
	err := RegisterService(&stubService{name: "unit-stub", priority: 11, shouldInit: false})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "已注册")
}

// TestRegisterServiceHook_AndExecuteHooks 服务级钩子注册与执行
//
// 【功能点】RegisterServiceHook 注册的钩子在 ExecuteHooks 对应阶段被调用
// 【测试流程】
//  1. 注册服务 unit-hook-svc
//  2. 注册 BeforeInit 钩子记录 serviceName
//  3. 调用 registry.ExecuteHooks(BeforeInit)
//  4. 断言钩子执行且 serviceName 正确
func TestRegisterServiceHook_AndExecuteHooks(t *testing.T) {
	svcWrapperCleanup(t)

	const svcName = "unit-hook-svc"
	require.NoError(t, RegisterService(&stubService{name: svcName, priority: 1, shouldInit: false}))

	var sawName string
	RegisterServiceHook(svcName, lifecycle.Hook{
		Phase:    lifecycle.BeforeInit,
		Priority: 1,
		Fn: func(ctx context.Context, name string) error {
			sawName = name
			return nil
		},
	})

	err := lifecycle.GetGlobalRegistry().ExecuteHooks(context.Background(), svcName, lifecycle.BeforeInit)
	require.NoError(t, err)
	assert.Equal(t, svcName, sawName)
}

// TestGetServiceState_DefaultUninitialized 服务状态默认值
//
// 【功能点】注册后未初始化时 GetServiceState 为 StateUninitialized
// 【测试流程】
//  1. 注册服务并读取状态
//  2. 断言为 lifecycle.StateUninitialized（与 core.StateUninitialized 别名一致）
func TestGetServiceState_DefaultUninitialized(t *testing.T) {
	svcWrapperCleanup(t)

	const svcName = "unit-state-svc"
	require.NoError(t, RegisterService(&stubService{name: svcName, priority: 1, shouldInit: false}))
	assert.Equal(t, StateUninitialized, GetServiceState(svcName))
	assert.Equal(t, lifecycle.StateUninitialized, GetServiceState(svcName))
}

// TestSetInitConfig_Wrapper 初始化配置封装
//
// 【功能点】SetInitConfig 写入 lifecycle 全局 InitConfig（通过并行初始化路径间接依赖）
// 【测试流程】
//  1. 调用 SetInitConfig 设置非默认值
//  2. defer ResetGlobalStateForTest 恢复默认配置（由 svcWrapperCleanup 负责）
//  3. 再次调用 SetInitConfig(DefaultInitConfig) 断言无 panic（封装可用）
func TestSetInitConfig_Wrapper(t *testing.T) {
	svcWrapperCleanup(t)

	custom := InitConfig{
		MaxConcurrency: 9,
		Timeout:        50 * time.Millisecond,
		RetryCount:     3,
		RetryInterval:  2 * time.Millisecond,
	}
	SetInitConfig(custom)

	SetInitConfig(DefaultInitConfig)
}

// TestAddMessageQueueConsumerProducer_AndAddSchedule 消息队列与定时任务累积
//
// 【功能点】AddMessageQueueConsumer / AddMessageQueueProducer / AddSchedule 追加全局列表并可由 getter 读取
// 【测试流程】
//  1. 构造指针类型的 MessageQueue 与消费者、生产者注册调用
//  2. 调用 AddSchedule
//  3. 断言 lifecycle.GetMessageQueueConsumerList 等与传入条目一致
func TestAddMessageQueueConsumerProducer_AndAddSchedule(t *testing.T) {
	svcWrapperCleanup(t)

	mqConsumer := &config.MessageQueue{QueueName: "q-cons"}
	mqProducer := &config.MessageQueue{QueueName: "q-prod"}

	AddMessageQueueConsumer(mqConsumer)
	AddMessageQueueProducer(mqProducer)

	AddSchedule(config.ScheduleInfo{Cron: "@every 1h", Name: "job-a"})

	cons := lifecycle.GetMessageQueueConsumerList()
	prod := lifecycle.GetMessageQueueProducerList()
	sched := lifecycle.GetScheduleList()

	require.Len(t, cons, 1)
	require.Len(t, prod, 1)
	require.Len(t, sched, 1)
	assert.Equal(t, mqConsumer, cons[0])
	assert.Equal(t, mqProducer, prod[0])
	assert.Equal(t, "@every 1h", sched[0].Cron)
	assert.Equal(t, "job-a", sched[0].Name)
}

// TestKindOfData_PointerAndValue kindOfData 类型判定
//
// 【功能点】kindOfData 对指针解引用后返回底层类别，非结构体返回对应 Kind
// 【测试流程】
//  1. 对结构体值、结构体指针调用 kindOfData，期望 Struct
//  2. 对字符串值与字符串指针调用，期望 String
func TestKindOfData_PointerAndValue(t *testing.T) {
	type sample struct{ X int }

	assert.Equal(t, kindOfData(sample{}), kindOfData(&sample{}))

	s := "x"
	assert.NotEqual(t, kindOfData(s), kindOfData(sample{}))
	assert.Equal(t, kindOfData(s), kindOfData(&s))
}

// TestDefaultValidator_ValidateStruct_StructAndNonStruct defaultValidator 校验路径
//
// 【功能点】ValidateStruct 仅对 struct 执行 playground 校验；非结构体直接返回 nil
// 【测试流程】
//  1. 构造 defaultValidator，对字符串调用 ValidateStruct 断言 nil
//  2. 对违反 binding 标签的结构体指针断言返回错误
//  3. 对合法结构体断言通过
func TestDefaultValidator_ValidateStruct_StructAndNonStruct(t *testing.T) {
	v := &defaultValidator{}

	assert.NoError(t, v.ValidateStruct("not-a-struct"))
	assert.NoError(t, v.ValidateStruct(struct{}{}))

	type row struct {
		Email string `binding:"required,email"`
	}
	require.Error(t, v.ValidateStruct(&row{Email: "bad"}))
	require.NoError(t, v.ValidateStruct(&row{Email: "ok@example.com"}))
}

// TestDefaultValidator_EngineLazyInit Engine 懒加载
//
// 【功能点】Engine() 触发 lazyinit 并返回非 nil 校验引擎
// 【测试流程】
//  1. 新建 defaultValidator 调用 Engine()
//  2. 断言返回值非 nil 且第二次调用仍可用
func TestDefaultValidator_EngineLazyInit(t *testing.T) {
	v := &defaultValidator{}
	eng := v.Engine()
	require.NotNil(t, eng)
	assert.NotNil(t, v.Engine())
}

// TestOverrideValidator_SetsBindingValidator overrideValidator 替换 Gin 校验器
//
// 【功能点】overrideValidator 将 binding.Validator 设为 *defaultValidator
// 【测试流程】
//  1. 备份 binding.Validator，defer 恢复
//  2. 调用 overrideValidator
//  3. 断言可通过 binding.Validator 完成字段校验
func TestOverrideValidator_SetsBindingValidator(t *testing.T) {
	prev := binding.Validator
	t.Cleanup(func() { binding.Validator = prev })

	overrideValidator()
	require.NotNil(t, binding.Validator)

	type payload struct {
		Name string `binding:"required"`
	}
	require.Error(t, binding.Validator.ValidateStruct(&payload{Name: ""}))
	require.NoError(t, binding.Validator.ValidateStruct(&payload{Name: "ok"}))
}

// TestRegisterBuiltinServices_Registrations 内置服务注册清单
//
// 【功能点】registerBuiltinServices 向全局注册表注册框架内置服务（不执行 Init）
// 【测试流程】
//  1. ResetGlobalStateForTest 后调用 registerBuiltinServices
//  2. 断言 logger、redis、rabbitmq 等各服务名称均可 GetService
func TestRegisterBuiltinServices_Registrations(t *testing.T) {
	svcWrapperCleanup(t)

	registerBuiltinServices()
	reg := lifecycle.GetGlobalRegistry()
	names := []string{
		"logger", "tracing", "redis", "mysql", "elasticsearch",
		"rabbitmq", "etcd", "schedule",
	}
	for _, n := range names {
		_, ok := reg.GetService(n)
		assert.True(t, ok, "missing service %q", n)
	}
}

// TestLoadConfig_ProdEnvSetsGinReleaseMode 生产环境与 Gin 模式
//
// 【功能点】loadConfig 在 -env=prod 时将 Gin 设为 ReleaseMode 并写入 app.Env
// 【测试流程】
//  1. 准备包含 config.default.yml 与 config.prod.yml 的临时目录
//  2. 使用命令行参数指定 prod 环境并调用 loadConfig
//  3. 断言 gin.Mode() 为 ReleaseMode 且 app.Env 为 prod
func TestLoadConfig_ProdEnvSetsGinReleaseMode(t *testing.T) {
	originalArgs := os.Args
	originalEnv := app.Env
	prevMode := gin.Mode()
	defer func() {
		os.Args = originalArgs
		app.Env = originalEnv
		gin.SetMode(prevMode)
	}()

	gin.SetMode(gin.TestMode)
	os.Args = []string{"program", "-env", constant.ProdEnv, "-config", "./test_conf_prod_wrap"}

	testConfDir := "test_conf_prod_wrap"
	require.NoError(t, os.MkdirAll(testConfDir, 0755))
	defer os.RemoveAll(testConfDir)

	defaultFile := filepath.Join(testConfDir, constant.DefaultConfigFileName)
	require.NoError(t, os.WriteFile(defaultFile, []byte("name: default_app\nport: 8080\n"), 0644))

	envFile := filepath.Join(testConfDir, constant.CustomConfigFileNamePrefix+constant.ProdEnv+constant.CustomConfigFileNameSuffix)
	require.NoError(t, os.WriteFile(envFile, []byte("name: prod_app\nport: 9090\n"), 0644))

	type wrapCfg struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	}
	cfg := &wrapCfg{}
	loadConfig(cfg)

	assert.Equal(t, gin.ReleaseMode, gin.Mode())
	assert.Equal(t, constant.ProdEnv, app.Env)
	assert.Equal(t, "prod_app", cfg.Name)
	assert.Equal(t, 9090, cfg.Port)
}

func metricsEngineTestSetup(t *testing.T) {
	t.Helper()
	originalConfig := app.BaseConfig
	t.Cleanup(func() { app.BaseConfig = originalConfig })

	clearMiddlewares()
	require.NoError(t, RegisterMiddleware("testMw", func() gin.HandlerFunc {
		return func(c *gin.Context) { c.Next() }
	}))
}

// TestInitEngine_MetricsRoute_CustomPath 自定义指标路径
//
// 【功能点】Metrics.Enabled 且 Path 非空时在自定义路径暴露 Prometheus Handler
// 【测试流程】
//  1. 配置 Path 为 /prometheus 并 initEngine
//  2. GET /prometheus 期望 200
func TestInitEngine_MetricsRoute_CustomPath(t *testing.T) {
	metricsEngineTestSetup(t)

	optionFuncList = make([]gin.OptionFunc, 0)
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{
			RoutePrefix: "",
			Middlewares: []string{"testMw"},
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Path:    "/prometheus",
		},
	}

	engine := initEngine()
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/prometheus", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestInitEngine_MetricsRoute_DefaultPath Path 为空时默认 /metrics
//
// 【功能点】Metrics.Path 为空时回退为 /metrics
// 【测试流程】
//  1. 启用 Metrics 且 Path 置空
//  2. initEngine 后 GET /metrics 期望 200
func TestInitEngine_MetricsRoute_DefaultPath(t *testing.T) {
	metricsEngineTestSetup(t)

	optionFuncList = make([]gin.OptionFunc, 0)
	app.BaseConfig = config.BaseConfig{
		Service: config.ServiceInfo{
			RoutePrefix: "",
			Middlewares: []string{"testMw"},
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Path:    "",
		},
	}

	engine := initEngine()
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	assert.Equal(t, http.StatusOK, w.Code)
}
