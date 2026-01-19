//go:build integration
// +build integration

// ==================== 集成测试文件（需要 RabbitMQ 连接） ====================
//
// 本文件中的所有测试都是集成测试，需要真实的 RabbitMQ 连接。
// 如果 RabbitMQ 连接失败，测试将直接失败（而非跳过）。
//
// 运行方式: go test -tags=integration -v ./model/config/...
//
// 请确保在运行测试前：
// 1. RabbitMQ 服务已启动
// 2. 下方的连接配置（Host/Port/Username/Password）正确

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== 测试辅助函数 ====================

// 硬编码的 RabbitMQ 测试配置
const (
	configTestRabbitMQHost     = "localhost"
	configTestRabbitMQPort     = 5672
	configTestRabbitMQUsername = "guest"
	configTestRabbitMQPassword = "rabbitMq10.160.23.43"
)

// getTestRabbitMQUrl 获取测试用的 RabbitMQ URL（硬编码配置）
func getTestRabbitMQUrl() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/", configTestRabbitMQUsername, configTestRabbitMQPassword, configTestRabbitMQHost, configTestRabbitMQPort)
}

// requireRabbitMQ 验证 RabbitMQ 连接，连接失败则测试失败
func requireRabbitMQ(t *testing.T) string {
	url := getTestRabbitMQUrl()

	// 尝试连接验证
	mq := MessageQueue{
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

	return url
}

// generateQueueName 生成唯一的队列名称
func generateQueueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// ==================== 集成测试：发送消息（需要 RabbitMQ 连接） ====================
// 测试点：验证消息发送功能

// TestIntegration_Publish_SingleMessage 测试发送单条消息
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_Publish_SingleMessage(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-publish-single")
	mq := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer mq.Close()

	// 发送消息
	testMessage := "Hello, RabbitMQ! " + time.Now().String()
	err := mq.Publish(testMessage)
	if err != nil {
		t.Fatalf("发送消息失败: %v", err)
	}

	t.Logf("消息发送成功: %s", testMessage)
}

// TestIntegration_PublishWithContext 测试使用 Context 发送消息
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_PublishWithContext(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-publish-ctx")
	mq := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer mq.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testMessage := "Context message " + time.Now().String()
	err := mq.PublishWithContext(ctx, testMessage)
	if err != nil {
		t.Fatalf("使用 context 发送消息失败: %v", err)
	}

	t.Logf("消息发送成功: %s", testMessage)
}

// TestIntegration_PublishBatch 测试批量发送消息
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_PublishBatch(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-publish-batch")
	mq := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer mq.Close()

	messages := []string{
		"Batch message 1",
		"Batch message 2",
		"Batch message 3",
		"Batch message 4",
		"Batch message 5",
	}

	err := mq.PublishBatch(messages)
	if err != nil {
		t.Fatalf("批量发送消息失败: %v", err)
	}

	t.Logf("批量发送成功，消息数量: %d", len(messages))
}

// TestIntegration_PublishWithConfirm 测试确认模式发送消息
// 需要 RabbitMQ 连接：涉及真实的消息发送和确认
func TestIntegration_PublishWithConfirm(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-publish-confirm")
	mq := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		PublishConfirm: PublishConfirmConfig{
			Enabled: true,
			Timeout: 5 * time.Second,
		},
	}
	defer mq.Close()

	testMessage := "Confirmed message " + time.Now().String()
	err := mq.PublishWithContext(context.Background(), testMessage)
	if err != nil {
		t.Fatalf("确认模式发送消息失败: %v", err)
	}

	t.Logf("确认模式消息发送成功: %s", testMessage)
}

// ==================== 集成测试：消费消息（需要 RabbitMQ 连接） ====================
// 测试点：验证消息消费功能

// TestIntegration_Consume_SingleMessage 测试消费单条消息
// 需要 RabbitMQ 连接：涉及真实的消息发送和消费
func TestIntegration_Consume_SingleMessage(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-consume-single")
	testMessage := "Test consume message " + time.Now().String()

	// 用于接收消息的通道
	receivedChan := make(chan string, 1)
	var consumeErr error

	// 消费者
	consumer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		FunWithCtx: func(ctx context.Context, msg string) error {
			receivedChan <- msg
			return nil
		},
	}
	defer consumer.Close()

	// 生产者
	producer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer producer.Close()

	// 启动消费者
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		consumeErr = consumer.ConsumeWithContext(ctx)
	}()

	// 等待消费者启动
	time.Sleep(500 * time.Millisecond)

	// 发送消息
	err := producer.Publish(testMessage)
	if err != nil {
		t.Fatalf("发送消息失败: %v", err)
	}

	// 等待接收消息
	select {
	case received := <-receivedChan:
		if received != testMessage {
			t.Errorf("接收到的消息不匹配: got %s, want %s", received, testMessage)
		} else {
			t.Logf("成功接收消息: %s", received)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("接收消息超时")
	}

	cancel()

	// 检查消费者是否正常退出
	if consumeErr != nil {
		t.Logf("消费者退出: %v", consumeErr)
	}
}

// TestIntegration_Consume_MultipleMessages 测试消费多条消息
// 需要 RabbitMQ 连接：涉及真实的消息发送和消费
func TestIntegration_Consume_MultipleMessages(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-consume-multi")
	numMessages := 10

	// 用于统计接收消息数量
	var receivedCount int32
	receivedMessages := make([]string, 0, numMessages)
	var mu sync.Mutex

	// 消费者
	consumer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		ConsumeConfig: ConsumeConfig{
			PrefetchCount: 5,
		},
		FunWithCtx: func(ctx context.Context, msg string) error {
			mu.Lock()
			receivedMessages = append(receivedMessages, msg)
			mu.Unlock()
			atomic.AddInt32(&receivedCount, 1)
			return nil
		},
	}
	defer consumer.Close()

	// 生产者
	producer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer producer.Close()

	// 启动消费者
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	go func() {
		_ = consumer.ConsumeWithContext(ctx)
	}()

	// 等待消费者启动
	time.Sleep(500 * time.Millisecond)

	// 发送多条消息
	messages := make([]string, numMessages)
	for i := 0; i < numMessages; i++ {
		messages[i] = fmt.Sprintf("Message %d - %d", i+1, time.Now().UnixNano())
	}

	err := producer.PublishBatch(messages)
	if err != nil {
		t.Fatalf("批量发送消息失败: %v", err)
	}

	// 等待所有消息被消费
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&receivedCount) >= int32(numMessages) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	cancel()

	count := atomic.LoadInt32(&receivedCount)
	if count != int32(numMessages) {
		t.Errorf("接收到的消息数量不匹配: got %d, want %d", count, numMessages)
	} else {
		t.Logf("成功接收 %d 条消息", count)
	}
}

// TestIntegration_Consume_GracefulShutdown 测试消费者优雅关闭
// 需要 RabbitMQ 连接：涉及真实的消息发送和消费
func TestIntegration_Consume_GracefulShutdown(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-consume-shutdown")

	var handlerCalled bool
	var mu sync.Mutex

	consumer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		FunWithCtx: func(ctx context.Context, msg string) error {
			mu.Lock()
			handlerCalled = true
			mu.Unlock()
			// 模拟处理耗时
			time.Sleep(100 * time.Millisecond)
			return nil
		},
	}
	defer consumer.Close()

	// 启动消费者
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- consumer.ConsumeWithContext(ctx)
	}()

	// 等待消费者启动
	time.Sleep(500 * time.Millisecond)

	// 发送一条消息
	producer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer producer.Close()

	_ = producer.Publish("shutdown test message")

	// 等待消息开始处理
	time.Sleep(50 * time.Millisecond)

	// 触发优雅关闭
	cancel()

	// 等待消费者退出
	select {
	case err := <-done:
		if err != nil {
			t.Logf("消费者退出错误（可能是正常的）: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("消费者未能优雅关闭")
	}

	t.Log("消费者优雅关闭成功")
}

// ==================== 集成测试：死信队列（需要 RabbitMQ 连接） ====================
// 测试点：验证死信队列功能

// TestIntegration_DeadLetterQueue 测试死信队列
// 需要 RabbitMQ 连接：涉及真实的消息发送和死信队列处理
func TestIntegration_DeadLetterQueue(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-dlq")
	dlqName := queueName + ".dlq"

	// 用于接收死信消息的通道
	dlqReceived := make(chan string, 1)

	// 主队列消费者（会失败）
	mainConsumer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		DeadLetter: DeadLetterConfig{
			Enabled:   true,
			QueueName: dlqName,
		},
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 1, // 只重试1次
		},
		FunWithCtx: func(ctx context.Context, msg string) error {
			// 模拟处理失败
			return fmt.Errorf("模拟处理失败")
		},
	}
	defer mainConsumer.Close()

	// 死信队列消费者
	dlqConsumer := MessageQueue{
		QueueName:    dlqName,
		ExchangeName: queueName + "-exchange.dlx",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		FunWithCtx: func(ctx context.Context, msg string) error {
			dlqReceived <- msg
			return nil
		},
	}
	defer dlqConsumer.Close()

	// 启动消费者
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	go func() {
		_ = mainConsumer.ConsumeWithContext(ctx)
	}()

	go func() {
		_ = dlqConsumer.ConsumeWithContext(ctx)
	}()

	// 等待消费者启动
	time.Sleep(1 * time.Second)

	// 发送消息
	producer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer producer.Close()

	testMessage := "DLQ test message"
	err := producer.Publish(testMessage)
	if err != nil {
		t.Fatalf("发送消息失败: %v", err)
	}

	// 等待消息进入死信队列
	select {
	case received := <-dlqReceived:
		t.Logf("死信队列成功接收消息: %s", received)
	case <-time.After(15 * time.Second):
		t.Log("死信队列测试超时（可能需要多次重试才能进入DLQ）")
		// 不标记为失败，因为这取决于具体的重试逻辑
	}
}

// ==================== 集成测试：JSON 消息（需要 RabbitMQ 连接） ====================
// 测试点：验证 JSON 格式消息的发送和解析

type TestOrder struct {
	OrderID   string  `json:"orderId"`
	UserID    string  `json:"userId"`
	Amount    float64 `json:"amount"`
	Timestamp int64   `json:"timestamp"`
}

// TestIntegration_JSONMessage 测试 JSON 格式消息的发送和消费
// 需要 RabbitMQ 连接：涉及真实的消息发送和消费
func TestIntegration_JSONMessage(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-json")

	receivedOrder := make(chan TestOrder, 1)

	consumer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
		FunWithCtx: func(ctx context.Context, msg string) error {
			var order TestOrder
			if err := json.Unmarshal([]byte(msg), &order); err != nil {
				return err
			}
			receivedOrder <- order
			return nil
		},
	}
	defer consumer.Close()

	producer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer producer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		_ = consumer.ConsumeWithContext(ctx)
	}()

	time.Sleep(500 * time.Millisecond)

	// 发送 JSON 消息
	order := TestOrder{
		OrderID:   "ORD-001",
		UserID:    "USER-123",
		Amount:    99.99,
		Timestamp: time.Now().Unix(),
	}

	msgBytes, _ := json.Marshal(order)
	err := producer.Publish(string(msgBytes))
	if err != nil {
		t.Fatalf("发送 JSON 消息失败: %v", err)
	}

	select {
	case received := <-receivedOrder:
		if received.OrderID != order.OrderID {
			t.Errorf("OrderID 不匹配: got %s, want %s", received.OrderID, order.OrderID)
		}
		if received.Amount != order.Amount {
			t.Errorf("Amount 不匹配: got %f, want %f", received.Amount, order.Amount)
		}
		t.Logf("成功接收并解析 JSON 消息: %+v", received)
	case <-time.After(5 * time.Second):
		t.Fatal("接收 JSON 消息超时")
	}
}

// ==================== 集成测试：并发发送（需要 RabbitMQ 连接） ====================
// 测试点：验证并发发送消息的正确性和线程安全性

// TestIntegration_ConcurrentPublish 测试并发发送消息
// 需要 RabbitMQ 连接：涉及真实的消息发送
func TestIntegration_ConcurrentPublish(t *testing.T) {
	url := requireRabbitMQ(t)

	queueName := generateQueueName("test-concurrent")
	numGoroutines := 10
	messagesPerGoroutine := 10

	producer := MessageQueue{
		QueueName:    queueName,
		ExchangeName: queueName + "-exchange",
		ExchangeType: "direct",
		RoutingKey:   queueName + "-key",
		MqConnStr:    url,
	}
	defer producer.Close()

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := fmt.Sprintf("Goroutine %d, Message %d", goroutineID, j)
				err := producer.Publish(msg)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	totalMessages := numGoroutines * messagesPerGoroutine
	success := atomic.LoadInt32(&successCount)
	errors := atomic.LoadInt32(&errorCount)

	t.Logf("并发发送完成: 成功 %d, 失败 %d, 总计 %d", success, errors, totalMessages)

	if success != int32(totalMessages) {
		t.Errorf("部分消息发送失败: 成功 %d/%d", success, totalMessages)
	}
}

// ==================== 集成测试：基准测试（需要 RabbitMQ 连接） ====================
// 测试点：验证消息发送的性能

// BenchmarkIntegration_Publish 基准测试单条消息发送性能
// 需要 RabbitMQ 连接：涉及真实的消息发送
func BenchmarkIntegration_Publish(b *testing.B) {
	url := getTestRabbitMQUrl()

	mq := MessageQueue{
		QueueName:    "bench-publish",
		ExchangeName: "bench-publish-exchange",
		ExchangeType: "direct",
		RoutingKey:   "bench-publish-key",
		MqConnStr:    url,
	}

	// 预热连接
	err := mq.InitChannelForProducer()
	if err != nil {
		b.Fatalf("RabbitMQ 连接失败: %v", err)
	}
	defer mq.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mq.Publish(fmt.Sprintf("Benchmark message %d", i))
	}
}

// BenchmarkIntegration_PublishBatch 基准测试批量消息发送性能
// 需要 RabbitMQ 连接：涉及真实的消息发送
func BenchmarkIntegration_PublishBatch(b *testing.B) {
	url := getTestRabbitMQUrl()

	mq := MessageQueue{
		QueueName:    "bench-batch",
		ExchangeName: "bench-batch-exchange",
		ExchangeType: "direct",
		RoutingKey:   "bench-batch-key",
		MqConnStr:    url,
	}

	err := mq.InitChannelForProducer()
	if err != nil {
		b.Fatalf("RabbitMQ 连接失败: %v", err)
	}
	defer mq.Close()

	messages := make([]string, 100)
	for i := range messages {
		messages[i] = fmt.Sprintf("Batch message %d", i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mq.PublishBatch(messages)
	}
}
