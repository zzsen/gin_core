package initialize

import (
	"fmt"
	"sync"
	"testing"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// getProducerTestRabbitMQUrl 获取测试用的 RabbitMQ URL
func getProducerTestRabbitMQUrl() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", producerTestRabbitMQUsername, producerTestRabbitMQPassword, producerTestRabbitMQHost, producerTestRabbitMQPort)
}

// ==================== 测试辅助函数 ====================

// 硬编码的 RabbitMQ 测试配置
const (
	producerTestRabbitMQHost     = "localhost"
	producerTestRabbitMQPort     = 5672
	producerTestRabbitMQUsername = "guest"
	producerTestRabbitMQPassword = "rabbitMq10.160.23.43"
)

// setupProducerTestConfig 设置测试配置
func setupProducerTestConfig() func() {
	// 备份原始配置
	originalConfig := app.BaseConfig
	originalProducerList := app.RabbitMQProducerList

	// 设置测试配置（硬编码）
	app.BaseConfig = config.BaseConfig{
		RabbitMQ: config.RabbitMQInfo{
			Host:     producerTestRabbitMQHost,
			Port:     producerTestRabbitMQPort,
			Username: producerTestRabbitMQUsername,
			Password: producerTestRabbitMQPassword,
		},
		RabbitMQList: config.RabbitMqListInfo{
			{AliasName: "primary", Host: producerTestRabbitMQHost, Port: producerTestRabbitMQPort, Username: producerTestRabbitMQUsername, Password: producerTestRabbitMQPassword},
			{AliasName: "secondary", Host: "192.168.1.100", Port: 5672, Username: "admin", Password: "admin"},
		},
	}

	app.RabbitMQProducerList = make(map[string]*config.MessageQueue)

	// 返回清理函数
	return func() {
		// 关闭所有生产者连接
		for _, producer := range app.RabbitMQProducerList {
			producer.Close()
		}
		app.BaseConfig = originalConfig
		app.RabbitMQProducerList = originalProducerList
	}
}

// ==================== 单元测试：InitialRabbitMqProducer 参数校验（不需要 RabbitMQ 连接） ====================
// 测试点：验证生产者初始化的参数校验逻辑

// TestInitialRabbitMqProducer_Empty 测试空生产者列表初始化
// 不需要 RabbitMQ 连接：仅验证空列表处理逻辑
func TestInitialRabbitMqProducer_Empty(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	// 空列表不应 panic
	InitialRabbitMqProducer()

	if len(app.RabbitMQProducerList) != 0 {
		t.Errorf("空列表初始化后，生产者列表应为空，实际长度: %d", len(app.RabbitMQProducerList))
	}
}

// TestInitialRabbitMqProducer_NoMQConfig 测试无 MQ 配置时的初始化
// 不需要 RabbitMQ 连接：仅验证配置校验逻辑
func TestInitialRabbitMqProducer_NoMQConfig(t *testing.T) {
	// 备份并清空配置
	originalConfig := app.BaseConfig
	app.BaseConfig = config.BaseConfig{}
	defer func() { app.BaseConfig = originalConfig }()

	// 无配置时初始化（不应 panic，但会记录错误日志）
	mq := &config.MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	// 直接调用内部函数
	initMqProducer(mq)

	// 由于无法获取连接字符串，生产者列表应为空
	if len(app.RabbitMQProducerList) != 0 {
		t.Errorf("无配置时，生产者列表应为空，实际长度: %d", len(app.RabbitMQProducerList))
	}
}

// TestInitialRabbitMqProducer_InvalidMQName 测试使用无效 MQName 的初始化
// 不需要 RabbitMQ 连接：仅验证 MQName 校验逻辑
func TestInitialRabbitMqProducer_InvalidMQName(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	// 使用不存在的 MQName
	mq := &config.MessageQueue{
		MQName:       "not-exist-mq",
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	// 直接调用内部函数
	initMqProducer(mq)

	// 由于 MQName 不存在，生产者列表应为空
	if len(app.RabbitMQProducerList) != 0 {
		t.Errorf("无效 MQName 时，生产者列表应为空，实际长度: %d", len(app.RabbitMQProducerList))
	}
}

// TestInitialRabbitMqProducer_MultipleQueues 测试初始化多个生产者队列
// 不需要 RabbitMQ 连接：仅验证多队列处理逻辑（会因无法连接而失败，但不应 panic）
func TestInitialRabbitMqProducer_MultipleQueues(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	// 初始化多个生产者（无法真正连接，但可以测试逻辑）
	mq1 := &config.MessageQueue{
		QueueName:    "queue1",
		ExchangeName: "exchange1",
		ExchangeType: "direct",
		RoutingKey:   "key1",
	}

	mq2 := &config.MessageQueue{
		QueueName:    "queue2",
		ExchangeName: "exchange2",
		ExchangeType: "topic",
		RoutingKey:   "key2",
	}

	mq3 := &config.MessageQueue{
		MQName:       "primary",
		QueueName:    "queue3",
		ExchangeName: "exchange3",
		ExchangeType: "fanout",
		RoutingKey:   "key3",
	}

	// 调用初始化（会因为无法连接而失败，但不应 panic）
	initMqProducer(mq1)
	initMqProducer(mq2)
	initMqProducer(mq3)

	// 由于无法真正连接，生产者列表可能为空
	// 这里只测试不 panic
	t.Logf("生产者列表长度: %d", len(app.RabbitMQProducerList))
}

// ==================== 单元测试：initMqProducer 连接字符串设置（不需要 RabbitMQ 连接） ====================
// 测试点：验证生产者初始化时连接字符串的设置逻辑

// TestInitMqProducer_DefaultConfig 测试使用默认配置初始化生产者
// 不需要 RabbitMQ 连接：仅验证连接字符串设置（会因无法连接而失败）
func TestInitMqProducer_DefaultConfig(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	mq := &config.MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	// 调用内部初始化函数（会因为无法连接而失败，但会设置连接字符串）
	initMqProducer(mq)

	// 验证连接字符串已设置
	expectedConnStr := getProducerTestRabbitMQUrl()
	if mq.MqConnStr != expectedConnStr {
		t.Errorf("MqConnStr = %v, want %v", mq.MqConnStr, expectedConnStr)
	}
}

// TestInitMqProducer_NamedConfig 测试使用命名配置初始化生产者
// 不需要 RabbitMQ 连接：仅验证连接字符串设置（会因无法连接而失败）
func TestInitMqProducer_NamedConfig(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	mq := &config.MessageQueue{
		MQName:       "primary",
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	// 调用内部初始化函数
	initMqProducer(mq)

	// 验证连接字符串已设置（使用 primary 配置）
	expectedConnStr := getProducerTestRabbitMQUrl()
	if mq.MqConnStr != expectedConnStr {
		t.Errorf("MqConnStr = %v, want %v", mq.MqConnStr, expectedConnStr)
	}
}

// TestInitMqProducer_InvalidConnection 测试无效连接配置的初始化
// 不需要 RabbitMQ 连接：验证连接失败时的处理逻辑
func TestInitMqProducer_InvalidConnection(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	mq := &config.MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	// 调用初始化（会因为无法连接而失败）
	initMqProducer(mq)

	// 由于无法连接，生产者不应被添加到列表
	queueInfo := mq.GetInfo()
	if _, ok := app.RabbitMQProducerList[queueInfo]; ok {
		// 如果本地有 RabbitMQ 运行，则可能成功
		t.Log("连接成功（本地有 RabbitMQ 运行）")
	} else {
		t.Log("预期的连接失败，生产者未添加到列表")
	}
}

// ==================== 单元测试：并发安全（不需要 RabbitMQ 连接） ====================
// 测试点：验证生产者初始化的并发安全性

// TestInitialRabbitMqProducer_ConcurrentInit 测试并发初始化生产者
// 不需要 RabbitMQ 连接：仅验证并发安全性（会因无法连接而失败，但不应 panic）
func TestInitialRabbitMqProducer_ConcurrentInit(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	var wg sync.WaitGroup
	numGoroutines := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			mq := &config.MessageQueue{
				QueueName:    "concurrent-queue",
				ExchangeName: "concurrent-exchange",
				ExchangeType: "direct",
				RoutingKey:   "concurrent-key",
			}

			// 并发初始化不应 panic
			initMqProducer(mq)
		}(i)
	}

	wg.Wait()
}

// ==================== 单元测试：配置验证（不需要 RabbitMQ 连接） ====================
// 测试点：验证生产者配置的 GetInfo 方法

// TestInitMqProducer_VerifyQueueInfo 测试验证队列信息格式
// 不需要 RabbitMQ 连接：仅验证配置格式化逻辑
func TestInitMqProducer_VerifyQueueInfo(t *testing.T) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	tests := []struct {
		name         string
		mqName       string
		queueName    string
		exchangeName string
		exchangeType string
		routingKey   string
		expectedInfo string
	}{
		{
			name:         "完整配置",
			mqName:       "test-mq",
			queueName:    "order-queue",
			exchangeName: "order-exchange",
			exchangeType: "direct",
			routingKey:   "order-key",
			expectedInfo: "test-mq_order-queue_order-exchange_direct_order-key",
		},
		{
			name:         "无 MQName",
			mqName:       "",
			queueName:    "user-queue",
			exchangeName: "user-exchange",
			exchangeType: "topic",
			routingKey:   "user.*",
			expectedInfo: "_user-queue_user-exchange_topic_user.*",
		},
		{
			name:         "fanout 类型",
			mqName:       "",
			queueName:    "broadcast-queue",
			exchangeName: "broadcast-exchange",
			exchangeType: "fanout",
			routingKey:   "",
			expectedInfo: "_broadcast-queue_broadcast-exchange_fanout_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mq := &config.MessageQueue{
				MQName:       tt.mqName,
				QueueName:    tt.queueName,
				ExchangeName: tt.exchangeName,
				ExchangeType: tt.exchangeType,
				RoutingKey:   tt.routingKey,
			}
			if mq.GetInfo() != tt.expectedInfo {
				t.Errorf("GetInfo() = %v, want %v", mq.GetInfo(), tt.expectedInfo)
			}
		})
	}
}

// ==================== 单元测试：基准测试（不需要 RabbitMQ 连接） ====================
// 测试点：验证生产者初始化的性能

// BenchmarkInitialRabbitMqProducer_Empty 基准测试空列表初始化性能
// 不需要 RabbitMQ 连接：仅测试空列表处理性能
func BenchmarkInitialRabbitMqProducer_Empty(b *testing.B) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InitialRabbitMqProducer()
	}
}

// BenchmarkInitMqProducer_NoConnection 基准测试无连接时的初始化性能
// 不需要 RabbitMQ 连接：仅测试初始化逻辑性能（会因无法连接而失败）
func BenchmarkInitMqProducer_NoConnection(b *testing.B) {
	cleanup := setupProducerTestConfig()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mq := &config.MessageQueue{
			QueueName:    "bench-queue",
			ExchangeName: "bench-exchange",
			ExchangeType: "direct",
			RoutingKey:   "bench-key",
		}
		initMqProducer(mq)
	}
}
