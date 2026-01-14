package services

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/model/config"
)

// MySQLService MySQL数据库服务
type MySQLService struct{}

// Name 返回服务名称
func (s *MySQLService) Name() string { return "mysql" }

// Priority 返回初始化优先级
func (s *MySQLService) Priority() int { return 10 }

// Dependencies 返回依赖
func (s *MySQLService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 根据配置判断是否需要初始化
func (s *MySQLService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.System.UseMysql
}

// Init 初始化MySQL
func (s *MySQLService) Init(ctx context.Context) error {
	// 验证配置
	if app.BaseConfig.Db == nil && len(app.BaseConfig.DbList) == 0 && len(app.BaseConfig.DbResolvers) == 0 {
		return fmt.Errorf("未找到有效的数据库配置")
	}

	// 初始化主数据库连接
	initialize.InitDB()
	// 初始化多数据库连接列表
	initialize.InitDBList()
	// 初始化数据库读写分离解析器
	initialize.InitDBResolver()
	return nil
}

// Close 关闭数据库连接
func (s *MySQLService) Close(ctx context.Context) error {
	// GORM 通常自动管理连接，无需手动关闭
	return nil
}

// HealthCheck 健康检查
func (s *MySQLService) HealthCheck(ctx context.Context) error {
	if app.DB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	db, err := app.DB.DB()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}
