// Package tracing 提供基于 OpenTelemetry 的分布式链路追踪功能
// 本文件实现了 OpenTelemetry SDK 的初始化和核心追踪功能
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/zzsen/gin_core/model/config"
)

// Tracer 全局追踪器实例
var Tracer trace.Tracer

// tracerProvider 全局 TracerProvider 实例
var tracerProvider *sdktrace.TracerProvider

// tracingConfig 全局链路追踪配置
var tracingConfig *config.TracingConfig

// IsEnabled 返回链路追踪是否已启用
func IsEnabled() bool {
	return tracingConfig != nil && tracingConfig.Enabled
}

// IsDBTracingEnabled 返回数据库追踪是否已启用
func IsDBTracingEnabled() bool {
	return IsEnabled() && tracingConfig.EnableDBTracing
}

// IsRedisTracingEnabled 返回 Redis 追踪是否已启用
func IsRedisTracingEnabled() bool {
	return IsEnabled() && tracingConfig.EnableRedisTracing
}

// IsHTTPClientTracingEnabled 返回 HTTP 客户端追踪是否已启用
func IsHTTPClientTracingEnabled() bool {
	return IsEnabled() && tracingConfig.EnableHTTPClientTracing
}

// InitTracer 初始化 OpenTelemetry Tracer
// 该函数会：
// 1. 根据配置创建对应的导出器（OTLP、Jaeger、stdout）
// 2. 创建资源信息，标识服务名称和环境
// 3. 配置采样策略
// 4. 设置全局 TracerProvider 和 Propagator
// 参数：
//   - cfg: 链路追踪配置
//
// 返回：
//   - func(context.Context) error: 关闭函数，用于优雅关闭追踪器
//   - error: 初始化错误
func InitTracer(cfg *config.TracingConfig) (func(context.Context) error, error) {
	tracingConfig = cfg

	if cfg == nil || !cfg.Enabled {
		// 返回 NoOp Tracer
		Tracer = otel.Tracer("noop")
		return func(ctx context.Context) error { return nil }, nil
	}

	ctx := context.Background()

	// 创建导出器
	exporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建导出器失败: %w", err)
	}

	// 创建资源信息
	res, err := createResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("创建资源失败: %w", err)
	}

	// 创建采样器
	sampler := createSampler(cfg)

	// 创建 TracerProvider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tracerProvider)

	// 设置全局 Propagator
	propagator := createPropagator(cfg)
	otel.SetTextMapPropagator(propagator)

	// 创建 Tracer 实例
	Tracer = tracerProvider.Tracer(cfg.ServiceName)

	// 返回关闭函数
	return tracerProvider.Shutdown, nil
}

// createExporter 根据配置创建不同的导出器
func createExporter(ctx context.Context, cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	switch cfg.ExporterType {
	case "otlp":
		return createOTLPExporter(ctx, cfg)
	case "stdout":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		return nil, fmt.Errorf("不支持的导出类型: %s，支持的类型: otlp, stdout", cfg.ExporterType)
	}
}

// createOTLPExporter 创建 OTLP gRPC 导出器
func createOTLPExporter(ctx context.Context, cfg *config.TracingConfig) (sdktrace.SpanExporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
		opts = append(opts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	return otlptracegrpc.New(ctx, opts...)
}

// createResource 创建资源信息
func createResource(ctx context.Context, cfg *config.TracingConfig) (*resource.Resource, error) {
	// 直接创建资源，避免与 resource.Default() 的 Schema URL 冲突
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.ServiceName),
		semconv.ServiceVersion("1.0.0"),
		attribute.String("library.language", "go"),
		attribute.String("telemetry.sdk.name", "opentelemetry"),
		attribute.String("telemetry.sdk.language", "go"),
	), nil
}

// createSampler 创建采样器
func createSampler(cfg *config.TracingConfig) sdktrace.Sampler {
	// 使用 ParentBased 采样器，尊重父 Span 的采样决策
	return sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.SampleRate),
	)
}

// createPropagator 创建上下文传播器
func createPropagator(cfg *config.TracingConfig) propagation.TextMapPropagator {
	switch cfg.PropagatorType {
	case "b3":
		// B3 格式需要额外的包，这里暂时使用 TraceContext
		return propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	default:
		// 默认使用 W3C Trace Context 标准
		return propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
	}
}

// SpanFromContext 从 context 获取当前 Span
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - trace.Span: 当前 Span，如果不存在则返回 NoOp Span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// StartSpan 开始一个新的 Span
// 参数：
//   - ctx: 父上下文
//   - name: Span 名称
//   - opts: Span 选项
//
// 返回：
//   - context.Context: 包含新 Span 的上下文
//   - trace.Span: 新创建的 Span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if Tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return Tracer.Start(ctx, name, opts...)
}

// GetTraceID 从 context 获取 TraceID
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - string: TraceID 字符串，如果无效则返回空字符串
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID 从 context 获取 SpanID
// 参数：
//   - ctx: 上下文
//
// 返回：
//   - string: SpanID 字符串，如果无效则返回空字符串
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// SetSpanError 设置 Span 错误状态
// 参数：
//   - span: 目标 Span
//   - err: 错误信息
func SetSpanError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}

// SetSpanAttributes 设置 Span 属性
// 参数：
//   - span: 目标 Span
//   - attrs: 属性键值对
func SetSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	span.SetAttributes(attrs...)
}

// AddSpanEvent 添加 Span 事件
// 参数：
//   - span: 目标 Span
//   - name: 事件名称
//   - attrs: 事件属性
func AddSpanEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, trace.WithAttributes(attrs...))
}
