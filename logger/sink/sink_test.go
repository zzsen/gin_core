// Package sink Sink 接口测试
//
// ==================== 测试说明 ====================
// 验证 FormattedEntry 与 Sink 接口可由假实现满足。
//
// 测试覆盖内容：
// 1. fakeSink 实现 Sink
// 2. Send / Close 基本行为
//
// 运行测试：go test -v ./logger/sink/... -run TestSinkInterface
// ==================================================
package sink

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSinkInterface_FakeImplements 测试 Sink 接口可由假实现满足
//
// 【功能点】FormattedEntry + Sink 契约
// 【测试流程】
// 1. 将 fakeSink 赋给 Sink 接口
// 2. 调用 Send / Close 无错误
func TestSinkInterface_FakeImplements(t *testing.T) {
	var s Sink = &fakeSink{}
	assert.Equal(t, "fake", s.Name())
	require.NoError(t, s.Send(context.Background(), []FormattedEntry{{Message: "hi"}}))
	require.NoError(t, s.Close(context.Background()))
}

type fakeSink struct{ sent []FormattedEntry }

func (f *fakeSink) Name() string     { return "fake" }
func (f *fakeSink) MinLevel() string { return "" }
func (f *fakeSink) Send(_ context.Context, entries []FormattedEntry) error {
	f.sent = append(f.sent, entries...)
	return nil
}
func (f *fakeSink) Close(context.Context) error { return nil }
