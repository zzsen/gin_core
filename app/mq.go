package app

import (
	"context"
	"fmt"
	"time"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// SendRabbitMqMsg 发送RabbitMQ消息
// 该函数支持向多个消息队列实例发送消息，并提供重试机制
// 参数：
//   - queueName: 队列名称
//   - exchangeName: 交换机名称
//   - exchangeType: 交换机类型（direct, fanout, topic, headers）
//   - routingKey: 路由键
//   - message: 消息内容
//   - mqConfigNames: 消息队列配置名称列表（可选，为空时使用默认配置）
//
// 返回：
//   - error: 如果所有消息队列都发送失败则返回错误，部分成功时返回最后一个错误
func SendRabbitMqMsg(queueName string, exchangeName string,
	exchangeType string, routingKey string, message string, mqConfigNames ...string) error {
	if len(mqConfigNames) == 0 {
		mqConfigNames = []string{""}
	}

	var lastErr error
	successCount := 0

	for _, mqConfigName := range mqConfigNames {
		messageQueue := config.MessageQueue{
			MQName:       mqConfigName,
			QueueName:    queueName,
			ExchangeName: exchangeName,
			ExchangeType: exchangeType,
			RoutingKey:   routingKey,
		}
		// 获取消息队列连接字符串
		mqConnStr := BaseConfig.RabbitMQ.Url()
		// 如果配置了消息队列名称, 则使用对应的消息队列
		if messageQueue.MQName != "" {
			mqConnStr = BaseConfig.RabbitMQList.Url(messageQueue.MQName)
		}
		if mqConnStr == "" {
			err := fmt.Errorf("[消息队列] 未找到对应的消息队列配置, MQName: %s", messageQueue.MQName)
			logger.Error("%v", err)
			lastErr = err
			continue
		}
		messageQueue.MqConnStr = mqConnStr

		// 发送消息，带重试机制
		err := sendRabbitMqMsgWithRetry(&messageQueue, message, 3, 100*time.Millisecond)
		if err != nil {
			lastErr = err
			logger.Error("[消息队列] 消息发送失败, queueInfo: %s, error: %v", messageQueue.GetInfo(), err)
		} else {
			successCount++
		}
	}

	// 如果所有消息队列都发送失败，返回错误
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("[消息队列] 所有消息队列发送失败: %w", lastErr)
	}

	// 部分成功时，记录警告但不返回错误（允许部分失败）
	if successCount > 0 && successCount < len(mqConfigNames) {
		logger.Warn("[消息队列] 部分消息队列发送失败, 成功: %d/%d", successCount, len(mqConfigNames))
	}

	return nil
}

// sendRabbitMqMsgWithRetry 发送RabbitMQ消息，带重试机制
// 参数：
//   - messageQueue: 消息队列配置（指针）
//   - message: 消息内容
//   - maxRetries: 最大重试次数（不包括首次尝试）
//   - retryInterval: 重试间隔时间
//
// 返回：
//   - error: 发送失败时返回错误
func sendRabbitMqMsgWithRetry(messageQueue *config.MessageQueue, message string, maxRetries int, retryInterval time.Duration) error {
	queueInfo := messageQueue.GetInfo()
	var lastErr error

	// 重试循环
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// 如果不是首次尝试，等待重试间隔
		if attempt > 0 {
			time.Sleep(retryInterval)
			logger.Info("[消息队列] 重试发送消息, queueInfo: %s, 尝试次数: %d/%d", queueInfo, attempt, maxRetries)
		}

		// 获取或初始化生产者
		producer, err := getOrInitProducer(messageQueue, queueInfo)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				continue // 继续重试
			}
			return fmt.Errorf("初始化生产者失败: %w", err)
		}

		// 检查并重新初始化通道（如果已关闭）
		if producer.Channel == nil || producer.Channel.IsClosed() {
			err = producer.InitChannelForProducer()
			if err != nil {
				lastErr = err
				if attempt < maxRetries {
					continue // 继续重试
				}
				return fmt.Errorf("重新初始化通道失败: %w", err)
			}
		}

		// 尝试发布消息
		err = producer.Publish(message)
		if err != nil {
			lastErr = err
			// 如果是连接相关错误，标记通道为关闭状态，下次重试时会重新初始化
			if producer.Channel != nil {
				producer.Channel.Close()
			}
			if attempt < maxRetries {
				continue // 继续重试
			}
			return fmt.Errorf("消息发布失败: %w", err)
		}

		// 发送成功
		if BaseConfig.RabbitMQ.LogMessageContent {
			logger.Info("[消息队列] 消息发布成功, queueInfo: %s, message: %s", queueInfo, message)
		} else {
			logger.Info("[消息队列] 消息发布成功, queueInfo: %s", queueInfo)
		}
		return nil
	}

	// 所有重试都失败
	return fmt.Errorf("消息发送失败，已重试 %d 次: %w", maxRetries, lastErr)
}

// getOrInitProducer 获取或初始化消息队列生产者
// 该函数是线程安全的，使用 sync.Map 提供并发安全的读写访问
// 参数：
//   - messageQueue: 消息队列配置（指针）
//   - queueInfo: 队列信息字符串
//
// 返回：
//   - *config.MessageQueue: 生产者实例
//   - error: 初始化失败时返回错误
func getOrInitProducer(messageQueue *config.MessageQueue, queueInfo string) (*config.MessageQueue, error) {
	// 快速路径：尝试从 sync.Map 中读取已存在的生产者
	if producer, ok := RabbitMQProducerList.Load(queueInfo); ok {
		return producer.(*config.MessageQueue), nil
	}

	// 慢路径：需要初始化新的生产者
	// 先初始化连接和通道
	err := messageQueue.InitChannelForProducer()
	if err != nil {
		return nil, fmt.Errorf("初始化发送者失败, queueInfo: %s, error: %w", queueInfo, err)
	}

	// 使用 LoadOrStore 确保并发安全，避免重复初始化
	// 如果另一个 goroutine 已经存储了该 key，则使用已存储的值
	actual, loaded := RabbitMQProducerList.LoadOrStore(queueInfo, messageQueue)
	if loaded {
		// 已存在，关闭我们刚初始化的连接，使用已有的
		if messageQueue.Channel != nil {
			messageQueue.Channel.Close()
		}
		if messageQueue.Conn != nil {
			messageQueue.Conn.Close()
		}
		return actual.(*config.MessageQueue), nil
	}

	logger.Info("[消息队列] 动态初始化发送者成功, queueInfo: %s", queueInfo)
	return messageQueue, nil
}

// SendRabbitMqMsgBatch 批量发送RabbitMQ消息
// 该函数支持向消息队列批量发送消息，提高发送效率
// 参数：
//   - queueName: 队列名称
//   - exchangeName: 交换机名称
//   - exchangeType: 交换机类型（direct, fanout, topic, headers）
//   - routingKey: 路由键
//   - messages: 消息内容列表
//   - mqConfigNames: 消息队列配置名称列表（可选，为空时使用默认配置）
//
// 返回：
//   - error: 如果所有消息队列都发送失败则返回错误
func SendRabbitMqMsgBatch(queueName string, exchangeName string,
	exchangeType string, routingKey string, messages []string, mqConfigNames ...string) error {
	return SendRabbitMqMsgBatchWithContext(context.Background(), queueName, exchangeName, exchangeType, routingKey, messages, mqConfigNames...)
}

// SendRabbitMqMsgBatchWithContext 批量发送RabbitMQ消息（带 context）
// 参数：
//   - ctx: context
//   - queueName: 队列名称
//   - exchangeName: 交换机名称
//   - exchangeType: 交换机类型（direct, fanout, topic, headers）
//   - routingKey: 路由键
//   - messages: 消息内容列表
//   - mqConfigNames: 消息队列配置名称列表（可选，为空时使用默认配置）
//
// 返回：
//   - error: 如果所有消息队列都发送失败则返回错误
func SendRabbitMqMsgBatchWithContext(ctx context.Context, queueName string, exchangeName string,
	exchangeType string, routingKey string, messages []string, mqConfigNames ...string) error {
	if len(messages) == 0 {
		return nil
	}

	if len(mqConfigNames) == 0 {
		mqConfigNames = []string{""}
	}

	var lastErr error
	successCount := 0

	for _, mqConfigName := range mqConfigNames {
		messageQueue := config.MessageQueue{
			MQName:       mqConfigName,
			QueueName:    queueName,
			ExchangeName: exchangeName,
			ExchangeType: exchangeType,
			RoutingKey:   routingKey,
		}
		// 获取消息队列连接字符串
		mqConnStr := BaseConfig.RabbitMQ.Url()
		// 如果配置了消息队列名称, 则使用对应的消息队列
		if messageQueue.MQName != "" {
			mqConnStr = BaseConfig.RabbitMQList.Url(messageQueue.MQName)
		}
		if mqConnStr == "" {
			err := fmt.Errorf("[消息队列] 未找到对应的消息队列配置, MQName: %s", messageQueue.MQName)
			logger.Error("%v", err)
			lastErr = err
			continue
		}
		messageQueue.MqConnStr = mqConnStr

		// 获取或初始化生产者
		queueInfo := messageQueue.GetInfo()
		producer, err := getOrInitProducer(&messageQueue, queueInfo)
		if err != nil {
			lastErr = err
			logger.Error("[消息队列] 初始化生产者失败, queueInfo: %s, error: %v", queueInfo, err)
			continue
		}

		// 批量发送消息
		err = producer.PublishBatchWithContext(ctx, messages)
		if err != nil {
			lastErr = err
			logger.Error("[消息队列] 批量消息发送失败, queueInfo: %s, error: %v", queueInfo, err)
		} else {
			successCount++
			logger.Info("[消息队列] 批量消息发布成功, queueInfo: %s, 消息数量: %d", queueInfo, len(messages))
		}
	}

	// 如果所有消息队列都发送失败，返回错误
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("[消息队列] 所有消息队列批量发送失败: %w", lastErr)
	}

	return nil
}

// SendRabbitMqMsgWithConfirm 发送RabbitMQ消息（启用 Publisher Confirms）
// 该函数支持消息确认机制，确保消息成功投递到 RabbitMQ
// 参数：
//   - queueName: 队列名称
//   - exchangeName: 交换机名称
//   - exchangeType: 交换机类型（direct, fanout, topic, headers）
//   - routingKey: 路由键
//   - message: 消息内容
//   - confirmTimeout: 确认超时时间
//   - mqConfigNames: 消息队列配置名称列表（可选，为空时使用默认配置）
//
// 返回：
//   - error: 如果所有消息队列都发送失败则返回错误
func SendRabbitMqMsgWithConfirm(queueName string, exchangeName string,
	exchangeType string, routingKey string, message string, confirmTimeout time.Duration, mqConfigNames ...string) error {
	if len(mqConfigNames) == 0 {
		mqConfigNames = []string{""}
	}

	var lastErr error
	successCount := 0

	for _, mqConfigName := range mqConfigNames {
		messageQueue := config.MessageQueue{
			MQName:       mqConfigName,
			QueueName:    queueName,
			ExchangeName: exchangeName,
			ExchangeType: exchangeType,
			RoutingKey:   routingKey,
			PublishConfirm: config.PublishConfirmConfig{
				Enabled: true,
				Timeout: confirmTimeout,
			},
		}
		// 获取消息队列连接字符串
		mqConnStr := BaseConfig.RabbitMQ.Url()
		// 如果配置了消息队列名称, 则使用对应的消息队列
		if messageQueue.MQName != "" {
			mqConnStr = BaseConfig.RabbitMQList.Url(messageQueue.MQName)
		}
		if mqConnStr == "" {
			err := fmt.Errorf("[消息队列] 未找到对应的消息队列配置, MQName: %s", messageQueue.MQName)
			logger.Error("%v", err)
			lastErr = err
			continue
		}
		messageQueue.MqConnStr = mqConnStr

		// 发送消息，带重试机制
		err := sendRabbitMqMsgWithRetry(&messageQueue, message, 3, 100*time.Millisecond)
		if err != nil {
			lastErr = err
			logger.Error("[消息队列] 消息发送失败, queueInfo: %s, error: %v", messageQueue.GetInfo(), err)
		} else {
			successCount++
		}
	}

	// 如果所有消息队列都发送失败，返回错误
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("[消息队列] 所有消息队列发送失败: %w", lastErr)
	}

	return nil
}
