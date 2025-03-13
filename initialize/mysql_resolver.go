package initialize

import (
	"reflect"
	"time"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/dbresolver"
)

func InitDBResolver() {
	if len(global.BaseConfig.DbResolvers) > 0 {
		global.DBResolver = initMultiDB(global.BaseConfig.DbResolvers)
		logger.Info("[db] db resolver已初始化")
	}
}

func initMultiDB(dbResolvers config.DbResolvers) *gorm.DB {
	defaultDBConfig := dbResolvers.DefaultConfig()
	var err error

	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,                        // 使用单数表名，启用该选项后，`User` 表将是`user`
			TablePrefix:   defaultDBConfig.TablePrefix, // 表名前缀，`User`表为`t_users`
			//NameReplacer:  strings.NewReplacer("CID", "Cid"), // 在转为数据库名称之前，使用NameReplacer更改结构/字段名称。
		},
	}
	loggerConfig := global.BaseConfig.Log.ToDbLoggerConfig()
	dbLogger := logger.InitLogger(loggerConfig)
	ignoreRecordNotFoundError := true
	if defaultDBConfig.IgnoreRecordNotFoundError != nil {
		ignoreRecordNotFoundError = *defaultDBConfig.IgnoreRecordNotFoundError
	}

	gormConfig.Logger = gormLogger.New(
		dbLogger,
		gormLogger.Config{
			SlowThreshold:             time.Duration(defaultDBConfig.SlowThreshold) * time.Millisecond, // 慢查询阈值
			LogLevel:                  gormLogger.Info,                                                 // 日志级别
			IgnoreRecordNotFoundError: ignoreRecordNotFoundError,                                       // 忽略ErrRecordNotFound（记录未找到）错误
			Colorful:                  true,                                                            // 彩色打印
		},
	)

	DB, err := gorm.Open(mysql.New(mysql.Config{
		DSN: defaultDBConfig.Dsn(),
	}), gormConfig)
	if err != nil {
		logger.Error("error occurs while initializing db resolver, error: %v", err)
		return nil
	}

	resolverPlugin := &dbresolver.DBResolver{}
	for _, resolver := range dbResolvers {
		if !resolver.IsValid() {
			logger.Error("exists invalid db resolver config, please check config")
			return nil
		}
		resolverPlugin.Register(dbresolver.Config{
			Sources:           resolver.SourceConfigs(),
			Replicas:          resolver.ReplicaConfigs(),
			TraceResolverMode: true,
		}, resolver.Tables...)
	}
	resolverPlugin.SetMaxIdleConns(defaultDBConfig.MaxIdleConns)
	resolverPlugin.SetMaxOpenConns(defaultDBConfig.MaxOpenConns)

	if err := DB.Use(resolverPlugin); err != nil {
		logger.Error("error occurs while db trying to use db resolver, error: %v", err)
		return nil
	}

	//创建前把createTime和updateTime字段填充好默认值
	DB.Callback().Create().Before("gorm:create").Register("update_create_time", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}
		timeFieldsToInit := []string{"CreateTime", "UpdateTime"}
		for _, field := range timeFieldsToInit {

			if timeField := db.Statement.Schema.LookUpField(field); timeField != nil {
				switch db.Statement.ReflectValue.Kind() {
				case reflect.Slice, reflect.Array:
					for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
						if _, isZero := timeField.ValueOf(db.Statement.Context, db.Statement.ReflectValue.Index(i)); isZero {
							timeField.Set(db.Statement.Context, db.Statement.ReflectValue.Index(i), time.Now())
						}
					}
				case reflect.Struct:
					if _, isZero := timeField.ValueOf(db.Statement.Context, db.Statement.ReflectValue); isZero {
						timeField.Set(db.Statement.Context, db.Statement.ReflectValue, time.Now())
					}
				}
			}
		}
	})

	return DB
}
