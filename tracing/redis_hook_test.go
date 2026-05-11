// Package tracing Redis 追踪钩子测试
//
// ==================== 测试说明 ====================
// 本文件包含 RedisTracingHook 的单元测试，不依赖 Redis 服务。
//
// 测试覆盖内容：
// 1. NewRedisTracingHook 创建（默认别名/自定义别名）
// 2. formatCmd 命令格式化（正常/长参数/多参数截断）
// 3. formatPipelineCmds 管道命令格式化（正常/超过 5 条截断/空）
// 4. DialHook/ProcessHook/ProcessPipelineHook 禁用追踪时的直通
//
// 运行测试：go test -v ./tracing/... -run "TestNewRedisTracingHook|TestFormatCmd|TestFormatPipeline|TestRedis"
// ==================================================
package tracing

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestNewRedisTracingHook 测试创建 Redis 追踪钩子
//
// 【功能点】默认别名为 "default"，自定义别名正确赋值
func TestNewRedisTracingHook(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	assert.Equal(t, "default", hook.aliasName)
	assert.Equal(t, "localhost:6379", hook.addr)
	assert.Equal(t, 0, hook.db)

	hook2 := NewRedisTracingHook("localhost:6379", "cache", 1)
	assert.Equal(t, "cache", hook2.aliasName)
	assert.Equal(t, 1, hook2.db)
}

// TestFormatCmd_Normal 测试正常命令格式化
func TestFormatCmd_Normal(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	cmd := redis.NewStringCmd(context.Background(), "GET", "mykey")
	result := hook.formatCmd(cmd)
	assert.Contains(t, result, "GET")
	assert.Contains(t, result, "mykey")
}

// TestFormatCmd_LongArg 测试长参数截断
//
// 【功能点】单个参数超过 100 字符应截断
func TestFormatCmd_LongArg(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	longVal := strings.Repeat("x", 200)
	cmd := redis.NewStringCmd(context.Background(), "SET", "key", longVal)
	result := hook.formatCmd(cmd)
	assert.Contains(t, result, "...")
	assert.Less(t, len(result), 700)
}

// TestFormatCmd_ManyArgs 测试多参数截断
//
// 【功能点】超过 10 个参数应截断
func TestFormatCmd_ManyArgs(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	args := []interface{}{"MGET"}
	for i := 0; i < 15; i++ {
		args = append(args, fmt.Sprintf("key%d", i))
	}
	cmd := redis.NewStringCmd(context.Background(), args...)
	result := hook.formatCmd(cmd)
	assert.Contains(t, result, "...")
}

// TestFormatCmd_Empty 测试空参数命令
func TestFormatCmd_Empty(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	cmd := redis.NewStringCmd(context.Background())
	result := hook.formatCmd(cmd)
	assert.Empty(t, result)
}

// TestFormatPipelineCmds_Normal 测试管道命令格式化
func TestFormatPipelineCmds_Normal(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	cmds := []redis.Cmder{
		redis.NewStringCmd(context.Background(), "GET", "key1"),
		redis.NewStringCmd(context.Background(), "SET", "key2", "val"),
	}
	result := hook.formatPipelineCmds(cmds)
	assert.Contains(t, result, "get")
	assert.Contains(t, result, "set")
}

// TestFormatPipelineCmds_TooMany 测试管道命令超过 5 条截断
func TestFormatPipelineCmds_TooMany(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	cmds := make([]redis.Cmder, 8)
	for i := range cmds {
		cmds[i] = redis.NewStringCmd(context.Background(), "GET", fmt.Sprintf("key%d", i))
	}
	result := hook.formatPipelineCmds(cmds)
	assert.Contains(t, result, "3 more commands")
}

// TestFormatPipelineCmds_Empty 测试空管道
func TestFormatPipelineCmds_Empty(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	result := hook.formatPipelineCmds(nil)
	assert.Empty(t, result)
}

// TestRedisDialHook_Disabled 测试禁用追踪时 DialHook 直通
func TestRedisDialHook_Disabled(t *testing.T) {
	resetGlobals()
	tracingConfig = nil

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	nextCalled := false
	dialFn := hook.DialHook(func(ctx context.Context, network, addr string) (net.Conn, error) {
		nextCalled = true
		return nil, nil
	})

	_, _ = dialFn(context.Background(), "tcp", "localhost:6379")
	assert.True(t, nextCalled)
}

// TestRedisDialHook_Enabled 测试启用追踪时 DialHook
func TestRedisDialHook_Enabled(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	nextCalled := false
	dialFn := hook.DialHook(func(ctx context.Context, network, addr string) (net.Conn, error) {
		nextCalled = true
		return nil, nil
	})

	_, _ = dialFn(context.Background(), "tcp", "localhost:6379")
	assert.True(t, nextCalled)
}

// TestRedisDialHook_Error 测试 DialHook 连接失败
func TestRedisDialHook_Error(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	dialFn := hook.DialHook(func(ctx context.Context, network, addr string) (net.Conn, error) {
		return nil, fmt.Errorf("connection refused")
	})

	_, err := dialFn(context.Background(), "tcp", "localhost:6379")
	require.Error(t, err)
}

// TestRedisProcessHook_Disabled 测试禁用追踪时 ProcessHook 直通
func TestRedisProcessHook_Disabled(t *testing.T) {
	resetGlobals()
	tracingConfig = nil

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	nextCalled := false
	processFn := hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error {
		nextCalled = true
		return nil
	})

	cmd := redis.NewStringCmd(context.Background(), "GET", "key")
	_ = processFn(context.Background(), cmd)
	assert.True(t, nextCalled)
}

// TestRedisProcessHook_Enabled 测试启用追踪时 ProcessHook
func TestRedisProcessHook_Enabled(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "cache", 0)
	nextCalled := false
	processFn := hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error {
		nextCalled = true
		return nil
	})

	cmd := redis.NewStringCmd(context.Background(), "GET", "key")
	err := processFn(context.Background(), cmd)
	require.NoError(t, err)
	assert.True(t, nextCalled)
}

// TestRedisProcessHook_Error 测试 ProcessHook 命令错误
func TestRedisProcessHook_Error(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	processFn := hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error {
		return fmt.Errorf("command error")
	})

	cmd := redis.NewStringCmd(context.Background(), "GET", "key")
	err := processFn(context.Background(), cmd)
	require.Error(t, err)
}

// TestRedisProcessPipelineHook_Disabled 测试禁用追踪时 ProcessPipelineHook 直通
func TestRedisProcessPipelineHook_Disabled(t *testing.T) {
	resetGlobals()
	tracingConfig = nil

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	nextCalled := false
	pipelineFn := hook.ProcessPipelineHook(func(ctx context.Context, cmds []redis.Cmder) error {
		nextCalled = true
		return nil
	})

	cmds := []redis.Cmder{redis.NewStringCmd(context.Background(), "GET", "key")}
	_ = pipelineFn(context.Background(), cmds)
	assert.True(t, nextCalled)
}

// TestRedisProcessPipelineHook_Enabled 测试启用追踪时 ProcessPipelineHook
func TestRedisProcessPipelineHook_Enabled(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	nextCalled := false
	pipelineFn := hook.ProcessPipelineHook(func(ctx context.Context, cmds []redis.Cmder) error {
		nextCalled = true
		return nil
	})

	cmds := []redis.Cmder{
		redis.NewStringCmd(context.Background(), "GET", "key1"),
		redis.NewStringCmd(context.Background(), "SET", "key2", "val"),
	}
	err := pipelineFn(context.Background(), cmds)
	require.NoError(t, err)
	assert.True(t, nextCalled)
}

// TestRedisProcessPipelineHook_Error 测试管道命令错误
func TestRedisProcessPipelineHook_Error(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	pipelineFn := hook.ProcessPipelineHook(func(ctx context.Context, cmds []redis.Cmder) error {
		return fmt.Errorf("pipeline error")
	})

	cmds := []redis.Cmder{redis.NewStringCmd(context.Background(), "GET", "key")}
	err := pipelineFn(context.Background(), cmds)
	require.Error(t, err)
}

// TestFormatCmd_TotalLenExceeded 测试总长度超限截断
//
// 【功能点】参数总长度超过 500 时应截断
func TestFormatCmd_TotalLenExceeded(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	args := []interface{}{"MSET"}
	for i := 0; i < 8; i++ {
		args = append(args, fmt.Sprintf("key%d", i), strings.Repeat("v", 80))
	}
	cmd := redis.NewStringCmd(context.Background(), args...)
	result := hook.formatCmd(cmd)
	assert.Contains(t, result, "...")
}
