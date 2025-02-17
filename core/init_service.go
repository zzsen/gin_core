package core

import (
	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

var messageQueueList []config.MessageQueue = make([]config.MessageQueue, 0)

func AddMessageQueue(messageQueue config.MessageQueue) {
	messageQueueList = append(messageQueueList, messageQueue)
	logger.Info("[消息队列] 添加消息队列成功, 队列信息: %s, 方法: %s",
		messageQueue.GetInfo(), messageQueue.GetFuncInfo())
}

var scheduleList []config.ScheduleInfo = make([]config.ScheduleInfo, 0)

func AddSchedule(schedule config.ScheduleInfo) {
	scheduleList = append(scheduleList, schedule)
	logger.Info("[定时任务] 添加定时任务成功, cron表达式: %s, 方法: %s",
		schedule.Cron, schedule.GetFuncInfo())
}

func initService() {
	// 初始化日志
	global.Logger = logger.InitLogger(global.BaseConfig.Log)
	// 初始化redis
	if global.BaseConfig.System.UseRedis {
		initialize.InitRedis()
		initialize.InitRedisList()
	}
	// 初始化数据库
	if global.BaseConfig.System.UseMysql {
		initialize.InitDB()
		initialize.InitDBList()
		initialize.InitDBResolver()
	}
	// 初始化es
	if global.BaseConfig.System.UseEs {
		initialize.InitElasticsearch()
	}
	// 初始化消息队列
	if global.BaseConfig.System.UseRabbitMQ && len(messageQueueList) > 0 {
		go initialize.InitialRabbitMq(messageQueueList...)
	}
	// 初始化定时任务
	if len(scheduleList) > 0 {
		c := cron.New()
		for _, schedule := range scheduleList {
			c.AddFunc(schedule.Cron, schedule.Cmd)
		}
		c.Start()
	}
}
