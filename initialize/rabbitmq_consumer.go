// Package initialize 提供各种服务的初始化功能
// 本文件专门负责RabbitMQ消息队列消费者的初始化，支持多队列配置和自动重连机制
package initialize

import (
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// queueToRestart 待重启的消息队列通道
// 当消息队列消费者出错时，将队列名称加入到该通道
// 用于实现消息队列的自动重连和故障恢复机制
var queueToRestart chan (string) = make(chan string)

// InitialRabbitMq 初始化RabbitMQ消息队列消费者
// 该函数会：
// 1. 创建消息队列配置映射表
// 2. 为每个消息队列启动独立的消费者协程
// 3. 启动故障恢复监听器，自动重启出错的消费者
// 参数：
//   - messageQueueList: 消息队列配置列表，支持可变参数
func InitialRabbitMq(messageQueueList ...config.MessageQueue) {
	// 创建消息队列配置映射表，用于快速查找和故障恢复
	messageQueueMap := map[string]*config.MessageQueue{}

	// 遍历所有消息队列配置并启动消费者
	for _, messageQueue := range messageQueueList {
		// 将消息队列配置存储到映射表中，键为队列信息
		messageQueueMap[messageQueue.GetInfo()] = &messageQueue
		// 为每个消息队列启动独立的消费者协程
		go startMqConsume(&messageQueue)
	}

	// 启动故障恢复监听器，当服务异常时自动重启消费者任务
	for queueName := range queueToRestart {
		// 从映射表中查找对应的消息队列配置
		messageQueue, ok := messageQueueMap[queueName]
		if ok {
			// 等待5秒后重试，避免频繁重连
			time.Sleep(5 * time.Second)
			logger.Info("[消息队列] 正在尝试重连, queueInfo: %s", messageQueue.GetInfo())
			// 重新启动消费者协程
			go startMqConsume(messageQueue)
		}
	}
}

// startMqConsume 启动单个消息队列消费者
// 该函数会：
// 1. 获取消息队列连接字符串
// 2. 根据配置选择对应的RabbitMQ实例
// 3. 启动消费者并处理消息
// 4. 如果出错，将队列名称发送到重启通道
// 参数：
//   - messageQueue: 消息队列配置信息
func startMqConsume(messageQueue *config.MessageQueue) {
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

	// 启动消费者并开始处理消息
	err := messageQueue.Consume()
	if err != nil {
		// 如果消费者出错，将队列名称发送到重启通道
		queueToRestart <- messageQueue.GetInfo()
		// 记录错误日志
		logger.Error("[消息队列]%v", err.Error())
	}
}
