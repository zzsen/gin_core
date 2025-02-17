package config

import "fmt"

type RabbitMQInfo struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

func (rabbitMQInfo *RabbitMQInfo) Url() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/",
		rabbitMQInfo.Username,
		rabbitMQInfo.Password,
		rabbitMQInfo.Host,
		rabbitMQInfo.Port)
}

type MessageQueue struct {
	QueueName    string
	ExchangeName string
	// direct, fanout, topic
	ExchangeType string
	RoutingKey   string
	Fun          func(string) error
}

func (m *MessageQueue) GetInfo() string {
	return fmt.Sprintf("%s_%s_%s_%s", m.QueueName, m.ExchangeName, m.ExchangeType, m.RoutingKey)
}
