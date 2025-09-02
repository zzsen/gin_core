// Package initialize 提供各种服务的初始化功能
// 本文件专门负责MySQL数据库的初始化，支持单数据库和多数据库列表配置
package initialize

import (
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// tableEntity 存储需要迁移的表实体，用于数据库表结构自动迁移
var tableEntity []any

// InitDB 初始化单个数据库连接
// 该函数会：
// 1. 检查数据库配置是否存在
// 2. 初始化单个数据库连接
// 3. 将数据库实例存储到全局app.DB中
func InitDB() {
	// 检查数据库配置是否存在
	if app.BaseConfig.Db == nil {
		logger.Error("[db] 未找到配置, 请检查配置")
		return
	}

	// 初始化单个数据库连接
	dbClient, err := initSingleDB(*app.BaseConfig.Db)
	if err != nil {
		panic(fmt.Errorf("[db] 初始化db失败, %s", err.Error()))
	}

	// 将数据库实例存储到全局变量中，供其他模块使用
	app.DB = dbClient
	logger.Info("[db] db已初始化")
}

// InitDBList 初始化多个数据库连接列表
// 该函数会：
// 1. 创建数据库实例映射表
// 2. 遍历所有数据库配置并初始化连接
// 3. 将数据库实例按别名存储到全局app.DBList中
func InitDBList() {
	// 初始化数据库实例映射表
	app.DBList = make(map[string]*gorm.DB)

	// 遍历所有数据库配置并初始化连接
	for _, dbConfig := range app.BaseConfig.DbList {
		dbClient, err := initSingleDB(dbConfig)
		if err != nil {
			panic(fmt.Errorf("[db] 初始化db [%s] 失败, %s", dbConfig.AliasName, err.Error()))
		}
		// 将数据库实例按别名存储到映射表中
		app.DBList[dbConfig.AliasName] = dbClient
	}
	logger.Info("[db] db列表已初始化")
}

// initSingleDB 初始化单个数据库连接
// 该函数会：
// 1. 创建GORM配置
// 2. 建立MySQL数据库连接
// 3. 初始化数据库回调函数
// 4. 配置数据库连接池
// 5. 执行数据库迁移（如果配置了迁移模式）
func initSingleDB(dbConfig config.DbInfo) (*gorm.DB, error) {
	var err error

	// 初始化GORM配置
	gormConfig := initGormConfig(dbConfig)

	// 建立MySQL数据库连接
	DB, err := gorm.Open(mysql.New(mysql.Config{
		DSN: dbConfig.Dsn(), // 数据库连接字符串
	}), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 mysql client 失败: %v", err)
	}

	// 初始化数据库回调函数（如自动时间字段填充等）
	initDBCallbacks(DB)

	// 配置数据库连接池参数
	initDBConnConfig(DB, dbConfig)

	// 执行数据库迁移（根据配置的迁移模式）
	Migrate(DB, dbConfig.Migrate)
	return DB, nil
}

// RegisterTable 注册需要迁移的表实体
// 该函数用于收集需要自动迁移的表结构定义
// 参数：
//   - table: 表结构实体（通常是GORM模型结构体）
func RegisterTable(table any) {
	tableEntity = append(tableEntity, table)
}

// Migrate 执行数据库表结构迁移
// 该函数会：
// 1. 检查迁移模式是否有效
// 2. 根据模式执行相应的迁移操作
// 3. 支持"create"和"update"两种迁移模式
// 参数：
//   - db: 数据库连接实例
//   - migrateMode: 迁移模式（"create"或"update"）
func Migrate(db *gorm.DB, migrateMode string) {
	// 检查迁移模式是否有效
	if migrateMode != "update" && migrateMode != "create" {
		return
	}

	// 如果是创建模式，先删除已存在的表
	if migrateMode == "create" {
		// 删除表
		_ = db.Migrator().DropTable(tableEntity...)
	}

	// 执行表结构自动迁移（创建表或更新表结构）
	_ = db.AutoMigrate(tableEntity...)
}
