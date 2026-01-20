// Package initialize 提供各种服务的初始化功能
// 本文件专门负责RabbitMQ消息队列发送者的初始化，支持多队列配置和连接管理
package initialize

import (
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// InitialRabbitMqProducer 初始化RabbitMQ消息队列发送者
// 该函数会：
// 1. 创建消息队列配置映射表
// 2. 为每个消息队列初始化连接和通道
// 3. 将初始化好的发送者存储到全局映射表中，供后续使用
// 参数：
//   - messageQueueList: 消息队列配置列表，支持可变参数
func InitialRabbitMqProducer(messageQueueList ...*config.MessageQueue) {
	// 遍历所有消息队列配置并初始化发送者
	for _, mq := range messageQueueList {
		// 初始化单个消息队列发送者
		initMqProducer(mq)
	}
}

// initMqProducer 初始化单个消息队列发送者
// 该函数会：
// 1. 获取消息队列连接字符串
// 2. 根据配置选择对应的RabbitMQ实例
// 3. 初始化连接和通道
// 4. 将初始化好的发送者存储到全局映射表中
// 参数：
//   - messageQueue: 消息队列配置信息
func initMqProducer(messageQueue *config.MessageQueue) {
	// 获取消息队列连接字符串，默认使用基础配置
	mqConnStr := app.BaseConfig.RabbitMQ.Url()

	// 如果配置了特定的消息队列名称，则使用对应的消息队列配置
	if messageQueue.MQName != "" {
		mqConnStr = app.BaseConfig.RabbitMQList.Url(messageQueue.MQName)
	}

	// 检查是否找到有效的连接字符串
	if mqConnStr == "" {
		logger.Error("[消息队列] 未找到对应的消息队列配置, MQName: %s", messageQueue.MQName)
		return
	}

	// 设置消息队列连接字符串
	messageQueue.MqConnStr = mqConnStr

	// 初始化连接和通道（不立即使用，只是预初始化）
	err := messageQueue.InitChannelForProducer()
	if err != nil {
		logger.Error("[消息队列] 初始化发送者失败, queueInfo: %s, error: %v", messageQueue.GetInfo(), err)
		return
	}

	// 将初始化好的发送者存储到全局映射表中
	app.RabbitMQProducerList[messageQueue.GetInfo()] = messageQueue
	logger.Info("[消息队列] 发送者初始化成功, queueInfo: %s", messageQueue.GetInfo())
}
