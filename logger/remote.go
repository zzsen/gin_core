package logger

import (
	"context"
	"sync"

	"github.com/zzsen/gin_core/logger/sink"
)

// 全局远程管线：由 InitLogger/attachRemoteSink 设置，CloseRemote 清空。
var (
	remoteMu       sync.RWMutex
	remotePipeline *sink.Pipeline
)

// SetRemotePipeline 设置全局远程投递管线。
//
// 【功能】供 InitLogger 装配与单测注入使用；并发安全。
func SetRemotePipeline(p *sink.Pipeline) {
	remoteMu.Lock()
	defer remoteMu.Unlock()
	remotePipeline = p
}

// GetRemotePipeline 返回当前远程管线指针（可能为 nil）。
//
// 【功能】主要用于测试断言是否已装配；业务代码一般无需调用。
func GetRemotePipeline() *sink.Pipeline {
	remoteMu.RLock()
	defer remoteMu.RUnlock()
	return remotePipeline
}

// FlushRemote 刷出远程缓冲中的待发送日志，不关闭管线。
//
// 【流程】
// 1. 读取当前 pipeline；无则直接返回
// 2. 调用 Pipeline.Flush，等待本轮发送完成
func FlushRemote(ctx context.Context) error {
	// 1. 取当前管线快照
	remoteMu.RLock()
	p := remotePipeline
	remoteMu.RUnlock()
	if p == nil {
		return nil
	}
	// 2. 刷出缓冲
	return p.Flush(ctx)
}

// CloseRemote 刷出并关闭远程管线，同时清空全局引用。
//
// 【功能】供 LoggerService.Close 在进程退出时调用，尽量减少缓冲丢失。
// 【流程】
// 1. 加锁取出 pipeline 并置空，避免重复 Close
// 2. 调用 Pipeline.Close（内部含最终 Flush）
func CloseRemote(ctx context.Context) error {
	// 1. 摘除全局引用
	remoteMu.Lock()
	p := remotePipeline
	remotePipeline = nil
	remoteMu.Unlock()
	if p == nil {
		return nil
	}
	// 2. 刷出并停止 worker
	return p.Close(ctx)
}
