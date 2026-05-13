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

	"github.com/stretchr/testify/assert"
	"github.com/zzsen/gin_core/core/lifecycle"
	"github.com/zzsen/gin_core/model/config"
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
