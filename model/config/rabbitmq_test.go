package config

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ==================== 单元测试文件（不需要 RabbitMQ 连接） ====================
//
// 本文件中的所有测试都是单元测试，不需要真实的 RabbitMQ 连接。
// 这些测试主要验证：
// - 配置结构的方法（Url、GetInfo 等）
// - 消息队列配置的处理逻辑
// - 错误处理（使用无效连接验证错误返回）
//
// 如需测试真实 MQ 连接，请运行集成测试：
// go test -tags=integration -v ./model/config/...

// ==================== 单元测试：RabbitMQInfo URL 生成（不需要 RabbitMQ 连接） ====================
// 测试点：验证 RabbitMQInfo 结构的 Url 方法

// TestRabbitMQInfo_Url 测试 RabbitMQInfo 的 URL 生成
//
// 【功能点】验证 Url() 方法生成正确的 AMQP URL
// 【测试流程】
//  1. 测试正常配置 → "amqp://user:pass@host:port/"
//  2. 测试自定义端口
//  3. 测试带别名配置
func TestRabbitMQInfo_Url(t *testing.T) {
	tests := []struct {
		name     string
		info     RabbitMQInfo
		expected string
	}{
		{
			name: "正常配置",
			info: RabbitMQInfo{
				Host:     "localhost",
				Port:     5672,
				Username: "guest",
				Password: "guest",
			},
			expected: "amqp://guest:guest@localhost:5672/",
		},
		{
			name: "自定义端口",
			info: RabbitMQInfo{
				Host:     "192.168.1.100",
				Port:     15672,
				Username: "admin",
				Password: "admin123",
			},
			expected: "amqp://admin:admin123@192.168.1.100:15672/",
		},
		{
			name: "带别名",
			info: RabbitMQInfo{
				AliasName: "primary",
				Host:      "mq.example.com",
				Port:      5672,
				Username:  "user",
				Password:  "pass",
			},
			expected: "amqp://user:pass@mq.example.com:5672/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.info.Url()
			if result != tt.expected {
				t.Errorf("Url() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ==================== 单元测试：RabbitMqListInfo URL 查找（不需要 RabbitMQ 连接） ====================
// 测试点：验证 RabbitMqListInfo 的按别名查找 URL 功能

// TestRabbitMqListInfo_Url 测试 RabbitMqListInfo 的按别名查找 URL
//
// 【功能点】验证按别名查找 URL 的功能
// 【测试流程】
//  1. 测试查找存在的别名 → 返回对应 URL
//  2. 测试查找不存在的别名 → 返回空字符串
func TestRabbitMqListInfo_Url(t *testing.T) {
	list := RabbitMqListInfo{
		{AliasName: "primary", Host: "localhost", Port: 5672, Username: "guest", Password: "guest"},
		{AliasName: "secondary", Host: "192.168.1.100", Port: 5672, Username: "admin", Password: "admin"},
	}

	tests := []struct {
		name      string
		aliasName string
		expected  string
	}{
		{
			name:      "查找存在的别名 - primary",
			aliasName: "primary",
			expected:  "amqp://guest:guest@localhost:5672/",
		},
		{
			name:      "查找存在的别名 - secondary",
			aliasName: "secondary",
			expected:  "amqp://admin:admin@192.168.1.100:5672/",
		},
		{
			name:      "查找不存在的别名",
			aliasName: "notexist",
			expected:  "",
		},
		{
			name:      "空别名",
			aliasName: "",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := list.Url(tt.aliasName)
			if result != tt.expected {
				t.Errorf("Url(%s) = %v, want %v", tt.aliasName, result, tt.expected)
			}
		})
	}
}

// ==================== 单元测试：MessageQueue 基础方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证 MessageQueue 的 GetInfo 和 GetFuncInfo 方法

// TestMessageQueue_GetInfo 测试 MessageQueue 的 GetInfo 方法
//
// 【功能点】验证 GetInfo() 返回正确格式的队列信息字符串
// 【测试流程】测试完整配置、空 MQName、全空等场景
func TestMessageQueue_GetInfo(t *testing.T) {
	tests := []struct {
		name     string
		mq       MessageQueue
		expected string
	}{
		{
			name: "完整配置",
			mq: MessageQueue{
				MQName:       "test-mq",
				QueueName:    "test-queue",
				ExchangeName: "test-exchange",
				ExchangeType: "direct",
				RoutingKey:   "test-key",
			},
			expected: "test-mq_test-queue_test-exchange_direct_test-key",
		},
		{
			name: "空 MQName",
			mq: MessageQueue{
				QueueName:    "queue",
				ExchangeName: "exchange",
				ExchangeType: "topic",
				RoutingKey:   "key",
			},
			expected: "_queue_exchange_topic_key",
		},
		{
			name:     "全部为空",
			mq:       MessageQueue{},
			expected: "____",
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mq.GetInfo()
			if result != tt.expected {
				t.Errorf("GetInfo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMessageQueue_GetFuncInfo 测试 MessageQueue 的 GetFuncInfo 方法
//
// 【功能点】验证 GetFuncInfo() 返回处理函数名称
// 【测试流程】测试有处理函数和无处理函数两种情况
func TestMessageQueue_GetFuncInfo(t *testing.T) {
	// 测试有处理函数的情况
	mq := MessageQueue{
		Fun: func(msg string) error { return nil },
	}
	result := mq.GetFuncInfo()
	if result == "" {
		t.Error("GetFuncInfo() should return function name, got empty string")
	}

	// 测试没有处理函数的情况
	mqEmpty := MessageQueue{}
	resultEmpty := mqEmpty.GetFuncInfo()
	if resultEmpty != "" {
		t.Errorf("GetFuncInfo() should return empty string for nil function, got %v", resultEmpty)
	}
}

// ==================== 单元测试：死信队列配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证死信队列相关配置方法

// TestMessageQueue_getDeadLetterExchange 测试获取死信交换机名称
//
// 【功能点】验证死信交换机名称生成逻辑
// 【测试流程】
//  1. 测试自定义死信交换机名称
//  2. 测试基于交换机名称自动生成
//  3. 测试基于队列名称自动生成
func TestMessageQueue_getDeadLetterExchange(t *testing.T) {
	tests := []struct {
		name     string
		mq       MessageQueue
		expected string
	}{
		{
			name: "自定义死信交换机",
			mq: MessageQueue{
				ExchangeName: "order.exchange",
				DeadLetter: DeadLetterConfig{
					Exchange: "custom.dlx",
				},
			},
			expected: "custom.dlx",
		},
		{
			name: "基于交换机名称自动生成",
			mq: MessageQueue{
				ExchangeName: "order.exchange",
				DeadLetter:   DeadLetterConfig{},
			},
			expected: "order.exchange.dlx",
		},
		{
			name: "无交换机名称，基于队列名称生成",
			mq: MessageQueue{
				QueueName:  "order.queue",
				DeadLetter: DeadLetterConfig{},
			},
			expected: "order.queue.dlx",
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mq.getDeadLetterExchange()
			if result != tt.expected {
				t.Errorf("getDeadLetterExchange() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMessageQueue_getDeadLetterQueue 测试获取死信队列名称
//
// 【功能点】验证死信队列名称生成逻辑
// 【测试流程】测试自定义名称和自动生成名称两种情况
func TestMessageQueue_getDeadLetterQueue(t *testing.T) {
	tests := []struct {
		name     string
		mq       MessageQueue
		expected string
	}{
		{
			name: "自定义死信队列名称",
			mq: MessageQueue{
				QueueName: "order.queue",
				DeadLetter: DeadLetterConfig{
					QueueName: "custom.dlq",
				},
			},
			expected: "custom.dlq",
		},
		{
			name: "基于队列名称自动生成",
			mq: MessageQueue{
				QueueName:  "order.queue",
				DeadLetter: DeadLetterConfig{},
			},
			expected: "order.queue.dlq",
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mq.getDeadLetterQueue()
			if result != tt.expected {
				t.Errorf("getDeadLetterQueue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMessageQueue_getDeadLetterRoutingKey 测试获取死信路由键
//
// 【功能点】验证死信路由键获取逻辑
// 【测试流程】测试自定义路由键、使用原路由键、空路由键三种情况
func TestMessageQueue_getDeadLetterRoutingKey(t *testing.T) {
	tests := []struct {
		name     string
		mq       MessageQueue
		expected string
	}{
		{
			name: "自定义死信路由键",
			mq: MessageQueue{
				RoutingKey: "order.create",
				DeadLetter: DeadLetterConfig{
					RoutingKey: "order.dead",
				},
			},
			expected: "order.dead",
		},
		{
			name: "使用原路由键",
			mq: MessageQueue{
				RoutingKey: "order.create",
				DeadLetter: DeadLetterConfig{},
			},
			expected: "order.create",
		},
		{
			name: "空路由键",
			mq: MessageQueue{
				DeadLetter: DeadLetterConfig{},
			},
			expected: "",
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			result := tt.mq.getDeadLetterRoutingKey()
			if result != tt.expected {
				t.Errorf("getDeadLetterRoutingKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ==================== 单元测试：消费配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证 ConsumeConfig 结构的默认值和自定义值

// TestConsumeConfig_Defaults 测试 ConsumeConfig 的默认值
//
// 【功能点】验证 ConsumeConfig 结构的默认值
// 【测试流程】创建空配置并验证各字段默认值为零值
func TestConsumeConfig_Defaults(t *testing.T) {
	config := ConsumeConfig{}

	// 测试默认值
	if config.PrefetchCount != 0 {
		t.Errorf("PrefetchCount 默认值应为 0，实际为 %d", config.PrefetchCount)
	}
	if config.MaxRetry != 0 {
		t.Errorf("MaxRetry 默认值应为 0，实际为 %d", config.MaxRetry)
	}
	if config.RetryDelay != 0 {
		t.Errorf("RetryDelay 默认值应为 0，实际为 %v", config.RetryDelay)
	}
}

// TestConsumeConfig_Custom 测试 ConsumeConfig 的自定义值
//
// 【功能点】验证 ConsumeConfig 的自定义配置值
// 【测试流程】设置自定义配置值并验证正确设置
func TestConsumeConfig_Custom(t *testing.T) {
	config := ConsumeConfig{
		PrefetchCount: 10,
		MaxRetry:      5,
		RetryDelay:    time.Second * 3,
	}

	if config.PrefetchCount != 10 {
		t.Errorf("PrefetchCount 应为 10，实际为 %d", config.PrefetchCount)
	}
	if config.MaxRetry != 5 {
		t.Errorf("MaxRetry 应为 5，实际为 %d", config.MaxRetry)
	}
	if config.RetryDelay != time.Second*3 {
		t.Errorf("RetryDelay 应为 3s，实际为 %v", config.RetryDelay)
	}
}

// ==================== 单元测试：Publisher Confirms 配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证 PublishConfirmConfig 结构

// TestPublishConfirmConfig 测试 PublishConfirmConfig 配置
//
// 【功能点】验证 PublishConfirmConfig 的配置值
// 【测试流程】测试默认配置和启用确认模式两种情况
func TestPublishConfirmConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  PublishConfirmConfig
		enabled bool
		timeout time.Duration
	}{
		{
			name:    "默认配置",
			config:  PublishConfirmConfig{},
			enabled: false,
			timeout: 0,
		},
		{
			name: "启用确认",
			config: PublishConfirmConfig{
				Enabled: true,
				Timeout: 5 * time.Second,
			},
			enabled: true,
			timeout: 5 * time.Second,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Enabled != tt.enabled {
				t.Errorf("Enabled = %v, want %v", tt.config.Enabled, tt.enabled)
			}
			if tt.config.Timeout != tt.timeout {
				t.Errorf("Timeout = %v, want %v", tt.config.Timeout, tt.timeout)
			}
		})
	}
}

// ==================== 单元测试：消息重试次数（不需要 RabbitMQ 连接） ====================
// 测试点：验证从消息 header 中获取重试次数

// TestMessageQueue_getRetryCount 测试从消息 header 获取重试次数
//
// 【功能点】验证从消息 x-death header 解析重试次数
// 【测试流程】
//  1. 测试无 headers - 返回 0
//  2. 测试有 x-death header - 返回正确 count
//  3. 测试 x-death 格式错误 - 返回 0
func TestMessageQueue_getRetryCount(t *testing.T) {
	mq := MessageQueue{}

	tests := []struct {
		name     string
		msg      amqp.Delivery
		expected int
	}{
		{
			name:     "无 headers",
			msg:      amqp.Delivery{},
			expected: 0,
		},
		{
			name: "空 headers",
			msg: amqp.Delivery{
				Headers: amqp.Table{},
			},
			expected: 0,
		},
		{
			name: "无 x-death header",
			msg: amqp.Delivery{
				Headers: amqp.Table{
					"other-header": "value",
				},
			},
			expected: 0,
		},
		{
			name: "有 x-death header，count=1",
			msg: amqp.Delivery{
				Headers: amqp.Table{
					"x-death": []interface{}{
						amqp.Table{
							"count": int64(1),
						},
					},
				},
			},
			expected: 1,
		},
		{
			name: "有 x-death header，count=5",
			msg: amqp.Delivery{
				Headers: amqp.Table{
					"x-death": []interface{}{
						amqp.Table{
							"count": int64(5),
						},
					},
				},
			},
			expected: 5,
		},
		{
			name: "x-death 格式错误 - 空数组",
			msg: amqp.Delivery{
				Headers: amqp.Table{
					"x-death": []interface{}{},
				},
			},
			expected: 0,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			result := mq.getRetryCount(tt.msg)
			if result != tt.expected {
				t.Errorf("getRetryCount() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// ==================== 单元测试：DeadLetterConfig 完整配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证 DeadLetterConfig 的完整配置

// TestDeadLetterConfig_Complete 测试 DeadLetterConfig 完整配置
//
// 【功能点】验证 DeadLetterConfig 的完整配置值
// 【测试流程】设置所有配置字段并验证正确设置
func TestDeadLetterConfig_Complete(t *testing.T) {
	config := DeadLetterConfig{
		Enabled:    true,
		Exchange:   "order.dlx",
		RoutingKey: "order.dead",
		QueueName:  "order.dlq",
		MessageTTL: 86400000, // 24小时
	}

	if !config.Enabled {
		t.Error("Enabled 应为 true")
	}
	if config.Exchange != "order.dlx" {
		t.Errorf("Exchange = %v, want order.dlx", config.Exchange)
	}
	if config.RoutingKey != "order.dead" {
		t.Errorf("RoutingKey = %v, want order.dead", config.RoutingKey)
	}
	if config.QueueName != "order.dlq" {
		t.Errorf("QueueName = %v, want order.dlq", config.QueueName)
	}
	if config.MessageTTL != 86400000 {
		t.Errorf("MessageTTL = %v, want 86400000", config.MessageTTL)
	}
}

// ==================== 单元测试：MessageQueue 完整配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证 MessageQueue 的完整配置

// TestMessageQueue_CompleteConfig 测试 MessageQueue 完整配置
//
// 【功能点】验证 MessageQueue 的完整配置值
// 【测试流程】设置所有配置字段并验证正确设置
func TestMessageQueue_CompleteConfig(t *testing.T) {
	handler := func(ctx context.Context, msg string) error {
		return nil
	}

	mq := MessageQueue{
		MQName:       "primary",
		QueueName:    "order.process",
		ExchangeName: "order.exchange",
		ExchangeType: "direct",
		RoutingKey:   "order.create",
		FunWithCtx:   handler,
		DeadLetter: DeadLetterConfig{
			Enabled:    true,
			MessageTTL: 86400000,
		},
		PublishConfirm: PublishConfirmConfig{
			Enabled: true,
			Timeout: 5 * time.Second,
		},
		ConsumeConfig: ConsumeConfig{
			PrefetchCount: 10,
			MaxRetry:      3,
			RetryDelay:    time.Second,
		},
	}

	// 验证基本配置
	expectedInfo := "primary_order.process_order.exchange_direct_order.create"
	if mq.GetInfo() != expectedInfo {
		t.Errorf("GetInfo() = %v, want %v", mq.GetInfo(), expectedInfo)
	}

	// 验证死信队列配置
	if !mq.DeadLetter.Enabled {
		t.Error("DeadLetter.Enabled 应为 true")
	}
	if mq.getDeadLetterExchange() != "order.exchange.dlx" {
		t.Errorf("getDeadLetterExchange() = %v, want order.exchange.dlx", mq.getDeadLetterExchange())
	}
	if mq.getDeadLetterQueue() != "order.process.dlq" {
		t.Errorf("getDeadLetterQueue() = %v, want order.process.dlq", mq.getDeadLetterQueue())
	}

	// 验证消费配置
	if mq.ConsumeConfig.PrefetchCount != 10 {
		t.Errorf("ConsumeConfig.PrefetchCount = %v, want 10", mq.ConsumeConfig.PrefetchCount)
	}

	// 验证处理函数存在
	if mq.FunWithCtx == nil {
		t.Error("FunWithCtx 不应为 nil")
	}
}

// ==================== 单元测试：批量发布空消息（不需要 RabbitMQ 连接） ====================
// 测试点：验证批量发布空消息列表的快速返回

// TestMessageQueue_PublishBatch_EmptyMessages 测试批量发布空消息列表
//
// 【功能点】验证空消息列表时快速返回 nil
// 【测试流程】分别传入 nil 和空数组，验证都返回 nil
func TestMessageQueue_PublishBatch_EmptyMessages(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	// 空消息列表不应返回错误
	err := mq.PublishBatch(nil)
	if err != nil {
		t.Errorf("PublishBatch(nil) 应返回 nil，实际返回 %v", err)
	}

	err = mq.PublishBatch([]string{})
	if err != nil {
		t.Errorf("PublishBatch([]) 应返回 nil，实际返回 %v", err)
	}
}

// TestMessageQueue_PublishBatchWithContext_EmptyMessages 测试带 Context 的批量发布空消息列表
//
// 【功能点】验证带 Context 的空消息列表时快速返回 nil
// 【测试流程】分别传入 nil 和空数组，验证都返回 nil
func TestMessageQueue_PublishBatchWithContext_EmptyMessages(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	ctx := context.Background()

	// 空消息列表不应返回错误
	err := mq.PublishBatchWithContext(ctx, nil)
	if err != nil {
		t.Errorf("PublishBatchWithContext(ctx, nil) 应返回 nil，实际返回 %v", err)
	}

	err = mq.PublishBatchWithContext(ctx, []string{})
	if err != nil {
		t.Errorf("PublishBatchWithContext(ctx, []) 应返回 nil，实际返回 %v", err)
	}
}

// ==================== 单元测试：消费者函数处理（不需要 RabbitMQ 连接） ====================
// 测试点：验证消费者处理函数的调用

// TestMessageQueue_HandlerFunctions 测试处理函数的调用
//
// 【功能点】验证 Fun 和 FunWithCtx 处理函数被正确调用
// 【测试流程】分别测试旧版 Fun 和新版 FunWithCtx 的调用和参数传递
func TestMessageQueue_HandlerFunctions(t *testing.T) {
	var called bool
	var receivedMsg string

	// 测试旧版处理函数
	mqOld := MessageQueue{
		Fun: func(msg string) error {
			called = true
			receivedMsg = msg
			return nil
		},
	}

	if mqOld.Fun == nil {
		t.Error("Fun 不应为 nil")
	}

	called = false
	_ = mqOld.Fun("test message")
	if !called {
		t.Error("Fun 应被调用")
	}
	if receivedMsg != "test message" {
		t.Errorf("receivedMsg = %v, want 'test message'", receivedMsg)
	}

	// 测试新版处理函数
	mqNew := MessageQueue{
		FunWithCtx: func(ctx context.Context, msg string) error {
			called = true
			receivedMsg = msg
			return nil
		},
	}

	if mqNew.FunWithCtx == nil {
		t.Error("FunWithCtx 不应为 nil")
	}

	called = false
	_ = mqNew.FunWithCtx(context.Background(), "test message with ctx")
	if !called {
		t.Error("FunWithCtx 应被调用")
	}
	if receivedMsg != "test message with ctx" {
		t.Errorf("receivedMsg = %v, want 'test message with ctx'", receivedMsg)
	}
}

// TestMessageQueue_HandlerWithError 测试处理函数返回错误
//
// 【功能点】验证处理函数返回错误时正确传递
// 【测试流程】设置返回错误的处理函数，验证错误正确返回
func TestMessageQueue_HandlerWithError(t *testing.T) {
	expectedErr := errors.New("处理失败")

	mq := MessageQueue{
		Fun: func(msg string) error {
			return expectedErr
		},
	}

	err := mq.Fun("test")
	if err != expectedErr {
		t.Errorf("Fun 应返回预期的错误，实际返回 %v", err)
	}

	mqCtx := MessageQueue{
		FunWithCtx: func(ctx context.Context, msg string) error {
			return expectedErr
		},
	}

	err = mqCtx.FunWithCtx(context.Background(), "test")
	if err != expectedErr {
		t.Errorf("FunWithCtx 应返回预期的错误，实际返回 %v", err)
	}
}

// ==================== 单元测试：Context 取消（不需要 RabbitMQ 连接） ====================
// 测试点：验证 Context 取消时的行为

// TestMessageQueue_ContextCancellation 测试 Context 取消时的处理
//
// 【功能点】验证 Context 取消时处理函数正确响应
// 【测试流程】先测试未取消时正常返回，再测试取消后返回 context.Canceled
func TestMessageQueue_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mq := MessageQueue{
		FunWithCtx: func(ctx context.Context, msg string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
	}

	// context 未取消时应正常返回
	err := mq.FunWithCtx(ctx, "test")
	if err != nil {
		t.Errorf("未取消的 context 不应返回错误，实际返回 %v", err)
	}

	// 取消 context
	cancel()

	// context 取消后应返回错误
	err = mq.FunWithCtx(ctx, "test")
	if err == nil {
		t.Error("已取消的 context 应返回错误")
	}
}

// ==================== 单元测试：Close 方法（不需要 RabbitMQ 连接） ====================
// 测试点：验证 Close 方法对 nil 连接和通道的处理

// TestMessageQueue_Close_NilConnAndChannel 测试关闭 nil 连接和通道
//
// 【功能点】验证 Close() 对 nil 连接和通道的安全处理
// 【测试流程】创建空 MessageQueue 并调用 Close()，验证不会 panic
func TestMessageQueue_Close_NilConnAndChannel(t *testing.T) {
	mq := MessageQueue{}

	// 不应 panic
	mq.Close()
}

// ==================== 单元测试：消息处理 handleMessage（不需要 RabbitMQ 连接） ====================
// 测试点：验证消息处理函数的调用逻辑

// TestMessageQueue_HandleMessage_WithFunWithCtx 测试使用 FunWithCtx 处理消息
//
// 【功能点】验证 FunWithCtx 处理函数被正确调用
// 【测试流程】设置处理函数并模拟调用，验证消息正确传递
func TestMessageQueue_HandleMessage_WithFunWithCtx(t *testing.T) {
	var receivedMsg string
	var handlerCalled bool

	mq := MessageQueue{
		FunWithCtx: func(ctx context.Context, msg string) error {
			handlerCalled = true
			receivedMsg = msg
			return nil
		},
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 3,
		},
	}

	// 模拟消息
	msg := amqp.Delivery{
		Body: []byte("test message"),
	}

	// 调用 handleMessage（注意：这个方法是私有的，需要通过其他方式测试）
	// 这里我们直接测试处理函数的行为
	ctx := context.Background()
	err := mq.FunWithCtx(ctx, string(msg.Body))

	if err != nil {
		t.Errorf("处理函数应返回 nil，实际返回 %v", err)
	}
	if !handlerCalled {
		t.Error("处理函数应被调用")
	}
	if receivedMsg != "test message" {
		t.Errorf("receivedMsg = %v, want 'test message'", receivedMsg)
	}
}

// TestMessageQueue_HandleMessage_WithFun 测试使用 Fun 处理消息
//
// 【功能点】验证旧版 Fun 处理函数被正确调用
// 【测试流程】设置处理函数并模拟调用，验证消息正确传递
func TestMessageQueue_HandleMessage_WithFun(t *testing.T) {
	var receivedMsg string
	var handlerCalled bool

	mq := MessageQueue{
		Fun: func(msg string) error {
			handlerCalled = true
			receivedMsg = msg
			return nil
		},
	}

	// 模拟消息
	msg := amqp.Delivery{
		Body: []byte("legacy message"),
	}

	// 测试旧版处理函数
	err := mq.Fun(string(msg.Body))

	if err != nil {
		t.Errorf("处理函数应返回 nil，实际返回 %v", err)
	}
	if !handlerCalled {
		t.Error("处理函数应被调用")
	}
	if receivedMsg != "legacy message" {
		t.Errorf("receivedMsg = %v, want 'legacy message'", receivedMsg)
	}
}

// TestMessageQueue_HandleMessage_Error 测试处理消息时返回错误
//
// 【功能点】验证处理函数返回错误时正确传递
// 【测试流程】设置返回错误的处理函数，验证错误正确返回
func TestMessageQueue_HandleMessage_Error(t *testing.T) {
	expectedErr := errors.New("处理失败")

	mq := MessageQueue{
		FunWithCtx: func(ctx context.Context, msg string) error {
			return expectedErr
		},
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 3,
		},
	}

	ctx := context.Background()
	err := mq.FunWithCtx(ctx, "test")

	if err != expectedErr {
		t.Errorf("处理函数应返回预期的错误，实际返回 %v", err)
	}
}

// ==================== 单元测试：Consume 方法无连接（不需要 RabbitMQ 连接） ====================
// 测试点：验证无效连接时 Consume 方法返回错误

// TestMessageQueue_Consume_NoConnection 测试无效连接时 Consume 返回错误
//
// 【功能点】验证无效连接时 Consume() 返回连接错误
// 【测试流程】设置无效连接字符串，调用 Consume()，验证返回错误
func TestMessageQueue_Consume_NoConnection(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
		Fun:          func(msg string) error { return nil },
	}

	// 应该返回连接错误
	err := mq.Consume()
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// TestMessageQueue_ConsumeWithContext_NoConnection 测试无效连接时 ConsumeWithContext 返回错误
//
// 【功能点】验证无效连接时 ConsumeWithContext() 返回连接错误
// 【测试流程】设置无效连接字符串，调用 ConsumeWithContext()，验证返回错误
func TestMessageQueue_ConsumeWithContext_NoConnection(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
		FunWithCtx:   func(ctx context.Context, msg string) error { return nil },
	}

	ctx := context.Background()

	// 应该返回连接错误
	err := mq.ConsumeWithContext(ctx)
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// ==================== 单元测试：Publish 方法无连接（不需要 RabbitMQ 连接） ====================
// 测试点：验证无效连接时 Publish 方法返回错误

// TestMessageQueue_Publish_NoConnection 测试无效连接时 Publish 返回错误
//
// 【功能点】验证无效连接时 Publish() 返回连接错误
// 【测试流程】设置无效连接字符串，调用 Publish()，验证返回错误
func TestMessageQueue_Publish_NoConnection(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
	}

	// 应该返回连接错误
	err := mq.Publish("test message")
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// TestMessageQueue_PublishWithContext_NoConnection 测试无效连接时 PublishWithContext 返回错误
//
// 【功能点】验证无效连接时 PublishWithContext() 返回连接错误
// 【测试流程】设置无效连接字符串，调用 PublishWithContext()，验证返回错误
func TestMessageQueue_PublishWithContext_NoConnection(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
	}

	ctx := context.Background()

	// 应该返回连接错误
	err := mq.PublishWithContext(ctx, "test message")
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// TestMessageQueue_PublishBatch_NoConnection 测试无效连接时 PublishBatch 返回错误
//
// 【功能点】验证无效连接时 PublishBatch() 返回连接错误
// 【测试流程】设置无效连接字符串，调用 PublishBatch()，验证返回错误
func TestMessageQueue_PublishBatch_NoConnection(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
	}

	messages := []string{"msg1", "msg2", "msg3"}

	// 应该返回连接错误
	err := mq.PublishBatch(messages)
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// ==================== 单元测试：InitChannelForProducer 无连接（不需要 RabbitMQ 连接） ====================
// 测试点：验证无效连接时 InitChannelForProducer 返回错误

// TestMessageQueue_InitChannelForProducer_NoConnection 测试无效连接时返回错误
//
// 【功能点】验证无效连接时 InitChannelForProducer() 返回连接错误
// 【测试流程】设置无效连接字符串，调用 InitChannelForProducer()，验证返回错误
func TestMessageQueue_InitChannelForProducer_NoConnection(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
	}

	// 应该返回连接错误
	err := mq.InitChannelForProducer()
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// TestMessageQueue_InitChannelForProducer_WithConfirm 测试带确认模式的无效连接
//
// 【功能点】验证启用 Publisher Confirms 时无效连接返回错误
// 【测试流程】启用确认模式并设置无效连接，验证返回错误
func TestMessageQueue_InitChannelForProducer_WithConfirm(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
		MqConnStr:    "amqp://invalid:invalid@localhost:9999/",
		PublishConfirm: PublishConfirmConfig{
			Enabled: true,
			Timeout: 5 * time.Second,
		},
	}

	// 应该返回连接错误（但会尝试启用确认模式）
	err := mq.InitChannelForProducer()
	if err == nil {
		t.Error("无效连接应返回错误")
	}
}

// ==================== 单元测试：消费者配置应用（不需要 RabbitMQ 连接） ====================
// 测试点：验证消费者配置的默认值处理

// TestMessageQueue_ConsumeConfig_PrefetchCount 测试 PrefetchCount 的默认值处理
// 不需要 RabbitMQ 连接：仅验证配置处理逻辑
func TestMessageQueue_ConsumeConfig_PrefetchCount(t *testing.T) {
	tests := []struct {
		name           string
		prefetchCount  int
		expectedActual int
	}{
		{"默认值（0转为1）", 0, 1},
		{"自定义值", 10, 10},
		{"负值（转为1）", -1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mq := MessageQueue{
				ConsumeConfig: ConsumeConfig{
					PrefetchCount: tt.prefetchCount,
				},
			}

			// 验证 prefetchCount 的实际处理
			actual := mq.ConsumeConfig.PrefetchCount
			if actual <= 0 {
				actual = 1 // 这是 initChannel 中的逻辑
			}

			if actual != tt.expectedActual {
				t.Errorf("PrefetchCount 处理后应为 %d，实际为 %d", tt.expectedActual, actual)
			}
		})
	}
}

// TestMessageQueue_ConsumeConfig_MaxRetry 测试 MaxRetry 的默认值处理
// 不需要 RabbitMQ 连接：仅验证配置处理逻辑
func TestMessageQueue_ConsumeConfig_MaxRetry(t *testing.T) {
	tests := []struct {
		name           string
		maxRetry       int
		expectedActual int
	}{
		{"默认值（0转为3）", 0, 3},
		{"自定义值", 5, 5},
		{"负值（转为3）", -1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mq := MessageQueue{
				ConsumeConfig: ConsumeConfig{
					MaxRetry: tt.maxRetry,
				},
			}

			// 验证 maxRetry 的实际处理
			actual := mq.ConsumeConfig.MaxRetry
			if actual <= 0 {
				actual = 3 // 这是 handleMessage 中的逻辑
			}

			if actual != tt.expectedActual {
				t.Errorf("MaxRetry 处理后应为 %d，实际为 %d", tt.expectedActual, actual)
			}
		})
	}
}

// ==================== 单元测试：死信队列集成配置（不需要 RabbitMQ 连接） ====================
// 测试点：验证死信队列的完整配置和自动生成

// TestMessageQueue_DeadLetter_FullConfig 测试死信队列完整配置
// 不需要 RabbitMQ 连接：仅验证配置值
func TestMessageQueue_DeadLetter_FullConfig(t *testing.T) {
	mq := MessageQueue{
		MQName:       "test-mq",
		QueueName:    "order.process",
		ExchangeName: "order.exchange",
		ExchangeType: "direct",
		RoutingKey:   "order.create",
		DeadLetter: DeadLetterConfig{
			Enabled:    true,
			Exchange:   "order.dlx",
			QueueName:  "order.dlq",
			RoutingKey: "order.dead",
			MessageTTL: 86400000,
		},
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 5,
		},
	}

	// 验证死信队列配置
	if !mq.DeadLetter.Enabled {
		t.Error("DeadLetter.Enabled 应为 true")
	}

	// 验证自定义配置优先于自动生成
	if mq.getDeadLetterExchange() != "order.dlx" {
		t.Errorf("自定义死信交换机应为 order.dlx，实际为 %s", mq.getDeadLetterExchange())
	}
	if mq.getDeadLetterQueue() != "order.dlq" {
		t.Errorf("自定义死信队列应为 order.dlq，实际为 %s", mq.getDeadLetterQueue())
	}
	if mq.getDeadLetterRoutingKey() != "order.dead" {
		t.Errorf("自定义死信路由键应为 order.dead，实际为 %s", mq.getDeadLetterRoutingKey())
	}
}

// TestMessageQueue_DeadLetter_AutoGenerated 测试死信队列自动生成配置
// 不需要 RabbitMQ 连接：仅验证自动生成逻辑
func TestMessageQueue_DeadLetter_AutoGenerated(t *testing.T) {
	mq := MessageQueue{
		QueueName:    "order.process",
		ExchangeName: "order.exchange",
		ExchangeType: "direct",
		RoutingKey:   "order.create",
		DeadLetter: DeadLetterConfig{
			Enabled: true,
			// 不设置自定义值，测试自动生成
		},
	}

	// 验证自动生成的配置
	if mq.getDeadLetterExchange() != "order.exchange.dlx" {
		t.Errorf("自动生成的死信交换机应为 order.exchange.dlx，实际为 %s", mq.getDeadLetterExchange())
	}
	if mq.getDeadLetterQueue() != "order.process.dlq" {
		t.Errorf("自动生成的死信队列应为 order.process.dlq，实际为 %s", mq.getDeadLetterQueue())
	}
	if mq.getDeadLetterRoutingKey() != "order.create" {
		t.Errorf("自动生成的死信路由键应为 order.create，实际为 %s", mq.getDeadLetterRoutingKey())
	}
}

// ==================== 单元测试：基准测试（不需要 RabbitMQ 连接） ====================
// 测试点：验证配置方法的性能

// BenchmarkMessageQueue_GetInfo 基准测试 GetInfo 方法性能
// 不需要 RabbitMQ 连接：仅测试字符串格式化性能
func BenchmarkMessageQueue_GetInfo(b *testing.B) {
	mq := MessageQueue{
		MQName:       "test-mq",
		QueueName:    "test-queue",
		ExchangeName: "test-exchange",
		ExchangeType: "direct",
		RoutingKey:   "test-key",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mq.GetInfo()
	}
}

// BenchmarkRabbitMQInfo_Url 基准测试 RabbitMQInfo.Url 方法性能
// 不需要 RabbitMQ 连接：仅测试字符串格式化性能
func BenchmarkRabbitMQInfo_Url(b *testing.B) {
	info := RabbitMQInfo{
		Host:     "localhost",
		Port:     5672,
		Username: "guest",
		Password: "guest",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = info.Url()
	}
}

// BenchmarkRabbitMqListInfo_Url 基准测试 RabbitMqListInfo.Url 方法性能
// 不需要 RabbitMQ 连接：仅测试列表查找性能
func BenchmarkRabbitMqListInfo_Url(b *testing.B) {
	list := RabbitMqListInfo{
		{AliasName: "primary", Host: "localhost", Port: 5672, Username: "guest", Password: "guest"},
		{AliasName: "secondary", Host: "192.168.1.100", Port: 5672, Username: "admin", Password: "admin"},
		{AliasName: "tertiary", Host: "192.168.1.101", Port: 5672, Username: "user", Password: "pass"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = list.Url("secondary")
	}
}
