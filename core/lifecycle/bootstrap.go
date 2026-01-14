package lifecycle

import (
	"context"

	"github.com/zzsen/gin_core/app"
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

// GetMessageQueueConsumerList 获取消息队列消费者列表
func GetMessageQueueConsumerList() []config.MessageQueue {
	return messageQueueConsumerList
}

// GetMessageQueueProducerList 获取消息队列发送者列表
func GetMessageQueueProducerList() []config.MessageQueue {
	return messageQueueProducerList
}

// GetScheduleList 获取定时任务列表
func GetScheduleList() []config.ScheduleInfo {
	return scheduleList
}

// InitServices 初始化所有服务组件
// 这是服务初始化的核心函数，使用并行初始化机制
//
// 初始化流程：
// 1. 使用依赖解析器分析服务依赖关系
// 2. 按层级并行初始化服务
//
// 特性：
// - 自动解析服务依赖关系
// - 同一层级的服务并行初始化
// - 支持自定义服务扩展
// - 提供初始化前后的钩子机制
//
// 错误处理：
// - 关键服务初始化失败会返回错误
// - 详细的错误信息帮助快速定位问题
func InitServices(ctx context.Context) error {
	return InitAllServices(ctx, &app.BaseConfig)
}

// CloseServices 关闭所有服务
// 按依赖关系逆序关闭服务，确保依赖的服务最后关闭
func CloseServices(ctx context.Context) error {
	return CloseAllServices(ctx, &app.BaseConfig)
}
