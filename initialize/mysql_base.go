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

func initDBCallbacks(gormDB *gorm.DB) {
	// 创建前把createTime和updateTime字段填充好默认值
	gormDB.Callback().Create().Before("gorm:create").Register("fill:createTime_updateTime", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}
		timeFieldsToInit := []string{"CreateTime", "UpdateTime", "CreatedAt", "UpdatedAt"}
		for _, field := range timeFieldsToInit {

			if timeField := db.Statement.Schema.LookUpField(field); timeField != nil {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						if _, isZero := timeField.ValueOf(db.Statement.Context, db.Statement.ReflectValue.Index(i)); isZero && db.Statement.ReflectValue.Index(i).CanAddr() {
							timeField.Set(db.Statement.Context, db.Statement.ReflectValue.Index(i), time.Now())
						}
					}
				case reflect.Struct:
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
		timeFieldsToInit := []string{"UpdateTime", "UpdatedAt"}
		for _, field := range timeFieldsToInit {
			if timeField := db.Statement.Schema.LookUpField(field); timeField != nil {
				db.Statement.Dest = map[string]any{timeField.DBName: db.Statement.DB.NowFunc()}
			}
		}
	})
}

var initOnce sync.Once
var dbLogger *logrus.Logger

func initDBLogger() *logrus.Logger {
	initOnce.Do(func() {
		// 初始化日志记录器
		loggerConfig := app.BaseConfig.Log.ToDbLoggerConfig()
		dbLogger = logger.InitLogger(loggerConfig)
	})
	return dbLogger
}

func initGormLoggerConfig(dbConfig config.DbInfo) gormLogger.Config {
	// 是否忽略记录未找到错误
	ignoreRecordNotFoundError := true
	if dbConfig.IgnoreRecordNotFoundError != nil {
		ignoreRecordNotFoundError = *dbConfig.IgnoreRecordNotFoundError
	}
	// 日志级别
	logLevel := gormLogger.Warn
	if dbConfig.LogLevel != nil {
		logLevel = gormLogger.LogLevel(*dbConfig.LogLevel)
	}
	// 慢查询阈值
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

func initGormConfig(dbConfig config.DbInfo) *gorm.Config {
	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,                 // 使用单数表名，启用该选项后，`User` 表将是`user`
			TablePrefix:   dbConfig.TablePrefix, // 表名前缀，`User`表为`t_users`
			//NameReplacer:  strings.NewReplacer("CID", "Cid"), // 在转为数据库名称之前，使用NameReplacer更改结构/字段名称。
		},
	}

	gormConfig.Logger = gormLogger.New(
		initDBLogger(),
		initGormLoggerConfig(dbConfig),
	)
	return gormConfig
}

func initDBConnConfig(gormDB *gorm.DB, dbConfig config.DbInfo) error {
	SqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("获取 sqlDB 失败: %v", err)
	}
	SqlDB.SetMaxIdleConns(max(dbConfig.MaxIdleConns, 10))
	SqlDB.SetMaxOpenConns(max(dbConfig.MaxOpenConns, 100))
	SqlDB.SetConnMaxIdleTime(time.Duration(max(dbConfig.ConnMaxIdleTime, 60)) * time.Second)
	SqlDB.SetConnMaxLifetime(time.Duration(max(dbConfig.ConnMaxLifetime, 60)) * time.Second)
	return nil
}
