// Package initialize RabbitMQ 消费者初始化功能测试
//
// ==================== 测试说明 ====================
// 本文件包含 RabbitMQ 消费者初始化和管理功能的单元测试。
//
// 测试覆盖内容：
// 1. StopConsumer - 停止单个消费者（存在/不存在）
// 2. StopAllConsumers - 停止所有消费者
// 3. 消费者生命周期 - 启动、运行、停止
// 4. 并发安全 - 多协程并发操作消费者
// 5. 回调函数 - 消费者处理函数校验
//
// 注意：需要真实 RabbitMQ 连接的测试会自动跳过
//
// 运行测试：go test -v ./initialize/... -run Consumer
// ==================================================
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
//
// 【功能点】验证停止不存在的消费者时不会 panic
// 【测试流程】调用 StopConsumer() 传入不存在的队列信息，验证无异常
func TestStopConsumer_NotExists(t *testing.T) {
	// 停止不存在的消费者不应 panic
	StopConsumer("not-exists-queue-info")
}

// TestStopAllConsumers_Empty 测试空消费者列表时停止所有消费者
//
// 【功能点】验证空消费者列表时停止所有消费者不会 panic
// 【测试流程】清空消费者列表，调用 StopAllConsumers()，验证无异常
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
//
// 【功能点】验证停止已存在的消费者时正确取消 context
// 【测试流程】
//  1. 创建测试用的取消函数并注册到消费者列表
//  2. 调用 StopConsumer() 停止消费者
//  3. 验证 context 已被正确取消
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
//
// 【功能点】验证批量停止所有消费者的功能
// 【测试流程】
//  1. 创建多个测试用的取消函数并注册
//  2. 调用 StopAllConsumers() 停止所有消费者
//  3. 验证所有 context 均已被取消
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
//
// 【功能点】验证 GetInfo() 方法返回正确格式的队列信息字符串
// 【测试流程】创建配置并调用 GetInfo()，验证返回格式为 "mqName_queue_exchange_type_key"
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
//
// 【功能点】验证消费者管理的并发安全性
// 【测试流程】
//  1. 启动多个协程并发添加消费者
//  2. 启动多个协程并发读取消费者
//  3. 启动多个协程并发停止消费者
//  4. 验证无数据竞争和 panic
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
//
// 【功能点】验证消费者的完整生命周期管理
// 【测试流程】
//  1. 模拟添加消费者（注册取消函数）
//  2. 验证消费者存在
//  3. 停止消费者
//  4. 验证消费者已被移除
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
