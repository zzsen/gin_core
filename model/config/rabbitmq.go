package config

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQInfo struct {
	AliasName         string `yaml:"aliasName"`         // 代表当前实例的名字
	Host              string `yaml:"host"`              // 主机地址
	Port              int    `yaml:"port"`              // 端口号
	Username          string `yaml:"username"`          // 用户名
	Password          string `yaml:"password"`          // 密码
	LogMessageContent bool   `yaml:"logMessageContent"` // 是否在日志中输出消息内容，默认 false（不输出），生产环境建议关闭以避免敏感信息泄露
}

func (rabbitMQInfo *RabbitMQInfo) Url() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/",
		rabbitMQInfo.Username,
		rabbitMQInfo.Password,
		rabbitMQInfo.Host,
		rabbitMQInfo.Port)
}

type RabbitMqListInfo []RabbitMQInfo

func (rabbitMqListInfo *RabbitMqListInfo) Url(aliasName string) string {
	for _, rabbitMQInfo := range *rabbitMqListInfo {
		if rabbitMQInfo.AliasName == aliasName {
			return rabbitMQInfo.Url()
		}
	}
	return ""
}

// DeadLetterConfig 死信队列配置
// 用于配置消息消费失败后的处理策略
type DeadLetterConfig struct {
	// Enabled 是否启用死信队列
	Enabled bool
	// Exchange 死信交换机名称，为空时自动生成（原交换机名称 + ".dlx"）
	Exchange string
	// RoutingKey 死信路由键，为空时使用原路由键
	RoutingKey string
	// QueueName 死信队列名称，为空时自动生成（原队列名称 + ".dlq"）
	QueueName string
	// MessageTTL 消息在死信队列中的存活时间（毫秒），0表示永不过期
	MessageTTL int64
}

// PublishConfirmConfig Publisher Confirms 配置
// 用于确保消息成功投递到 RabbitMQ
type PublishConfirmConfig struct {
	// Enabled 是否启用发布确认
	Enabled bool
	// Timeout 确认超时时间
	Timeout time.Duration
}

// ConsumeConfig 消费者配置
type ConsumeConfig struct {
	// PrefetchCount 预取数量，控制消费者一次从队列获取的消息数量
	PrefetchCount int
	// MaxRetry 最大重试次数，超过后消息将被发送到死信队列
	MaxRetry int
	// RetryDelay 重试延迟时间
	RetryDelay time.Duration
}

type MessageQueue struct {
	MQName       string
	QueueName    string
	ExchangeName string
	// direct(根据路由精准匹配), fanout(广播, queue和routing都设空), topic(路由模糊匹配), headers(根据header匹配)
	ExchangeType string
	RoutingKey   string
	MqConnStr    string
	Conn         *amqp.Connection
	Channel      *amqp.Channel
	// Fun 消费函数（旧版兼容，建议使用 FunWithCtx）
	Fun func(string) error
	// FunWithCtx 带 context 的消费函数，支持优雅关闭
	FunWithCtx func(ctx context.Context, msg string) error
	// DeadLetter 死信队列配置
	DeadLetter DeadLetterConfig
	// PublishConfirm Publisher Confirms 配置
	PublishConfirm PublishConfirmConfig
	// ConsumeConfig 消费者配置
	ConsumeConfig ConsumeConfig
	// confirmChan 用于接收发布确认
	confirmChan chan amqp.Confirmation
	// confirmLock 用于保护确认通道的并发访问
	confirmLock sync.Mutex
}

func (m *MessageQueue) GetInfo() string {
	var b strings.Builder
	b.Grow(len(m.MQName) + len(m.QueueName) + len(m.ExchangeName) + len(m.ExchangeType) + len(m.RoutingKey) + 4)
	b.WriteString(m.MQName)
	b.WriteByte('_')
	b.WriteString(m.QueueName)
	b.WriteByte('_')
	b.WriteString(m.ExchangeName)
	b.WriteByte('_')
	b.WriteString(m.ExchangeType)
	b.WriteByte('_')
	b.WriteString(m.RoutingKey)
	return b.String()
}

func (m *MessageQueue) GetFuncInfo() string {
	// 使用 reflect.ValueOf 获取传入函数的反射值
	value := reflect.ValueOf(m.Fun)
	if value.Kind() != reflect.Func {
		return ""
	}
	// 获取函数的指针
	pc := value.Pointer()
	// 根据函数指针获取函数的详细信息
	funcInfo := runtime.FuncForPC(pc)
	if funcInfo == nil {
		return ""
	}
	// 返回函数名
	return funcInfo.Name()
}

func (m *MessageQueue) initConn() error {
	queueInfo := m.GetInfo()
	if m.Conn == nil || m.Conn.IsClosed() {
		conn, err := amqp.Dial(m.MqConnStr)
		if err != nil {
			return fmt.Errorf("连接失败, queueInfo: %s, error: %w", queueInfo, err)
		}
		m.Conn = conn
	}
	return nil
}

func (m *MessageQueue) initChannel() error {
	if m.Channel == nil || m.Channel.IsClosed() {
		queueInfo := m.GetInfo()

		if err := m.initConn(); err != nil {
			return err
		}

		ch, err := m.Conn.Channel()
		if err != nil {
			return fmt.Errorf("开启通道失败: queueInfo: %s, error: %w", queueInfo, err)
		}

		if m.ExchangeName != "" {
			err = ch.ExchangeDeclare(
				m.ExchangeName, // name
				m.ExchangeType, // type
				true,           // durable
				false,          // auto-deleted
				false,          // internal
				false,          // no-wait
				nil,            // arguments
			)
			if err != nil {
				return fmt.Errorf("声明交换机失败: queueInfo: %s, error: %w", queueInfo, err)
			}
		}

		// 构建队列参数
		queueArgs := amqp.Table{}

		// 配置死信队列
		if m.DeadLetter.Enabled {
			// 初始化死信交换机和队列
			if err := m.initDeadLetterQueue(ch); err != nil {
				return err
			}

			// 设置死信交换机
			dlxExchange := m.getDeadLetterExchange()
			queueArgs["x-dead-letter-exchange"] = dlxExchange

			// 设置死信路由键
			dlxRoutingKey := m.getDeadLetterRoutingKey()
			if dlxRoutingKey != "" {
				queueArgs["x-dead-letter-routing-key"] = dlxRoutingKey
			}
		}

		var argsPtr amqp.Table
		if len(queueArgs) > 0 {
			argsPtr = queueArgs
		}

		q, err := ch.QueueDeclare(
			m.QueueName, // name
			true,        // durable
			false,       // delete when unused
			false,       // exclusive
			false,       // no-wait
			argsPtr,     // arguments
		)
		if err != nil {
			return fmt.Errorf("创建队列失败: queueInfo: %s, error: %w", queueInfo, err)
		}

		err = ch.QueueBind(
			q.Name,         // queue name
			m.RoutingKey,   // routing key
			m.ExchangeName, // exchange
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("队列绑定失败: queueInfo: %s, error: %w", queueInfo, err)
		}

		// 设置 QoS，使用配置的 PrefetchCount，默认为 1
		prefetchCount := m.ConsumeConfig.PrefetchCount
		if prefetchCount <= 0 {
			prefetchCount = 1
		}
		err = ch.Qos(
			prefetchCount, // prefetch count
			0,             // prefetch size
			false,         // global
		)
		if err != nil {
			return fmt.Errorf("设置QoS异常: queueInfo: %s, error: %w", queueInfo, err)
		}

		m.Channel = ch
	}
	return nil
}

// initDeadLetterQueue 初始化死信队列
func (m *MessageQueue) initDeadLetterQueue(ch *amqp.Channel) error {
	queueInfo := m.GetInfo()
	dlxExchange := m.getDeadLetterExchange()
	dlxQueue := m.getDeadLetterQueue()
	dlxRoutingKey := m.getDeadLetterRoutingKey()

	// 声明死信交换机
	err := ch.ExchangeDeclare(
		dlxExchange,    // name
		m.ExchangeType, // type - 使用与主交换机相同的类型
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return fmt.Errorf("声明死信交换机失败: queueInfo: %s, error: %w", queueInfo, err)
	}

	// 构建死信队列参数
	dlqArgs := amqp.Table{}
	if m.DeadLetter.MessageTTL > 0 {
		dlqArgs["x-message-ttl"] = m.DeadLetter.MessageTTL
	}

	var dlqArgsPtr amqp.Table
	if len(dlqArgs) > 0 {
		dlqArgsPtr = dlqArgs
	}

	// 声明死信队列
	_, err = ch.QueueDeclare(
		dlxQueue,   // name
		true,       // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		dlqArgsPtr, // arguments
	)
	if err != nil {
		return fmt.Errorf("创建死信队列失败: queueInfo: %s, error: %w", queueInfo, err)
	}

	// 绑定死信队列到死信交换机
	err = ch.QueueBind(
		dlxQueue,      // queue name
		dlxRoutingKey, // routing key
		dlxExchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("死信队列绑定失败: queueInfo: %s, error: %w", queueInfo, err)
	}

	return nil
}

// getDeadLetterExchange 获取死信交换机名称
func (m *MessageQueue) getDeadLetterExchange() string {
	if m.DeadLetter.Exchange != "" {
		return m.DeadLetter.Exchange
	}
	if m.ExchangeName != "" {
		return m.ExchangeName + ".dlx"
	}
	return m.QueueName + ".dlx"
}

// getDeadLetterQueue 获取死信队列名称
func (m *MessageQueue) getDeadLetterQueue() string {
	if m.DeadLetter.QueueName != "" {
		return m.DeadLetter.QueueName
	}
	return m.QueueName + ".dlq"
}

// getDeadLetterRoutingKey 获取死信路由键
func (m *MessageQueue) getDeadLetterRoutingKey() string {
	if m.DeadLetter.RoutingKey != "" {
		return m.DeadLetter.RoutingKey
	}
	return m.RoutingKey
}

// InitChannelForProducer 初始化发送者通道
// 该方法专门为消息发送者设计，只初始化连接、通道和交换机
// 不进行队列声明和绑定，这些操作由消费者负责
// 返回值：
//   - error: 初始化失败时返回错误信息
func (m *MessageQueue) InitChannelForProducer() error {
	if m.Channel == nil || m.Channel.IsClosed() {
		queueInfo := m.GetInfo()

		if err := m.initConn(); err != nil {
			return err
		}

		ch, err := m.Conn.Channel()
		if err != nil {
			return fmt.Errorf("开启通道失败: queueInfo: %s, error: %w", queueInfo, err)
		}

		// 发送者只需要声明交换机，不需要声明队列和绑定
		if m.ExchangeName != "" {
			err = ch.ExchangeDeclare(
				m.ExchangeName, // name
				m.ExchangeType, // type
				true,           // durable
				false,          // auto-deleted
				false,          // internal
				false,          // no-wait
				nil,            // arguments
			)
			if err != nil {
				return fmt.Errorf("声明交换机失败: queueInfo: %s, error: %w", queueInfo, err)
			}
		}

		// 如果启用了 Publisher Confirms，设置确认模式
		if m.PublishConfirm.Enabled {
			err = ch.Confirm(false)
			if err != nil {
				return fmt.Errorf("设置确认模式失败: queueInfo: %s, error: %w", queueInfo, err)
			}

			// 初始化确认通道
			m.confirmLock.Lock()
			m.confirmChan = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
			m.confirmLock.Unlock()
		}

		m.Channel = ch
	}
	return nil
}

func (m *MessageQueue) Close() {
	if m.Conn != nil && !m.Conn.IsClosed() {
		m.Conn.Close()
	}
	if m.Channel != nil && !m.Channel.IsClosed() {
		m.Channel.Close()
	}
}

// Consume 启动消费者（无 context 版本，保持向后兼容）
func (m *MessageQueue) Consume() error {
	return m.ConsumeWithContext(context.Background())
}

// ConsumeWithContext 启动消费者（带 context 版本，支持优雅关闭）
// 当 context 被取消时，消费者会优雅地停止处理新消息
func (m *MessageQueue) ConsumeWithContext(ctx context.Context) error {
	err := m.initChannel()
	if err != nil {
		return err
	}

	closeChan := make(chan *amqp.Error, 1)
	notifyClose := m.Channel.NotifyClose(closeChan)

	msgs, err := m.Channel.Consume(
		m.QueueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)

	queueInfo := m.GetInfo()
	if err != nil {
		return fmt.Errorf("注册消费者失败: queueInfo: %s, error: %w", queueInfo, err)
	}

	for {
		select {
		case <-ctx.Done():
			// context 被取消，优雅关闭
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("消息通道已关闭, queueInfo: %s", queueInfo)
			}
			m.handleMessage(ctx, msg)
		case <-notifyClose:
			return fmt.Errorf("连接失败, queueInfo: %s", queueInfo)
		}
	}
}

// handleMessage 处理单条消息
func (m *MessageQueue) handleMessage(ctx context.Context, msg amqp.Delivery) {
	var err error
	msgBody := string(msg.Body)

	// 优先使用带 context 的处理函数
	if m.FunWithCtx != nil {
		err = m.FunWithCtx(ctx, msgBody)
	} else if m.Fun != nil {
		err = m.Fun(msgBody)
	} else {
		// 没有处理函数，直接确认
		msg.Ack(false)
		return
	}

	if err == nil {
		// 处理成功，确认消息
		msg.Ack(false)
		return
	}

	// 处理失败，检查重试次数
	retryCount := m.getRetryCount(msg)
	maxRetry := m.ConsumeConfig.MaxRetry
	if maxRetry <= 0 {
		maxRetry = 3 // 默认重试 3 次
	}

	if retryCount < maxRetry {
		// 重试：拒绝消息并重新入队
		// 注意：这里使用 Nack 并 requeue，消息会立即重新投递
		// 如果需要延迟重试，需要配合延迟队列或 TTL 实现
		msg.Nack(false, true)
	} else {
		// 超过重试次数，拒绝消息（如果配置了死信队列，消息会进入死信队列）
		msg.Nack(false, false)
	}
}

// getRetryCount 获取消息的重试次数
func (m *MessageQueue) getRetryCount(msg amqp.Delivery) int {
	if msg.Headers == nil {
		return 0
	}

	// 检查 x-death header（RabbitMQ 自动添加）
	if xDeath, ok := msg.Headers["x-death"]; ok {
		if deaths, ok := xDeath.([]interface{}); ok && len(deaths) > 0 {
			if death, ok := deaths[0].(amqp.Table); ok {
				if count, ok := death["count"]; ok {
					if countInt, ok := count.(int64); ok {
						return int(countInt)
					}
				}
			}
		}
	}

	return 0
}

// Publish 发布单条消息
func (m *MessageQueue) Publish(message string) error {
	return m.PublishWithContext(context.Background(), message)
}

// PublishWithContext 发布单条消息（带 context）
func (m *MessageQueue) PublishWithContext(ctx context.Context, message string) error {
	err := m.InitChannelForProducer()
	if err != nil {
		return err
	}

	// 设置发布超时
	timeout := 5 * time.Second
	if m.PublishConfirm.Timeout > 0 {
		timeout = m.PublishConfirm.Timeout
	}

	pubCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err = m.Channel.PublishWithContext(pubCtx,
		m.ExchangeName, // exchange
		m.RoutingKey,   // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(message),
			DeliveryMode: amqp.Persistent, // 持久化消息
		})
	if err != nil {
		return fmt.Errorf("消息发布失败, queueInfo: %s, error: %w", m.GetInfo(), err)
	}

	// 如果启用了 Publisher Confirms，等待确认
	if m.PublishConfirm.Enabled {
		if err := m.waitForConfirm(pubCtx); err != nil {
			return err
		}
	}

	return nil
}

// PublishBatch 批量发布消息
// 参数：
//   - messages: 要发布的消息列表
//
// 返回：
//   - error: 发布失败时返回错误，包含失败的消息索引
func (m *MessageQueue) PublishBatch(messages []string) error {
	return m.PublishBatchWithContext(context.Background(), messages)
}

// PublishBatchWithContext 批量发布消息（带 context）
// 参数：
//   - ctx: context
//   - messages: 要发布的消息列表
//
// 返回：
//   - error: 发布失败时返回错误
func (m *MessageQueue) PublishBatchWithContext(ctx context.Context, messages []string) error {
	if len(messages) == 0 {
		return nil
	}

	err := m.InitChannelForProducer()
	if err != nil {
		return err
	}

	// 设置发布超时
	timeout := 5 * time.Second
	if m.PublishConfirm.Timeout > 0 {
		timeout = m.PublishConfirm.Timeout
	}

	pubCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var failedIndexes []int
	var firstErr error

	for i, message := range messages {
		select {
		case <-pubCtx.Done():
			return fmt.Errorf("批量发布超时, queueInfo: %s, 已发布: %d/%d", m.GetInfo(), i, len(messages))
		default:
			err := m.Channel.PublishWithContext(pubCtx,
				m.ExchangeName, // exchange
				m.RoutingKey,   // routing key
				false,          // mandatory
				false,          // immediate
				amqp.Publishing{
					ContentType:  "text/plain",
					Body:         []byte(message),
					DeliveryMode: amqp.Persistent,
				})
			if err != nil {
				failedIndexes = append(failedIndexes, i)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}

	// 如果启用了 Publisher Confirms，等待所有确认
	if m.PublishConfirm.Enabled && len(failedIndexes) == 0 {
		// 等待所有消息确认
		for i := 0; i < len(messages); i++ {
			if err := m.waitForConfirm(pubCtx); err != nil {
				return fmt.Errorf("批量发布确认失败, queueInfo: %s, 消息索引: %d: %w", m.GetInfo(), i, err)
			}
		}
	}

	if len(failedIndexes) > 0 {
		return fmt.Errorf("批量发布部分失败, queueInfo: %s, 失败索引: %v: %w", m.GetInfo(), failedIndexes, firstErr)
	}

	return nil
}

// waitForConfirm 等待发布确认
func (m *MessageQueue) waitForConfirm(ctx context.Context) error {
	m.confirmLock.Lock()
	confirmChan := m.confirmChan
	m.confirmLock.Unlock()

	if confirmChan == nil {
		return fmt.Errorf("确认通道未初始化, queueInfo: %s", m.GetInfo())
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("等待确认超时, queueInfo: %s", m.GetInfo())
	case confirm, ok := <-confirmChan:
		if !ok {
			return fmt.Errorf("确认通道已关闭, queueInfo: %s", m.GetInfo())
		}
		if !confirm.Ack {
			return fmt.Errorf("消息未被确认, queueInfo: %s, deliveryTag: %d", m.GetInfo(), confirm.DeliveryTag)
		}
		return nil
	}
}
