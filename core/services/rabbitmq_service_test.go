package services

import (
	"context"
	"testing"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/model/config"
)

// ==================== 测试辅助函数 ====================

// 硬编码的 RabbitMQ 测试配置
const (
	serviceTestRabbitMQHost     = "localhost"
	serviceTestRabbitMQPort     = 5672
	serviceTestRabbitMQUsername = "admin"
	serviceTestRabbitMQPassword = "admin"
)

// setupRabbitMQServiceTestConfig 设置测试配置
func setupRabbitMQServiceTestConfig() func() {
	// 备份原始配置
	originalConfig := app.BaseConfig
	originalProducerList := app.RabbitMQProducerList

	// 设置测试配置（硬编码）
	app.BaseConfig = config.BaseConfig{
		RabbitMQ: config.RabbitMQInfo{
			Host:     serviceTestRabbitMQHost,
			Port:     serviceTestRabbitMQPort,
			Username: serviceTestRabbitMQUsername,
			Password: serviceTestRabbitMQPassword,
		},
		RabbitMQList: config.RabbitMqListInfo{
			{AliasName: "primary", Host: serviceTestRabbitMQHost, Port: serviceTestRabbitMQPort, Username: serviceTestRabbitMQUsername, Password: serviceTestRabbitMQPassword},
		},
		System: config.SystemInfo{
			UseRabbitMQ: true,
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

// ==================== 单元测试：NewRabbitMQService 服务创建（不需要 RabbitMQ 连接） ====================
// 测试点：验证 RabbitMQ 服务的创建逻辑

// TestNewRabbitMQService 测试创建 RabbitMQ 服务
// 不需要 RabbitMQ 连接：仅验证服务实例化逻辑
func TestNewRabbitMQService(t *testing.T) {
	consumers := []*config.MessageQueue{
		{QueueName: "consumer1"},
		{QueueName: "consumer2"},
	}
	producers := []*config.MessageQueue{
		{QueueName: "producer1"},
	}

	service := NewRabbitMQService(consumers, producers)

	if service == nil {
		t.Fatal("NewRabbitMQService 应返回非空实例")
	}

	if len(service.consumerList) != 2 {
		t.Errorf("consumerList 长度应为 2，实际为 %d", len(service.consumerList))
	}

	if len(service.producerList) != 1 {
		t.Errorf("producerList 长度应为 1，实际为 %d", len(service.producerList))
	}
}

// TestNewRabbitMQService_EmptyLists 测试使用空列表创建服务
// 不需要 RabbitMQ 连接：仅验证服务实例化逻辑
func TestNewRabbitMQService_EmptyLists(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	if service == nil {
		t.Fatal("NewRabbitMQService 应返回非空实例")
	}

	if service.consumerList != nil {
		t.Errorf("consumerList 应为 nil，实际为 %v", service.consumerList)
	}

	if service.producerList != nil {
		t.Errorf("producerList 应为 nil，实际为 %v", service.producerList)
	}
}

// ==================== 单元测试：Name 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务名称返回

// TestRabbitMQService_Name 测试获取服务名称
// 不需要 RabbitMQ 连接：仅验证返回值
func TestRabbitMQService_Name(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	name := service.Name()
	expected := "rabbitmq"

	if name != expected {
		t.Errorf("Name() = %v, want %v", name, expected)
	}
}

// ==================== 单元测试：Priority 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务优先级返回

// TestRabbitMQService_Priority 测试获取服务优先级
// 不需要 RabbitMQ 连接：仅验证返回值
func TestRabbitMQService_Priority(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	priority := service.Priority()
	expected := 30

	if priority != expected {
		t.Errorf("Priority() = %v, want %v", priority, expected)
	}
}

// ==================== 单元测试：Dependencies 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务依赖返回

// TestRabbitMQService_Dependencies 测试获取服务依赖
// 不需要 RabbitMQ 连接：仅验证返回值
func TestRabbitMQService_Dependencies(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	deps := service.Dependencies()

	if len(deps) != 1 {
		t.Errorf("Dependencies() 长度应为 1，实际为 %d", len(deps))
	}

	if deps[0] != "logger" {
		t.Errorf("Dependencies()[0] = %v, want 'logger'", deps[0])
	}
}

// ==================== 单元测试：ShouldInit 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务初始化条件判断

// TestRabbitMQService_ShouldInit_True 测试 UseRabbitMQ=true 时应初始化
// 不需要 RabbitMQ 连接：仅验证条件判断逻辑
func TestRabbitMQService_ShouldInit_True(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	cfg := &config.BaseConfig{
		System: config.SystemInfo{
			UseRabbitMQ: true,
		},
	}

	if !service.ShouldInit(cfg) {
		t.Error("ShouldInit() 应返回 true 当 UseRabbitMQ=true")
	}
}

// TestRabbitMQService_ShouldInit_False 测试 UseRabbitMQ=false 时不应初始化
// 不需要 RabbitMQ 连接：仅验证条件判断逻辑
func TestRabbitMQService_ShouldInit_False(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	cfg := &config.BaseConfig{
		System: config.SystemInfo{
			UseRabbitMQ: false,
		},
	}

	if service.ShouldInit(cfg) {
		t.Error("ShouldInit() 应返回 false 当 UseRabbitMQ=false")
	}
}

// TestRabbitMQService_ShouldInit_DefaultConfig 测试默认配置时不应初始化
// 不需要 RabbitMQ 连接：仅验证条件判断逻辑
func TestRabbitMQService_ShouldInit_DefaultConfig(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	cfg := &config.BaseConfig{}

	if service.ShouldInit(cfg) {
		t.Error("ShouldInit() 应返回 false 当使用默认配置")
	}
}

// ==================== 单元测试：Init 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务初始化逻辑（会因无法连接而失败，但不应返回错误）

// TestRabbitMQService_Init_EmptyLists 测试空列表时的初始化
// 不需要 RabbitMQ 连接：仅验证空列表处理逻辑
func TestRabbitMQService_Init_EmptyLists(t *testing.T) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	service := NewRabbitMQService(nil, nil)

	ctx := context.Background()
	err := service.Init(ctx)

	if err != nil {
		t.Errorf("Init() 不应返回错误，实际返回 %v", err)
	}
}

// TestRabbitMQService_Init_WithProducers 测试包含生产者时的初始化
// 不需要 RabbitMQ 连接：验证初始化逻辑（会因无法连接而失败，但不应返回错误）
func TestRabbitMQService_Init_WithProducers(t *testing.T) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	producers := []*config.MessageQueue{
		{
			QueueName:    "test-producer",
			ExchangeName: "test-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test-key",
		},
	}

	service := NewRabbitMQService(nil, producers)

	ctx := context.Background()
	err := service.Init(ctx)

	if err != nil {
		t.Errorf("Init() 不应返回错误，实际返回 %v", err)
	}

	// 由于无法真正连接 RabbitMQ，生产者可能初始化失败
	// 这里只测试不返回错误
}

// TestRabbitMQService_Init_WithConsumers 测试包含消费者时的初始化
// 不需要 RabbitMQ 连接：验证初始化逻辑（消费者在协程中启动，不会阻塞）
func TestRabbitMQService_Init_WithConsumers(t *testing.T) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	consumers := []*config.MessageQueue{
		{
			QueueName:    "test-consumer",
			ExchangeName: "test-exchange",
			ExchangeType: "direct",
			RoutingKey:   "test-key",
			FunWithCtx: func(ctx context.Context, msg string) error {
				return nil
			},
		},
	}

	service := NewRabbitMQService(consumers, nil)

	ctx := context.Background()
	err := service.Init(ctx)

	if err != nil {
		t.Errorf("Init() 不应返回错误，实际返回 %v", err)
	}

	// 消费者在协程中启动，不会阻塞
}

// TestRabbitMQService_Init_Full 测试完整初始化（消费者和生产者）
// 不需要 RabbitMQ 连接：验证初始化逻辑（会因无法连接而失败，但不应返回错误）
func TestRabbitMQService_Init_Full(t *testing.T) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	consumers := []*config.MessageQueue{
		{
			QueueName:    "consumer-queue",
			ExchangeName: "consumer-exchange",
			ExchangeType: "direct",
			RoutingKey:   "consumer-key",
			FunWithCtx: func(ctx context.Context, msg string) error {
				return nil
			},
		},
	}

	producers := []*config.MessageQueue{
		{
			QueueName:    "producer-queue",
			ExchangeName: "producer-exchange",
			ExchangeType: "direct",
			RoutingKey:   "producer-key",
		},
	}

	service := NewRabbitMQService(consumers, producers)

	ctx := context.Background()
	err := service.Init(ctx)

	if err != nil {
		t.Errorf("Init() 不应返回错误，实际返回 %v", err)
	}
}

// ==================== 单元测试：Close 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务关闭逻辑

// TestRabbitMQService_Close_EmptyProducerList 测试空生产者列表时的关闭
// 不需要 RabbitMQ 连接：仅验证关闭逻辑
func TestRabbitMQService_Close_EmptyProducerList(t *testing.T) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	service := NewRabbitMQService(nil, nil)

	ctx := context.Background()
	err := service.Close(ctx)

	if err != nil {
		t.Errorf("Close() 不应返回错误，实际返回 %v", err)
	}
}

// TestRabbitMQService_Close_WithProducers 测试包含生产者时的关闭
// 不需要 RabbitMQ 连接：仅验证关闭逻辑
func TestRabbitMQService_Close_WithProducers(t *testing.T) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	// 模拟添加生产者到全局列表
	producer := &config.MessageQueue{
		QueueName:    "close-test-queue",
		ExchangeName: "close-test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "close-test-key",
	}
	app.RabbitMQProducerList["test-producer"] = producer

	service := NewRabbitMQService(nil, nil)

	ctx := context.Background()
	err := service.Close(ctx)

	if err != nil {
		t.Errorf("Close() 不应返回错误，实际返回 %v", err)
	}
}

// ==================== 单元测试：SetConsumerList 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证设置消费者列表逻辑

// TestRabbitMQService_SetConsumerList 测试设置消费者列表
// 不需要 RabbitMQ 连接：仅验证列表设置逻辑
func TestRabbitMQService_SetConsumerList(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	if service.consumerList != nil {
		t.Error("初始 consumerList 应为 nil")
	}

	consumers := []*config.MessageQueue{
		{QueueName: "queue1"},
		{QueueName: "queue2"},
		{QueueName: "queue3"},
	}

	service.SetConsumerList(consumers)

	if len(service.consumerList) != 3 {
		t.Errorf("设置后 consumerList 长度应为 3，实际为 %d", len(service.consumerList))
	}
}

// TestRabbitMQService_SetConsumerList_Replace 测试替换消费者列表
// 不需要 RabbitMQ 连接：仅验证列表替换逻辑
func TestRabbitMQService_SetConsumerList_Replace(t *testing.T) {
	initial := []*config.MessageQueue{
		{QueueName: "initial-queue"},
	}

	service := NewRabbitMQService(initial, nil)

	if len(service.consumerList) != 1 {
		t.Errorf("初始 consumerList 长度应为 1，实际为 %d", len(service.consumerList))
	}

	newConsumers := []*config.MessageQueue{
		{QueueName: "new-queue1"},
		{QueueName: "new-queue2"},
	}

	service.SetConsumerList(newConsumers)

	if len(service.consumerList) != 2 {
		t.Errorf("替换后 consumerList 长度应为 2，实际为 %d", len(service.consumerList))
	}

	if service.consumerList[0].QueueName != "new-queue1" {
		t.Errorf("第一个消费者队列名应为 'new-queue1'，实际为 %s", service.consumerList[0].QueueName)
	}
}

// ==================== 单元测试：SetProducerList 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证设置生产者列表逻辑

// TestRabbitMQService_SetProducerList 测试设置生产者列表
// 不需要 RabbitMQ 连接：仅验证列表设置逻辑
func TestRabbitMQService_SetProducerList(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	if service.producerList != nil {
		t.Error("初始 producerList 应为 nil")
	}

	producers := []*config.MessageQueue{
		{QueueName: "producer1"},
		{QueueName: "producer2"},
	}

	service.SetProducerList(producers)

	if len(service.producerList) != 2 {
		t.Errorf("设置后 producerList 长度应为 2，实际为 %d", len(service.producerList))
	}
}

// TestRabbitMQService_SetProducerList_Replace 测试替换生产者列表
// 不需要 RabbitMQ 连接：仅验证列表替换逻辑
func TestRabbitMQService_SetProducerList_Replace(t *testing.T) {
	initial := []*config.MessageQueue{
		{QueueName: "initial-producer"},
	}

	service := NewRabbitMQService(nil, initial)

	if len(service.producerList) != 1 {
		t.Errorf("初始 producerList 长度应为 1，实际为 %d", len(service.producerList))
	}

	newProducers := []*config.MessageQueue{
		{QueueName: "new-producer1"},
		{QueueName: "new-producer2"},
		{QueueName: "new-producer3"},
	}

	service.SetProducerList(newProducers)

	if len(service.producerList) != 3 {
		t.Errorf("替换后 producerList 长度应为 3，实际为 %d", len(service.producerList))
	}
}

// ==================== 单元测试：服务接口完整性（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务实现了所有必需的接口方法

// TestRabbitMQService_ImplementsServiceInterface 测试服务实现了完整的服务接口
// 不需要 RabbitMQ 连接：仅验证接口实现
func TestRabbitMQService_ImplementsServiceInterface(t *testing.T) {
	service := NewRabbitMQService(nil, nil)

	// 验证所有必需的方法都存在
	_ = service.Name()
	_ = service.Priority()
	_ = service.Dependencies()
	_ = service.ShouldInit(&config.BaseConfig{})
	_ = service.Init(context.Background())
	_ = service.Close(context.Background())
}

// ==================== 单元测试：基准测试（不需要 RabbitMQ 连接） ====================
// 测试点：验证服务方法的性能

// BenchmarkNewRabbitMQService 基准测试创建服务的性能
// 不需要 RabbitMQ 连接：仅测试实例化性能
func BenchmarkNewRabbitMQService(b *testing.B) {
	consumers := []*config.MessageQueue{
		{QueueName: "consumer1"},
		{QueueName: "consumer2"},
	}
	producers := []*config.MessageQueue{
		{QueueName: "producer1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewRabbitMQService(consumers, producers)
	}
}

// BenchmarkRabbitMQService_Name 基准测试获取服务名称的性能
// 不需要 RabbitMQ 连接：仅测试方法调用性能
func BenchmarkRabbitMQService_Name(b *testing.B) {
	service := NewRabbitMQService(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Name()
	}
}

// BenchmarkRabbitMQService_Priority 基准测试获取服务优先级的性能
// 不需要 RabbitMQ 连接：仅测试方法调用性能
func BenchmarkRabbitMQService_Priority(b *testing.B) {
	service := NewRabbitMQService(nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Priority()
	}
}

// BenchmarkRabbitMQService_ShouldInit 基准测试判断是否应初始化的性能
// 不需要 RabbitMQ 连接：仅测试条件判断性能
func BenchmarkRabbitMQService_ShouldInit(b *testing.B) {
	service := NewRabbitMQService(nil, nil)
	cfg := &config.BaseConfig{
		System: config.SystemInfo{
			UseRabbitMQ: true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.ShouldInit(cfg)
	}
}

// BenchmarkRabbitMQService_Init_Empty 基准测试空列表初始化的性能
// 不需要 RabbitMQ 连接：仅测试初始化逻辑性能
func BenchmarkRabbitMQService_Init_Empty(b *testing.B) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	service := NewRabbitMQService(nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Init(ctx)
	}
}

// BenchmarkRabbitMQService_Close_Empty 基准测试空列表关闭的性能
// 不需要 RabbitMQ 连接：仅测试关闭逻辑性能
func BenchmarkRabbitMQService_Close_Empty(b *testing.B) {
	cleanup := setupRabbitMQServiceTestConfig()
	defer cleanup()

	service := NewRabbitMQService(nil, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.Close(ctx)
	}
}
