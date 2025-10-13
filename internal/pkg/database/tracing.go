package database

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const (
	// 用于GORM追踪的仪器名称
	instrumentationName = "internal/pkg/database/tracing"
)

// GormTracingPlugin 是一个实现了gorm.Plugin接口的追踪插件
// 它为所有数据库操作添加OpenTelemetry追踪功能
type GormTracingPlugin struct {
	// 可选的追踪器，如果为nil则使用全局追踪器
	tracer trace.Tracer
}

// NewGormTracingPlugin 创建一个新的GORM追踪插件
func NewGormTracingPlugin() *GormTracingPlugin {
	return &GormTracingPlugin{
		tracer: otel.GetTracerProvider().Tracer(instrumentationName),
	}
}

// Name 返回插件名称
func (p *GormTracingPlugin) Name() string {
	return "GormTracingPlugin"
}

// Initialize 初始化插件，注册GORM回调
func (p *GormTracingPlugin) Initialize(db *gorm.DB) error {
	// 查询操作
	if err := db.Callback().Query().Before("gorm:query").Register("tracing:before_query", p.beforeQuery); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("tracing:after_query", p.afterQuery); err != nil {
		return err
	}

	// 创建操作
	if err := db.Callback().Create().Before("gorm:create").Register("tracing:before_create", p.beforeCreate); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register("tracing:after_create", p.afterCreate); err != nil {
		return err
	}

	// 更新操作
	if err := db.Callback().Update().Before("gorm:update").Register("tracing:before_update", p.beforeUpdate); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("tracing:after_update", p.afterUpdate); err != nil {
		return err
	}

	// 删除操作
	if err := db.Callback().Delete().Before("gorm:delete").Register("tracing:before_delete", p.beforeDelete); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("tracing:after_delete", p.afterDelete); err != nil {
		return err
	}

	// 原始SQL操作
	if err := db.Callback().Raw().Before("gorm:raw").Register("tracing:before_raw", p.beforeRaw); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("tracing:after_raw", p.afterRaw); err != nil {
		return err
	}

	return nil
}

// 辅助函数：从GORM DB中提取上下文
func extractContext(db *gorm.DB) context.Context {
	if db.Statement == nil {
		return context.Background()
	}
	return db.Statement.Context
}

// 辅助函数：设置span的通用属性
func setSpanAttributes(span trace.Span, db *gorm.DB) {
	// 设置一些基本的数据库属性
	attributes := []attribute.KeyValue{
		attribute.String("db.system", "mysql"), // 或其他数据库类型
		attribute.String("db.name", db.Dialector.Name()),
	}

	// 添加操作类型
	if db.Statement.Schema != nil {
		attributes = append(attributes, attribute.String("db.table", db.Statement.Schema.Table))
	} else if db.Statement.Table != "" {
		attributes = append(attributes, attribute.String("db.table", db.Statement.Table))
	}

	// 推断操作类型
	opType := "UNKNOWN"
	switch {
	case db.Statement.ReflectValue.Kind() == reflect.Slice && db.Statement.SQL.String() != "" && strings.HasPrefix(strings.ToUpper(db.Statement.SQL.String()), "SELECT"):
		opType = "SELECT"
	case db.Statement.ReflectValue.Kind() == reflect.Slice && db.Statement.SQL.String() != "" && strings.HasPrefix(strings.ToUpper(db.Statement.SQL.String()), "INSERT"):
		opType = "INSERT"
	case db.Statement.SQL.String() != "" && strings.HasPrefix(strings.ToUpper(db.Statement.SQL.String()), "UPDATE"):
		opType = "UPDATE"
	case db.Statement.SQL.String() != "" && strings.HasPrefix(strings.ToUpper(db.Statement.SQL.String()), "DELETE"):
		opType = "DELETE"
	}
	attributes = append(attributes, attribute.String("db.operation", opType))

	// 当SQL语句准备好时添加
	if db.Statement.SQL.String() != "" {
		attributes = append(attributes, attribute.String("db.statement", db.Statement.SQL.String()))
	}

	// 添加影响的行数
	if db.Statement.RowsAffected > 0 {
		attributes = append(attributes, attribute.Int64("db.rows_affected", db.Statement.RowsAffected))
	}

	span.SetAttributes(attributes...)
}

// 查询操作的回调
func (p *GormTracingPlugin) beforeQuery(db *gorm.DB) {
	ctx := extractContext(db)
	spanName := fmt.Sprintf("%s SELECT", db.Statement.Table)
	ctx, span := p.tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	// 存储span以便在afterQuery中使用
	db.Statement.Context = ctx
	db.Set("tracing:span", span)
}

func (p *GormTracingPlugin) afterQuery(db *gorm.DB) {
	spanValue, exists := db.Get("tracing:span")
	if !exists {
		return
	}

	if span, ok := spanValue.(trace.Span); ok {
		defer span.End()

		setSpanAttributes(span, db)

		// 记录错误（如果有）
		if db.Error != nil && !errors.Is(db.Error, gorm.ErrRecordNotFound) {
			span.SetStatus(codes.Error, db.Error.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// 创建操作的回调
func (p *GormTracingPlugin) beforeCreate(db *gorm.DB) {
	ctx := extractContext(db)
	spanName := fmt.Sprintf("%s INSERT", db.Statement.Table)
	ctx, span := p.tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	// 存储span以便在afterCreate中使用
	db.Statement.Context = ctx
	db.Set("tracing:span", span)
}

func (p *GormTracingPlugin) afterCreate(db *gorm.DB) {
	spanValue, exists := db.Get("tracing:span")
	if !exists {
		return
	}

	if span, ok := spanValue.(trace.Span); ok {
		defer span.End()

		setSpanAttributes(span, db)

		// 记录错误（如果有）
		if db.Error != nil {
			span.SetStatus(codes.Error, db.Error.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// 更新操作的回调
func (p *GormTracingPlugin) beforeUpdate(db *gorm.DB) {
	ctx := extractContext(db)
	spanName := fmt.Sprintf("%s UPDATE", db.Statement.Table)
	ctx, span := p.tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	// 存储span以便在afterUpdate中使用
	db.Statement.Context = ctx
	db.Set("tracing:span", span)
}

func (p *GormTracingPlugin) afterUpdate(db *gorm.DB) {
	spanValue, exists := db.Get("tracing:span")
	if !exists {
		return
	}

	if span, ok := spanValue.(trace.Span); ok {
		defer span.End()

		setSpanAttributes(span, db)

		// 记录错误（如果有）
		if db.Error != nil {
			span.SetStatus(codes.Error, db.Error.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// 删除操作的回调
func (p *GormTracingPlugin) beforeDelete(db *gorm.DB) {
	ctx := extractContext(db)
	spanName := fmt.Sprintf("%s DELETE", db.Statement.Table)
	ctx, span := p.tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	// 存储span以便在afterDelete中使用
	db.Statement.Context = ctx
	db.Set("tracing:span", span)
}

func (p *GormTracingPlugin) afterDelete(db *gorm.DB) {
	spanValue, exists := db.Get("tracing:span")
	if !exists {
		return
	}

	if span, ok := spanValue.(trace.Span); ok {
		defer span.End()

		setSpanAttributes(span, db)

		// 记录错误（如果有）
		if db.Error != nil {
			span.SetStatus(codes.Error, db.Error.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

// 原始SQL操作的回调
func (p *GormTracingPlugin) beforeRaw(db *gorm.DB) {
	ctx := extractContext(db)
	spanName := "SQL RAW"
	if db.Statement.Table != "" {
		spanName = fmt.Sprintf("%s RAW", db.Statement.Table)
	}

	ctx, span := p.tracer.Start(
		ctx,
		spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)

	// 存储span以便在afterRaw中使用
	db.Statement.Context = ctx
	db.Set("tracing:span", span)
}

func (p *GormTracingPlugin) afterRaw(db *gorm.DB) {
	spanValue, exists := db.Get("tracing:span")
	if !exists {
		return
	}

	if span, ok := spanValue.(trace.Span); ok {
		defer span.End()

		setSpanAttributes(span, db)

		// 记录错误（如果有）
		if db.Error != nil {
			span.SetStatus(codes.Error, db.Error.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}
