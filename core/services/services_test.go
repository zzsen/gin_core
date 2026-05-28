// Package services 内置服务单元测试
//
// ==================== 测试说明 ====================
// 本文件包含内置服务（非 RabbitMQ）的单元测试，不需要外部依赖。
//
// 测试覆盖内容：
// 1. LoggerService: Name/Priority/Dependencies/ShouldInit
// 2. TracingService: Name/Priority/Dependencies/ShouldInit
// 3. RedisService: Name/Priority/Dependencies/ShouldInit
// 4. MySQLService: Name/Priority/Dependencies/ShouldInit
// 5. ElasticsearchService: Name/Priority/Dependencies/ShouldInit
// 6. EtcdService: Name/Priority/Dependencies/ShouldInit
// 7. ScheduleService: Name/Priority/Dependencies/ShouldInit/SetScheduleList
// 8. Service 接口实现验证
//
// 运行测试：go test -v ./core/services/... -run "TestLogger|TestTracing|TestRedis|TestMySQL|TestElasticsearch|TestEtcd|TestSchedule|TestServicesImplement"
// ==================================================
package services

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/model/config"

	"gorm.io/gorm"
)

// ==================== LoggerService ====================

func TestLoggerService_Name(t *testing.T) {
	assert.Equal(t, "logger", (&LoggerService{}).Name())
}

func TestLoggerService_Priority(t *testing.T) {
	assert.Equal(t, 0, (&LoggerService{}).Priority())
}

func TestLoggerService_Dependencies(t *testing.T) {
	assert.Nil(t, (&LoggerService{}).Dependencies())
}

func TestLoggerService_ShouldInit(t *testing.T) {
	assert.True(t, (&LoggerService{}).ShouldInit(&config.BaseConfig{}))
}

// ==================== TracingService ====================

func TestTracingService_Name(t *testing.T) {
	assert.Equal(t, "tracing", (&TracingService{}).Name())
}

func TestTracingService_Priority(t *testing.T) {
	assert.Equal(t, 5, (&TracingService{}).Priority())
}

func TestTracingService_Dependencies(t *testing.T) {
	assert.Equal(t, []string{"logger"}, (&TracingService{}).Dependencies())
}

func TestTracingService_ShouldInit_Enabled(t *testing.T) {
	cfg := &config.BaseConfig{Tracing: &config.TracingConfig{Enabled: true}}
	assert.True(t, (&TracingService{}).ShouldInit(cfg))
}

func TestTracingService_ShouldInit_Disabled(t *testing.T) {
	assert.False(t, (&TracingService{}).ShouldInit(&config.BaseConfig{}))
}

// ==================== RedisService ====================

func TestRedisService_Name(t *testing.T) {
	assert.Equal(t, "redis", (&RedisService{}).Name())
}

func TestRedisService_Priority(t *testing.T) {
	assert.Equal(t, 10, (&RedisService{}).Priority())
}

func TestRedisService_Dependencies(t *testing.T) {
	assert.Equal(t, []string{"logger"}, (&RedisService{}).Dependencies())
}

func TestRedisService_ShouldInit(t *testing.T) {
	assert.False(t, (&RedisService{}).ShouldInit(&config.BaseConfig{}))
	cfg := &config.BaseConfig{}
	cfg.System.UseRedis = true
	assert.True(t, (&RedisService{}).ShouldInit(cfg))
}

// ==================== MySQLService ====================

func TestMySQLService_Name(t *testing.T) {
	assert.Equal(t, "mysql", (&MySQLService{}).Name())
}

func TestMySQLService_Priority(t *testing.T) {
	assert.Equal(t, 10, (&MySQLService{}).Priority())
}

func TestMySQLService_Dependencies(t *testing.T) {
	assert.Equal(t, []string{"logger"}, (&MySQLService{}).Dependencies())
}

func TestMySQLService_ShouldInit(t *testing.T) {
	assert.False(t, (&MySQLService{}).ShouldInit(&config.BaseConfig{}))
	cfg := &config.BaseConfig{}
	cfg.System.UseMysql = true
	assert.True(t, (&MySQLService{}).ShouldInit(cfg))
}

// ==================== ElasticsearchService ====================

func TestElasticsearchService_Name(t *testing.T) {
	assert.Equal(t, "elasticsearch", (&ElasticsearchService{}).Name())
}

func TestElasticsearchService_Priority(t *testing.T) {
	assert.Equal(t, 20, (&ElasticsearchService{}).Priority())
}

func TestElasticsearchService_Dependencies(t *testing.T) {
	assert.Equal(t, []string{"logger"}, (&ElasticsearchService{}).Dependencies())
}

func TestElasticsearchService_ShouldInit(t *testing.T) {
	assert.False(t, (&ElasticsearchService{}).ShouldInit(&config.BaseConfig{}))
	cfg := &config.BaseConfig{}
	cfg.System.UseEs = true
	assert.True(t, (&ElasticsearchService{}).ShouldInit(cfg))
}

// ==================== EtcdService ====================

func TestEtcdService_Name(t *testing.T) {
	assert.Equal(t, "etcd", (&EtcdService{}).Name())
}

func TestEtcdService_Priority(t *testing.T) {
	assert.Equal(t, 20, (&EtcdService{}).Priority())
}

func TestEtcdService_Dependencies(t *testing.T) {
	assert.Equal(t, []string{"logger"}, (&EtcdService{}).Dependencies())
}

func TestEtcdService_ShouldInit(t *testing.T) {
	assert.False(t, (&EtcdService{}).ShouldInit(&config.BaseConfig{}))
	cfg := &config.BaseConfig{}
	cfg.System.UseEtcd = true
	assert.True(t, (&EtcdService{}).ShouldInit(cfg))
}

// ==================== ScheduleService ====================

func TestScheduleService_Name(t *testing.T) {
	svc := NewScheduleService(nil)
	assert.Equal(t, "schedule", svc.Name())
}

func TestScheduleService_Priority(t *testing.T) {
	svc := NewScheduleService(nil)
	assert.Equal(t, 100, svc.Priority())
}

func TestScheduleService_Dependencies(t *testing.T) {
	svc := NewScheduleService(nil)
	assert.Equal(t, []string{"logger"}, svc.Dependencies())
}

func TestScheduleService_ShouldInit(t *testing.T) {
	svc := NewScheduleService(nil)
	assert.False(t, svc.ShouldInit(&config.BaseConfig{}))

	cfg := &config.BaseConfig{}
	cfg.System.UseSchedule = true
	// scheduleList 为空时仍返回 false
	assert.False(t, svc.ShouldInit(cfg))

	svcWithList := NewScheduleService([]config.ScheduleInfo{{Cron: "@every 1s", Cmd: func() {}}})
	assert.True(t, svcWithList.ShouldInit(cfg))
}

func TestScheduleService_SetScheduleList(t *testing.T) {
	svc := NewScheduleService(nil)
	list := []config.ScheduleInfo{{Cron: "@every 1s", Cmd: func() {}}}
	svc.SetScheduleList(list)
	assert.Len(t, svc.scheduleList, 1)
}

// ==================== Close 方法测试 ====================

func TestLoggerService_Close(t *testing.T) {
	svc := &LoggerService{}
	assert.NoError(t, svc.Close(context.Background()))
}

func TestScheduleService_Close_NilCron(t *testing.T) {
	svc := NewScheduleService(nil)
	assert.NoError(t, svc.Close(context.Background()))
}

func TestTracingService_Close(t *testing.T) {
	svc := &TracingService{}
	assert.NoError(t, svc.Close(context.Background()))
}

// TestMySQLService_Init_NoDbConfig_ReturnsError 测试 MySQL 服务在无数据库配置时 Init 返回错误
//
// 【功能点】验证 Db、DbList、DbResolvers 均为空时 Init 不走 initialize 链并返回明确错误
// 【测试流程】
// 1. 保存并重置 app.BaseConfig 为无数据库字段的结构体
// 2. 调用 MySQLService.Init
// 3. 断言返回错误且文案包含「未找到有效的数据库配置」
func TestMySQLService_Init_NoDbConfig_ReturnsError(t *testing.T) {
	orig := app.BaseConfig
	defer func() { app.BaseConfig = orig }()

	app.BaseConfig = config.BaseConfig{}
	svc := &MySQLService{}
	err := svc.Init(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到有效的数据库配置")
}

// TestMySQLService_Close_WithSQLite 测试 MySQL 服务 Close 委托 CloseAllDB 关闭内存 SQLite
//
// 【功能点】验证 Close 能关闭已注入的 app.DB（glebarez/sqlite 内存库）
// 【测试流程】
// 1. 保存 app.DB、DBResolver、DBList 并在结束时恢复
// 2. 打开 :memory: SQLite 并赋值给 app.DB
// 3. 调用 MySQLService.Close，断言无错误
func TestMySQLService_Close_WithSQLite(t *testing.T) {
	origDB := app.DB
	origResolver := app.DBResolver
	origList := app.DBList
	defer func() {
		app.DB = origDB
		app.DBResolver = origResolver
		app.DBList = origList
	}()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	app.DB = db
	app.DBResolver = nil
	app.DBList = map[string]*gorm.DB{}

	svc := &MySQLService{}
	assert.NoError(t, svc.Close(context.Background()))
}

// TestMySQLService_HealthCheck_DBNotInitialized 测试 DB 为 nil 时健康检查失败
//
// 【功能点】验证 HealthCheck 在 app.DB 未设置时返回「数据库未初始化」类错误
// 【测试流程】
// 1. 保存并重置 app.DB 为 nil
// 2. 调用 HealthCheck
// 3. 断言返回错误
func TestMySQLService_HealthCheck_DBNotInitialized(t *testing.T) {
	orig := app.DB
	defer func() { app.DB = orig }()

	app.DB = nil
	svc := &MySQLService{}
	err := svc.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "数据库未初始化")
}

// TestMySQLService_HealthCheck_SQLiteOk 测试内存 SQLite 下健康检查 Ping 成功
//
// 【功能点】验证 HealthCheck 对有效 gorm.DB 执行 PingContext 成功
// 【测试流程】
// 1. 保存并替换 app.DB 为内存 SQLite 实例
// 2. 调用 HealthCheck
// 3. 断言无错误
func TestMySQLService_HealthCheck_SQLiteOk(t *testing.T) {
	orig := app.DB
	defer func() { app.DB = orig }()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	app.DB = db

	svc := &MySQLService{}
	assert.NoError(t, svc.HealthCheck(context.Background()))
}

// TestRedisService_Close_Miniredis 测试关闭注入的 miniredis 客户端
//
// 【功能点】验证 RedisService.Close 在非 nil 客户端上调用 Close 成功
// 【测试流程】
// 1. 保存 app.Redis，启动 miniredis 并创建 go-redis Client 赋值给 app.Redis
// 2. 调用 Close
// 3. 断言无错误并恢复全局 Redis
func TestRedisService_Close_Miniredis(t *testing.T) {
	orig := app.Redis
	defer func() { app.Redis = orig }()

	mr := miniredis.RunT(t)
	app.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := &RedisService{}
	assert.NoError(t, svc.Close(context.Background()))
}

// TestRedisService_HealthCheck_NotInitialized 测试 Redis 未初始化时的健康检查
//
// 【功能点】验证 app.Redis 为 nil 时返回明确错误
// 【测试流程】
// 1. 保存并将 app.Redis 置为 nil
// 2. 调用 HealthCheck
// 3. 断言错误包含 redis 未初始化语义
func TestRedisService_HealthCheck_NotInitialized(t *testing.T) {
	orig := app.Redis
	defer func() { app.Redis = orig }()

	app.Redis = nil
	svc := &RedisService{}
	err := svc.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis未初始化")
}

// TestRedisService_HealthCheck_MiniredisOk 测试 miniredis 下 Ping 成功路径
//
// 【功能点】验证 HealthCheck 委托 Redis.Ping 在可用实例上返回成功
// 【测试流程】
// 1. 保存 app.Redis，注入指向 miniredis 的客户端
// 2. 调用 HealthCheck
// 3. 断言无错误
func TestRedisService_HealthCheck_MiniredisOk(t *testing.T) {
	orig := app.Redis
	defer func() { app.Redis = orig }()

	mr := miniredis.RunT(t)
	app.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})

	svc := &RedisService{}
	assert.NoError(t, svc.HealthCheck(context.Background()))
	mr.Close()
}

// TestEtcdService_Close_ClientNil 测试 Etcd 客户端为 nil 时 Close 无副作用
//
// 【功能点】验证 app.Etcd 为 nil 时 Close 直接返回 nil
// 【测试流程】
// 1. 保存并将 app.Etcd 置 nil
// 2. 调用 EtcdService.Close
// 3. 断言无错误
func TestEtcdService_Close_ClientNil(t *testing.T) {
	orig := app.Etcd
	defer func() { app.Etcd = orig }()

	app.Etcd = nil
	svc := &EtcdService{}
	assert.NoError(t, svc.Close(context.Background()))
}

// TestEtcdService_Init_NoEtcdConfig_ReturnsError 测试 Etcd 配置缺失时 Init 错误路径
//
// 【功能点】验证 app.BaseConfig.Etcd 为 nil 时返回配置错误且不调用外部初始化
// 【测试流程】
// 1. 保存并重置 BaseConfig.Etcd 为 nil
// 2. 调用 Init
// 3. 断言错误文案包含 Etcd 配置无效描述
func TestEtcdService_Init_NoEtcdConfig_ReturnsError(t *testing.T) {
	orig := app.BaseConfig
	defer func() { app.BaseConfig = orig }()

	app.BaseConfig = config.BaseConfig{Etcd: nil}
	svc := &EtcdService{}
	err := svc.Init(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到有效的Etcd配置")
}

// TestElasticsearchService_Init_NoEsConfig_ReturnsError 测试 ES 配置为空时 Init 错误路径
//
// 【功能点】验证 BaseConfig.Es 为 nil 时返回错误
// 【测试流程】
// 1. 保存并重置 BaseConfig.Es
// 2. 调用 ElasticsearchService.Init
// 3. 断言错误
func TestElasticsearchService_Init_NoEsConfig_ReturnsError(t *testing.T) {
	orig := app.BaseConfig
	defer func() { app.BaseConfig = orig }()

	app.BaseConfig = config.BaseConfig{Es: nil}
	svc := &ElasticsearchService{}
	err := svc.Init(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到有效的Elasticsearch配置")
}

// TestElasticsearchService_HealthCheck_ESNil 测试 ES 客户端未初始化时的健康检查
//
// 【功能点】验证 app.ES 为 nil 时 HealthCheck 返回 elasticsearch 未初始化错误
// 【测试流程】
// 1. 保存并将 app.ES 置 nil
// 2. 调用 HealthCheck
// 3. 断言错误
func TestElasticsearchService_HealthCheck_ESNil(t *testing.T) {
	orig := app.ES
	defer func() { app.ES = orig }()

	app.ES = nil
	svc := &ElasticsearchService{}
	err := svc.HealthCheck(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "elasticsearch未初始化")
}

// ==================== 接口实现验证 ====================

func TestServicesImplementInterface(t *testing.T) {
	var _ lifecycle.Service = &LoggerService{}
	var _ lifecycle.Service = &TracingService{}
	var _ lifecycle.Service = &RedisService{}
	var _ lifecycle.Service = &MySQLService{}
	var _ lifecycle.Service = &ElasticsearchService{}
	var _ lifecycle.Service = &EtcdService{}
	var _ lifecycle.Service = &ScheduleService{}
}
