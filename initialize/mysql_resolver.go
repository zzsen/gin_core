package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

func InitDBResolver() {
	if len(app.BaseConfig.DbResolvers) > 0 {
		dbClient, err := initMultiDB(app.BaseConfig.DbResolvers)
		if err != nil {
			panic(fmt.Errorf("[db] 初始化db resolver失败, %s", err.Error()))
		}
		app.DBResolver = dbClient
		logger.Info("[db] db resolver已初始化")
	}
}

func initMultiDB(dbResolvers config.DbResolvers) (*gorm.DB, error) {
	defaultDBConfig := dbResolvers.DefaultConfig()

	gormConfig := initGormConfig(defaultDBConfig)

	DB, err := gorm.Open(mysql.New(mysql.Config{
		DSN: defaultDBConfig.Dsn(),
	}), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("初始化 mysql client 失败: %v", err)
	}

	resolverPlugin := &dbresolver.DBResolver{}
	for _, resolver := range dbResolvers {
		if !resolver.IsValid() {
			return nil, fmt.Errorf("无效的db resolver配置, 请检查配置")
		}
		resolverPlugin.Register(dbresolver.Config{
			Sources:           resolver.SourceConfigs(),
			Replicas:          resolver.ReplicaConfigs(),
			TraceResolverMode: true,
		}, resolver.Tables...)
	}

	resolverPlugin.SetMaxIdleConns(max(defaultDBConfig.MaxIdleConns, 10))
	resolverPlugin.SetMaxOpenConns(max(defaultDBConfig.MaxOpenConns, 100))
	resolverPlugin.SetConnMaxIdleTime(time.Duration(max(defaultDBConfig.ConnMaxIdleTime, 60)) * time.Second)
	resolverPlugin.SetConnMaxLifetime(time.Duration(max(defaultDBConfig.ConnMaxLifetime, 60)) * time.Second)

	if err := DB.Use(resolverPlugin); err != nil {
		return nil, fmt.Errorf("启用db resolver plugin失败: %v", err)
	}

	initDBCallbacks(DB)

	return DB, nil
}
