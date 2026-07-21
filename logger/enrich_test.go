// Package logger 字段注入测试
//
// ==================== 测试说明 ====================
// 验证 EnrichFields 对 resource 与 fieldMap 的处理。
//
// 测试覆盖内容：
// 1. resource 非空字段注入
// 2. fieldMap 重命名
//
// 运行测试：go test -v ./logger/... -run TestEnrichFields
// ==================================================
package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zzsen/gin_core/model/config"
)

// TestEnrichFields_InjectsResource 测试 resource 字段注入
//
// 【功能点】非空 ServiceName/Env/Host/Version 写入 fields
// 【测试流程】
// 1. 传入 base 与 resource
// 2. 断言公共字段存在且保留原有 userId
func TestEnrichFields_InjectsResource(t *testing.T) {
	out := EnrichFields(map[string]any{"userId": 1}, &config.LogResourceConfig{
		ServiceName: "svc", Env: "dev", Host: "h1", Version: "1.0.0",
	}, nil)
	assert.Equal(t, "svc", out["serviceName"])
	assert.Equal(t, "dev", out["env"])
	assert.Equal(t, "h1", out["host"])
	assert.Equal(t, "1.0.0", out["version"])
	assert.Equal(t, 1, out["userId"])
}

// TestEnrichFields_FieldMapRenames 测试 fieldMap 重命名
//
// 【功能点】映射存在的键被重命名，原键删除
// 【测试流程】
// 1. serviceName → service
// 2. 断言新键存在、旧键不存在
func TestEnrichFields_FieldMapRenames(t *testing.T) {
	out := EnrichFields(map[string]any{"serviceName": "svc"}, &config.LogResourceConfig{ServiceName: "svc"}, map[string]string{
		"serviceName": "service",
	})
	assert.Equal(t, "svc", out["service"])
	_, hasOld := out["serviceName"]
	assert.False(t, hasOld)
}
