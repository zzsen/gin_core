package core

import (
	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

var messageQueueConsumerList []config.MessageQueue = make([]config.MessageQueue, 0)

func AddMessageQueueConsumer(messageQueue config.MessageQueue) {
	messageQueueConsumerList = append(messageQueueConsumerList, messageQueue)
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
	logger.Logger = logger.InitLogger(app.BaseConfig.Log)
	app.Logger = logger.Logger
	// 初始化redis
	if app.BaseConfig.System.UseRedis {
		if app.BaseConfig.Redis == nil && len(app.BaseConfig.RedisList) == 0 {
			panic("[redis] not valid redis config, please check config")
		} else {
			initialize.InitRedis()
			initialize.InitRedisList()
		}
	}
	// 初始化数据库
	if app.BaseConfig.System.UseMysql {
		if app.BaseConfig.Db == nil && len(app.BaseConfig.DbList) == 0 && len(app.BaseConfig.DbResolvers) == 0 {
			panic("[db] not valid db config, please check config")
		} else {
			initialize.InitDB()
			initialize.InitDBList()
			initialize.InitDBResolver()
		}
	}
	// 初始化es
	if app.BaseConfig.System.UseEs {
		if app.BaseConfig.Es == nil {
			panic("[es] not valid es config, please check config")
		} else {
			initialize.InitElasticsearch()
		}
	}
	// 初始化消息队列
	if app.BaseConfig.System.UseRabbitMQ && len(messageQueueConsumerList) > 0 {
		go initialize.InitialRabbitMq(messageQueueConsumerList...)
	}
	if app.BaseConfig.System.UseEtcd {
		if app.BaseConfig.Etcd == nil {
			panic("[etcd] not valid etcd config, please check config")
		} else {
			initialize.InitEtcd()
		}
	}
	// 初始化定时任务
	if app.BaseConfig.System.UseSchedule && len(scheduleList) > 0 {
		c := cron.New()
		for _, schedule := range scheduleList {
			c.AddFunc(schedule.Cron, schedule.Cmd)
		}
		c.Start()
	}
}
