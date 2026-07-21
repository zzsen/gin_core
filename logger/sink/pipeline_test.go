// Package sink Pipeline 异步管线测试
//
// ==================== 测试说明 ====================
// 验证 Enqueue 非阻塞、队列满丢弃、Flush/Close。
//
// 测试覆盖内容：
// 1. Enqueue 不因 Sink.Send 阻塞而卡住
// 2. 队列满丢弃最旧并计数
// 3. Flush 刷出待发送条目
//
// 运行测试：go test -v ./logger/sink/... -run TestPipeline_
// ==================================================
package sink

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingSink struct {
	block chan struct{}
	sent  []FormattedEntry
	mu    sync.Mutex
}

func (b *blockingSink) Name() string     { return "blocking" }
func (b *blockingSink) MinLevel() string { return "" }
func (b *blockingSink) Send(_ context.Context, entries []FormattedEntry) error {
	<-b.block
	b.mu.Lock()
	b.sent = append(b.sent, entries...)
	b.mu.Unlock()
	return nil
}
func (b *blockingSink) Close(context.Context) error { return nil }

type recordingSink struct {
	mu   sync.Mutex
	sent []FormattedEntry
}

func (r *recordingSink) Name() string     { return "recording" }
func (r *recordingSink) MinLevel() string { return "" }
func (r *recordingSink) Send(_ context.Context, entries []FormattedEntry) error {
	r.mu.Lock()
	r.sent = append(r.sent, entries...)
	r.mu.Unlock()
	return nil
}
func (r *recordingSink) Close(context.Context) error { return nil }

// TestPipeline_EnqueueDoesNotCallSendSync Enqueue 不因 Send 阻塞
//
// 【功能点】热路径仅入队
// 【测试流程】
// 1. Sink.Send 阻塞
// 2. Enqueue 在超时内返回
func TestPipeline_EnqueueDoesNotCallSendSync(t *testing.T) {
	fs := &blockingSink{block: make(chan struct{})}
	p := NewPipeline(fs, PipelineConfig{QueueSize: 10, BatchSize: 10, FlushInterval: time.Hour})
	done := make(chan struct{})
	go func() {
		p.Enqueue(FormattedEntry{Message: "a"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Enqueue blocked on Send")
	}
	close(fs.block)
	require.NoError(t, p.Close(context.Background()))
}

// TestPipeline_QueueFullDropsOldest 队列满丢弃最旧
//
// 【功能点】背压 drop-oldest + Dropped 计数
// 【测试流程】
// 1. QueueSize=2，连续 Enqueue 3 条
// 2. Dropped >= 1
func TestPipeline_QueueFullDropsOldest(t *testing.T) {
	// 阻塞 Sink 防止 worker 立刻消费，以便测队列满丢弃
	bs := &blockingSink{block: make(chan struct{})}
	p := NewPipeline(bs, PipelineConfig{QueueSize: 2, BatchSize: 10, FlushInterval: time.Hour})
	p.Enqueue(FormattedEntry{Message: "1"})
	p.Enqueue(FormattedEntry{Message: "2"})
	p.Enqueue(FormattedEntry{Message: "3"})
	assert.GreaterOrEqual(t, p.Dropped(), uint64(1))
	close(bs.block)
	require.NoError(t, p.Close(context.Background()))
}

// TestPipeline_FlushSendsPending Flush 刷出缓冲
//
// 【功能点】Flush 将队列中条目发送到 Sink
// 【测试流程】
// 1. Enqueue 后 Flush
// 2. recordingSink 收到条目
func TestPipeline_FlushSendsPending(t *testing.T) {
	fs := &recordingSink{}
	p := NewPipeline(fs, PipelineConfig{QueueSize: 10, BatchSize: 10, FlushInterval: time.Hour})
	p.Enqueue(FormattedEntry{Message: "pending"})
	require.NoError(t, p.Flush(context.Background()))
	fs.mu.Lock()
	n := len(fs.sent)
	fs.mu.Unlock()
	assert.Greater(t, n, 0)
	require.NoError(t, p.Close(context.Background()))
}

type failAlwaysSink struct{ calls atomic.Int32 }

func (f *failAlwaysSink) Name() string     { return "fail" }
func (f *failAlwaysSink) MinLevel() string { return "" }
func (f *failAlwaysSink) Send(context.Context, []FormattedEntry) error {
	f.calls.Add(1)
	return assert.AnError
}
func (f *failAlwaysSink) Close(context.Context) error { return nil }

// TestPipeline_RetriesThenCountsFailure 重试耗尽后记 Failures
func TestPipeline_RetriesThenCountsFailure(t *testing.T) {
	fs := &failAlwaysSink{}
	p := NewPipeline(fs, PipelineConfig{
		QueueSize: 10, BatchSize: 10, FlushInterval: time.Hour,
		MaxAttempts: 2, Backoff: time.Millisecond,
	})
	p.Enqueue(FormattedEntry{Message: "x"})
	_ = p.Flush(context.Background())
	assert.GreaterOrEqual(t, fs.calls.Load(), int32(2))
	assert.GreaterOrEqual(t, p.Failures(), uint64(1))
	require.NoError(t, p.Close(context.Background()))
}

// TestPipeline_CallerNotBlockedWhenSinkDown Enqueue 在 Sink 失败时仍快速返回
func TestPipeline_CallerNotBlockedWhenSinkDown(t *testing.T) {
	fs := &failAlwaysSink{}
	p := NewPipeline(fs, PipelineConfig{
		QueueSize: 10, BatchSize: 1, FlushInterval: time.Hour,
		MaxAttempts: 3, Backoff: 50 * time.Millisecond,
	})
	done := make(chan struct{})
	go func() {
		p.Enqueue(FormattedEntry{Message: "fast"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Enqueue blocked while sink failing")
	}
	require.NoError(t, p.Close(context.Background()))
}
