package initialize

import (
	"context"
	"sync"
	"testing"

	"github.com/zzsen/gin_core/model/config"
)

// ==================== 单元测试：消费者管理（不需要 RabbitMQ 连接） ====================
// 测试点：验证消费者的启动、停止和管理逻辑

// TestStopConsumer_NotExists 测试停止不存在的消费者不应 panic
// 不需要 RabbitMQ 连接：仅验证消费者管理逻辑
func TestStopConsumer_NotExists(t *testing.T) {
	// 停止不存在的消费者不应 panic
	StopConsumer("not-exists-queue-info")
}

// TestStopAllConsumers_Empty 测试空消费者列表时停止所有消费者
// 不需要 RabbitMQ 连接：仅验证消费者管理逻辑
func TestStopAllConsumers_Empty(t *testing.T) {
	// 清空现有的取消函数
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelFuncs = make(map[string]context.CancelFunc)
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	// 空列表不应 panic
	StopAllConsumers()
}

// TestStopConsumer_Exists 测试停止已存在的消费者
// 不需要 RabbitMQ 连接：仅验证消费者停止逻辑
func TestStopConsumer_Exists(t *testing.T) {
	// 备份原始数据
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelFuncs = make(map[string]context.CancelFunc)
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	// 创建一个测试用的取消函数
	ctx, cancel := context.WithCancel(context.Background())
	queueInfo := "test_queue_exchange_direct_key"

	consumerCancelLock.Lock()
	consumerCancelFuncs[queueInfo] = cancel
	consumerCancelLock.Unlock()

	// 验证消费者存在
	consumerCancelLock.Lock()
	_, exists := consumerCancelFuncs[queueInfo]
	consumerCancelLock.Unlock()

	if !exists {
		t.Error("消费者应该存在")
	}

	// 停止消费者
	StopConsumer(queueInfo)

	// 验证 context 已被取消
	select {
	case <-ctx.Done():
		// 正确取消
	default:
		t.Error("context 应该已被取消")
	}
}

// TestStopAllConsumers 测试停止所有消费者
// 不需要 RabbitMQ 连接：仅验证消费者批量停止逻辑
func TestStopAllConsumers(t *testing.T) {
	// 备份原始数据
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelFuncs = make(map[string]context.CancelFunc)
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	// 创建多个测试用的取消函数
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	ctx3, cancel3 := context.WithCancel(context.Background())

	consumerCancelLock.Lock()
	consumerCancelFuncs["queue1"] = cancel1
	consumerCancelFuncs["queue2"] = cancel2
	consumerCancelFuncs["queue3"] = cancel3
	consumerCancelLock.Unlock()

	// 停止所有消费者
	StopAllConsumers()

	// 验证所有 context 已被取消
	contexts := []context.Context{ctx1, ctx2, ctx3}
	for i, ctx := range contexts {
		select {
		case <-ctx.Done():
			// 正确取消
		default:
			t.Errorf("context %d 应该已被取消", i+1)
		}
	}
}

// ==================== 单元测试：MessageQueue 配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证 MessageQueue 配置的 GetInfo 方法

// TestMessageQueue_GetInfo 测试消息队列配置的 GetInfo 方法
// 不需要 RabbitMQ 连接：仅验证配置格式化逻辑
func TestMessageQueue_GetInfo(t *testing.T) {
	mq := config.MessageQueue{
		MQName:       "test-mq",
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	expected := "test-mq_test-queue_test-exchange_direct_test-key"
	if mq.GetInfo() != expected {
		t.Errorf("GetInfo() = %v, want %v", mq.GetInfo(), expected)
	}
}

// ==================== 单元测试：并发安全（不需要 RabbitMQ 连接） ====================
// 测试点：验证消费者管理的并发安全性

// TestConsumerCancelFuncs_ConcurrentAccess 测试消费者取消函数的并发访问安全性
// 不需要 RabbitMQ 连接：仅验证锁机制
func TestConsumerCancelFuncs_ConcurrentAccess(t *testing.T) {
	// 备份原始数据
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelFuncs = make(map[string]context.CancelFunc)
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	var wg sync.WaitGroup
	numGoroutines := 100

	// 并发添加
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, cancel := context.WithCancel(context.Background())
			queueInfo := "queue_" + string(rune('A'+idx%26))

			consumerCancelLock.Lock()
			consumerCancelFuncs[queueInfo] = cancel
			consumerCancelLock.Unlock()
		}(i)
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			queueInfo := "queue_" + string(rune('A'+idx%26))

			consumerCancelLock.Lock()
			_ = consumerCancelFuncs[queueInfo]
			consumerCancelLock.Unlock()
		}(i)
	}

	// 并发停止
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			queueInfo := "queue_" + string(rune('A'+idx%26))
			StopConsumer(queueInfo)
		}(i)
	}

	wg.Wait()
}

// ==================== 单元测试：消费者生命周期（不需要 RabbitMQ 连接） ====================
// 测试点：验证消费者的完整生命周期管理
// 注意：InitialRabbitMq 和 InitialRabbitMqWithContext 的测试需要 RabbitMQ 连接，
// 这些是集成测试，在此省略以避免测试阻塞

// TestConsumerLifecycle 测试消费者的创建和停止生命周期
// 不需要 RabbitMQ 连接：仅验证生命周期管理逻辑
func TestConsumerLifecycle(t *testing.T) {
	// 备份原始数据
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelFuncs = make(map[string]context.CancelFunc)
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	// 模拟添加消费者
	ctx, cancel := context.WithCancel(context.Background())
	queueInfo := "lifecycle_test_queue"

	consumerCancelLock.Lock()
	consumerCancelFuncs[queueInfo] = cancel
	consumerCancelLock.Unlock()

	// 验证消费者存在
	consumerCancelLock.Lock()
	_, exists := consumerCancelFuncs[queueInfo]
	consumerCancelLock.Unlock()

	if !exists {
		t.Fatal("消费者应该存在")
	}

	// 验证 context 未取消
	select {
	case <-ctx.Done():
		t.Fatal("context 不应该已被取消")
	default:
		// 正确 - context 未取消
	}

	// 停止消费者
	StopConsumer(queueInfo)

	// 验证 context 已取消
	select {
	case <-ctx.Done():
		// 正确 - context 已取消
	default:
		t.Fatal("context 应该已被取消")
	}
}

// ==================== 单元测试：基准测试（不需要 RabbitMQ 连接） ====================
// 测试点：验证消费者管理的性能

// BenchmarkStopConsumer 基准测试停止消费者的性能
// 不需要 RabbitMQ 连接：仅测试停止逻辑性能
func BenchmarkStopConsumer(b *testing.B) {
	// 备份原始数据
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 每次迭代创建新的取消函数
		consumerCancelLock.Lock()
		consumerCancelFuncs = make(map[string]context.CancelFunc)
		_, cancel := context.WithCancel(context.Background())
		consumerCancelFuncs["bench_queue"] = cancel
		consumerCancelLock.Unlock()

		StopConsumer("bench_queue")
	}
}

// BenchmarkConcurrentAccess 基准测试并发访问消费者管理的性能
// 不需要 RabbitMQ 连接：仅测试并发访问性能
func BenchmarkConcurrentAccess(b *testing.B) {
	// 备份原始数据
	consumerCancelLock.Lock()
	originalFuncs := consumerCancelFuncs
	consumerCancelFuncs = make(map[string]context.CancelFunc)
	consumerCancelLock.Unlock()

	defer func() {
		consumerCancelLock.Lock()
		consumerCancelFuncs = originalFuncs
		consumerCancelLock.Unlock()
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			queueInfo := "queue_" + string(rune('A'+i%26))
			_, cancel := context.WithCancel(context.Background())

			consumerCancelLock.Lock()
			consumerCancelFuncs[queueInfo] = cancel
			consumerCancelLock.Unlock()

			StopConsumer(queueInfo)
			i++
		}
	})
}
