package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 待重启的消息队列通道, 出错时将队列名称加入到通道
var queueToRestart chan (string) = make(chan string)

func InitialRabbitMq(messageQueueList ...config.MessageQueue) {
	messageQueueMap := map[string]config.MessageQueue{}

	for _, messageQueue := range messageQueueList {
		messageQueueMap[messageQueue.GetInfo()] = messageQueue
		go startMqConsume(messageQueue)
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

func startMqConsume(messageQueue config.MessageQueue) {
	queueInfo := messageQueue.GetInfo()
	conn, err := amqp.Dial(global.BaseConfig.RabbitMQ.Url())
	if err != nil {
		queueToRestart <- queueInfo
		logger.Error(fmt.Sprintf("[消息队列] 连接失败, queueInfo: %s, error: %v", queueInfo, err))
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		queueToRestart <- queueInfo
		logger.Error(fmt.Sprintf("[消息队列] 开启通道失败: queueInfo: %s, error: %v", queueInfo, err))
		return
	}
	defer ch.Close()

	if messageQueue.ExchangeName != "" {
		err = ch.ExchangeDeclare(
			messageQueue.ExchangeName, // name
			messageQueue.ExchangeType, // type
			true,                      // durable
			false,                     // auto-deleted
			false,                     // internal
			false,                     // no-wait
			nil,                       // arguments
		)
		if err != nil {
			queueToRestart <- queueInfo
			logger.Error(fmt.Sprintf("[消息队列] 声明交换机失败: queueInfo: %s, error: %v", queueInfo, err))
			return
		}
		defer ch.Close()
	}

	q, err := ch.QueueDeclare(
		messageQueue.QueueName, // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		queueToRestart <- queueInfo
		logger.Error(fmt.Sprintf("[消息队列] 创建队列失败: queueInfo: %s, error: %v", queueInfo, err))
		return
	}

	if messageQueue.RoutingKey != "" {
		err = ch.QueueBind(
			q.Name,                    // queue name
			messageQueue.RoutingKey,   // routing key
			messageQueue.ExchangeName, // exchange
			false,
			nil,
		)
	}
	if err != nil {
		queueToRestart <- queueInfo
		logger.Error(fmt.Sprintf("[消息队列] 队列绑定失败: queueInfo: %s, error: %v", queueInfo, err))
		return
	}

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		queueToRestart <- queueInfo
		logger.Error(fmt.Sprintf("[消息队列] 设置QoS异常: queueInfo: %s, error: %v", queueInfo, err))
		return
	}

	closeChan := make(chan *amqp.Error, 1)
	notifyClose := ch.NotifyClose(closeChan)

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		queueToRestart <- queueInfo
		logger.Error(fmt.Sprintf("[消息队列] 注册消费者失败: queueInfo: %s, error: %v", queueInfo, err))
		return
	}
	logger.Info("[消息队列] 连接成功, queueInfo: %s", queueInfo)

	for {
		select {
		case msg := <-msgs:
			if err := messageQueue.Fun(string(msg.Body)); err == nil {
				msg.Ack(false)
			}
		case <-notifyClose:
			logger.Error("[消息队列] 连接失败, queueInfo: %s", queueInfo)
			queueToRestart <- queueInfo
			return
		}
	}
}
