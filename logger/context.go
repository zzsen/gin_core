package logger

import (
	"context"

	"github.com/sirupsen/logrus"
)

// ctxKey 日志关联字段在 context 中的私有键类型，避免与其他包 string key 冲突。
type ctxKey string

// 上下文键：与中间件 c.Set("traceId") 等约定对齐（WithContext 同时兼容 string key）。
const (
	CtxTraceID   ctxKey = "traceId"
	CtxRequestID ctxKey = "requestId"
)

// ContextWithTraceID 将 traceId 写入 context，供 WithContext 读取。
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, CtxTraceID, traceID)
}

// ContextWithRequestID 将 requestId 写入 context，供 WithContext 读取。
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, CtxRequestID, requestID)
}

// WithContext 返回带关联字段的 logrus.Entry（增量 API，不改变旧 Info* 签名）。
//
// 【功能】从 ctx 提取 traceId / requestId，写入结构化字段后返回 Entry。
// 【流程】
// 1. 读取私有键 CtxTraceID / CtxRequestID
// 2. 兼容 string 键 "traceId" / "requestId"（中间件常见写法）
// 3. 无关联字段时返回空 Entry；有则 WithFields
func WithContext(ctx context.Context) *logrus.Entry {
	fields := logrus.Fields{}
	if ctx != nil {
		// 1. 私有键
		if v := ctx.Value(CtxTraceID); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields["traceId"] = s
			}
		}
		if v := ctx.Value(CtxRequestID); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields["requestId"] = s
			}
		}
		// 2. 兼容字符串键
		if v := ctx.Value("traceId"); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields["traceId"] = s
			}
		}
		if v := ctx.Value("requestId"); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields["requestId"] = s
			}
		}
	}
	// 3. 构造 Entry
	if len(fields) == 0 {
		return logrus.NewEntry(Logger)
	}
	return Logger.WithFields(fields)
}
