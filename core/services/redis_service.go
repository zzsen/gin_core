package services

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// RedisService Redis服务
type RedisService struct{}

// Name 返回服务名称
func (s *RedisService) Name() string { return "redis" }

// Priority 返回初始化优先级
func (s *RedisService) Priority() int { return 10 }

// Dependencies 返回依赖
func (s *RedisService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 根据配置判断是否需要初始化
func (s *RedisService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.System.UseRedis
}

// Init 初始化Redis
func (s *RedisService) Init(ctx context.Context) error {
	// 验证配置
	if app.BaseConfig.Redis == nil && len(app.BaseConfig.RedisList) == 0 {
		return fmt.Errorf("未找到有效的Redis配置")
	}

	// 初始化主Redis连接
	initialize.InitRedis()
	// 初始化多Redis实例连接列表
	initialize.InitRedisList()
	return nil
}

// Close 关闭Redis连接
func (s *RedisService) Close(ctx context.Context) error {
	if app.Redis != nil {
		if err := app.Redis.Close(); err != nil {
			logger.Error("[Redis] 关闭连接失败: %v", err)
			return err
		}
		logger.Info("[Redis] 连接已关闭")
	}
	return nil
}

// HealthCheck 健康检查
func (s *RedisService) HealthCheck(ctx context.Context) error {
	if app.Redis == nil {
		return fmt.Errorf("redis未初始化")
	}
	return app.Redis.Ping(ctx).Err()
}
