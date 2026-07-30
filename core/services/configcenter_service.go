package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/configcenter"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// ConfigCenterService Etcd 配置中心服务（overlay + Watch）
type ConfigCenterService struct {
	mu   sync.Mutex
	stop func()
}

// Name 返回服务名称
func (s *ConfigCenterService) Name() string { return "configcenter" }

// Priority 返回初始化优先级（在 etcd 之后、存储类服务之前）
func (s *ConfigCenterService) Priority() int { return 25 }

// Dependencies 依赖 etcd 客户端
func (s *ConfigCenterService) Dependencies() []string { return []string{"etcd"} }

// ShouldInit 仅在 useEtcd 且 configCenter.enabled 时初始化
func (s *ConfigCenterService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg != nil && cfg.System.UseEtcd && cfg.Etcd != nil && cfg.Etcd.ConfigCenterEnabled()
}

// Init 启动配置中心：加载 overlay，并按需启动 Watch。
//
// 【流程】
// 1. 校验 app.Etcd；nil 时按 configCenter.required 决定失败或 warn 跳过
// 2. LoadOverlay 合并 Etcd YAML 到 app.BaseConfig
// 3. ConfigCenterWatch 为 true 时 StartWatch，保存 stop 供 Close
func (s *ConfigCenterService) Init(ctx context.Context) error {
	// 步骤1：Etcd 客户端
	if app.Etcd == nil {
		cc := app.BaseConfig.Etcd
		if cc != nil && cc.ConfigCenterRequired() {
			return fmt.Errorf("%w: app.Etcd is nil", configcenter.ErrOverlayRequired)
		}
		logger.Warn("[configcenter] app.Etcd 为空，跳过 overlay")
		return nil
	}
	// 步骤2：启动 overlay
	cc := app.BaseConfig.Etcd.ConfigCenter
	if err := configcenter.LoadOverlay(ctx, app.Etcd, cc, app.CipherKey); err != nil {
		return err
	}
	// 步骤3：可选 Watch
	if app.BaseConfig.Etcd.ConfigCenterWatch() {
		stop := configcenter.StartWatch(ctx, app.Etcd, app.BaseConfig.Etcd, app.CipherKey)
		s.mu.Lock()
		s.stop = stop
		s.mu.Unlock()
	}
	return nil
}

// Close 停止配置中心 Watch（幂等）；无 Watch 时为空操作。
func (s *ConfigCenterService) Close(ctx context.Context) error {
	s.mu.Lock()
	stop := s.stop
	s.stop = nil
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
	return nil
}
