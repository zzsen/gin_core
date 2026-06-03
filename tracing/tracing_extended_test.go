// Package tracing 链路追踪模块扩展单元测试
//
// ==================== 测试说明 ====================
// 本文件补充 OTLP 导出器、HTTP Transport 边界、Redis Nil 错误处理、formatCmd 总长度截断、
// GORM 插件回调路径（含 Statement.Context 为空）等场景，用于提升 tracing 包语句覆盖率。
//
// 测试覆盖内容：
// 1. InitTracer OTLP Insecure / TLS 导出器
// 2. HTTP RoundTrip Tracer 为 nil 但 Tracing 已启用的场景
// 3. Redis ProcessHook RedisNil 错误忽略
// 4. GormTracingPlugin Initialize 回调注册
// 5. GORM before 回调 Context 为空
// 6. formatCmd 总长度超限截断
// 7. GORM Query 启用/禁用 DB Tracing 的行为差异
// 8. GORM after 回调 Span 缺失、类型错误、ErrRecordNotFound 处理
// 9. GORM after 长 SQL 截断
// 10. GORM RawSQL 错误记录到 Span
//
// 运行测试：go test -short -count=1 ./tracing/...
// ==================================================
package tracing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sqlite "github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"gorm.io/gorm"
)

type tracingGormModel struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func openTestSQLiteDB(t *testing.T) *gorm.DB {
	t.Helper()
	// modernc/sqlite 下 ":memory:" 会为每个连接创建独立库；共享缓存才能使迁移与查询看到同一 schema
	db, err := gorm.Open(sqlite.Open("file:tracing_shared_mem?mode=memory&cache=shared"), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)
	return db
}

// TestInitTracer_OTLP_Insecure OTLP 明文初始化
//
// 【功能点】ExporterType=otlp 且 Insecure=true 时应成功创建导出器与 TracerProvider
// 【测试流程】
// 1. 配置 OTLP Endpoint 与 Insecure
// 2. InitTracer 成功返回 Shutdown
// 3. 调用 Shutdown 清理全局状态
func TestInitTracer_OTLP_Insecure(t *testing.T) {
	resetGlobals()
	cfg := &config.TracingConfig{
		Enabled:      true,
		ServiceName:  "otlp-insecure-test",
		ExporterType: "otlp",
		Endpoint:     "127.0.0.1:4317",
		Insecure:     true,
		SampleRate:   1.0,
	}
	shutdown, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	require.NotNil(t, Tracer)
	require.NotNil(t, tracerProvider)
	assert.NoError(t, shutdown(context.Background()))
	resetGlobals()
}

// TestInitTracer_OTLP_TLS OTLP TLS 初始化
//
// 【功能点】Insecure=false 时应走默认 TLS 选项分支并成功初始化
// 【测试流程】配置 Insecure=false → InitTracer → Shutdown
func TestInitTracer_OTLP_TLS(t *testing.T) {
	resetGlobals()
	cfg := &config.TracingConfig{
		Enabled:      true,
		ServiceName:  "otlp-tls-test",
		ExporterType: "otlp",
		Endpoint:     "127.0.0.1:4317",
		Insecure:     false,
		SampleRate:   1.0,
	}
	shutdown, err := InitTracer(cfg)
	require.NoError(t, err)
	require.NotNil(t, shutdown)
	assert.NoError(t, shutdown(context.Background()))
	resetGlobals()
}

// TestHTTPRoundTrip_TracerNilButTracingEnabled HTTP 追踪启用但 Tracer 未设置
//
// 【功能点】EnableHTTPClientTracing=true 且 Tracer=nil 时 RoundTrip 仍应完成请求（StartSpan 退化）
// 【测试流程】重置全局 → 仅设置 tracingConfig → RoundTrip 本地测试服务
func TestHTTPRoundTrip_TracerNilButTracingEnabled(t *testing.T) {
	resetGlobals()
	Tracer = nil
	tracingConfig = &config.TracingConfig{
		Enabled:                 true,
		EnableHTTPClientTracing: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := NewTracingTransport(nil)
	req, err := http.NewRequestWithContext(context.Background(), "GET", server.URL+"/p", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()
	resetGlobals()
}

// TestRedisProcessHook_RedisNilIgnored ProcessHook 忽略 redis.Nil
//
// 【功能点】命令返回 redis.Nil 时不应标记 Span 错误状态
// 【测试流程】启用 Redis 追踪 → ProcessHook 返回 redis.Nil → 无 panic
func TestRedisProcessHook_RedisNilIgnored(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableRedisTracing: true}

	hook := NewRedisTracingHook("localhost:6379", "", 0)
	processFn := hook.ProcessHook(func(ctx context.Context, cmd redis.Cmder) error {
		return redis.Nil
	})
	cmd := redis.NewStringCmd(context.Background(), "GET", "missing")
	err := processFn(context.Background(), cmd)
	assert.ErrorIs(t, err, redis.Nil)
}

// TestGormTracingPlugin_Initialize GORM 插件注册
//
// 【功能点】Initialize 应成功注册各类回调
// 【测试流程】内存 SQLite → AutoMigrate → Initialize（避免迁移触发插件回调异常）
func TestGormTracingPlugin_Initialize(t *testing.T) {
	resetGlobals()
	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))
	p := NewGormTracingPlugin("memdb")
	require.NoError(t, p.Initialize(db))
}

// TestGormTracing_before_NilContext Statement.Context 为空时跳过 Span
//
// 【功能点】启用 DB 追踪但 Statement.Context==nil 时 before 应提前返回且不 panic
// 【测试流程】构造最小 *gorm.DB（Context=nil）→ 调用 before 闭包 → 无异常结束
func TestGormTracing_before_NilContext(t *testing.T) {
	resetGlobals()
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	inner := openTestSQLiteDB(t)
	db := &gorm.DB{}
	stmt := &gorm.Statement{DB: db, Context: nil}
	db.Statement = stmt
	db.Config = inner.Config

	NewGormTracingPlugin("x").before("db.probe")(db)
}

// TestFormatCmd_TotalLenExceededWithinArgCap Redis formatCmd 总字符上限（先于参数个数上限）
//
// 【功能点】累计长度超过 maxTotalLen 时在未达到「最多 10 个参数」分支即截断
// 【测试流程】构造短命令名 + 多条未超长参数使 totalLen>500 → formatCmd 含省略标记
func TestFormatCmd_TotalLenExceededWithinArgCap(t *testing.T) {
	hook := NewRedisTracingHook("localhost:6379", "", 0)
	args := []interface{}{"X"}
	for i := 0; i < 6; i++ {
		args = append(args, strings.Repeat("a", 100))
	}
	cmd := redis.NewStringCmd(context.Background(), args...)
	result := hook.formatCmd(cmd)
	assert.Contains(t, result, "...")
}

// TestGormTracing_Query_WithDBTracing 启用库追踪的 CRUD 路径
//
// 【功能点】EnableDBTracing=true 时 before/after 与 StartSpan 协作不产生错误
// 【测试流程】设置 Tracer 与配置 → AutoMigrate → Initialize → Create/First/Update/Row/Count/Delete
func TestGormTracing_Query_WithDBTracing(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("gorm-test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))
	require.NoError(t, NewGormTracingPlugin("mem").Initialize(db))

	ctx := context.Background()
	require.NoError(t, db.WithContext(ctx).Create(&tracingGormModel{Name: "a"}).Error)

	var got tracingGormModel
	require.NoError(t, db.WithContext(ctx).First(&got, 1).Error)
	assert.Equal(t, "a", got.Name)

	require.NoError(t, db.WithContext(ctx).Model(&tracingGormModel{}).Where("id = ?", 1).Update("name", "b").Error)

	row := db.WithContext(ctx).Raw("SELECT 1").Row()
	require.NoError(t, row.Err())

	var cnt int64
	require.NoError(t, db.WithContext(ctx).Model(&tracingGormModel{}).Count(&cnt).Error)
	assert.EqualValues(t, 1, cnt)

	require.NoError(t, db.WithContext(ctx).Delete(&tracingGormModel{}, 1).Error)
}

// TestGormTracing_Query_DBTracingDisabled 关闭库追踪
//
// 【功能点】EnableDBTracing=false 时插件回调提前返回，查询仍可执行
// 【测试流程】关闭 DB 追踪 → AutoMigrate → Create
func TestGormTracing_Query_DBTracingDisabled(t *testing.T) {
	resetGlobals()
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: false}

	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))
	require.NoError(t, NewGormTracingPlugin().Initialize(db))
	require.NoError(t, db.Create(&tracingGormModel{Name: "off"}).Error)
}

// TestGormTracing_after_NoSpanInInstance 未缓存 Span 时的 after
//
// 【功能点】InstanceGet(gormSpanKey) 失败时 after 应直接返回且不 panic
// 【测试流程】启用 DB 追踪 → Session 带 Context/SQL 但不 InstanceSet span → 调用 after
func TestGormTracing_after_NoSpanInInstance(t *testing.T) {
	resetGlobals()
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))

	sess := db.Session(&gorm.Session{})
	sess.Statement.Context = context.Background()
	sess.Statement.SQL.WriteString("SELECT 1")

	NewGormTracingPlugin().after(sess)
}

// TestGormTracing_after_WrongSpanType Instance 中非 Span 类型
//
// 【功能点】gormSpanKey 对应值类型错误时 after 应忽略并返回
// 【测试流程】构造最小 *gorm.DB → InstanceSet 字符串 → 调用 after
func TestGormTracing_after_WrongSpanType(t *testing.T) {
	resetGlobals()
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	inner := openTestSQLiteDB(t)
	db := &gorm.DB{}
	stmt := &gorm.Statement{DB: db, Context: context.Background()}
	db.Statement = stmt
	db.Config = inner.Config
	db.Statement.SQL.WriteString("SELECT 1")
	db.InstanceSet(gormSpanKey, "not-a-span")

	NewGormTracingPlugin().after(db)
}

// TestGormTracing_after_ErrRecordNotFound 记录未找到
//
// 【功能点】db.Error 为 gorm.ErrRecordNotFound 时不应 SetStatus Error
// 【测试流程】启用追踪 → First 不存在主键 → 断言 ErrRecordNotFound
func TestGormTracing_after_ErrRecordNotFound(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("gorm-test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))
	require.NoError(t, NewGormTracingPlugin().Initialize(db))

	var m tracingGormModel
	err := db.Where("id = ?", 99999).First(&m).Error
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// TestGormTracing_after_LongSQLTruncation 超长 SQL 截断
//
// 【功能点】SQL 超过 1000 字符时 after 应追加截断标记
// 【测试流程】构造长 WHERE 字面量 → Raw Scan → 不应 panic
func TestGormTracing_after_LongSQLTruncation(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("gorm-test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))
	require.NoError(t, NewGormTracingPlugin().Initialize(db))

	longLit := strings.Repeat("a", 1100)
	q := "SELECT 1 WHERE LENGTH('" + longLit + "') > 0"
	var x int
	require.NoError(t, db.Raw(q).Scan(&x).Error)
}

// TestGormTracing_RawSQL_ErrorRecordsOnSpan Raw SQL 错误
//
// 【功能点】db.Error 既不是 nil 也不是 ErrRecordNotFound 时 after 应 RecordError
// 【测试流程】建表并插入一行 → 引用不存在列的 Raw Scan → 返回底层 SQL 错误
func TestGormTracing_RawSQL_ErrorRecordsOnSpan(t *testing.T) {
	resetGlobals()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	Tracer = tp.Tracer("gorm-test")
	tracingConfig = &config.TracingConfig{Enabled: true, EnableDBTracing: true}

	db := openTestSQLiteDB(t)
	require.NoError(t, db.AutoMigrate(&tracingGormModel{}))
	require.NoError(t, NewGormTracingPlugin().Initialize(db))
	require.NoError(t, db.Create(&tracingGormModel{Name: "row"}).Error)

	var x int
	err := db.Raw("SELECT __nonexistent_column__ FROM tracing_gorm_models WHERE id = ?", 1).Scan(&x).Error
	require.Error(t, err)
	assert.NotErrorIs(t, err, gorm.ErrRecordNotFound)
}
