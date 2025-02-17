package config

import (
	"fmt"
	"reflect"
	"runtime"
)

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
