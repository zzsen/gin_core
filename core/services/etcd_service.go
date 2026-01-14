package services

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// EtcdService Etcd服务
type EtcdService struct{}

// Name 返回服务名称
func (s *EtcdService) Name() string { return "etcd" }

// Priority 返回初始化优先级
func (s *EtcdService) Priority() int { return 20 }

// Dependencies 返回依赖
func (s *EtcdService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 根据配置判断是否需要初始化
func (s *EtcdService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.System.UseEtcd
}

// Init 初始化Etcd
func (s *EtcdService) Init(ctx context.Context) error {
	// 验证配置
	if app.BaseConfig.Etcd == nil {
		return fmt.Errorf("未找到有效的Etcd配置")
	}

	// 初始化Etcd客户端
	initialize.InitEtcd()
	return nil
}

// Close 关闭Etcd连接
func (s *EtcdService) Close(ctx context.Context) error {
	if app.Etcd != nil {
		if err := app.Etcd.Close(); err != nil {
			logger.Error("[Etcd] 关闭连接失败: %v", err)
			return err
		}
		logger.Info("[Etcd] 连接已关闭")
	}
	return nil
}

// HealthCheck 健康检查
func (s *EtcdService) HealthCheck(ctx context.Context) error {
	if app.Etcd == nil {
		return fmt.Errorf("etcd未初始化")
	}
	_, err := app.Etcd.Status(ctx, app.BaseConfig.Etcd.Addresses[0])
	return err
}
