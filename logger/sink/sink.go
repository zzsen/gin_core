// Package sink 提供远程日志 Sink 抽象与异步投递管线。
//
// 上层（logger 包）负责将 logrus Entry 转为 FormattedEntry 并入队；
// 具体协议适配器（如 Loki）实现 Sink.Send 完成网络发送。
package sink

import (
	"context"
	"time"
)

// FormattedEntry 已格式化的日志条目，供 Sink 批量发送。
type FormattedEntry struct {
	Level   string         // 级别名，如 info / error
	Time    time.Time      // 日志时间；零值时由适配器补 Now
	Message string         // 原始消息
	Fields  map[string]any // enrich 后的结构化字段
	Line    []byte         // 序列化后的行内容（如 JSON），优先作为远程 payload
}

// Sink 远程（或可插拔）日志输出接口。
//
// 实现方须保证 Send 可被 Pipeline worker 调用；Close 用于释放连接等资源。
type Sink interface {
	// Name 返回 Sink 名称（如 loki），用于日志与排障标识
	Name() string
	// MinLevel 返回最低级别名；空表示不限制（过滤一般由 Hook.Levels 完成）
	MinLevel() string
	// Send 批量发送；entries 为空时应立即返回 nil
	Send(ctx context.Context, entries []FormattedEntry) error
	// Close 释放资源；可重复调用语义由实现自行保证
	Close(ctx context.Context) error
}
