// Package logger 远程管线关闭测试
//
// ==================== 测试说明 ====================
// 验证 CloseRemote / Logger 侧 Flush 能刷出 pending 条目。
//
// 运行测试：go test -v ./logger/... -run TestCloseRemote
// ==================================================
package logger

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/logger/sink"
)

type closeRecordingSink struct {
	mu   sync.Mutex
	sent []sink.FormattedEntry
}

func (r *closeRecordingSink) Name() string     { return "rec" }
func (r *closeRecordingSink) MinLevel() string { return "" }
func (r *closeRecordingSink) Send(_ context.Context, entries []sink.FormattedEntry) error {
	r.mu.Lock()
	r.sent = append(r.sent, entries...)
	r.mu.Unlock()
	return nil
}
func (r *closeRecordingSink) Close(context.Context) error { return nil }

// TestCloseRemote_FlushesPipeline CloseRemote 刷出 pending
//
// 【功能点】关闭时 Flush
// 【测试流程】
// 1. SetRemotePipeline + Enqueue
// 2. CloseRemote 后 sink 收到条目
func TestCloseRemote_FlushesPipeline(t *testing.T) {
	fs := &closeRecordingSink{}
	p := sink.NewPipeline(fs, sink.PipelineConfig{QueueSize: 10, BatchSize: 10, FlushInterval: time.Hour})
	SetRemotePipeline(p)
	t.Cleanup(func() { _ = CloseRemote(context.Background()) })
	p.Enqueue(sink.FormattedEntry{Message: "pending"})
	require.NoError(t, CloseRemote(context.Background()))
	fs.mu.Lock()
	n := len(fs.sent)
	fs.mu.Unlock()
	assert.Greater(t, n, 0)
}
