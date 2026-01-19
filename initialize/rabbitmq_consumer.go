// Package initialize 提供各种服务的初始化功能
// 本文件专门负责RabbitMQ消息队列消费者的初始化，支持多队列配置和自动重连机制
package initialize

import (
	"context"
	"sync"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// queueToRestart 待重启的消息队列通道
// 当消息队列消费者出错时，将队列名称加入到该通道
// 用于实现消息队列的自动重连和故障恢复机制
var queueToRestart chan (string) = make(chan string)

// consumerCancelFuncs 存储所有消费者的取消函数，用于优雅关闭
var consumerCancelFuncs = make(map[string]context.CancelFunc)
var consumerCancelLock sync.Mutex

// InitialRabbitMq 初始化RabbitMQ消息队列消费者（向后兼容版本）
// 该函数会：
// 1. 创建消息队列配置映射表
// 2. 为每个消息队列启动独立的消费者协程
// 3. 启动故障恢复监听器，自动重启出错的消费者
// 参数：
//   - messageQueueList: 消息队列配置列表，支持可变参数
func InitialRabbitMq(messageQueueList ...config.MessageQueue) {
	InitialRabbitMqWithContext(context.Background(), messageQueueList...)
}

// InitialRabbitMqWithContext 初始化RabbitMQ消息队列消费者（支持 context 优雅关闭）
// 该函数会：
// 1. 创建消息队列配置映射表
// 2. 为每个消息队列启动独立的消费者协程
// 3. 启动故障恢复监听器，自动重启出错的消费者
// 4. 当 context 取消时，所有消费者会优雅关闭
// 参数：
//   - ctx: 用于控制消费者生命周期的 context
//   - messageQueueList: 消息队列配置列表，支持可变参数
func InitialRabbitMqWithContext(ctx context.Context, messageQueueList ...config.MessageQueue) {
	// 创建消息队列配置映射表，用于快速查找和故障恢复
	messageQueueMap := map[string]*config.MessageQueue{}

	// 遍历所有消息队列配置并启动消费者
	for i := range messageQueueList {
		mq := &messageQueueList[i]
		// 将消息队列配置存储到映射表中，键为队列信息
		messageQueueMap[mq.GetInfo()] = mq
		// 为每个消息队列启动独立的消费者协程
		go startMqConsumeWithContext(ctx, mq)
	}

	// 启动故障恢复监听器协程
	go func() {
		for {
			select {
			case <-ctx.Done():
				// context 取消，停止故障恢复
				logger.Info("[消息队列] 收到关闭信号，停止故障恢复监听器")
				return
			case queueName := <-queueToRestart:
				// 从映射表中查找对应的消息队列配置
				messageQueue, ok := messageQueueMap[queueName]
				if ok {
					// 等待5秒后重试，避免频繁重连
					time.Sleep(5 * time.Second)

					// 检查 context 是否已取消
					select {
					case <-ctx.Done():
						return
					default:
						logger.Info("[消息队列] 正在尝试重连, queueInfo: %s", messageQueue.GetInfo())
						// 重新启动消费者协程
						go startMqConsumeWithContext(ctx, messageQueue)
					}
				}
			}
		}
	}()
}

// startMqConsumeWithContext 启动单个消息队列消费者（支持 context）
// 该函数会：
// 1. 获取消息队列连接字符串
// 2. 根据配置选择对应的RabbitMQ实例
// 3. 启动消费者并处理消息
// 4. 如果出错，将队列名称发送到重启通道
// 参数：
//   - ctx: 用于控制消费者生命周期的 context
//   - messageQueue: 消息队列配置信息
func startMqConsumeWithContext(ctx context.Context, messageQueue *config.MessageQueue) {
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

	// 创建子 context 用于单个消费者
	consumerCtx, cancel := context.WithCancel(ctx)
	queueInfo := messageQueue.GetInfo()

	// 存储取消函数
	consumerCancelLock.Lock()
	consumerCancelFuncs[queueInfo] = cancel
	consumerCancelLock.Unlock()

	// 确保退出时清理
	defer func() {
		consumerCancelLock.Lock()
		delete(consumerCancelFuncs, queueInfo)
		consumerCancelLock.Unlock()
	}()

	logger.Info("[消息队列] 消费者启动, queueInfo: %s", queueInfo)

	// 启动消费者并开始处理消息
	err := messageQueue.ConsumeWithContext(consumerCtx)
	if err != nil {
		// 检查是否是因为 context 取消
		select {
		case <-ctx.Done():
			logger.Info("[消息队列] 消费者已优雅关闭, queueInfo: %s", queueInfo)
			return
		default:
			// 如果消费者出错，将队列名称发送到重启通道
			queueToRestart <- queueInfo
			// 记录错误日志
			logger.Error("[消息队列] %v", err.Error())
		}
	}
}

// StopConsumer 停止指定的消费者
// 参数：
//   - queueInfo: 队列信息（由 MessageQueue.GetInfo() 返回）
func StopConsumer(queueInfo string) {
	consumerCancelLock.Lock()
	defer consumerCancelLock.Unlock()

	if cancel, ok := consumerCancelFuncs[queueInfo]; ok {
		cancel()
		logger.Info("[消息队列] 消费者已停止, queueInfo: %s", queueInfo)
	}
}

// StopAllConsumers 停止所有消费者
func StopAllConsumers() {
	consumerCancelLock.Lock()
	defer consumerCancelLock.Unlock()

	for queueInfo, cancel := range consumerCancelFuncs {
		cancel()
		logger.Info("[消息队列] 消费者已停止, queueInfo: %s", queueInfo)
	}
}
