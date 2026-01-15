// Package tracing 提供基于 OpenTelemetry 的分布式链路追踪功能
// 本文件实现了 Redis 操作的追踪钩子
package tracing

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// RedisTracingHook Redis 链路追踪钩子
// 实现了 redis.Hook 接口，用于追踪 Redis 操作
type RedisTracingHook struct {
	// addr Redis 服务器地址
	addr string
	// aliasName Redis 实例别名
	aliasName string
	// db Redis 数据库编号
	db int
}

// NewRedisTracingHook 创建新的 Redis 追踪钩子实例
// 参数：
//   - addr: Redis 服务器地址
//   - aliasName: Redis 实例别名，可选
//   - db: Redis 数据库编号
//
// 返回：
//   - *RedisTracingHook: 追踪钩子实例
func NewRedisTracingHook(addr string, aliasName string, db int) *RedisTracingHook {
	if aliasName == "" {
		aliasName = "default"
	}
	return &RedisTracingHook{
		addr:      addr,
		aliasName: aliasName,
		db:        db,
	}
}

// DialHook 连接建立钩子
// 实现 redis.Hook 接口
func (h *RedisTracingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !IsRedisTracingEnabled() {
			return next(ctx, network, addr)
		}

		ctx, span := StartSpan(ctx, "redis.dial",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemRedis,
				semconv.NetPeerName(addr),
				attribute.String("net.transport", network),
			),
		)
		defer span.End()

		conn, err := next(ctx, network, addr)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return conn, err
	}
}

// ProcessHook 单个命令处理钩子
// 实现 redis.Hook 接口
// 该函数会：
// 1. 在命令执行前创建 Span
// 2. 记录命令名称和参数
// 3. 在命令执行后记录错误（如果有）
// 4. 结束 Span
func (h *RedisTracingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if !IsRedisTracingEnabled() {
			return next(ctx, cmd)
		}

		// 创建 Span，使用命令名称作为操作名
		spanName := fmt.Sprintf("redis.%s", strings.ToLower(cmd.Name()))
		ctx, span := StartSpan(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemRedis,
				semconv.NetPeerName(h.addr),
				attribute.String("redis.alias", h.aliasName),
				attribute.Int("db.redis.database_index", h.db),
				attribute.String("db.statement", h.formatCmd(cmd)),
			),
		)
		defer span.End()

		// 执行命令
		err := next(ctx, cmd)
		if err != nil && err != redis.Nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}

// ProcessPipelineHook 管道命令处理钩子
// 实现 redis.Hook 接口
// 该函数会追踪 Pipeline 中的批量命令执行
func (h *RedisTracingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if !IsRedisTracingEnabled() {
			return next(ctx, cmds)
		}

		// 创建 Pipeline Span
		ctx, span := StartSpan(ctx, "redis.pipeline",
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemRedis,
				semconv.NetPeerName(h.addr),
				attribute.String("redis.alias", h.aliasName),
				attribute.Int("db.redis.database_index", h.db),
				attribute.Int("redis.pipeline.commands_count", len(cmds)),
				attribute.String("db.statement", h.formatPipelineCmds(cmds)),
			),
		)
		defer span.End()

		// 执行管道命令
		err := next(ctx, cmds)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}

// formatCmd 格式化单个 Redis 命令为字符串
// 会截断过长的参数，避免追踪数据过大
func (h *RedisTracingHook) formatCmd(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) == 0 {
		return ""
	}

	var parts []string
	totalLen := 0
	maxTotalLen := 500 // 最大总长度

	for i, arg := range args {
		if i > 10 { // 最多记录10个参数
			parts = append(parts, "...")
			break
		}

		s := fmt.Sprintf("%v", arg)
		// 截断过长的单个参数
		if len(s) > 100 {
			s = s[:100] + "..."
		}

		totalLen += len(s)
		if totalLen > maxTotalLen {
			parts = append(parts, "...")
			break
		}

		parts = append(parts, s)
	}

	return strings.Join(parts, " ")
}

// formatPipelineCmds 格式化管道命令为字符串
func (h *RedisTracingHook) formatPipelineCmds(cmds []redis.Cmder) string {
	if len(cmds) == 0 {
		return ""
	}

	var parts []string
	for i, cmd := range cmds {
		if i >= 5 { // 最多记录5个命令
			parts = append(parts, fmt.Sprintf("... and %d more commands", len(cmds)-5))
			break
		}
		parts = append(parts, cmd.Name())
	}

	return strings.Join(parts, ", ")
}
