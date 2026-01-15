// Package tracing 提供基于 OpenTelemetry 的分布式链路追踪功能
// 本文件实现了 GORM 数据库操作的追踪插件
package tracing

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	// gormSpanKey 用于在 GORM 实例中存储 Span 的键名
	gormSpanKey = "otel:span"
	// gormPluginName 插件名称
	gormPluginName = "otel-tracing"
)

// GormTracingPlugin GORM 链路追踪插件
// 实现了 gorm.Plugin 接口，用于追踪数据库操作
type GormTracingPlugin struct {
	// dbName 数据库名称，用于标识追踪来源
	dbName string
}

// NewGormTracingPlugin 创建新的 GORM 追踪插件实例
// 参数：
//   - dbName: 数据库名称，可选，用于在追踪中标识数据库
//
// 返回：
//   - *GormTracingPlugin: 追踪插件实例
func NewGormTracingPlugin(dbName ...string) *GormTracingPlugin {
	name := "default"
	if len(dbName) > 0 && dbName[0] != "" {
		name = dbName[0]
	}
	return &GormTracingPlugin{dbName: name}
}

// Name 返回插件名称
// 实现 gorm.Plugin 接口
func (p *GormTracingPlugin) Name() string {
	return gormPluginName
}

// Initialize 初始化插件，注册 GORM 回调函数
// 实现 gorm.Plugin 接口
// 该函数会：
// 1. 注册创建、查询、更新、删除等操作的 Before 回调（创建 Span）
// 2. 注册对应操作的 After 回调（结束 Span 并记录信息）
func (p *GormTracingPlugin) Initialize(db *gorm.DB) error {
	// 注册 Before 回调 - 在数据库操作执行前创建 Span
	if err := db.Callback().Create().Before("gorm:create").Register("otel:before_create", p.before("db.create")); err != nil {
		return fmt.Errorf("注册 before_create 回调失败: %w", err)
	}
	if err := db.Callback().Query().Before("gorm:query").Register("otel:before_query", p.before("db.query")); err != nil {
		return fmt.Errorf("注册 before_query 回调失败: %w", err)
	}
	if err := db.Callback().Update().Before("gorm:update").Register("otel:before_update", p.before("db.update")); err != nil {
		return fmt.Errorf("注册 before_update 回调失败: %w", err)
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register("otel:before_delete", p.before("db.delete")); err != nil {
		return fmt.Errorf("注册 before_delete 回调失败: %w", err)
	}
	if err := db.Callback().Row().Before("gorm:row").Register("otel:before_row", p.before("db.row")); err != nil {
		return fmt.Errorf("注册 before_row 回调失败: %w", err)
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register("otel:before_raw", p.before("db.raw")); err != nil {
		return fmt.Errorf("注册 before_raw 回调失败: %w", err)
	}

	// 注册 After 回调 - 在数据库操作执行后结束 Span
	if err := db.Callback().Create().After("gorm:create").Register("otel:after_create", p.after); err != nil {
		return fmt.Errorf("注册 after_create 回调失败: %w", err)
	}
	if err := db.Callback().Query().After("gorm:query").Register("otel:after_query", p.after); err != nil {
		return fmt.Errorf("注册 after_query 回调失败: %w", err)
	}
	if err := db.Callback().Update().After("gorm:update").Register("otel:after_update", p.after); err != nil {
		return fmt.Errorf("注册 after_update 回调失败: %w", err)
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("otel:after_delete", p.after); err != nil {
		return fmt.Errorf("注册 after_delete 回调失败: %w", err)
	}
	if err := db.Callback().Row().After("gorm:row").Register("otel:after_row", p.after); err != nil {
		return fmt.Errorf("注册 after_row 回调失败: %w", err)
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("otel:after_raw", p.after); err != nil {
		return fmt.Errorf("注册 after_raw 回调失败: %w", err)
	}

	return nil
}

// before 返回操作执行前的回调函数
// 该回调函数会：
// 1. 从上下文中创建新的子 Span
// 2. 设置 Span 的基本属性（数据库系统、表名等）
// 3. 将 Span 存储到 GORM 实例中，供 after 回调使用
func (p *GormTracingPlugin) before(operationName string) func(*gorm.DB) {
	return func(db *gorm.DB) {
		if !IsDBTracingEnabled() {
			return
		}

		ctx := db.Statement.Context
		if ctx == nil {
			return
		}

		// 创建子 Span
		ctx, span := StartSpan(ctx, operationName,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemMySQL,
				attribute.String("db.name", p.dbName),
				attribute.String("db.table", db.Statement.Table),
				attribute.String("db.operation", operationName),
			),
		)

		// 更新上下文并存储 Span
		db.Statement.Context = ctx
		db.InstanceSet(gormSpanKey, span)
	}
}

// after 操作执行后的回调函数
// 该回调函数会：
// 1. 从 GORM 实例中获取之前存储的 Span
// 2. 记录 SQL 语句和影响行数
// 3. 如果发生错误，记录错误信息
// 4. 结束 Span
func (p *GormTracingPlugin) after(db *gorm.DB) {
	if !IsDBTracingEnabled() {
		return
	}

	// 获取之前存储的 Span
	v, ok := db.InstanceGet(gormSpanKey)
	if !ok {
		return
	}

	span, ok := v.(trace.Span)
	if !ok {
		return
	}
	defer span.End()

	// 记录 SQL 语句（注意：生产环境可能需要脱敏处理）
	sql := db.Statement.SQL.String()
	if sql != "" {
		// 限制 SQL 长度，避免过长
		if len(sql) > 1000 {
			sql = sql[:1000] + "...(truncated)"
		}
		span.SetAttributes(semconv.DBStatement(sql))
	}

	// 记录影响的行数
	span.SetAttributes(attribute.Int64("db.rows_affected", db.Statement.RowsAffected))

	// 记录错误
	if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
		span.RecordError(db.Error)
		span.SetStatus(codes.Error, db.Error.Error())
	}
}
