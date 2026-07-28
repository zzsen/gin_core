package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/discovery"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// EtcdService Etcd 客户端服务（初始化 / 关闭 / 健康检查，非配置中心）
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

// Init 初始化 Etcd。
//
// 流程：
// 1. 校验配置块存在
// 2. 调用 initialize.InitEtcd
// 3. 配置错误始终返回；连接错误按 required 阻断或降级
func (s *EtcdService) Init(ctx context.Context) error {
	if app.BaseConfig.Etcd == nil {
		return fmt.Errorf("未找到有效的Etcd配置")
	}

	// 挂载 discovery 生命周期钩子（幂等；enabled=false 时 Ready 空操作）
	discovery.InstallHooks()

	err := initialize.InitEtcd()
	if err == nil {
		return nil
	}
	if errors.Is(err, initialize.ErrEtcdConfig) {
		return err
	}
	if app.BaseConfig.Etcd.IsRequired() {
		return err
	}
	logger.Warn("[Etcd] 初始化失败，required=false，降级继续: %v", err)
	app.Etcd = nil
	return nil
}

// Close 关闭 Etcd 连接
func (s *EtcdService) Close(ctx context.Context) error {
	if app.Etcd != nil {
		if err := app.Etcd.Close(); err != nil {
			logger.Error("[Etcd] 关闭连接失败: %v", err)
			return err
		}
		logger.Info("[Etcd] 连接已关闭")
		app.Etcd = nil
	}
	return nil
}

// HealthCheck 按 health.strategy（any|all）探活；非法 strategy warn 后回退 any
func (s *EtcdService) HealthCheck(ctx context.Context) error {
	if app.Etcd == nil {
		return fmt.Errorf("etcd未初始化")
	}
	cfg := app.BaseConfig.Etcd
	if cfg == nil || len(cfg.Endpoints) == 0 {
		return fmt.Errorf("etcd.endpoints 为空")
	}
	strategy := cfg.HealthStrategy()
	if strategy != "any" && strategy != "all" {
		logger.Warn("[Etcd] 不支持的 etcd.health.strategy=%q，回退为默认 any", strategy)
		strategy = "any"
	}

	var firstErr error
	checked := 0
	for _, ep := range cfg.Endpoints {
		ep = strings.TrimSpace(ep)
		if ep == "" {
			continue
		}
		checked++
		_, err := app.Etcd.Status(ctx, ep)
		if err != nil {
			if strategy == "all" {
				return fmt.Errorf("etcd健康检查失败(strategy=all): %s: %w", ep, err)
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if strategy == "any" {
			return nil
		}
	}
	if checked == 0 {
		return fmt.Errorf("etcd健康检查失败: 无有效 endpoints")
	}
	if strategy == "all" {
		return nil
	}
	if firstErr != nil {
		return fmt.Errorf("etcd健康检查失败(strategy=any): %w", firstErr)
	}
	return fmt.Errorf("etcd健康检查失败(strategy=any)")
}
