// Package initialize 提供各种服务的初始化功能
// 本文件专门负责MySQL数据库解析器的初始化，支持读写分离和分库分表
package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// InitDBResolver 初始化数据库解析器
// 该函数会：
// 1. 检查是否配置了数据库解析器
// 2. 初始化多数据库配置（支持读写分离、分库分表）
// 3. 将解析器实例存储到全局app.DBResolver中
func InitDBResolver() {
	// 检查是否配置了数据库解析器
	if len(app.BaseConfig.DbResolvers) > 0 {
		// 初始化多数据库配置
		dbClient, err := initMultiDB(app.BaseConfig.DbResolvers)
		if err != nil {
			panic(exception.NewInitError("db", "初始化解析器", err))
		}
		// 将数据库解析器实例存储到全局变量中，供其他模块使用
		app.DBResolver = dbClient
		logger.Info("[db] db resolver已初始化")
	}
}

// initMultiDB 初始化多数据库配置
// 该函数会：
// 1. 获取默认数据库配置并创建GORM配置
// 2. 建立与默认数据库的连接
// 3. 配置数据库解析器插件（支持读写分离、分库分表）
// 4. 设置连接池参数
// 5. 启用解析器插件并初始化数据库回调函数
func initMultiDB(dbResolvers config.DbResolvers) (*gorm.DB, error) {
	// 获取默认数据库配置
	defaultDBConfig := dbResolvers.DefaultConfig()

	// 初始化GORM配置
	gormConfig := initGormConfig(defaultDBConfig)

	// 使用默认配置建立MySQL连接
	DB, err := gorm.Open(mysql.New(mysql.Config{
		DSN: defaultDBConfig.Dsn(), // 数据库连接字符串
	}), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 mysql client 失败: %v", err)
	}

	// 创建数据库解析器插件
	resolverPlugin := &dbresolver.DBResolver{}

	// 遍历所有解析器配置并注册到插件中
	for _, resolver := range dbResolvers {
		// 验证解析器配置的有效性
		if !resolver.IsValid() {
			return nil, fmt.Errorf("无效的db resolver配置, 请检查配置")
		}

		// 注册解析器配置，支持读写分离和分库分表
		resolverPlugin.Register(dbresolver.Config{
			Sources:           resolver.SourceConfigs(),  // 主库配置（写操作）
			Replicas:          resolver.ReplicaConfigs(), // 从库配置（读操作）
			TraceResolverMode: true,                      // 启用解析器模式追踪
		}, resolver.Tables...) // 指定该解析器适用的表名
	}

	// 设置连接池参数，使用配置值或默认值
	resolverPlugin.SetMaxIdleConns(max(defaultDBConfig.MaxIdleConns, 10))                                    // 最大空闲连接数，默认10
	resolverPlugin.SetMaxOpenConns(max(defaultDBConfig.MaxOpenConns, 100))                                   // 最大打开连接数，默认100
	resolverPlugin.SetConnMaxIdleTime(time.Duration(max(defaultDBConfig.ConnMaxIdleTime, 60)) * time.Second) // 连接最大空闲时间，默认60秒
	resolverPlugin.SetConnMaxLifetime(time.Duration(max(defaultDBConfig.ConnMaxLifetime, 60)) * time.Second) // 连接最大生命周期，默认60秒

	// 将解析器插件应用到数据库连接
	if err := DB.Use(resolverPlugin); err != nil {
		return nil, fmt.Errorf("启用db resolver plugin失败: %v", err)
	}

	// 初始化数据库回调函数（如自动时间字段填充等）
	initDBCallbacks(DB)

	return DB, nil
}
