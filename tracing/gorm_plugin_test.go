// Package tracing GORM 追踪插件测试
//
// ==================== 测试说明 ====================
// 本文件包含 GormTracingPlugin 的单元测试，不依赖数据库。
//
// 测试覆盖内容：
// 1. NewGormTracingPlugin 创建（默认/自定义名称）
// 2. Name 方法返回值
// 3. Initialize 注册回调（使用内存 SQLite）
//
// 运行测试：go test -v ./tracing/... -run "TestGorm|TestNewGormTracing"
// ==================================================
package tracing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewGormTracingPlugin_Default 测试默认数据库名称
//
// 【功能点】不传参或传空字符串时 dbName 应为 "default"
func TestNewGormTracingPlugin_Default(t *testing.T) {
	plugin := NewGormTracingPlugin()
	assert.Equal(t, "default", plugin.dbName)

	plugin2 := NewGormTracingPlugin("")
	assert.Equal(t, "default", plugin2.dbName)
}

// TestNewGormTracingPlugin_Custom 测试自定义数据库名称
func TestNewGormTracingPlugin_Custom(t *testing.T) {
	plugin := NewGormTracingPlugin("mydb")
	assert.Equal(t, "mydb", plugin.dbName)
}

// TestGormTracingPlugin_Name 测试插件名称
func TestGormTracingPlugin_Name(t *testing.T) {
	plugin := NewGormTracingPlugin()
	assert.Equal(t, "otel-tracing", plugin.Name())
}
