//go:build integration
// +build integration

// ==================== 集成测试文件（需要 RabbitMQ 连接） ====================
//
// 本文件中的所有测试都是集成测试，需要真实的 RabbitMQ 连接。
// 如果 RabbitMQ 连接失败，测试将直接失败（而非跳过）。
//
// 运行方式: go test -tags=integration -v ./app/...
//
// 请确保在运行测试前：
// 1. RabbitMQ 服务已启动
// 2. 下方的连接配置（Host/Port/Username/Password）正确

package app

import (
	"context"
	"encoding/json"
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
	integrationTestRabbitMQHost     = "localhost"
	integrationTestRabbitMQPort     = 5672
	integrationTestRabbitMQUsername = "guest"
	integrationTestRabbitMQPassword = "rabbitMq10.160.23.43"
)

// getIntegrationTestRabbitMQUrl 获取集成测试用的 RabbitMQ URL（硬编码配置）
func getIntegrationTestRabbitMQUrl() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", integrationTestRabbitMQUsername, integrationTestRabbitMQPassword, integrationTestRabbitMQHost, integrationTestRabbitMQPort)
}

// requireRabbitMQConnection 验证 RabbitMQ 连接，连接失败则测试失败
func requireRabbitMQConnection(t *testing.T) {
	url := getIntegrationTestRabbitMQUrl()

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

// setupIntegrationTestConfig 设置集成测试配置
func setupIntegrationTestConfig() func() {
	originalConfig := BaseConfig

	// 设置测试配置（硬编码）
	BaseConfig = config.BaseConfig{
		RabbitMQ: config.RabbitMQInfo{
			Host:     integrationTestRabbitMQHost,
			Port:     integrationTestRabbitMQPort,
			Username: integrationTestRabbitMQUsername,
			Password: integrationTestRabbitMQPassword,
		},
		RabbitMQList: config.RabbitMqListInfo{
			{AliasName: "primary", Host: integrationTestRabbitMQHost, Port: integrationTestRabbitMQPort, Username: integrationTestRabbitMQUsername, Password: integrationTestRabbitMQPassword},
		},
	}

	// 清空生产者列表（使用 sync.Map）
	clearIntegrationRabbitMQProducerList()

	return func() {
		BaseConfig = originalConfig
		// 关闭所有生产者连接并清空列表
		RabbitMQProducerList.Range(func(key, value any) bool {
			producer := value.(*config.MessageQueue)
			producer.Close()
			RabbitMQProducerList.Delete(key)
			return true
		})
	}
}

// clearIntegrationRabbitMQProducerList 清空生产者列表（用于集成测试）
func clearIntegrationRabbitMQProducerList() {
	RabbitMQProducerList.Range(func(key, value any) bool {
		RabbitMQProducerList.Delete(key)
		return true
	})
}

// generateQueueName 生成唯一的队列名称
func generateQueueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// ==================== 集成测试：SendRabbitMqMsg 消息发送（需要 RabbitMQ 连接） ====================
// 测试点：验证 SendRabbitMqMsg 函数的消息发送功能

// TestIntegration_SendRabbitMqMsg 测试普通消息发送
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_SendRabbitMqMsg(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-send")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"
	testMessage := "Integration test message " + time.Now().String()

	err := SendRabbitMqMsg(queueName, exchangeName, "direct", routingKey, testMessage)
	if err != nil {
		t.Fatalf("SendRabbitMqMsg 失败: %v", err)
	}

	t.Logf("消息发送成功: %s", testMessage)
}

// TestIntegration_SendRabbitMqMsg_MultipleInstances 测试使用多实例配置发送消息
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_SendRabbitMqMsg_MultipleInstances(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-multi")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"
	testMessage := "Multi-instance test message"

	// 使用默认配置和命名配置
	err := SendRabbitMqMsg(queueName, exchangeName, "direct", routingKey, testMessage, "", "primary")
	if err != nil {
		t.Fatalf("多实例发送失败: %v", err)
	}

	t.Log("多实例消息发送成功")
}

// ==================== 集成测试：SendRabbitMqMsgBatch 批量发送（需要 RabbitMQ 连接） ====================
// 测试点：验证批量消息发送功能

// TestIntegration_SendRabbitMqMsgBatch 测试批量消息发送
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_SendRabbitMqMsgBatch(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-batch")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"

	messages := make([]string, 100)
	for i := range messages {
		messages[i] = fmt.Sprintf("Batch message %d", i)
	}

	err := SendRabbitMqMsgBatch(queueName, exchangeName, "direct", routingKey, messages)
	if err != nil {
		t.Fatalf("批量发送失败: %v", err)
	}

	t.Logf("批量发送成功，消息数量: %d", len(messages))
}

// TestIntegration_SendRabbitMqMsgBatchWithContext 测试带 Context 的批量消息发送
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_SendRabbitMqMsgBatchWithContext(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-batch-ctx")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages := make([]string, 50)
	for i := range messages {
		messages[i] = fmt.Sprintf("Context batch message %d", i)
	}

	err := SendRabbitMqMsgBatchWithContext(ctx, queueName, exchangeName, "direct", routingKey, messages)
	if err != nil {
		t.Fatalf("使用 context 批量发送失败: %v", err)
	}

	t.Logf("使用 context 批量发送成功，消息数量: %d", len(messages))
}

// TestIntegration_SendRabbitMqMsgBatchWithContext_Cancelled 测试已取消的 Context 应快速返回
// 需要 RabbitMQ 连接：需要建立连接才能验证取消行为
func TestIntegration_SendRabbitMqMsgBatchWithContext_Cancelled(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-batch-cancel")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"

	// 创建已取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	messages := make([]string, 1000)
	for i := range messages {
		messages[i] = fmt.Sprintf("Cancelled message %d", i)
	}

	start := time.Now()
	_ = SendRabbitMqMsgBatchWithContext(ctx, queueName, exchangeName, "direct", routingKey, messages)
	elapsed := time.Since(start)

	// 应该快速返回
	if elapsed > 5*time.Second {
		t.Errorf("已取消的 context 应该快速返回，实际耗时: %v", elapsed)
	}

	t.Logf("已取消的 context 处理耗时: %v", elapsed)
}

// ==================== 集成测试：SendRabbitMqMsgWithConfirm 确认模式发送（需要 RabbitMQ 连接） ====================
// 测试点：验证确认模式消息发送功能

// TestIntegration_SendRabbitMqMsgWithConfirm 测试确认模式消息发送
// 需要 RabbitMQ 连接：涉及真实的消息发送和确认
func TestIntegration_SendRabbitMqMsgWithConfirm(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-confirm")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"
	testMessage := "Confirmed message " + time.Now().String()

	err := SendRabbitMqMsgWithConfirm(queueName, exchangeName, "direct", routingKey, testMessage, 5*time.Second)
	if err != nil {
		t.Fatalf("确认模式发送失败: %v", err)
	}

	t.Log("确认模式消息发送成功")
}

// ==================== 集成测试：端到端消息收发（需要 RabbitMQ 连接） ====================
// 测试点：验证消息发送和消费的完整流程

// TestIntegration_EndToEnd_SendAndConsume 测试端到端消息收发
// 需要 RabbitMQ 连接：涉及真实的消息发送和消费
func TestIntegration_EndToEnd_SendAndConsume(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-e2e")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"

	// 消息计数
	var receivedCount int32
	receivedMessages := make(chan string, 10)

	// 消费者
	consumer := config.MessageQueue{
		QueueName:    queueName,
		ExchangeName: exchangeName,
		ExchangeType: "direct",
		RoutingKey:   routingKey,
		MqConnStr:    BaseConfig.RabbitMQ.Url(),
		FunWithCtx: func(ctx context.Context, msg string) error {
			atomic.AddInt32(&receivedCount, 1)
			receivedMessages <- msg
			return nil
		},
	}
	defer consumer.Close()

	// 启动消费者
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() {
		_ = consumer.ConsumeWithContext(ctx)
	}()

	// 等待消费者启动
	time.Sleep(1 * time.Second)

	// 发送消息
	testMessages := []string{
		"E2E message 1",
		"E2E message 2",
		"E2E message 3",
	}

	for _, msg := range testMessages {
		err := SendRabbitMqMsg(queueName, exchangeName, "direct", routingKey, msg)
		if err != nil {
			t.Fatalf("发送消息失败: %v", err)
		}
	}

	// 等待所有消息被消费
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&receivedCount) >= int32(len(testMessages)) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	count := atomic.LoadInt32(&receivedCount)
	if count != int32(len(testMessages)) {
		t.Errorf("接收到的消息数量不匹配: got %d, want %d", count, len(testMessages))
	} else {
		t.Logf("端到端测试成功：发送 %d 条，接收 %d 条", len(testMessages), count)
	}
}

// ==================== 集成测试：JSON 消息端到端（需要 RabbitMQ 连接） ====================
// 测试点：验证 JSON 格式消息的发送和解析

type OrderMessage struct {
	OrderID   string  `json:"orderId"`
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	Timestamp int64   `json:"timestamp"`
}

// TestIntegration_EndToEnd_JSONMessage 测试 JSON 格式消息的端到端收发
// 需要 RabbitMQ 连接：涉及真实的消息发送和消费
func TestIntegration_EndToEnd_JSONMessage(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-json-e2e")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"

	receivedOrders := make(chan OrderMessage, 1)

	// 消费者
	consumer := config.MessageQueue{
		QueueName:    queueName,
		ExchangeName: exchangeName,
		ExchangeType: "direct",
		RoutingKey:   routingKey,
		MqConnStr:    BaseConfig.RabbitMQ.Url(),
		FunWithCtx: func(ctx context.Context, msg string) error {
			var order OrderMessage
			if err := json.Unmarshal([]byte(msg), &order); err != nil {
				return err
			}
			receivedOrders <- order
			return nil
		},
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		_ = consumer.ConsumeWithContext(ctx)
	}()

	time.Sleep(1 * time.Second)

	// 发送订单消息
	order := OrderMessage{
		OrderID:   "ORD-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		UserID:    "USER-001",
		Amount:    199.99,
		Timestamp: time.Now().Unix(),
	}

	orderBytes, _ := json.Marshal(order)
	err := SendRabbitMqMsg(queueName, exchangeName, "direct", routingKey, string(orderBytes))
	if err != nil {
		t.Fatalf("发送订单消息失败: %v", err)
	}

	select {
	case received := <-receivedOrders:
		if received.OrderID != order.OrderID {
			t.Errorf("OrderID 不匹配: got %s, want %s", received.OrderID, order.OrderID)
		}
		if received.Amount != order.Amount {
			t.Errorf("Amount 不匹配: got %f, want %f", received.Amount, order.Amount)
		}
		t.Logf("JSON 端到端测试成功: %+v", received)
	case <-time.After(5 * time.Second):
		t.Fatal("接收订单消息超时")
	}
}

// ==================== 集成测试：并发发送（需要 RabbitMQ 连接） ====================
// 测试点：验证并发发送消息的正确性和线程安全性

// TestIntegration_ConcurrentSend 测试并发发送消息
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_ConcurrentSend(t *testing.T) {
	requireRabbitMQConnection(t)
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	queueName := generateQueueName("test-concurrent-send")
	exchangeName := queueName + "-exchange"
	routingKey := queueName + "-key"

	numGoroutines := 10
	messagesPerGoroutine := 10

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := fmt.Sprintf("Concurrent message from goroutine %d, msg %d", id, j)
				err := SendRabbitMqMsg(queueName, exchangeName, "direct", routingKey, msg)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	total := numGoroutines * messagesPerGoroutine
	success := atomic.LoadInt32(&successCount)
	errors := atomic.LoadInt32(&errorCount)

	t.Logf("并发发送完成: 成功 %d, 失败 %d, 总计 %d", success, errors, total)

	if success != int32(total) {
		t.Errorf("部分消息发送失败: 成功 %d/%d", success, total)
	}
}

// ==================== 集成测试：基准测试（需要 RabbitMQ 连接） ====================
// 测试点：验证消息发送的性能

// BenchmarkIntegration_SendRabbitMqMsg 基准测试单条消息发送性能
// 需要 RabbitMQ 连接：涉及真实的消息发送
func BenchmarkIntegration_SendRabbitMqMsg(b *testing.B) {
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	mq := config.MessageQueue{
		QueueName:    "bench-send",
		ExchangeName: "bench-send-exchange",
		ExchangeType: "direct",
		RoutingKey:   "bench-send-key",
		MqConnStr:    BaseConfig.RabbitMQ.Url(),
	}

	err := mq.InitChannelForProducer()
	if err != nil {
		b.Fatalf("RabbitMQ 连接失败: %v", err)
	}
	defer mq.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SendRabbitMqMsg("bench-send", "bench-send-exchange", "direct", "bench-send-key", fmt.Sprintf("Bench msg %d", i))
	}
}

// BenchmarkIntegration_SendRabbitMqMsgBatch 基准测试批量消息发送性能
// 需要 RabbitMQ 连接：涉及真实的消息发送
func BenchmarkIntegration_SendRabbitMqMsgBatch(b *testing.B) {
	cleanup := setupIntegrationTestConfig()
	defer cleanup()

	mq := config.MessageQueue{
		QueueName:    "bench-batch-send",
		ExchangeName: "bench-batch-send-exchange",
		ExchangeType: "direct",
		RoutingKey:   "bench-batch-send-key",
		MqConnStr:    BaseConfig.RabbitMQ.Url(),
	}

	err := mq.InitChannelForProducer()
	if err != nil {
		b.Fatalf("RabbitMQ 连接失败: %v", err)
	}
	defer mq.Close()

	messages := make([]string, 100)
	for i := range messages {
		messages[i] = fmt.Sprintf("Batch bench msg %d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SendRabbitMqMsgBatch("bench-batch-send", "bench-batch-send-exchange", "direct", "bench-batch-send-key", messages)
	}
}
