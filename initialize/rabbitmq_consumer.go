package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// 待重启的消息队列通道, 出错时将队列名称加入到通道
var queueToRestart chan (string) = make(chan string)

func InitialRabbitMq(messageQueueList ...config.MessageQueue) {
	messageQueueMap := map[string]*config.MessageQueue{}

	for _, messageQueue := range messageQueueList {
		messageQueueMap[messageQueue.GetInfo()] = &messageQueue
		go startMqConsume(&messageQueue)
	}

	// 当服务异常时, 重启消费者任务
	for queueName := range queueToRestart {
		messageQueue, ok := messageQueueMap[queueName]
		if ok {
			time.Sleep(5 * time.Second)
			logger.Info("[消息队列] 正在尝试重连, queueInfo: %s", messageQueue.GetInfo())
			go startMqConsume(messageQueue)
		}
	}
}

func startMqConsume(messageQueue *config.MessageQueue) {
	// 获取消息队列连接字符串
	mqConnStr := app.BaseConfig.RabbitMQ.Url()
	// 如果配置了消息队列名称, 则使用对应的消息队列
	if messageQueue.MQName != "" {
		mqConnStr = app.BaseConfig.RabbitMQList.Url(messageQueue.MQName)
	}
	if mqConnStr == "" {
		logger.Error("[消息队列] 未找到对应的消息队列配置, MQName: %s", messageQueue.MQName)
		return
	}
	messageQueue.MqConnStr = mqConnStr
	err := messageQueue.Consume()
	if err != nil {
		queueToRestart <- messageQueue.GetInfo()
		logger.Error(fmt.Sprintf("[消息队列]%v", err.Error()))
	}
}
