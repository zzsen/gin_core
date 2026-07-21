package services

import (
	"context"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// LoggerService 日志服务
type LoggerService struct{}

// Name 返回服务名称
func (s *LoggerService) Name() string { return "logger" }

// Priority 返回初始化优先级（日志最先初始化）
func (s *LoggerService) Priority() int { return 0 }

// Dependencies 返回依赖（日志无依赖）
func (s *LoggerService) Dependencies() []string { return nil }

// ShouldInit 日志总是需要初始化
func (s *LoggerService) ShouldInit(cfg *config.BaseConfig) bool {
	return true
}

// Init 初始化日志系统。
//
// 【流程】
// 1. 按 app.BaseConfig.Log 调用 InitLogger（含文件轮转与可选远程 Sink）
// 2. 同步到包级 logger.Logger 与 app.Logger
func (s *LoggerService) Init(ctx context.Context) error {
	logger.Logger = logger.InitLogger(app.BaseConfig.Log)
	app.Logger = logger.Logger
	return nil
}

// Close 关闭日志服务。
//
// 【功能】进程退出时刷出远程缓冲并关闭 Sink 管线，减少 at-most-once 场景下的丢失。
// 【流程】委托 logger.CloseRemote（无远程管线时为空操作）
func (s *LoggerService) Close(ctx context.Context) error {
	return logger.CloseRemote(ctx)
}
