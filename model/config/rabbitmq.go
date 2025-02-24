package config

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQInfo struct {
	AliasName string `yaml:"aliasName"` // 代表当前实例的名字
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
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

type MessageQueue struct {
	MQName       string
	QueueName    string
	ExchangeName string
	// direct(根据路由精准匹配), fanout(广播, queue和routing都设空), topic(路由模糊匹配), headers(根据header匹配)
	ExchangeType string
	// ExchangeDurable    bool
	// ExchangeAutoDelete bool
	RoutingKey string
	MqConnStr  string
	Conn       *amqp.Connection
	Channel    *amqp.Channel
	Fun        func(string) error
}

func (m *MessageQueue) GetInfo() string {
	return fmt.Sprintf("%s_%s_%s_%s_%s", m.MQName, m.QueueName, m.ExchangeName, m.ExchangeType, m.RoutingKey)
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
			return fmt.Errorf("连接失败, queueInfo: %s, error: %v", queueInfo, err)
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

			return fmt.Errorf("开启通道失败: queueInfo: %s, error: %v", queueInfo, err)
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
				return fmt.Errorf("声明交换机失败: queueInfo: %s, error: %v", queueInfo, err)
			}
		}

		q, err := ch.QueueDeclare(
			m.QueueName, // name
			true,        // durable
			false,       // delete when unused
			false,       // exclusive
			false,       // no-wait
			nil,         // arguments
		)
		if err != nil {
			return fmt.Errorf("创建队列失败: queueInfo: %s, error: %v", queueInfo, err)
		}

		err = ch.QueueBind(
			q.Name,         // queue name
			m.RoutingKey,   // routing key
			m.ExchangeName, // exchange
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("队列绑定失败: queueInfo: %s, error: %v", queueInfo, err)
		}

		err = ch.Qos(
			1,     // prefetch count
			0,     // prefetch size
			false, // global
		)
		if err != nil {
			return fmt.Errorf("设置QoS异常: queueInfo: %s, error: %v", queueInfo, err)
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

func (m *MessageQueue) Consume() error {
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
		return fmt.Errorf("注册消费者失败: queueInfo: %s, error: %v", queueInfo, err)
	}
	for {
		select {
		case msg := <-msgs:
			if err := m.Fun(string(msg.Body)); err == nil {
				msg.Ack(false)
			}
		case <-notifyClose:
			return fmt.Errorf("连接失败, queueInfo: %s", queueInfo)
		}
	}
}

func (m *MessageQueue) Publish(message string) error {
	err := m.initChannel()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.Channel.PublishWithContext(ctx,
		m.ExchangeName, // exchange
		m.RoutingKey,   // routing key
		false,          // mandatory
		false,          // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		return fmt.Errorf("消息发布失败, queueInfo: %s, error: %v", m.GetInfo(), err)
	}
	return nil
}
