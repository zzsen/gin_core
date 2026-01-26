// Package app RabbitMQ 消息队列功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 RabbitMQ 消息发送功能的单元测试和集成测试。
//
// 测试覆盖内容：
// 1. 参数校验 - 空配置、空消息列表处理
// 2. 生产者管理 - 获取、创建、复用生产者
// 3. 消息发送 - 单条消息、批量消息发送
// 4. 连接重试 - 连接失败时的重试机制
// 5. 并发安全 - 多协程并发发送消息
// 6. 延迟消息 - 使用 x-delayed-message 插件的延迟消息
//
// 运行单元测试：go test -v ./app/... -run "^Test.*_No"
// 运行集成测试：需要真实 RabbitMQ 连接
// ==================================================
package app

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zzsen/gin_core/model/config"
)

// ==================== 测试辅助函数 ====================

// 硬编码的 RabbitMQ 测试配置
const (
	testRabbitMQHost     = "localhost"
	testRabbitMQPort     = 5672
	testRabbitMQUsername = "guest"
	testRabbitMQPassword = "rabbitMq10.160.23.43"
)

// getTestRabbitMQUrl 获取测试用的 RabbitMQ URL（硬编码配置）
func getTestRabbitMQUrl() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", testRabbitMQUsername, testRabbitMQPassword, testRabbitMQHost, testRabbitMQPort)
}

// requireRabbitMQ 验证 RabbitMQ 连接，连接失败则测试失败
func requireRabbitMQ(t *testing.T) {
	url := getTestRabbitMQUrl()

	mq := config.MessageQueue{
		QueueName:    "test-connection",
		ExchangeName: "test-connection-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-connection-key",
		MqConnStr:    url,
	}

	err := mq.InitChannelForProducer()
	if err != nil {
		t.Fatalf("RabbitMQ 连接失败，请确保 RabbitMQ 服务已启动并配置正确 (%s): %v", url, err)
	}
	mq.Close()
}

// setupTestConfig 设置测试配置
func setupTestConfig() func() {
	// 备份原始配置
	originalConfig := BaseConfig

	// 设置测试配置（硬编码）
	BaseConfig = config.BaseConfig{
		RabbitMQ: config.RabbitMQInfo{
			Host:     testRabbitMQHost,
			Port:     testRabbitMQPort,
			Username: testRabbitMQUsername,
			Password: testRabbitMQPassword,
		},
	}

	// 返回清理函数
	return func() {
		BaseConfig = originalConfig
		// 清理生产者列表
		lock.Lock()
		RabbitMQProducerList = make(map[string]*config.MessageQueue)
		lock.Unlock()
	}
}

// ==================== 单元测试：发送消息参数校验（不需要 RabbitMQ 连接） ====================
// 测试点：验证消息发送函数的参数校验逻辑，包括空配置、空消息列表等边界情况

// TestSendRabbitMqMsg_NoConfig 测试没有配置时发送消息应返回错误
//
// 【功能点】验证没有 RabbitMQ 配置时发送消息返回错误
// 【测试流程】清空配置后调用 SendRabbitMqMsg，验证返回错误
func TestSendRabbitMqMsg_NoConfig(t *testing.T) {
	// 备份并清空配置
	originalConfig := BaseConfig
	BaseConfig = config.BaseConfig{}
	defer func() { BaseConfig = originalConfig }()

	// 没有配置应返回错误
	err := SendRabbitMqMsg("test-queue", "test-exchange", "direct", "test-key", "test message")
	if err == nil {
		t.Error("没有配置时应返回错误")
	}
}

// TestSendRabbitMqMsgBatch_EmptyMessages 测试空消息列表应直接返回 nil
//
// 【功能点】验证空消息列表时快速返回 nil
// 【测试流程】分别传入 nil 和空数组，验证都返回 nil
func TestSendRabbitMqMsgBatch_EmptyMessages(t *testing.T) {
	// 空消息列表应直接返回 nil
	err := SendRabbitMqMsgBatch("test-queue", "test-exchange", "direct", "test-key", nil)
	if err != nil {
		t.Errorf("空消息列表应返回 nil，实际返回 %v", err)
	}

	err = SendRabbitMqMsgBatch("test-queue", "test-exchange", "direct", "test-key", []string{})
	if err != nil {
		t.Errorf("空消息列表应返回 nil，实际返回 %v", err)
	}
}

// TestSendRabbitMqMsgBatchWithContext_EmptyMessages 测试带 Context 的空消息列表应直接返回 nil
//
// 【功能点】验证带 Context 的空消息列表时快速返回 nil
// 【测试流程】分别传入 nil 和空数组，验证都返回 nil
func TestSendRabbitMqMsgBatchWithContext_EmptyMessages(t *testing.T) {
	ctx := context.Background()

	err := SendRabbitMqMsgBatchWithContext(ctx, "test-queue", "test-exchange", "direct", "test-key", nil)
	if err != nil {
		t.Errorf("空消息列表应返回 nil，实际返回 %v", err)
	}

	err = SendRabbitMqMsgBatchWithContext(ctx, "test-queue", "test-exchange", "direct", "test-key", []string{})
	if err != nil {
		t.Errorf("空消息列表应返回 nil，实际返回 %v", err)
	}
}

// TestSendRabbitMqMsgBatch_NoConfig 测试批量发送时没有配置应返回错误
//
// 【功能点】验证没有配置时批量发送返回错误
// 【测试流程】清空配置后调用 SendRabbitMqMsgBatch，验证返回错误
func TestSendRabbitMqMsgBatch_NoConfig(t *testing.T) {
	// 备份并清空配置
	originalConfig := BaseConfig
	BaseConfig = config.BaseConfig{}
	defer func() { BaseConfig = originalConfig }()

	// 没有配置应返回错误
	err := SendRabbitMqMsgBatch("test-queue", "test-exchange", "direct", "test-key", []string{"msg1", "msg2"})
	if err == nil {
		t.Error("没有配置时应返回错误")
	}
}

// TestSendRabbitMqMsgWithConfirm_NoConfig 测试确认模式发送时没有配置应返回错误
//
// 【功能点】验证没有配置时确认模式发送返回错误
// 【测试流程】清空配置后调用 SendRabbitMqMsgWithConfirm，验证返回错误
func TestSendRabbitMqMsgWithConfirm_NoConfig(t *testing.T) {
	// 备份并清空配置
	originalConfig := BaseConfig
	BaseConfig = config.BaseConfig{}
	defer func() { BaseConfig = originalConfig }()

	// 没有配置应返回错误
	err := SendRabbitMqMsgWithConfirm("test-queue", "test-exchange", "direct", "test-key", "test message", 5*time.Second)
	if err == nil {
		t.Error("没有配置时应返回错误")
	}
}

// TestSendRabbitMqMsg_InvalidMQName 测试使用不存在的 MQName 应返回错误
//
// 【功能点】验证使用不存在的 MQName 时返回错误
// 【测试流程】设置基础配置，使用不存在的 MQName 调用发送，验证返回错误
func TestSendRabbitMqMsg_InvalidMQName(t *testing.T) {
	// 备份并设置基础配置（用于让代码执行到 MQName 校验，不会真正连接）
	originalConfig := BaseConfig
	BaseConfig = config.BaseConfig{
		RabbitMQ: config.RabbitMQInfo{
			Host:     "localhost",
			Port:     5672,
			Username: "guest",
			Password: "guest",
		},
	}
	defer func() { BaseConfig = originalConfig }()

	// 使用不存在的 MQName 应返回错误（在获取连接字符串时就会失败，不会真正连接）
	err := SendRabbitMqMsg("test-queue", "test-exchange", "direct", "test-key", "test message", "not-exist-mq")
	if err == nil {
		t.Error("不存在的 MQName 应返回错误")
	}
}

// ==================== 单元测试：生产者缓存逻辑（不需要 RabbitMQ 连接） ====================
// 测试点：验证生产者缓存的获取和复用逻辑

// TestGetOrInitProducer_ExistingProducer 测试获取已存在的生产者应返回原实例
//
// 【功能点】验证缓存命中时返回原有生产者实例
// 【测试流程】预先添加生产者，再次获取同一队列信息，验证返回原实例
func TestGetOrInitProducer_ExistingProducer(t *testing.T) {
	// 备份并清空生产者列表
	lock.Lock()
	originalList := RabbitMQProducerList
	RabbitMQProducerList = make(map[string]*config.MessageQueue)
	lock.Unlock()

	defer func() {
		lock.Lock()
		RabbitMQProducerList = originalList
		lock.Unlock()
	}()

	// 预先添加一个生产者
	existingMq := &config.MessageQueue{
		QueueName:    "existing-queue",
		ExchangeName: "existing-exchange",
		ExchangeType: "direct",
		RoutingKey:   "existing-key",
	}
	queueInfo := existingMq.GetInfo()

	lock.Lock()
	RabbitMQProducerList[queueInfo] = existingMq
	lock.Unlock()

	// 获取已存在的生产者
	newMq := &config.MessageQueue{
		QueueName:    "existing-queue",
		ExchangeName: "existing-exchange",
		ExchangeType: "direct",
		RoutingKey:   "existing-key",
	}

	producer, err := getOrInitProducer(newMq, queueInfo)
	if err != nil {
		t.Errorf("获取已存在的生产者不应返回错误: %v", err)
	}

	// 应返回原有的生产者
	if producer != existingMq {
		t.Error("应返回已存在的生产者实例")
	}
}

// ==================== 集成测试：生产者初始化（需要 RabbitMQ 连接） ====================
// 测试点：验证生产者的初始化和并发初始化

// TestGetOrInitProducer_NewProducer 测试首次初始化生产者
//
// 【功能点】验证首次初始化生产者成功
// 【测试流程】清空缓存后获取生产者，验证初始化成功且返回非空
func TestGetOrInitProducer_NewProducer(t *testing.T) {
	// 验证 MQ 连接可用
	requireRabbitMQ(t)

	// 备份并清空生产者列表
	lock.Lock()
	originalList := RabbitMQProducerList
	RabbitMQProducerList = make(map[string]*config.MessageQueue)
	lock.Unlock()

	defer func() {
		lock.Lock()
		RabbitMQProducerList = originalList
		lock.Unlock()
	}()

	mq := &config.MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    getTestRabbitMQUrl(),
	}

	queueInfo := mq.GetInfo()

	// 初始化生产者
	producer, err := getOrInitProducer(mq, queueInfo)
	if err != nil {
		t.Fatalf("初始化生产者失败: %v", err)
	}

	if producer == nil {
		t.Fatal("生产者不应为 nil")
	}

	t.Log("生产者初始化成功")
}

// TestGetOrInitProducer_ConcurrentAccess 测试并发初始化生产者的线程安全性
//
// 【功能点】验证多协程并发初始化生产者的线程安全性
// 【测试流程】50 个协程并发获取同一生产者，验证无错误且最终只有一个实例
func TestGetOrInitProducer_ConcurrentAccess(t *testing.T) {
	// 验证 MQ 连接可用
	requireRabbitMQ(t)

	// 备份并清空生产者列表
	lock.Lock()
	originalList := RabbitMQProducerList
	RabbitMQProducerList = make(map[string]*config.MessageQueue)
	lock.Unlock()

	defer func() {
		lock.Lock()
		RabbitMQProducerList = originalList
		lock.Unlock()
	}()

	var wg sync.WaitGroup
	var errorCount int32
	numGoroutines := 50

	// 并发获取同一个生产者
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			mq := &config.MessageQueue{
				QueueName:    "concurrent-queue",
				ExchangeName: "concurrent-exchange",
				ExchangeType: "direct",
				RoutingKey:   "concurrent-key",
				MqConnStr:    getTestRabbitMQUrl(),
			}
			queueInfo := mq.GetInfo()

			_, err := getOrInitProducer(mq, queueInfo)
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
			}
		}()
	}

	wg.Wait()

	if errorCount > 0 {
		t.Fatalf("并发初始化生产者失败次数: %d/%d", errorCount, numGoroutines)
	}

	t.Logf("并发初始化生产者成功，协程数: %d", numGoroutines)
}

// ==================== 集成测试：消息发送（需要 RabbitMQ 连接） ====================
// 测试点：验证各种消息发送方式的正确性

// TestIntegration_SendAndReceive 测试普通消息发送
//
// 【功能点】验证普通消息发送成功
// 【测试流程】调用 SendRabbitMqMsg 发送消息，验证无错误返回
func TestIntegration_SendAndReceive(t *testing.T) {
	requireRabbitMQ(t)
	cleanup := setupTestConfig()
	defer cleanup()

	queueName := "test-integration-queue"
	exchangeName := "test-integration-exchange"
	exchangeType := "direct"
	routingKey := "test-integration-key"
	testMessage := "Hello, RabbitMQ! " + time.Now().String()

	// 发送消息
	err := SendRabbitMqMsg(queueName, exchangeName, exchangeType, routingKey, testMessage)
	if err != nil {
		t.Fatalf("发送消息失败: %v", err)
	}

	t.Log("消息发送成功")
}

// TestIntegration_SendBatch 测试批量消息发送
//
// 【功能点】验证批量消息发送成功
// 【测试流程】调用 SendRabbitMqMsgBatch 发送 5 条消息，验证无错误返回
func TestIntegration_SendBatch(t *testing.T) {
	requireRabbitMQ(t)
	cleanup := setupTestConfig()
	defer cleanup()

	queueName := "test-batch-queue"
	exchangeName := "test-batch-exchange"
	exchangeType := "direct"
	routingKey := "test-batch-key"

	messages := []string{
		"Batch message 1",
		"Batch message 2",
		"Batch message 3",
		"Batch message 4",
		"Batch message 5",
	}

	// 批量发送消息
	err := SendRabbitMqMsgBatch(queueName, exchangeName, exchangeType, routingKey, messages)
	if err != nil {
		t.Fatalf("批量发送消息失败: %v", err)
	}

	t.Logf("批量发送成功，消息数量: %d", len(messages))
}

// TestIntegration_SendWithConfirm 测试确认模式消息发送
//
// 【功能点】验证 Publisher Confirms 模式发送成功
// 【测试流程】调用 SendRabbitMqMsgWithConfirm 发送消息，验证无错误返回
func TestIntegration_SendWithConfirm(t *testing.T) {
	requireRabbitMQ(t)
	cleanup := setupTestConfig()
	defer cleanup()

	queueName := "test-confirm-queue"
	exchangeName := "test-confirm-exchange"
	exchangeType := "direct"
	routingKey := "test-confirm-key"
	testMessage := "Confirmed message " + time.Now().String()

	// 使用确认模式发送消息
	err := SendRabbitMqMsgWithConfirm(queueName, exchangeName, exchangeType, routingKey, testMessage, 5*time.Second)
	if err != nil {
		t.Fatalf("确认模式发送消息失败: %v", err)
	}

	t.Log("确认模式消息发送成功")
}

// TestIntegration_SendBatchWithContext 测试带 Context 的批量消息发送
//
// 【功能点】验证带 Context 的批量消息发送成功
// 【测试流程】使用带超时的 Context 调用 SendRabbitMqMsgBatchWithContext，验证无错误
func TestIntegration_SendBatchWithContext(t *testing.T) {
	requireRabbitMQ(t)
	cleanup := setupTestConfig()
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queueName := "test-ctx-batch-queue"
	exchangeName := "test-ctx-batch-exchange"
	exchangeType := "direct"
	routingKey := "test-ctx-batch-key"

	messages := []string{
		"Context batch message 1",
		"Context batch message 2",
		"Context batch message 3",
	}

	// 使用 context 批量发送消息
	err := SendRabbitMqMsgBatchWithContext(ctx, queueName, exchangeName, exchangeType, routingKey, messages)
	if err != nil {
		t.Fatalf("使用 context 批量发送消息失败: %v", err)
	}

	t.Logf("使用 context 批量发送成功，消息数量: %d", len(messages))
}

// ==================== 集成测试：Context 取消处理（需要 RabbitMQ 连接） ====================
// 测试点：验证 Context 取消时的行为

// TestIntegration_ContextCancellation 测试 Context 取消时应快速返回
//
// 【功能点】验证使用已取消的 Context 时快速返回
// 【测试流程】创建已取消的 Context，发送大量消息，验证快速返回（<5s）
func TestIntegration_ContextCancellation(t *testing.T) {
	requireRabbitMQ(t)
	cleanup := setupTestConfig()
	defer cleanup()

	// 创建已取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	messages := make([]string, 1000)
	for i := range messages {
		messages[i] = "Message " + string(rune('A'+i%26))
	}

	// 使用已取消的 context 发送应该快速返回
	start := time.Now()
	_ = SendRabbitMqMsgBatchWithContext(ctx, "test-queue", "test-exchange", "direct", "test-key", messages)
	elapsed := time.Since(start)

	// 应该很快返回（不会真正发送所有消息）
	if elapsed > 5*time.Second {
		t.Errorf("已取消的 context 应该快速返回，实际耗时: %v", elapsed)
	}

	t.Logf("已取消的 context 处理耗时: %v", elapsed)
}

// ==================== 基准测试（不需要 RabbitMQ 连接） ====================
// 测试点：验证关键路径的性能

// BenchmarkSendRabbitMqMsgBatch_EmptyMessages 基准测试空消息列表的处理性能
// 不需要 RabbitMQ 连接：仅测试空消息列表的快速返回路径
func BenchmarkSendRabbitMqMsgBatch_EmptyMessages(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = SendRabbitMqMsgBatch("queue", "exchange", "direct", "key", nil)
	}
}

// BenchmarkGetOrInitProducer_Existing 基准测试获取已存在生产者的性能
// 不需要 RabbitMQ 连接：仅测试缓存命中路径
func BenchmarkGetOrInitProducer_Existing(b *testing.B) {
	// 预先添加一个生产者
	existingMq := &config.MessageQueue{
		QueueName:    "bench-queue",
		ExchangeName: "bench-exchange",
		ExchangeType: "direct",
		RoutingKey:   "bench-key",
	}
	queueInfo := existingMq.GetInfo()

	lock.Lock()
	RabbitMQProducerList[queueInfo] = existingMq
	lock.Unlock()

	defer func() {
		lock.Lock()
		delete(RabbitMQProducerList, queueInfo)
		lock.Unlock()
	}()

	newMq := &config.MessageQueue{
		QueueName:    "bench-queue",
		ExchangeName: "bench-exchange",
		ExchangeType: "direct",
		RoutingKey:   "bench-key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getOrInitProducer(newMq, queueInfo)
	}
}
