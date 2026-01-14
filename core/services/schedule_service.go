package services

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// ScheduleService 定时任务服务
type ScheduleService struct {
	scheduleList []config.ScheduleInfo
	cron         *cron.Cron
}

// NewScheduleService 创建定时任务服务
func NewScheduleService(scheduleList []config.ScheduleInfo) *ScheduleService {
	return &ScheduleService{
		scheduleList: scheduleList,
	}
}

// Name 返回服务名称
func (s *ScheduleService) Name() string { return "schedule" }

// Priority 返回初始化优先级（最后初始化）
func (s *ScheduleService) Priority() int { return 100 }

// Dependencies 返回依赖
func (s *ScheduleService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 根据配置判断是否需要初始化
func (s *ScheduleService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.System.UseSchedule && len(s.scheduleList) > 0
}

// Init 初始化定时任务
func (s *ScheduleService) Init(ctx context.Context) error {
	// 创建新的Cron调度器实例
	s.cron = cron.New()

	// 注册所有定时任务到调度器
	for _, schedule := range s.scheduleList {
		_, err := s.cron.AddFunc(schedule.Cron, schedule.Cmd)
		if err != nil {
			logger.Error("[定时任务] 添加任务失败, cron: %s, error: %v", schedule.Cron, err)
			return err
		}
	}

	// 启动调度器
	s.cron.Start()
	logger.Info("[定时任务] 调度器已启动，共注册 %d 个任务", len(s.scheduleList))
	return nil
}

// Close 关闭定时任务
func (s *ScheduleService) Close(ctx context.Context) error {
	if s.cron != nil {
		s.cron.Stop()
		logger.Info("[定时任务] 调度器已停止")
	}
	return nil
}

// SetScheduleList 设置定时任务列表
func (s *ScheduleService) SetScheduleList(list []config.ScheduleInfo) {
	s.scheduleList = list
}
