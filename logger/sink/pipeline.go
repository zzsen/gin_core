package sink

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// PipelineConfig 异步投递管线配置。
type PipelineConfig struct {
	QueueSize     int           // 缓冲队列容量；<=0 默认 1024
	BatchSize     int           // 批量发送条数；<=0 默认 32
	FlushInterval time.Duration // 定时刷出间隔；<=0 默认 500ms
	MaxAttempts   int           // 发送最大尝试次数（含首次）；<=0 视为 1
	Backoff       time.Duration // 重试间隔；0 表示重试间不 Sleep
}

// Pipeline 异步缓冲 + 批量发送管线。
//
// 热路径仅 Enqueue；worker 负责 batch / 重试 / 调用 Sink.Send。
// 语义为 at-most-once：队列满丢弃最旧，发送最终失败后丢弃并计数。
type Pipeline struct {
	sink Sink
	cfg  PipelineConfig

	ch      chan FormattedEntry
	flushCh chan chan error
	done    chan struct{}

	dropped   atomic.Uint64
	success   atomic.Uint64
	failures  atomic.Uint64
	closed    atomic.Bool
	wg        sync.WaitGroup
	closeOnce sync.Once
}

// NewPipeline 创建管线并启动后台 worker。
//
// 【流程】
// 1. 规范化 QueueSize / BatchSize / FlushInterval / MaxAttempts 默认值
// 2. 初始化 channel 与计数器
// 3. 启动 loop goroutine
func NewPipeline(s Sink, cfg PipelineConfig) *Pipeline {
	// 1. 默认值
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 1024
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 32
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 500 * time.Millisecond
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	// 2. 构造实例
	p := &Pipeline{
		sink:    s,
		cfg:     cfg,
		ch:      make(chan FormattedEntry, cfg.QueueSize),
		flushCh: make(chan chan error, 1),
		done:    make(chan struct{}),
	}

	// 3. 启动 worker
	p.wg.Add(1)
	go p.loop()
	return p
}

// Enqueue 非阻塞入队；队列满时丢弃最旧并增加 Dropped 计数。
//
// 【流程】
// 1. 已关闭则直接记 dropped
// 2. 尝试入队；成功则返回
// 3. 失败则弹出最旧再试；仍失败则记 dropped（不阻塞调用方）
func (p *Pipeline) Enqueue(e FormattedEntry) {
	// 1. 关闭后拒绝入队
	if p.closed.Load() {
		p.dropped.Add(1)
		return
	}

	// 2. 快速路径
	select {
	case p.ch <- e:
		return
	default:
	}

	// 3. 丢弃最旧后重试入队
	select {
	case <-p.ch:
		p.dropped.Add(1)
	default:
	}
	select {
	case p.ch <- e:
	default:
		p.dropped.Add(1)
	}
}

// Dropped 返回因背压或关闭而丢弃的条目数。
func (p *Pipeline) Dropped() uint64 {
	return p.dropped.Load()
}

// Success 返回发送成功的批次数。
func (p *Pipeline) Success() uint64 { return p.success.Load() }

// Failures 返回重试耗尽后仍失败的批次数。
func (p *Pipeline) Failures() uint64 { return p.failures.Load() }

// Flush 刷出当前缓冲并等待本轮发送完成。
//
// 【流程】管线已 done 则直接返回；否则通过 requestFlush 同步等待 worker 回执。
func (p *Pipeline) Flush(ctx context.Context) error {
	select {
	case <-p.done:
		return nil
	default:
	}
	return p.requestFlush(ctx)
}

// requestFlush 向 worker 发送刷出请求并等待结果。
//
// 【流程】
// 1. 将响应 channel 交给 flushCh
// 2. 等待 worker 回写 error，或 ctx/done 取消
func (p *Pipeline) requestFlush(ctx context.Context) error {
	resp := make(chan error, 1)
	// 1. 提交刷出请求
	select {
	case p.flushCh <- resp:
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return nil
	}
	// 2. 等待回执
	select {
	case err := <-resp:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return nil
	}
}

// Close 刷出剩余条目、停止 worker 并关闭底层 Sink（只执行一次）。
//
// 【流程】
// 1. 标记 closed，阻止新 Enqueue
// 2. requestFlush 刷出已有缓冲
// 3. 关闭 done，等待 loop 退出
// 4. 调用 Sink.Close
func (p *Pipeline) Close(ctx context.Context) error {
	var err error
	p.closeOnce.Do(func() {
		// 1. 拒绝新入队
		p.closed.Store(true)
		// 2. 刷出
		_ = p.requestFlush(ctx)
		// 3. 停止 worker
		close(p.done)
		p.wg.Wait()
		// 4. 关闭 Sink
		if p.sink != nil {
			err = p.sink.Close(ctx)
		}
	})
	return err
}

// loop 后台 worker：按批量大小 / 定时器 / Flush 请求驱动 Sink.Send。
//
// 【流程】
// 1. 从 ch 收条目，达 BatchSize 则发送
// 2. ticker 到期发送当前 batch
// 3. flushCh：排空 channel 后发送，并向调用方回写 error
// 4. done：最终排空并退出
func (p *Pipeline) loop() {
	defer p.wg.Done()
	batch := make([]FormattedEntry, 0, p.cfg.BatchSize)
	ticker := time.NewTicker(p.cfg.FlushInterval)
	defer ticker.Stop()

	// send 发送当前 batch，含有限重试；成功/失败更新计数
	send := func() error {
		if len(batch) == 0 || p.sink == nil {
			batch = batch[:0]
			return nil
		}
		toSend := batch
		batch = make([]FormattedEntry, 0, p.cfg.BatchSize)
		var err error
		for attempt := 1; attempt <= p.cfg.MaxAttempts; attempt++ {
			err = p.sink.Send(context.Background(), toSend)
			if err == nil {
				p.success.Add(1)
				return nil
			}
			if attempt < p.cfg.MaxAttempts && p.cfg.Backoff > 0 {
				time.Sleep(p.cfg.Backoff)
			}
		}
		p.failures.Add(1)
		return err
	}

	for {
		select {
		case e := <-p.ch:
			// 1. 积攒 batch
			batch = append(batch, e)
			if len(batch) >= p.cfg.BatchSize {
				_ = send()
			}
		case <-ticker.C:
			// 2. 定时刷出
			_ = send()
		case resp := <-p.flushCh:
			// 3. 同步 Flush：排空队列后发送
		drain:
			for {
				select {
				case e := <-p.ch:
					batch = append(batch, e)
					if len(batch) >= p.cfg.BatchSize {
						_ = send()
					}
				default:
					break drain
				}
			}
			err := send()
			resp <- err
		case <-p.done:
			// 4. 关闭前最终排空
		final:
			for {
				select {
				case e := <-p.ch:
					batch = append(batch, e)
				default:
					break final
				}
			}
			_ = send()
			return
		}
	}
}
