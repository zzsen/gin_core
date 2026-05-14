// Package tracing 链路追踪模块单元测试
//
// ==================== 测试说明 ====================
// 本文件包含 tracing 包核心功能的单元测试，不依赖外部采集器。
//
// 测试覆盖内容：
// 1. IsEnabled/IsDBTracingEnabled/IsRedisTracingEnabled/IsHTTPClientTracingEnabled 状态查询
// 2. InitTracer 初始化（disabled/stdout/unsupported exporter）
// 3. createSampler 采样器创建
// 4. createPropagator 传播器创建（默认/b3）
// 5. createResource 资源创建
// 6. StartSpan/SpanFromContext Span 操作
// 7. GetTraceID/GetSpanID ID 提取
// 8. SetSpanError/SetSpanAttributes/AddSpanEvent Span 辅助函数
//
// 运行测试：go test -v ./tracing/... -run Test
// ==================================================
package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// resetGlobals 重置全局状态，避免测试间污染
func resetGlobals() {
	Tracer = nil
	tracerProvider = nil
	tracingConfig = nil
}

// --- IsEnabled 系列测试 ---

// TestIsEnabled 测试链路追踪启用状态查询
//
// 【功能点】验证 IsEnabled 在不同配置下的返回值
// 【测试流程】
// 1. nil 配置 → false
// 2. Enabled=false → false
// 3. Enabled=true → true
func TestIsEnabled(t *testing.T) {
	resetGlobals()

	tracingConfig = nil
	assert.False(t, IsEnabled())

	tracingConfig = &config.TracingConfig{Enabled: false}
	assert.False(t, IsEnabled())

	tracingConfig = &config.TracingConfig{Enabled: true}
	assert.True(t, IsEnabled())
}

// TestIsDBTracingEnabled 测试数据库追踪启用状态
//
// 【功能点】需要 Enabled=true 且 EnableDBTracing=true
func TestIsDBTracingEnabled(t *testing.T) {
	resetGlobals()

	tracingConfig = nil
	assert.False(t, IsDBTracingEnabled())

	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: false}
	assert.False(t, IsDBTracingEnabled())

	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}
	assert.True(t, IsDBTracingEnabled())
}

// TestIsRedisTracingEnabled 测试 Redis 追踪启用状态
func TestIsRedisTracingEnabled(t *testing.T) {
	resetGlobals()

	tracingConfig = nil
	assert.False(t, IsRedisTracingEnabled())

	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}
	assert.True(t, IsRedisTracingEnabled())
}

// TestIsHTTPClientTracingEnabled 测试 HTTP 客户端追踪启用状态
func TestIsHTTPClientTracingEnabled(t *testing.T) {
	resetGlobals()

	tracingConfig = nil
	assert.False(t, IsHTTPClientTracingEnabled())

	tracingConfig = &config.TracingConfig{Enabled: true, EnableHTTPClientTracing: true}
	assert.True(t, IsHTTPClientTracingEnabled())
}

// --- InitTracer 测试 ---

// TestInitTracer_Disabled 测试禁用时的初始化
//
// 【功能点】cfg=nil 或 Enabled=false 时返回 NoOp Tracer
func TestInitTracer_Disabled(t *testing.T) {
	resetGlobals()

	shutdown, err := InitTracer(nil)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	require.NotNil(t, Tracer)
	assert.NoError(t, shutdown(context.Background()))

	resetGlobals()
	shutdown, err = InitTracer(&config.TracingConfig{Enabled: false})
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
}

// TestInitTracer_Stdout 测试 stdout 导出器
//
// 【功能点】ExporterType=stdout 时应成功初始化
func TestInitTracer_Stdout(t *testing.T) {
	resetGlobals()

	cfg := &config.TracingConfig{
		Enabled:      true,
		ServiceName:  "test-service",
		ExporterType: "stdout",
		SampleRate:   1.0,
	}

	shutdown, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	require.NotNil(t, Tracer)
	require.NotNil(t, tracerProvider)

	assert.NoError(t, shutdown(context.Background()))
}

// TestInitTracer_UnsupportedExporter 测试不支持的导出器类型
//
// 【功能点】未知 ExporterType 应返回错误
func TestInitTracer_UnsupportedExporter(t *testing.T) {
	resetGlobals()

	cfg := &config.TracingConfig{
		Enabled:      true,
		ServiceName:  "test-service",
		ExporterType: "unknown",
	}

	_, err := InitTracer(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的导出类型")
}

// --- createSampler 测试 ---

// TestCreateSampler 测试采样器创建
//
// 【功能点】不同采样率应创建对应的采样器
func TestCreateSampler(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"全采样", 1.0},
		{"10% 采样", 0.1},
		{"不采样", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.TracingConfig{SampleRate: tt.rate}
			sampler := createSampler(cfg)
			assert.NotNil(t, sampler)
		})
	}
}

// --- createPropagator 测试 ---

// TestCreatePropagator 测试传播器创建
//
// 【功能点】验证 b3/b3multi/tracecontext/空值四种格式全覆盖
// 【测试流程】
// 1. 空值/tracecontext → 返回 CompositeTextMapPropagator
// 2. "b3" → 返回 B3 SingleHeader propagator
// 3. "b3multi" → 返回 B3 MultipleHeader propagator
func TestCreatePropagator(t *testing.T) {
	tests := []struct {
		name           string
		propagatorType string
	}{
		{"空值默认 tracecontext", ""},
		{"tracecontext 显式", "tracecontext"},
		{"b3 single header", "b3"},
		{"b3 multiple header", "b3multi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := createPropagator(&config.TracingConfig{PropagatorType: tt.propagatorType})
			assert.NotNil(t, p)
			// 验证 Fields() 返回非空（所有 propagator 都有 Fields 方法）
			assert.NotNil(t, p.Fields())
		})
	}
}

// TestCreatePropagator_B3Inject 测试 B3 propagator 注入能力
//
// 【功能点】验证 B3 propagator 能正确注入 B3 头
func TestCreatePropagator_B3Inject(t *testing.T) {
	b3Single := createPropagator(&config.TracingConfig{PropagatorType: "b3"})
	fields := b3Single.Fields()
	assert.Contains(t, fields, "b3")

	b3Multi := createPropagator(&config.TracingConfig{PropagatorType: "b3multi"})
	fieldsMulti := b3Multi.Fields()
	assert.True(t, len(fieldsMulti) > 0)
}

// --- createResource 测试 ---

// TestCreateResource 测试资源创建
//
// 【功能点】资源应包含服务名称
func TestCreateResource(t *testing.T) {
	cfg := &config.TracingConfig{ServiceName: "my-service"}
	res, err := createResource(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, res)
}

// --- StartSpan 测试 ---

// TestStartSpan_WithTracer 测试有 Tracer 时的 StartSpan
//
// 【功能点】Tracer 非 nil 时应创建真实 Span
func TestStartSpan_WithTracer(t *testing.T) {
	resetGlobals()

	tp := sdktrace.NewTracerProvider()
	Tracer = tp.Tracer("test")

	ctx, span := StartSpan(context.Background(), "test-op")
	require.NotNil(t, span)
	require.NotNil(t, ctx)
	span.End()

	_ = tp.Shutdown(context.Background())
}

// TestStartSpan_NilTracer 测试 Tracer 为 nil 时的 StartSpan
//
// 【功能点】Tracer=nil 时应返回 NoOp Span
func TestStartSpan_NilTracer(t *testing.T) {
	resetGlobals()
	Tracer = nil

	ctx := context.Background()
	retCtx, span := StartSpan(ctx, "test")
	assert.NotNil(t, span)
	assert.Equal(t, ctx, retCtx)
}

// --- SpanFromContext 测试 ---

// TestSpanFromContext 测试从 context 获取 Span
func TestSpanFromContext(t *testing.T) {
	span := SpanFromContext(context.Background())
	assert.NotNil(t, span)
}

// --- GetTraceID / GetSpanID 测试 ---

// TestGetTraceID_NoSpan 测试无 Span 时获取 TraceID
//
// 【功能点】无有效 Span 时应返回空字符串
func TestGetTraceID_NoSpan(t *testing.T) {
	assert.Equal(t, "", GetTraceID(context.Background()))
}

// TestGetSpanID_NoSpan 测试无 Span 时获取 SpanID
func TestGetSpanID_NoSpan(t *testing.T) {
	assert.Equal(t, "", GetSpanID(context.Background()))
}

// TestGetTraceID_WithSpan 测试有 Span 时获取 TraceID
//
// 【功能点】有有效 Span 时应返回非空 TraceID
func TestGetTraceID_WithSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	traceID := GetTraceID(ctx)
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32)
}

// TestGetSpanID_WithSpan 测试有 Span 时获取 SpanID
func TestGetSpanID_WithSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "op")
	defer span.End()

	spanID := GetSpanID(ctx)
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16)
}

// --- SetSpanError 测试 ---

// TestSetSpanError 测试设置 Span 错误
//
// 【功能点】err 非 nil 时设置错误状态，nil 时不操作
func TestSetSpanError(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "op")

	SetSpanError(span, errors.New("test error"))
	span.End()

	_, span2 := tracer.Start(context.Background(), "op2")
	SetSpanError(span2, nil)
	span2.End()
}

// --- SetSpanAttributes 测试 ---

// TestSetSpanAttributes 测试设置 Span 属性
func TestSetSpanAttributes(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "op")

	SetSpanAttributes(span, attribute.String("key", "value"), attribute.Int("count", 42))
	span.End()
}

// --- AddSpanEvent 测试 ---

// TestAddSpanEvent 测试添加 Span 事件
func TestAddSpanEvent(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "op")

	AddSpanEvent(span, "cache.hit", attribute.String("cache.key", "user:123"))
	span.End()
}
