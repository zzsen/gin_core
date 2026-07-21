// Package logger Context 关联字段测试
//
// ==================== 测试说明 ====================
// 验证 WithContext 注入 traceId，以及旧 Info API 兼容。
//
// 运行测试：go test -v ./logger/... -run "TestWithContext|TestLegacyInfo"
// ==================================================
package logger

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// TestWithContext_InjectsTraceID WithContext 注入 traceId
func TestWithContext_InjectsTraceID(t *testing.T) {
	var buf bytes.Buffer
	Logger = logrus.New()
	Logger.SetOutput(&buf)
	Logger.SetFormatter(&logrus.JSONFormatter{})
	ctx := ContextWithTraceID(context.Background(), "tid-1")
	WithContext(ctx).Info("hello")
	assert.Contains(t, buf.String(), "tid-1")
}

// TestLegacyInfo_StillWorks 旧 Info API 无需 context
func TestLegacyInfo_StillWorks(t *testing.T) {
	Logger = logrus.New()
	Logger.SetOutput(io.Discard)
	Info("ok")
}
