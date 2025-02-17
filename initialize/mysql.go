package initialize

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
	gormLogger "gorm.io/gorm/logger"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var tableEntity []interface{}

func InitDB() {
	global.DB = initSingleDB(global.BaseConfig.Db)
	logger.Info("[db] db已初始化")
}

func InitDBList() {
	global.DBList = make(map[string]*gorm.DB)
	for _, dbConfig := range global.BaseConfig.DbList {
		global.DBList[dbConfig.AliasName] = initSingleDB(dbConfig)
	}
}

func initSingleDB(dbConfig config.DbInfo) *gorm.DB {
	var err error

	gormConfig := &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,                 // 使用单数表名，启用该选项后，`User` 表将是`user`
			TablePrefix:   dbConfig.TablePrefix, // 表名前缀，`User`表为`t_users`
			//NameReplacer:  strings.NewReplacer("CID", "Cid"), // 在转为数据库名称之前，使用NameReplacer更改结构/字段名称。
		},
	}
	if dbConfig.EnableLog {
		newLogger := gormLogger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer（日志输出的目标，前缀和日志包含的内容——译者注）
			gormLogger.Config{
				SlowThreshold:             time.Duration(dbConfig.SlowThreshold) * time.Millisecond, // 慢查询阈值
				LogLevel:                  gormLogger.Info,                                          // 日志级别
				IgnoreRecordNotFoundError: true,                                                     // 忽略ErrRecordNotFound（记录未找到）错误
				Colorful:                  true,                                                     // 彩色打印
			},
		)
		gormConfig.Logger = newLogger
	}

	DB, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dbConfig.Dsn(),
	}), gormConfig)
	if err != nil {
		logger.Error("[db] error occurs while initializing db, error: %v", err)
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
						fmt.Println(isZero)
						timeField.Set(db.Statement.Context, db.Statement.ReflectValue, time.Now())
					}
				}
			}
		}
	})

	SqlDB, err := DB.DB()
	if err != nil {
		logger.Error("[db] get sqlDB err: %v", err)
		return nil
	}
	SqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	SqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)

	Migrate(DB, dbConfig.Migrate)
	return DB
}

func RegisterTable(table interface{}) {
	tableEntity = append(tableEntity, table)
}

func Migrate(db *gorm.DB, migrateMode string) {
	if migrateMode != "update" && migrateMode != "create" {
		return
	}
	if migrateMode == "create" {
		//删除表
		_ = db.Migrator().DropTable(tableEntity...)
	}

	//更新表
	_ = db.AutoMigrate(tableEntity...)
}
