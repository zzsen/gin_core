package services

import (
	"context"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/initialize"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// RabbitMQService RabbitMQ消息队列服务
type RabbitMQService struct {
	consumerList []*config.MessageQueue
	producerList []*config.MessageQueue
}

// NewRabbitMQService 创建RabbitMQ服务
func NewRabbitMQService(consumerList, producerList []*config.MessageQueue) *RabbitMQService {
	return &RabbitMQService{
		consumerList: consumerList,
		producerList: producerList,
	}
}

// Name 返回服务名称
func (s *RabbitMQService) Name() string { return "rabbitmq" }

// Priority 返回初始化优先级
func (s *RabbitMQService) Priority() int { return 30 }

// Dependencies 返回依赖
func (s *RabbitMQService) Dependencies() []string { return []string{"logger"} }

// ShouldInit 根据配置判断是否需要初始化
func (s *RabbitMQService) ShouldInit(cfg *config.BaseConfig) bool {
	return cfg.System.UseRabbitMQ
}

// Init 初始化RabbitMQ
func (s *RabbitMQService) Init(ctx context.Context) error {
	// 初始化消息队列生产者
	if len(s.producerList) > 0 {
		initialize.InitialRabbitMqProducer(s.producerList...)
	}

	// 在协程中启动消息队列消费者，避免阻塞服务启动
	if len(s.consumerList) > 0 {
		go initialize.InitialRabbitMq(s.consumerList...)
	}

	return nil
}

// Close 关闭RabbitMQ连接
func (s *RabbitMQService) Close(ctx context.Context) error {
	// 遍历 sync.Map 中的所有生产者并关闭
	app.RabbitMQProducerList.Range(func(key, value any) bool {
		producer := value.(*config.MessageQueue)
		producer.Close()
		logger.Info("[RabbitMQ] 已关闭生产者: %s", producer.GetInfo())
		return true // 继续遍历
	})
	return nil
}

// SetConsumerList 设置消费者列表
func (s *RabbitMQService) SetConsumerList(list []*config.MessageQueue) {
	s.consumerList = list
}

// SetProducerList 设置生产者列表
func (s *RabbitMQService) SetProducerList(list []*config.MessageQueue) {
	s.producerList = list
}
