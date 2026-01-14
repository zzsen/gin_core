package services

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/model/config"
)

// ElasticsearchService Elasticsearch服务
type ElasticsearchService struct{}

// Name 返回服务名称
func (s *ElasticsearchService) Name() string { return "elasticsearch" }

// Priority 返回初始化优先级
func (s *ElasticsearchService) Priority() int { return 20 }

// Dependencies 返回依赖
func (s *ElasticsearchService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 根据配置判断是否需要初始化
func (s *ElasticsearchService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.System.UseEs
}

// Init 初始化Elasticsearch
func (s *ElasticsearchService) Init(ctx context.Context) error {
	// 验证配置
	if app.BaseConfig.Es == nil {
		return fmt.Errorf("未找到有效的Elasticsearch配置")
	}

	// 初始化Elasticsearch客户端
	initialize.InitElasticsearch()
	return nil
}

// Close 关闭Elasticsearch连接
func (s *ElasticsearchService) Close(ctx context.Context) error {
	// ES客户端通常不需要显式关闭
	return nil
}

// HealthCheck 健康检查
func (s *ElasticsearchService) HealthCheck(ctx context.Context) error {
	if app.ES == nil {
		return fmt.Errorf("elasticsearch未初始化")
	}
	_, err := app.ES.Info().Do(ctx)
	return err
}
