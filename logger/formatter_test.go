// Package logger Formatter 解析与工厂测试
//
// ==================== 测试说明 ====================
// 覆盖 resolveFormat / newFormatter 的默认、大小写、非法回退与 JSON。
//
// 测试覆盖内容：
// 1. normalizeFormat / resolveFormat 优先级
// 2. newFormatter 返回 TextFormatter / JSONFormatter
// 3. 时间戳布局一致
//
// 运行测试：go test -v ./logger/... -run Format
// ==================================================
package logger

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveFormat 测试格式解析优先级
//
// 【功能点】级别 > 全局 > text；非法值视为空
// 【测试流程】
// 1. 全空 → text
// 2. 仅全局 json → json
// 3. 级别覆盖全局
// 4. 非法级别回退全局 / text
func TestResolveFormat(t *testing.T) {
	assert.Equal(t, "text", resolveFormat("", ""))
	assert.Equal(t, "json", resolveFormat("json", ""))
	assert.Equal(t, "json", resolveFormat("JSON", ""))
	assert.Equal(t, "text", resolveFormat("json", "text"))
	assert.Equal(t, "json", resolveFormat("text", "JSON"))
	assert.Equal(t, "json", resolveFormat("json", "bogus"))
	assert.Equal(t, "text", resolveFormat("bogus", ""))
	assert.Equal(t, "json", resolveFormat("", " json "))
	assert.Equal(t, "json", resolveFormat("json", "   ")) // 空白级别视为未配置，用全局
}

// TestNewFormatter 测试 Formatter 工厂
//
// 【功能点】text/空 → TextFormatter；json → JSONFormatter；时间戳布局一致
// 【测试流程】
// 1. 空与 TEXT → TextFormatter 选项
// 2. json → JSONFormatter
// 3. 未知值 → TextFormatter
func TestNewFormatter(t *testing.T) {
	const ts = "2006-01-02 15:04:05"

	tf, ok := newFormatter("").(*logrus.TextFormatter)
	require.True(t, ok)
	assert.True(t, tf.FullTimestamp)
	assert.Equal(t, ts, tf.TimestampFormat)
	assert.True(t, tf.DisableLevelTruncation)

	tf2, ok := newFormatter("TEXT").(*logrus.TextFormatter)
	require.True(t, ok)
	assert.Equal(t, ts, tf2.TimestampFormat)

	jf, ok := newFormatter("json").(*logrus.JSONFormatter)
	require.True(t, ok)
	assert.Equal(t, ts, jf.TimestampFormat)

	_, ok = newFormatter("unknown").(*logrus.TextFormatter)
	assert.True(t, ok)
}
