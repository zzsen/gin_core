// Package configcenter 白名单 Apply 测试
//
// ==================== 测试说明 ====================
// 验证 log.level / rateLimit 白名单应用与冷字段忽略。
//
// 运行测试：go test ./configcenter/ -count=1 -run ApplyWhitelist -v
// ==================================================

package configcenter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// TestApplyWhitelist_LogLevel 应用日志级别
func TestApplyWhitelist_LogLevel(t *testing.T) {
	_ = logger.SetLevel("info")
	prev := &config.BaseConfig{Log: config.LoggersConfig{Loggers: []config.LoggerConfig{{Level: "info"}}}}
	next := &config.BaseConfig{Log: config.LoggersConfig{Loggers: []config.LoggerConfig{{Level: "warn"}}}}
	require.NoError(t, ApplyWhitelist(prev, next, []string{"log.level"}))
	assert.Equal(t, "warn", prev.Log.Loggers[0].Level)
	assert.Equal(t, "warning", logger.GetLevel()) // logrus 规范化 warn → warning
}

// TestApplyWhitelist_IgnoresServicePort 冷字段忽略
func TestApplyWhitelist_IgnoresServicePort(t *testing.T) {
	prev := &config.BaseConfig{
		Service: config.ServiceInfo{Port: 8055},
		Log:     config.LoggersConfig{Loggers: []config.LoggerConfig{{Level: "info"}}},
	}
	next := &config.BaseConfig{
		Service: config.ServiceInfo{Port: 9999},
		Log:     config.LoggersConfig{Loggers: []config.LoggerConfig{{Level: "debug"}}},
	}
	require.NoError(t, ApplyWhitelist(prev, next, []string{"log.level"}))
	assert.Equal(t, 8055, prev.Service.Port)
	assert.Equal(t, "debug", prev.Log.Loggers[0].Level)
}

// TestApplyWhitelist_RateLimit 覆盖限流配置
func TestApplyWhitelist_RateLimit(t *testing.T) {
	prev := &config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 100}}
	next := &config.BaseConfig{RateLimit: config.RateLimitConfig{DefaultRate: 10, Enabled: true}}
	require.NoError(t, ApplyWhitelist(prev, next, []string{"rateLimit.*"}))
	assert.Equal(t, 10, prev.RateLimit.DefaultRate)
	assert.True(t, prev.RateLimit.Enabled)
}

// TestApplyWhitelist_NilArgs 空指针报错
func TestApplyWhitelist_NilArgs(t *testing.T) {
	require.Error(t, ApplyWhitelist(nil, &config.BaseConfig{}, nil))
	require.Error(t, ApplyWhitelist(&config.BaseConfig{}, nil, nil))
}
