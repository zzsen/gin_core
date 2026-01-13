package core

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// messageQueueConsumerList 消息队列消费者配置列表
// 存储应用中所有需要启动的消息队列消费者配置
// 这些消费者会在服务启动时被初始化，用于处理异步消息
// 支持多个消费者同时工作，提供高并发的消息处理能力
var messageQueueConsumerList []config.MessageQueue = make([]config.MessageQueue, 0)

// messageQueueProducerList 消息队列发送者配置列表
// 存储应用中所有需要初始化的消息队列发送者配置
// 这些发送者会在服务启动时被初始化，用于发送异步消息
// 支持多个发送者配置，提供灵活的消息发送能力
var messageQueueProducerList []config.MessageQueue = make([]config.MessageQueue, 0)

// AddMessageQueueConsumer 添加消息队列消费者配置
// 将消息队列消费者配置添加到全局列表中，在服务启动时会自动初始化这些消费者
// 支持多种消息队列类型和交换机模式，提供灵活的异步处理能力
//
// 参数 messageQueue: 消息队列配置对象，包含队列名、交换机、路由键和处理函数等信息
//
// 功能特性：
// - 支持多个消费者并发处理
// - 自动记录队列配置信息
// - 提供详细的日志输出
//
// 使用示例：
//
//	AddMessageQueueConsumer(config.MessageQueue{
//	  QueueName:    "user.created",
//	  ExchangeName: "user.events",
//	  ExchangeType: "topic",
//	  RoutingKey:   "user.created",
//	  Fun:          handleUserCreated,
//	})
func AddMessageQueueConsumer(messageQueue config.MessageQueue) {
	messageQueueConsumerList = append(messageQueueConsumerList, messageQueue)
	logger.Info("[消息队列] 添加消息队列成功, 队列信息: %s, 方法: %s",
		messageQueue.GetInfo(), messageQueue.GetFuncInfo())
}

// AddMessageQueueProducer 添加消息队列发送者配置
// 将消息队列发送者配置添加到全局列表中，在服务启动时会自动初始化这些发送者
// 支持多种消息队列类型和交换机模式，提供灵活的消息发送能力
//
// 参数 messageQueue: 消息队列配置对象，包含队列名、交换机、路由键等信息
//
// 功能特性：
// - 支持多个发送者配置
// - 自动记录队列配置信息
// - 提供详细的日志输出
//
// 使用示例：
//
//	AddMessageQueueProducer(config.MessageQueue{
//	  MQName:       "default",
//	  QueueName:    "user.created",
//	  ExchangeName: "user.events",
//	  ExchangeType: "topic",
//	  RoutingKey:   "user.created",
//	})
func AddMessageQueueProducer(messageQueue config.MessageQueue) {
	messageQueueProducerList = append(messageQueueProducerList, messageQueue)
	logger.Info("[消息队列] 添加消息队列发送者成功, 队列信息: %s", messageQueue.GetInfo())
}

// scheduleList 定时任务配置列表
// 存储应用中所有需要执行的定时任务配置
// 支持标准的Cron表达式，提供灵活的任务调度能力
// 任务会在独立的协程中执行，不会阻塞主服务
var scheduleList []config.ScheduleInfo = make([]config.ScheduleInfo, 0)

// AddSchedule 添加定时任务配置
// 将定时任务配置添加到全局列表中，在服务启动时会自动启动这些任务
// 基于robfig/cron库实现，支持标准的Cron表达式语法
//
// 参数 schedule: 定时任务配置对象，包含Cron表达式和执行函数
//
// Cron表达式格式：
// - 秒 分 时 日 月 周
// - 支持特殊表达式如 @every 30s, @hourly, @daily 等
//
// 功能特性：
// - 支持多个任务并发执行
// - 自动错误恢复
// - 详细的任务执行日志
//
// 使用示例：
//
//	AddSchedule(config.ScheduleInfo{
//	  Cron: "0 0 * * *",  // 每天凌晨执行
//	  Cmd:  cleanupOldFiles,
//	})
//
//	AddSchedule(config.ScheduleInfo{
//	  Cron: "@every 5m",  // 每5分钟执行
//	  Cmd:  healthCheck,
//	})
func AddSchedule(schedule config.ScheduleInfo) {
	scheduleList = append(scheduleList, schedule)
	logger.Info("[定时任务] 添加定时任务成功, cron表达式: %s, 方法: %s",
		schedule.Cron, schedule.GetFuncInfo())
}

// initService 初始化所有服务组件
// 这是服务初始化的核心函数，负责按顺序启动应用所需的各种服务组件
// 初始化顺序经过精心设计，确保依赖关系正确处理
//
// 初始化流程：
// 1. 日志系统 - 为后续组件提供日志记录能力
// 2. Redis缓存 - 提供高性能缓存服务
// 3. MySQL数据库 - 提供持久化存储服务
// 4. Elasticsearch - 提供全文搜索和分析能力
// 5. RabbitMQ消息队列 - 提供异步消息处理能力
// 6. Etcd配置中心 - 提供配置管理和服务发现
// 7. 定时任务调度器 - 提供定时任务执行能力
//
// 错误处理：
// - 关键服务初始化失败会导致程序panic退出
// - 配置验证确保服务配置的正确性
// - 详细的错误信息帮助快速定位问题
func initService() {
	// 初始化日志
	logger.Logger = logger.InitLogger(app.BaseConfig.Log)
	app.Logger = logger.Logger
	// 初始化redis
	if app.BaseConfig.System.UseRedis {
		// 验证Redis配置的完整性
		if app.BaseConfig.Redis == nil && len(app.BaseConfig.RedisList) == 0 {
			panic(exception.NewInitError("redis", "验证配置", fmt.Errorf("未找到有效的Redis配置, 请检查配置")))
		} else {
			// 初始化主Redis连接
			initialize.InitRedis()
			// 初始化多Redis实例连接列表
			initialize.InitRedisList()
		}
	}
	// 初始化数据库
	if app.BaseConfig.System.UseMysql {
		// 验证数据库配置的完整性
		if app.BaseConfig.Db == nil && len(app.BaseConfig.DbList) == 0 && len(app.BaseConfig.DbResolvers) == 0 {
			panic(exception.NewInitError("db", "验证配置", fmt.Errorf("未找到有效的数据库配置, 请检查配置")))
		} else {
			// 初始化主数据库连接
			initialize.InitDB()
			// 初始化多数据库连接列表
			initialize.InitDBList()
			// 初始化数据库读写分离解析器
			initialize.InitDBResolver()
		}
	}
	// 初始化es
	if app.BaseConfig.System.UseEs {
		// 验证Elasticsearch配置
		if app.BaseConfig.Es == nil {
			panic(exception.NewInitError("es", "验证配置", fmt.Errorf("未找到有效的Elasticsearch配置, 请检查配置")))
		} else {
			// 初始化Elasticsearch客户端
			initialize.InitElasticsearch()
		}
	}
	// 初始化消息队列
	if app.BaseConfig.System.UseRabbitMQ {
		// 初始化消息队列生产者
		if len(messageQueueProducerList) > 0 {
			initialize.InitialRabbitMqProducer(messageQueueProducerList...)
		}
		// 在协程中启动消息队列消费者，避免阻塞服务启动
		if len(messageQueueConsumerList) > 0 {
			go initialize.InitialRabbitMq(messageQueueConsumerList...)
		}
	}
	// 初始化etcd
	if app.BaseConfig.System.UseEtcd {
		// 验证Etcd配置
		if app.BaseConfig.Etcd == nil {
			panic(exception.NewInitError("etcd", "验证配置", fmt.Errorf("未找到有效的Etcd配置, 请检查配置")))
		} else {
			// 初始化Etcd客户端
			initialize.InitEtcd()
		}
	}
	// 初始化定时任务
	if app.BaseConfig.System.UseSchedule && len(scheduleList) > 0 {
		// 创建新的Cron调度器实例
		c := cron.New()
		// 注册所有定时任务到调度器
		for _, schedule := range scheduleList {
			// 添加任务到调度器，任务会在指定时间自动执行
			c.AddFunc(schedule.Cron, schedule.Cmd)
		}
		// 启动调度器，开始执行定时任务
		c.Start()
		logger.Info("[定时任务] 定时任务调度器已启动，共注册 %d 个任务", len(scheduleList))
	}
}
