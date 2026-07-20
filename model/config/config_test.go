// Package config 配置模型测试
//
// ==================== 测试说明 ====================
// 本文件包含各配置模型 Getter 方法的单元测试。
//
// 测试覆盖内容：
// 1. RateLimitConfig Getter 方法（默认值/自定义值）
// 2. RateLimitRule Getter 方法
// 3. CORSConfig Getter 方法
// 4. ServiceInfo.GetShutdownTimeout
// 5. DbInfo.Dsn（默认字符集/时区、自定义值）
// 6. DbResolver.IsValid / SourceConfigs / ReplicaConfigs
// 7. DbResolvers.IsValid / DefaultConfig
// 8. ScheduleInfo.GetFuncInfo
//
// 运行测试：go test -v ./model/config/... -run "TestConfig|TestRateLimit|TestCORS|TestService|TestDb|TestSchedule"
// ==================================================
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ==================== RateLimitConfig ====================

func TestRateLimitConfig_GetDefaultRate(t *testing.T) {
	assert.Equal(t, 100, (&RateLimitConfig{}).GetDefaultRate())
	assert.Equal(t, 50, (&RateLimitConfig{DefaultRate: 50}).GetDefaultRate())
}

func TestRateLimitConfig_GetDefaultBurst(t *testing.T) {
	assert.Equal(t, 200, (&RateLimitConfig{}).GetDefaultBurst())
	assert.Equal(t, 300, (&RateLimitConfig{DefaultBurst: 300}).GetDefaultBurst())
}

func TestRateLimitConfig_GetStore(t *testing.T) {
	assert.Equal(t, "memory", (&RateLimitConfig{}).GetStore())
	assert.Equal(t, "redis", (&RateLimitConfig{Store: "redis"}).GetStore())
}

func TestRateLimitConfig_GetMessage(t *testing.T) {
	assert.Equal(t, "请求过于频繁，请稍后再试", (&RateLimitConfig{}).GetMessage())
	assert.Equal(t, "自定义消息", (&RateLimitConfig{Message: "自定义消息"}).GetMessage())
}

func TestRateLimitConfig_GetCleanupInterval(t *testing.T) {
	assert.Equal(t, 60, (&RateLimitConfig{}).GetCleanupInterval())
	assert.Equal(t, 120, (&RateLimitConfig{CleanupInterval: 120}).GetCleanupInterval())
}

// ==================== RateLimitRule ====================

func TestRateLimitRule_GetRate(t *testing.T) {
	assert.Equal(t, 0, (&RateLimitRule{}).GetRate())
	assert.Equal(t, 10, (&RateLimitRule{Rate: 10}).GetRate())
}

func TestRateLimitRule_GetBurst(t *testing.T) {
	assert.Equal(t, 0, (&RateLimitRule{}).GetBurst())
	assert.Equal(t, 20, (&RateLimitRule{Rate: 10}).GetBurst())
	assert.Equal(t, 50, (&RateLimitRule{Burst: 50}).GetBurst())
}

func TestRateLimitRule_GetKeyType(t *testing.T) {
	assert.Equal(t, "ip", (&RateLimitRule{}).GetKeyType())
	assert.Equal(t, "user", (&RateLimitRule{KeyType: "user"}).GetKeyType())
	assert.Equal(t, "global", (&RateLimitRule{KeyType: "global"}).GetKeyType())
}

// ==================== CORSConfig ====================

func TestCORSConfig_GetAllowOrigins(t *testing.T) {
	assert.Equal(t, []string{"*"}, (&CORSConfig{}).GetAllowOrigins())
	assert.Equal(t, []string{"http://localhost"}, (&CORSConfig{AllowOrigins: []string{"http://localhost"}}).GetAllowOrigins())
}

func TestCORSConfig_GetAllowMethods(t *testing.T) {
	defaults := (&CORSConfig{}).GetAllowMethods()
	assert.Contains(t, defaults, "GET")
	assert.Contains(t, defaults, "POST")
	assert.Contains(t, defaults, "OPTIONS")
}

func TestCORSConfig_GetAllowHeaders(t *testing.T) {
	defaults := (&CORSConfig{}).GetAllowHeaders()
	assert.Contains(t, defaults, "Content-Type")
	assert.Contains(t, defaults, "Authorization")
}

func TestCORSConfig_GetMaxAge(t *testing.T) {
	assert.Equal(t, 86400, (&CORSConfig{}).GetMaxAge())
	assert.Equal(t, 3600, (&CORSConfig{MaxAge: 3600}).GetMaxAge())
}

// ==================== ServiceInfo ====================

func TestServiceInfo_GetShutdownTimeout(t *testing.T) {
	assert.Equal(t, 5, (&ServiceInfo{}).GetShutdownTimeout())
	assert.Equal(t, 10, (&ServiceInfo{ShutdownTimeout: 10}).GetShutdownTimeout())
}

// ==================== DbInfo ====================

func TestDbInfo_Dsn_Defaults(t *testing.T) {
	db := &DbInfo{Username: "root", Password: "pass", Host: "127.0.0.1", Port: 3306, DBName: "test"}
	dsn := db.Dsn()
	assert.Contains(t, dsn, "charset=utf8mb4")
	assert.Contains(t, dsn, "loc=Local")
	assert.Contains(t, dsn, "root:pass@tcp(127.0.0.1:3306)/test")
}

func TestDbInfo_Dsn_Custom(t *testing.T) {
	db := &DbInfo{Username: "u", Password: "p", Host: "db.local", Port: 3307, DBName: "mydb", Charset: "utf8", Loc: "UTC"}
	dsn := db.Dsn()
	assert.Contains(t, dsn, "charset=utf8")
	assert.Contains(t, dsn, "loc=UTC")
	assert.Contains(t, dsn, "db.local:3307")
}

// ==================== DbResolver ====================

func TestDbResolver_IsValid(t *testing.T) {
	assert.False(t, (&DbResolver{}).IsValid())
	assert.True(t, (&DbResolver{Sources: []DbInfo{{Host: "127.0.0.1"}}}).IsValid())
}

func TestDbResolver_SourceConfigs(t *testing.T) {
	r := DbResolver{Sources: []DbInfo{
		{Username: "root", Host: "127.0.0.1", Port: 3306, DBName: "db1"},
	}}
	configs := r.SourceConfigs()
	assert.Len(t, configs, 1)
}

func TestDbResolver_ReplicaConfigs(t *testing.T) {
	r := DbResolver{Replicas: []DbInfo{
		{Username: "root", Host: "127.0.0.1", Port: 3306, DBName: "db1"},
		{Username: "root", Host: "127.0.0.1", Port: 3307, DBName: "db2"},
	}}
	configs := r.ReplicaConfigs()
	assert.Len(t, configs, 2)
}

func TestDbResolvers_IsValid(t *testing.T) {
	valid := DbResolvers{{Sources: []DbInfo{{Host: "h"}}}}
	assert.True(t, valid.IsValid())

	invalid := DbResolvers{{Sources: []DbInfo{{Host: "h"}}}, {}}
	assert.False(t, invalid.IsValid())
}

func TestDbResolvers_DefaultConfig(t *testing.T) {
	resolvers := DbResolvers{{Sources: []DbInfo{{Host: "first"}}}}
	assert.Equal(t, "first", resolvers.DefaultConfig().Host)
}

// ==================== ScheduleInfo ====================

func TestScheduleInfo_GetFuncInfo(t *testing.T) {
	s := ScheduleInfo{Cron: "@every 5s", Cmd: func() {}}
	info := s.GetFuncInfo()
	assert.NotEmpty(t, info)
}

// TestScheduleInfo_GetFuncInfo_NilCmd 测试 Cmd 为 nil 时
func TestScheduleInfo_GetFuncInfo_NilCmd(t *testing.T) {
	s := ScheduleInfo{Cron: "@every 5s"}
	info := s.GetFuncInfo()
	assert.Empty(t, info)
}

// ==================== LoggersConfig ====================

func TestLoggersConfig_ToDbLoggerConfig(t *testing.T) {
	cfg := LoggersConfig{
		FilePath: "/var/log/app",
		Loggers: []LoggerConfig{
			{FileName: "info", FilePath: "/var/log/info"},
			{FileName: "error", FilePath: ""},
		},
	}
	dbCfg := cfg.ToDbLoggerConfig()
	assert.Equal(t, "/var/log/appDB", dbCfg.FilePath)
	assert.Equal(t, "infoDB", dbCfg.Loggers[0].FileName)
	assert.Equal(t, "/var/log/infoDB", dbCfg.Loggers[0].FilePath)
	assert.Equal(t, "errorDB", dbCfg.Loggers[1].FileName)
	assert.Empty(t, dbCfg.Loggers[1].FilePath)
}

func TestLoggersConfig_ToDbLoggerConfig_EmptyPath(t *testing.T) {
	cfg := LoggersConfig{
		FilePath: "",
		Loggers:  []LoggerConfig{{FileName: "", FilePath: ""}},
	}
	dbCfg := cfg.ToDbLoggerConfig()
	assert.Empty(t, dbCfg.FilePath)
	assert.Empty(t, dbCfg.Loggers[0].FileName)
}

// TestLoggersConfig_ToDbLoggerConfig_PreservesFormat 测试 Format 透传
//
// 【功能点】ToDbLoggerConfig 不修改全局/按级 Format 字段
// 【测试流程】
// 1. 设置全局 Format=json、级别 Format=text
// 2. 调用 ToDbLoggerConfig
// 3. Format 保持不变，FileName 仍加 DB 后缀
func TestLoggersConfig_ToDbLoggerConfig_PreservesFormat(t *testing.T) {
	cfg := LoggersConfig{
		FilePath: "/var/log/app",
		Format:   "json",
		Loggers: []LoggerConfig{
			{FileName: "info", FilePath: "/var/log/info", Format: "text"},
		},
	}
	dbCfg := cfg.ToDbLoggerConfig()
	assert.Equal(t, "json", dbCfg.Format)
	assert.Equal(t, "text", dbCfg.Loggers[0].Format)
	assert.Equal(t, "infoDB", dbCfg.Loggers[0].FileName)
}

// TestLoggersConfig_YAMLUnmarshalFormat YAML 反序列化 format 字段
//
// 【功能点】全局与按级 format 可从 YAML 正确映射
// 【测试流程】
// 1. 反序列化含 format / loggers[].format 的 YAML
// 2. 字段值与 YAML 一致
func TestLoggersConfig_YAMLUnmarshalFormat(t *testing.T) {
	const raw = `
filePath: "./log"
format: json
loggers:
  - level: info
    fileName: info
    format: text
`
	var cfg LoggersConfig
	err := yaml.Unmarshal([]byte(raw), &cfg)
	require.NoError(t, err)
	assert.Equal(t, "json", cfg.Format)
	require.Len(t, cfg.Loggers, 1)
	assert.Equal(t, "info", cfg.Loggers[0].Level)
	assert.Equal(t, "text", cfg.Loggers[0].Format)
}

// ==================== CORSConfig 自定义值分支 ====================

func TestCORSConfig_GetAllowMethods_Custom(t *testing.T) {
	cfg := &CORSConfig{AllowMethods: []string{"GET", "POST"}}
	assert.Equal(t, []string{"GET", "POST"}, cfg.GetAllowMethods())
}

func TestCORSConfig_GetAllowHeaders_Custom(t *testing.T) {
	cfg := &CORSConfig{AllowHeaders: []string{"X-Custom"}}
	assert.Equal(t, []string{"X-Custom"}, cfg.GetAllowHeaders())
}
