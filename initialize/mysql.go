package initialize

import (
	"fmt"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var tableEntity []interface{}

func InitDB() {
	if global.BaseConfig.Db == nil {
		logger.Error("[db] 未找到配置, 请检查配置")
		return
	}
	dbClient, err := initSingleDB(*global.BaseConfig.Db)
	if err != nil {
		panic(fmt.Errorf("[db] 初始化db失败, %s", err.Error()))
	}
	global.DB = dbClient
	logger.Info("[db] db已初始化")
}

func InitDBList() {
	global.DBList = make(map[string]*gorm.DB)
	for _, dbConfig := range global.BaseConfig.DbList {
		dbClient, err := initSingleDB(dbConfig)
		if err != nil {
			panic(fmt.Errorf("[db] 初始化db [%s] 失败, %s", dbConfig.AliasName, err.Error()))
		}
		global.DBList[dbConfig.AliasName] = dbClient
	}
	logger.Info("[db] db列表已初始化")
}

func initSingleDB(dbConfig config.DbInfo) (*gorm.DB, error) {
	var err error

	gormConfig := initGormConfig(dbConfig)

	DB, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dbConfig.Dsn(),
	}), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 mysql client 失败: %v", err)
	}

	initDBCallbacks(DB)

	initDBConnConfig(DB, dbConfig)

	Migrate(DB, dbConfig.Migrate)
	return DB, nil
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
