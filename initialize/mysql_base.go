// Package initialize 提供各种服务的初始化功能
// 本文件专门负责MySQL数据库的基础配置和GORM回调函数设置
package initialize

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// initDBCallbacks 初始化GORM数据库回调函数
// 该函数会：
// 1. 注册创建记录前的回调，自动填充时间字段
// 2. 注册更新记录前的回调，自动更新修改时间字段
func initDBCallbacks(gormDB *gorm.DB) {
	// 创建前把createTime和updateTime字段填充好默认值
	gormDB.Callback().Create().Before("gorm:create").Register("fill:createTime_updateTime", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}
		// 定义需要自动填充的时间字段名称
		timeFieldsToInit := []string{"CreateTime", "UpdateTime", "CreatedAt", "UpdatedAt"}
		for _, field := range timeFieldsToInit {

			if timeField := db.Statement.Schema.LookUpField(field); timeField != nil {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					// 处理批量创建的情况，遍历每个元素
					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						if _, isZero := timeField.ValueOf(db.Statement.Context, db.Statement.ReflectValue.Index(i)); isZero && db.Statement.ReflectValue.Index(i).CanAddr() {
							timeField.Set(db.Statement.Context, db.Statement.ReflectValue.Index(i), time.Now())
						}
					}
				case reflect.Struct:
					// 处理单个记录创建的情况
					if _, isZero := timeField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); isZero && db.Statement.ReflectValue.CanAddr() {
						timeField.Set(db.Statement.Context, db.Statement.ReflectValue, time.Now())
					}
				}
			}
		}
	})

	// 更新前修改updateTime字段
	gormDB.Callback().Update().Before("gorm:update").Register("update:updateTime", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}
		// 定义需要自动更新的时间字段名称
		timeFieldsToInit := []string{"UpdateTime", "UpdatedAt"}
		for _, field := range timeFieldsToInit {
			if timeField := db.Statement.Schema.LookUpField(field); timeField != nil && db.Statement.ReflectValue.CanAddr() {
				timeField.Set(db.Statement.Context, db.Statement.ReflectValue, time.Now())
			}
		}
	})
}

// 使用sync.Once确保数据库日志记录器只初始化一次
var initOnce sync.Once
var dbLogger *logrus.Logger

// initDBLogger 初始化数据库专用的日志记录器
// 使用单例模式确保只创建一个日志记录器实例
func initDBLogger() *logrus.Logger {
	initOnce.Do(func() {
		// 初始化日志记录器
		loggerConfig := app.BaseConfig.Log.ToDbLoggerConfig()
		dbLogger = logger.InitLogger(loggerConfig)
	})
	return dbLogger
}

// initGormLoggerConfig 初始化GORM日志配置
// 该函数会：
// 1. 设置是否忽略记录未找到错误
// 2. 设置日志级别
// 3. 设置慢查询阈值
func initGormLoggerConfig(dbConfig config.DbInfo) gormLogger.Config {
	// 是否忽略记录未找到错误，默认忽略
	ignoreRecordNotFoundError := true
	if dbConfig.IgnoreRecordNotFoundError != nil {
		ignoreRecordNotFoundError = *dbConfig.IgnoreRecordNotFoundError
	}
	// 日志级别，默认Warn级别
	logLevel := gormLogger.Warn
	if dbConfig.LogLevel != nil {
		logLevel = gormLogger.LogLevel(*dbConfig.LogLevel)
	}
	// 慢查询阈值，默认使用常量值
	slowThreshold := constant.DefaultDBSlowThreshold
	if dbConfig.SlowThreshold != nil {
		slowThreshold = *dbConfig.SlowThreshold
	}

	return gormLogger.Config{
		SlowThreshold:             time.Duration(slowThreshold) * time.Millisecond, // 慢查询阈值, 单位: 毫秒
		LogLevel:                  logLevel,                                        // 日志级别
		IgnoreRecordNotFoundError: ignoreRecordNotFoundError,                       // 忽略ErrRecordNotFound（记录未找到）错误
		Colorful:                  true,                                            // 彩色打印
	}
}

// initGormConfig 初始化GORM配置
// 该函数会：
// 1. 设置数据库迁移时禁用外键约束
// 2. 配置命名策略（单数表名、表前缀等）
// 3. 设置GORM日志记录器
func initGormConfig(dbConfig config.DbInfo) *gorm.Config {
	// 设置单数表名，默认使用单数表名
	singularTable := true
	if dbConfig.SingularTable != nil {
		singularTable = *dbConfig.SingularTable
	}

	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true, // 迁移时禁用外键约束
		NamingStrategy: schema.NamingStrategy{
			SingularTable: singularTable,        // 使用单数表名，启用该选项后，`User` 表将是`user`
			TablePrefix:   dbConfig.TablePrefix, // 表名前缀，`User`表为`t_users`
			//NameReplacer:  strings.NewReplacer("CID", "Cid"), // 在转为数据库名称之前，使用NameReplacer更改结构/字段名称。
		},
	}

	// 设置GORM日志记录器
	gormConfig.Logger = gormLogger.New(
		initDBLogger(),
		initGormLoggerConfig(dbConfig),
	)
	return gormConfig
}

// initDBConnConfig 初始化数据库连接池配置
// 该函数会：
// 1. 获取底层sql.DB实例
// 2. 设置最大空闲连接数
// 3. 设置最大打开连接数
// 4. 设置连接最大空闲时间
// 5. 设置连接最大生命周期
func initDBConnConfig(gormDB *gorm.DB, dbConfig config.DbInfo) error {
	// 获取底层的sql.DB实例以配置连接池
	SqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("获取 sqlDB 失败: %w", err)
	}

	// 设置连接池参数，使用配置值或默认值
	SqlDB.SetMaxIdleConns(max(dbConfig.MaxIdleConns, 10))                                    // 最大空闲连接数，默认10
	SqlDB.SetMaxOpenConns(max(dbConfig.MaxOpenConns, 100))                                   // 最大打开连接数，默认100
	SqlDB.SetConnMaxIdleTime(time.Duration(max(dbConfig.ConnMaxIdleTime, 60)) * time.Second) // 连接最大空闲时间，默认60秒
	SqlDB.SetConnMaxLifetime(time.Duration(max(dbConfig.ConnMaxLifetime, 60)) * time.Second) // 连接最大生命周期，默认60秒
	return nil
}
