// Package services 内置服务测试
//
// ==================== 测试说明 ====================
// 以表驱动方式集中验证各内置服务的元数据（Name/Priority/Dependencies）及 ShouldInit 在不同配置下的行为；
// 并覆盖 Init/HealthCheck/Close 中仅依赖配置与全局指针判空的逻辑分支，不连接 MySQL、Redis 等外部服务。
//
// 测试覆盖内容：
// 1. 全部内置服务的 Name()、Priority()、Dependencies() 元数据一致性
// 2. Logger / Tracing 及 System 开关类服务的 ShouldInit 边界（空配置、不完整配置、显式开关）
// 3. ScheduleService：ShouldInit 任务列表组合；Init 非法 Cron；合法 Cron 启动后可 Close
// 4. Redis / MySQL / Elasticsearch / Etcd：缺少配置时 Init 的前置校验错误分支
// 5. Redis / MySQL / Elasticsearch / Etcd：全局客户端为空时的 HealthCheck 与 Close 分支
// 6. lifecycle.Service 接口赋值兼容性（含 RabbitMQ、Schedule）
//
// 运行测试：go test -v ./core/services/... -run Test
// ==================================================

package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/model/config"
)

// serviceMetadata 抽取内置服务共用的元数据方法，便于表驱动测试
type serviceMetadata interface {
	Name() string
	Priority() int
	Dependencies() []string
}

// TestBuiltinServicesMetadata_Table 内置服务元数据表驱动校验
//
// 【功能点】统一断言各 Service 的 Name、Priority、Dependencies 与实现保持一致
// 【测试流程】
// 1. 构造包含全部内置服务实例的行表
// 2. 逐行比对 Name、Priority、Dependencies 期望值
func TestBuiltinServicesMetadata_Table(t *testing.T) {
	tests := []struct {
		name         string
		svc          serviceMetadata
		wantName     string
		wantPriority int
		wantDeps     []string
	}{
		{"LoggerService", &LoggerService{}, "logger", 0, nil},
		{"TracingService", &TracingService{}, "tracing", 5, []string{"logger"}},
		{"RedisService", &RedisService{}, "redis", 10, []string{"logger"}},
		{"MySQLService", &MySQLService{}, "mysql", 10, []string{"logger"}},
		{"ElasticsearchService", &ElasticsearchService{}, "elasticsearch", 20, []string{"logger"}},
		{"EtcdService", &EtcdService{}, "etcd", 20, []string{"logger"}},
		{"ScheduleService", NewScheduleService(nil), "schedule", 100, []string{"logger"}},
		{"RabbitMQService", NewRabbitMQService(nil, nil), "rabbitmq", 30, []string{"logger"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, tt.svc.Name())
			assert.Equal(t, tt.wantPriority, tt.svc.Priority())
			assert.Equal(t, tt.wantDeps, tt.svc.Dependencies())
		})
	}
}

// TestLoggerService_ShouldInit_Table 日志服务 ShouldInit 表驱动
//
// 【功能点】LoggerService.ShouldInit 在任何业务配置下均应返回 true
// 【测试流程】
// 1. 准备空配置与带 System 默认值的配置
// 2. 调用 ShouldInit 并期望均为 true
func TestLoggerService_ShouldInit_Table(t *testing.T) {
	svc := &LoggerService{}
	tests := []struct {
		name string
		cfg  *config.BaseConfig
	}{
		{"空 BaseConfig", &config.BaseConfig{}},
		{"仅 System 零值", &config.BaseConfig{System: config.SystemInfo{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, svc.ShouldInit(tt.cfg))
		})
	}
}

// TestTracingService_ShouldInit_Table 链路追踪 ShouldInit 表驱动
//
// 【功能点】仅在 Tracing 配置存在且 Enabled 为 true 时应初始化
// 【测试流程】
// 1. 覆盖 Tracing 未配置、指针 nil、存在但未启用、启用四种情况
// 2. 分别断言 ShouldInit 布尔结果
func TestTracingService_ShouldInit_Table(t *testing.T) {
	svc := &TracingService{}
	tests := []struct {
		name string
		cfg  *config.BaseConfig
		want bool
	}{
		{
			name: "Tracing 字段未设置（零值为 nil）",
			cfg:  &config.BaseConfig{},
			want: false,
		},
		{
			name: "Tracing 显式 nil",
			cfg:  &config.BaseConfig{Tracing: nil},
			want: false,
		},
		{
			name: "Tracing 存在但未启用",
			cfg:  &config.BaseConfig{Tracing: &config.TracingConfig{Enabled: false}},
			want: false,
		},
		{
			name: "Tracing 启用",
			cfg:  &config.BaseConfig{Tracing: &config.TracingConfig{Enabled: true}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, svc.ShouldInit(tt.cfg))
		})
	}
}

// systemFlagShouldIniter 由 System 开关控制的服务的 ShouldInit 共性抽象
type systemFlagShouldIniter interface {
	ShouldInit(cfg *config.BaseConfig) bool
}

// TestSystemFlagServices_ShouldInit_Table System 开关类服务 ShouldInit 表驱动
//
// 【功能点】Redis/MySQL/ES/Etcd/RabbitMQ 仅当对应 System 开关为 true 时 ShouldInit 为 true
// 【测试流程】
// 1. 对每项服务在默认配置下断言 false
// 2. 打开对应 System 字段后断言 true
func TestSystemFlagServices_ShouldInit_Table(t *testing.T) {
	tests := []struct {
		name   string
		svc    systemFlagShouldIniter
		enable func(*config.BaseConfig)
	}{
		{
			name: "RedisService",
			svc:  &RedisService{},
			enable: func(c *config.BaseConfig) {
				c.System.UseRedis = true
			},
		},
		{
			name: "MySQLService",
			svc:  &MySQLService{},
			enable: func(c *config.BaseConfig) {
				c.System.UseMysql = true
			},
		},
		{
			name: "ElasticsearchService",
			svc:  &ElasticsearchService{},
			enable: func(c *config.BaseConfig) {
				c.System.UseEs = true
			},
		},
		{
			name: "EtcdService",
			svc:  &EtcdService{},
			enable: func(c *config.BaseConfig) {
				c.System.UseEtcd = true
			},
		},
		{
			name: "RabbitMQService",
			svc:  NewRabbitMQService(nil, nil),
			enable: func(c *config.BaseConfig) {
				c.System.UseRabbitMQ = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_默认关闭", func(t *testing.T) {
			assert.False(t, tt.svc.ShouldInit(&config.BaseConfig{}))
		})
		t.Run(tt.name+"_开关开启", func(t *testing.T) {
			cfg := &config.BaseConfig{}
			tt.enable(cfg)
			assert.True(t, tt.svc.ShouldInit(cfg))
		})
	}
}

// TestScheduleService_ShouldInit_Table 定时任务 ShouldInit 表驱动
//
// 【功能点】同时满足 UseSchedule 与非空任务列表时才应初始化
// 【测试流程】
// 1. 组合「开关 × 任务列表是否为空」四种情形
// 2. 断言 ShouldInit 仅在为 true 且列表非空时为 true
func TestScheduleService_ShouldInit_Table(t *testing.T) {
	nonEmpty := []config.ScheduleInfo{{Cron: "@daily", Cmd: func() {}}}
	tests := []struct {
		name string
		svc  *ScheduleService
		cfg  *config.BaseConfig
		want bool
	}{
		{
			name: "默认关闭且无任务",
			svc:  NewScheduleService(nil),
			cfg:  &config.BaseConfig{},
			want: false,
		},
		{
			name: "开启调度但任务列表为空",
			svc:  NewScheduleService(nil),
			cfg: func() *config.BaseConfig {
				c := &config.BaseConfig{}
				c.System.UseSchedule = true
				return c
			}(),
			want: false,
		},
		{
			name: "有任务但未开启调度",
			svc:  NewScheduleService(nonEmpty),
			cfg:  &config.BaseConfig{},
			want: false,
		},
		{
			name: "开启调度且任务非空",
			svc:  NewScheduleService(nonEmpty),
			cfg: func() *config.BaseConfig {
				c := &config.BaseConfig{}
				c.System.UseSchedule = true
				return c
			}(),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.svc.ShouldInit(tt.cfg))
		})
	}
}

// TestRedisService_Init_Table_ConfigValidation Redis Init 配置校验
//
// 【功能点】在未提供 Redis 单实例与列表配置时 Init 应返回配置错误且不触发外部连接
// 【测试流程】
// 1. 备份并恢复 app.BaseConfig
// 2. 设置 Redis、RedisList 均为空
// 3. 调用 Init 断言错误文案包含预期中文提示
func TestRedisService_Init_Table_ConfigValidation(t *testing.T) {
	orig := app.BaseConfig
	defer func() {
		app.BaseConfig = orig
	}()

	svc := &RedisService{}
	tests := []struct {
		name string
		cfg  config.BaseConfig
	}{
		{"完全空配置", config.BaseConfig{}},
		{"Redis 与 RedisList 仍为空", config.BaseConfig{System: config.SystemInfo{UseRedis: true}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.BaseConfig = tt.cfg
			err := svc.Init(context.Background())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "未找到有效的Redis配置")
		})
	}
}

// TestMySQLService_Init_Table_ConfigValidation MySQL Init 配置校验
//
// 【功能点】Db、DbList、DbResolvers 均为空时 Init 应返回数据库配置缺失错误
// 【测试流程】
// 1. 备份 app.BaseConfig
// 2. 写入无数据库相关字段的配置
// 3. 调用 Init 断言错误信息
func TestMySQLService_Init_Table_ConfigValidation(t *testing.T) {
	orig := app.BaseConfig
	defer func() {
		app.BaseConfig = orig
	}()

	app.BaseConfig = config.BaseConfig{}
	err := (&MySQLService{}).Init(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到有效的数据库配置")
}

// TestElasticsearchService_Init_Table_ConfigValidation ES Init 配置校验
//
// 【功能点】Es 配置指针为 nil 时不应继续初始化客户端
// 【测试流程】
// 1. 备份 app.BaseConfig
// 2. 设置 Es 为 nil，可选打开 UseEs（Init 仍应先校验指针）
// 3. 断言返回 Elasticsearch 配置缺失错误
func TestElasticsearchService_Init_Table_ConfigValidation(t *testing.T) {
	orig := app.BaseConfig
	defer func() {
		app.BaseConfig = orig
	}()

	tests := []struct {
		name string
		cfg  config.BaseConfig
	}{
		{"Es 为 nil", config.BaseConfig{}},
		{"开启 UseEs 但 Es 仍为 nil", func() config.BaseConfig {
			c := config.BaseConfig{}
			c.System.UseEs = true
			c.Es = nil
			return c
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.BaseConfig = tt.cfg
			err := (&ElasticsearchService{}).Init(context.Background())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "未找到有效的Elasticsearch配置")
		})
	}
}

// TestEtcdService_Init_Table_ConfigValidation Etcd Init 配置校验
//
// 【功能点】Etcd 配置指针为 nil 时返回配置缺失错误
// 【测试流程】
// 1. 备份 app.BaseConfig 并写入 Etcd=nil
// 2. 调用 Init 校验错误文案
func TestEtcdService_Init_Table_ConfigValidation(t *testing.T) {
	orig := app.BaseConfig
	defer func() {
		app.BaseConfig = orig
	}()

	tests := []struct {
		name string
		cfg  config.BaseConfig
	}{
		{"Etcd 为 nil", config.BaseConfig{}},
		{"开启 UseEtcd 但 Etcd 未配置", func() config.BaseConfig {
			c := config.BaseConfig{}
			c.System.UseEtcd = true
			return c
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app.BaseConfig = tt.cfg
			err := (&EtcdService{}).Init(context.Background())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "未找到有效的Etcd配置")
		})
	}
}

// TestScheduleService_Init_Table_CronValidation 定时任务 Init Cron 校验
//
// 【功能点】非法 Cron 应返回错误；合法 Cron 应成功启动并可安全 Close
// 【测试流程】
// 1. 使用无效 Cron 字符串调用 Init，期望 Error
// 2. 使用合法 Cron（@daily）Init 成功后调用 Close 清理
func TestScheduleService_Init_Table_CronValidation(t *testing.T) {
	t.Run("非法Cron返回错误", func(t *testing.T) {
		svc := NewScheduleService([]config.ScheduleInfo{
			{Cron: "!!!invalid-cron!!!", Cmd: func() {}},
		})
		err := svc.Init(context.Background())
		assert.Error(t, err)
	})

	t.Run("合法Cron可启动并关闭", func(t *testing.T) {
		svc := NewScheduleService([]config.ScheduleInfo{
			{Cron: "@daily", Cmd: func() {}},
		})
		err := svc.Init(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, svc.cron)
		assert.NoError(t, svc.Close(context.Background()))
	})
}

// TestRedisService_Close_Table_NoClient Redis Close 无连接分支
//
// 【功能点】全局 Redis 客户端为 nil 时 Close 应快速返回且无错误
// 【测试流程】
// 1. 备份 app.Redis 并置为 nil
// 2. 调用 Close 断言 nil error
func TestRedisService_Close_Table_NoClient(t *testing.T) {
	orig := app.Redis
	defer func() {
		app.Redis = orig
	}()

	app.Redis = nil
	assert.NoError(t, (&RedisService{}).Close(context.Background()))
}

// TestEtcdService_Close_Table_NoClient Etcd Close 无连接分支
//
// 【功能点】全局 Etcd 客户端为 nil 时不尝试关闭连接
// 【测试流程】
// 1. 备份 app.Etcd 并置 nil
// 2. 调用 Close 断言无错误
func TestEtcdService_Close_Table_NoClient(t *testing.T) {
	orig := app.Etcd
	defer func() {
		app.Etcd = orig
	}()

	app.Etcd = nil
	assert.NoError(t, (&EtcdService{}).Close(context.Background()))
}

// TestElasticsearchService_Close_AlwaysNilError Elasticsearch Close 行为
//
// 【功能点】Close 当前实现恒返回 nil（无需外部连接）
// 【测试流程】直接调用 Close 断言成功
func TestElasticsearchService_Close_AlwaysNilError(t *testing.T) {
	assert.NoError(t, (&ElasticsearchService{}).Close(context.Background()))
}

// TestRedisService_HealthCheck_Table_NoClient Redis HealthCheck 未初始化分支
//
// 【功能点】app.Redis 为 nil 时应返回明确错误
// 【测试流程】备份 Redis、置 nil、调用 HealthCheck
func TestRedisService_HealthCheck_Table_NoClient(t *testing.T) {
	orig := app.Redis
	defer func() {
		app.Redis = orig
	}()

	app.Redis = nil
	err := (&RedisService{}).HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis未初始化")
}

// TestMySQLService_HealthCheck_Table_NoClient MySQL HealthCheck 未初始化分支
//
// 【功能点】DB 为 nil 时返回数据库未初始化错误
// 【测试流程】备份 app.DB、置 nil、调用 HealthCheck
func TestMySQLService_HealthCheck_Table_NoClient(t *testing.T) {
	orig := app.DB
	defer func() {
		app.DB = orig
	}()

	app.DB = nil
	err := (&MySQLService{}).HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数据库未初始化")
}

// TestElasticsearchService_HealthCheck_Table_NoClient ES HealthCheck 未初始化分支
//
// 【功能点】ES 客户端为 nil 时不发起网络请求
// 【测试流程】备份 app.ES、置 nil、断言错误文案
func TestElasticsearchService_HealthCheck_Table_NoClient(t *testing.T) {
	orig := app.ES
	defer func() {
		app.ES = orig
	}()

	app.ES = nil
	err := (&ElasticsearchService{}).HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "elasticsearch未初始化")
}

// TestEtcdService_HealthCheck_Table_NoClient Etcd HealthCheck 未初始化分支
//
// 【功能点】Etcd 客户端为 nil 时返回 etcd 未初始化错误
// 【测试流程】备份 app.Etcd、置 nil、调用 HealthCheck
func TestEtcdService_HealthCheck_Table_NoClient(t *testing.T) {
	orig := app.Etcd
	defer func() {
		app.Etcd = orig
	}()

	app.Etcd = nil
	err := (&EtcdService{}).HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "etcd未初始化")
}

// TestLifecycleService_InterfaceAssignment_Table 生命周期接口兼容性表驱动
//
// 【功能点】验证各内置服务均可赋值给 lifecycle.Service，并在运行时可读名称
// 【测试流程】
// 1. 构造 []lifecycle.Service 包含 logger 至 rabbitmq
// 2. 逐实例断言 Name() 非空
func TestLifecycleService_InterfaceAssignment_Table(t *testing.T) {
	svcs := []lifecycle.Service{
		&LoggerService{},
		&TracingService{},
		&RedisService{},
		&MySQLService{},
		&ElasticsearchService{},
		&EtcdService{},
		NewScheduleService(nil),
		NewRabbitMQService(nil, nil),
	}

	for _, svc := range svcs {
		t.Run(svc.Name(), func(t *testing.T) {
			assert.NotEmpty(t, svc.Name())
		})
	}
}
