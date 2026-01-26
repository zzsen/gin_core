// Package circuitbreaker 熔断器测试
//
// ==================== 测试说明 ====================
// 本文件包含熔断器的单元测试，不需要外部依赖。
//
// 测试覆盖内容：
// 1. 基础功能 - 熔断器创建、状态字符串
// 2. 状态转换 - Closed→Open、Open→HalfOpen、HalfOpen→Closed、HalfOpen→Open
// 3. 请求拦截 - Open状态拒绝请求、HalfOpen状态限制并发数
// 4. 计数器 - 成功/失败计数、周期性重置
// 5. 回调 - 状态变更回调
// 6. 手动操作 - 手动重置
// 7. Context - 上下文取消
// 8. 并发安全 - 多协程并发执行
// 9. 注册中心 - Get/List/Stats/ResetAll
// 10. 便捷函数 - 全局Execute函数
// 11. 性能基准 - 成功/失败/并发执行
//
// 运行测试：go test -v ./circuitbreaker/...
// ==================================================
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ==================== 基础功能测试 ====================
// 测试熔断器的创建和基础属性

// TestNew 测试创建熔断器
//
// 【功能点】验证默认配置创建熔断器
// 【测试流程】
//  1. 使用 nil 配置调用 New() 创建熔断器
//  2. 验证返回实例非空
//  3. 验证初始状态为 Closed（关闭状态，允许请求通过）
//  4. 验证默认名称为 "default"
func TestNew(t *testing.T) {
	cb := New(nil)
	if cb == nil {
		t.Fatal("New() 应返回非空实例")
	}

	if cb.State() != StateClosed {
		t.Errorf("初始状态应为 Closed，实际为 %v", cb.State())
	}

	if cb.Name() != "default" {
		t.Errorf("默认名称应为 'default'，实际为 %s", cb.Name())
	}
}

// TestNewWithConfig 测试使用自定义配置创建熔断器
//
// 【功能点】验证自定义配置创建熔断器
// 【测试流程】
//  1. 创建自定义配置（指定名称、最大请求数、超时时间、失败阈值）
//  2. 使用配置创建熔断器
//  3. 验证熔断器名称与配置一致
func TestNewWithConfig(t *testing.T) {
	config := &Config{
		Name:             "test-service",
		MaxRequests:      5,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
	}

	cb := New(config)
	if cb.Name() != "test-service" {
		t.Errorf("名称应为 'test-service'，实际为 %s", cb.Name())
	}
}

// TestStateString 测试状态字符串表示
//
// 【功能点】验证状态枚举的字符串转换
// 【测试流程】
//  1. 遍历所有状态值（Closed/Open/HalfOpen/未知）
//  2. 调用 String() 方法
//  3. 验证返回的字符串与期望值一致
func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "closed"},
		{StateOpen, "open"},
		{StateHalfOpen, "half-open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}

// ==================== 状态转换测试 ====================
// 测试熔断器的状态机转换逻辑
//
// 状态机说明：
//   Closed（关闭）→ Open（打开）：连续失败次数达到阈值 或 失败率超过阈值
//   Open（打开）→ HalfOpen（半开）：超时时间到期后自动转换
//   HalfOpen（半开）→ Closed（关闭）：连续成功次数达到 MaxRequests
//   HalfOpen（半开）→ Open（打开）：任何一次失败立即触发

// TestClosedToOpen_ConsecutiveFailures 测试连续失败触发熔断
//
// 【功能点】验证连续失败次数达到阈值时触发熔断
// 【测试流程】
//  1. 创建熔断器，设置连续失败阈值为 3
//  2. 连续执行 3 次失败的请求
//  3. 验证熔断器状态变为 Open
func TestClosedToOpen_ConsecutiveFailures(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(3),
		WithMinRequests(100), // 设置高值，确保不会因失败率触发
	)
	cb := New(config)

	ctx := context.Background()
	failErr := errors.New("service error")

	// 连续失败 3 次
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func() error {
			return failErr
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("连续失败 3 次后状态应为 Open，实际为 %v", cb.State())
	}
}

// TestClosedToOpen_FailureRatio 测试失败率触发熔断
//
// 【功能点】验证失败率超过阈值时触发熔断
// 【测试流程】
//  1. 创建熔断器，设置最小请求数 10、失败率阈值 50%
//  2. 发送 10 次请求，其中 6 次失败（60% > 50%）
//  3. 验证熔断器状态变为 Open
func TestClosedToOpen_FailureRatio(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(100), // 设置高值，确保不会因连续失败触发
		WithMinRequests(10),
		WithFailureRatio(0.5),
	)
	cb := New(config)

	ctx := context.Background()
	failErr := errors.New("service error")

	// 交替发送成功和失败请求，确保失败率检查时请求数已达到阈值
	// 最后一个请求是失败的，触发失败率检查
	// 请求序列：失败、成功、失败、成功、失败、成功、失败、成功、失败、失败
	// 10次请求，6次失败（60% > 50%）
	failPattern := []bool{true, false, true, false, true, false, true, false, true, true}
	for _, shouldFail := range failPattern {
		fail := shouldFail
		_ = cb.Execute(ctx, func() error {
			if fail {
				return failErr
			}
			return nil
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("失败率超过阈值后状态应为 Open，实际为 %v", cb.State())
	}
}

// TestOpenToHalfOpen 测试超时后进入半开状态
//
// 【功能点】验证 Open 状态超时后自动转换为 HalfOpen
// 【测试流程】
//  1. 创建熔断器，设置超时时间 100ms
//  2. 触发熔断（执行失败请求）
//  3. 等待 150ms（超过超时时间）
//  4. 验证熔断器状态变为 HalfOpen
func TestOpenToHalfOpen(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(100*time.Millisecond),
	)
	cb := New(config)

	ctx := context.Background()

	// 触发熔断
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	if cb.State() != StateOpen {
		t.Fatalf("状态应为 Open，实际为 %v", cb.State())
	}

	// 等待超时
	time.Sleep(150 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Errorf("超时后状态应为 HalfOpen，实际为 %v", cb.State())
	}
}

// TestHalfOpenToClosed 测试半开状态下成功恢复
//
// 【功能点】验证 HalfOpen 状态下连续成功后恢复到 Closed
// 【测试流程】
//  1. 创建熔断器，设置超时 50ms、MaxRequests 2
//  2. 触发熔断 → 等待进入 HalfOpen
//  3. 连续成功执行 2 次请求
//  4. 验证熔断器状态恢复为 Closed
func TestHalfOpenToClosed(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(50*time.Millisecond),
		WithMaxRequests(2),
	)
	cb := New(config)

	ctx := context.Background()

	// 触发熔断
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	// 等待进入半开状态
	time.Sleep(60 * time.Millisecond)

	// 连续成功 2 次
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, func() error {
			return nil
		})
		if err != nil {
			t.Fatalf("半开状态下执行失败: %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("连续成功后状态应为 Closed，实际为 %v", cb.State())
	}
}

// TestHalfOpenToOpen 测试半开状态下失败重新熔断
//
// 【功能点】验证 HalfOpen 状态下任何失败立即触发熔断
// 【测试流程】
//  1. 创建熔断器，设置超时 50ms
//  2. 触发熔断 → 等待进入 HalfOpen
//  3. 执行一次失败请求
//  4. 验证熔断器状态重新变为 Open
func TestHalfOpenToOpen(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(50*time.Millisecond),
		WithMaxRequests(3),
	)
	cb := New(config)

	ctx := context.Background()

	// 触发熔断
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	// 等待进入半开状态
	time.Sleep(60 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Fatalf("状态应为 HalfOpen，实际为 %v", cb.State())
	}

	// 在半开状态下失败
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	if cb.State() != StateOpen {
		t.Errorf("半开状态下失败后状态应为 Open，实际为 %v", cb.State())
	}
}

// ==================== 请求拦截测试 ====================
// 测试熔断器对请求的拦截行为

// TestOpenState_RejectsRequests 测试打开状态拒绝请求
//
// 【功能点】验证 Open 状态下所有请求被立即拒绝
// 【测试流程】
//  1. 创建熔断器并触发熔断
//  2. 尝试执行新请求
//  3. 验证请求被拒绝，返回 ErrCircuitOpen 错误
//  4. 验证业务函数未被执行
func TestOpenState_RejectsRequests(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(1*time.Hour), // 长超时，确保不会进入半开状态
	)
	cb := New(config)

	ctx := context.Background()

	// 触发熔断
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	// 后续请求应被拒绝
	err := cb.Execute(ctx, func() error {
		t.Error("不应该执行此函数")
		return nil
	})

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("应返回 ErrCircuitOpen，实际返回 %v", err)
	}
}

// TestHalfOpenState_LimitsRequests 测试半开状态限制请求数
//
// 【功能点】验证 HalfOpen 状态下只允许有限数量的探测请求
// 【测试流程】
//  1. 创建熔断器，设置 MaxRequests=2
//  2. 触发熔断 → 等待进入 HalfOpen
//  3. 并发发送 10 个请求
//  4. 验证只有 ≤2 个请求成功执行，其余被拒绝
func TestHalfOpenState_LimitsRequests(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(50*time.Millisecond),
		WithMaxRequests(2),
	)
	cb := New(config)

	ctx := context.Background()

	// 触发熔断
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	// 等待进入半开状态
	time.Sleep(60 * time.Millisecond)

	// 发送多个并发请求
	var wg sync.WaitGroup
	var successCount, rejectedCount int32

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Execute(ctx, func() error {
				time.Sleep(10 * time.Millisecond) // 模拟处理时间
				return nil
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else if errors.Is(err, ErrTooManyRequests) {
				atomic.AddInt32(&rejectedCount, 1)
			}
		}()
	}

	wg.Wait()

	// 应该只有 MaxRequests(2) 个请求成功
	if successCount > 2 {
		t.Errorf("半开状态下成功请求数应 <= 2，实际为 %d", successCount)
	}
}

// ==================== 计数器测试 ====================
// 测试熔断器的请求统计功能

// TestCounts 测试计数统计
//
// 【功能点】验证熔断器正确统计成功/失败/连续失败次数
// 【测试流程】
//  1. 创建熔断器（高阈值防止熔断）
//  2. 执行 3 次成功请求 + 2 次失败请求
//  3. 获取计数统计
//  4. 验证：总请求数=5、成功数=3、失败数=2、连续失败数=2
func TestCounts(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(100), // 高阈值，防止熔断
	)
	cb := New(config)

	ctx := context.Background()

	// 3 次成功
	for i := 0; i < 3; i++ {
		_ = cb.Execute(ctx, func() error { return nil })
	}

	// 2 次失败
	for i := 0; i < 2; i++ {
		_ = cb.Execute(ctx, func() error { return errors.New("error") })
	}

	counts := cb.Counts()

	if counts.Requests != 5 {
		t.Errorf("总请求数应为 5，实际为 %d", counts.Requests)
	}

	if counts.TotalSuccesses != 3 {
		t.Errorf("成功数应为 3，实际为 %d", counts.TotalSuccesses)
	}

	if counts.TotalFailures != 2 {
		t.Errorf("失败数应为 2，实际为 %d", counts.TotalFailures)
	}

	if counts.ConsecutiveFailures != 2 {
		t.Errorf("连续失败数应为 2，实际为 %d", counts.ConsecutiveFailures)
	}
}

// TestCountsReset_AfterInterval 测试统计周期后计数器重置
//
// 【功能点】验证统计周期（Interval）结束后计数器自动重置
// 【测试流程】
//  1. 创建熔断器，设置 Interval=50ms
//  2. 发送 5 个请求，验证计数为 5
//  3. 等待 60ms（超过周期）
//  4. 发送 1 个请求触发重置
//  5. 验证计数重置为 1
func TestCountsReset_AfterInterval(t *testing.T) {
	config := NewConfig("test",
		WithInterval(50*time.Millisecond),
		WithFailureThreshold(100),
	)
	cb := New(config)

	ctx := context.Background()

	// 发送一些请求
	for i := 0; i < 5; i++ {
		_ = cb.Execute(ctx, func() error { return nil })
	}

	if cb.Counts().Requests != 5 {
		t.Fatalf("请求数应为 5，实际为 %d", cb.Counts().Requests)
	}

	// 等待周期结束
	time.Sleep(60 * time.Millisecond)

	// 发送新请求触发重置
	_ = cb.Execute(ctx, func() error { return nil })

	if cb.Counts().Requests != 1 {
		t.Errorf("周期重置后请求数应为 1，实际为 %d", cb.Counts().Requests)
	}
}

// ==================== 回调测试 ====================
// 测试熔断器的事件回调功能

// TestOnStateChange 测试状态变更回调
//
// 【功能点】验证状态变更时触发 OnStateChange 回调
// 【测试流程】
//  1. 创建熔断器，注册状态变更回调
//  2. 触发熔断（Closed→Open）
//  3. 等待回调执行
//  4. 验证回调被调用 1 次，状态从 Closed 变为 Open
func TestOnStateChange(t *testing.T) {
	var callCount int
	var lastFrom, lastTo State
	var mu sync.Mutex

	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(50*time.Millisecond),
		WithMaxRequests(1),
		WithOnStateChange(func(name string, from, to State) {
			mu.Lock()
			defer mu.Unlock()
			callCount++
			lastFrom = from
			lastTo = to
		}),
	)
	cb := New(config)

	ctx := context.Background()

	// 触发 Closed -> Open
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	// 等待回调执行
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if callCount != 1 {
		t.Errorf("回调应被调用 1 次，实际 %d 次", callCount)
	}
	if lastFrom != StateClosed || lastTo != StateOpen {
		t.Errorf("状态转换应为 Closed->Open，实际为 %v->%v", lastFrom, lastTo)
	}
	mu.Unlock()
}

// ==================== 手动操作测试 ====================
// 测试熔断器的手动控制功能

// TestReset 测试手动重置
//
// 【功能点】验证手动重置熔断器到 Closed 状态
// 【测试流程】
//  1. 创建熔断器并触发熔断
//  2. 确认状态为 Open
//  3. 调用 Reset() 手动重置
//  4. 验证状态恢复为 Closed
//  5. 验证可以正常执行请求
func TestReset(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(1),
		WithTimeout(1*time.Hour),
	)
	cb := New(config)

	ctx := context.Background()

	// 触发熔断
	_ = cb.Execute(ctx, func() error {
		return errors.New("error")
	})

	if cb.State() != StateOpen {
		t.Fatalf("状态应为 Open，实际为 %v", cb.State())
	}

	// 手动重置
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("重置后状态应为 Closed，实际为 %v", cb.State())
	}

	// 确认可以正常执行
	err := cb.Execute(ctx, func() error { return nil })
	if err != nil {
		t.Errorf("重置后应能正常执行，错误: %v", err)
	}
}

// ==================== Context 测试 ====================
// 测试熔断器对 Context 的支持

// TestContextCancellation 测试上下文取消
//
// 【功能点】验证 Context 取消时请求立即返回
// 【测试流程】
//  1. 创建已取消的 Context
//  2. 尝试执行请求
//  3. 验证返回 context.Canceled 错误
//  4. 验证业务函数未被执行
func TestContextCancellation(t *testing.T) {
	cb := New(nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err := cb.Execute(ctx, func() error {
		t.Error("不应该执行此函数")
		return nil
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("应返回 context.Canceled，实际返回 %v", err)
	}
}

// ==================== 并发安全测试 ====================
// 测试熔断器在高并发场景下的正确性

// TestConcurrentExecution 测试并发执行
//
// 【功能点】验证熔断器在多协程并发访问时的线程安全性
// 【测试流程】
//  1. 创建熔断器（高阈值防止熔断）
//  2. 启动 100 个协程，每个协程执行 100 次请求
//  3. 等待所有协程完成
//  4. 验证成功请求数 = 10000
//  5. 验证统计计数与实际执行次数一致
func TestConcurrentExecution(t *testing.T) {
	config := NewConfig("test",
		WithFailureThreshold(10000), // 高阈值，防止熔断
		WithMinRequests(100000),     // 高阈值，防止因失败率熔断
	)
	cb := New(config)

	ctx := context.Background()
	var wg sync.WaitGroup
	var successCount int32
	concurrency := 100
	iterations := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				err := cb.Execute(ctx, func() error {
					return nil // 全部成功，避免熔断
				})
				if err == nil {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()

	expectedRequests := int32(concurrency * iterations)

	if successCount != expectedRequests {
		t.Errorf("成功请求数应为 %d，实际为 %d", expectedRequests, successCount)
	}

	counts := cb.Counts()
	if counts.Requests != uint32(expectedRequests) {
		t.Errorf("总请求数应为 %d，实际为 %d", expectedRequests, counts.Requests)
	}
}

// ==================== 注册中心测试 ====================
// 测试熔断器注册中心的管理功能

// TestRegistry_Get 测试从注册中心获取熔断器
//
// 【功能点】验证注册中心的单例获取和隔离性
// 【测试流程】
//  1. 创建注册中心
//  2. 获取同名熔断器两次，验证返回相同实例
//  3. 获取不同名熔断器，验证返回不同实例
func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry(nil)

	cb1 := registry.Get("service-a")
	cb2 := registry.Get("service-a")

	if cb1 != cb2 {
		t.Error("同名熔断器应返回相同实例")
	}

	cb3 := registry.Get("service-b")
	if cb1 == cb3 {
		t.Error("不同名熔断器应返回不同实例")
	}
}

// TestRegistry_List 测试列出所有熔断器
//
// 【功能点】验证注册中心的列表功能
// 【测试流程】
//  1. 创建注册中心
//  2. 获取 3 个不同名称的熔断器
//  3. 调用 List() 获取名称列表
//  4. 验证列表包含 3 个熔断器名称
func TestRegistry_List(t *testing.T) {
	registry := NewRegistry(nil)

	registry.Get("service-a")
	registry.Get("service-b")
	registry.Get("service-c")

	names := registry.List()

	if len(names) != 3 {
		t.Errorf("应有 3 个熔断器，实际有 %d 个", len(names))
	}
}

// TestRegistry_Stats 测试获取统计信息
//
// 【功能点】验证注册中心的统计信息聚合功能
// 【测试流程】
//  1. 创建注册中心并获取熔断器
//  2. 执行 5 次请求
//  3. 调用 Stats() 获取统计信息
//  4. 验证统计信息包含正确的请求计数
func TestRegistry_Stats(t *testing.T) {
	registry := NewRegistry(func(name string) *Config {
		return NewConfig(name, WithFailureThreshold(100))
	})

	cb := registry.Get("service-a")
	ctx := context.Background()

	// 发送一些请求
	for i := 0; i < 5; i++ {
		_ = cb.Execute(ctx, func() error { return nil })
	}

	stats := registry.Stats()

	if stat, ok := stats["service-a"]; ok {
		if stat.Counts.Requests != 5 {
			t.Errorf("请求数应为 5，实际为 %d", stat.Counts.Requests)
		}
	} else {
		t.Error("应包含 service-a 的统计信息")
	}
}

// TestRegistry_ResetAll 测试重置所有熔断器
//
// 【功能点】验证注册中心的批量重置功能
// 【测试流程】
//  1. 创建注册中心，获取多个熔断器并触发熔断
//  2. 确认所有熔断器处于 Open 状态
//  3. 调用 ResetAll() 批量重置
//  4. 验证所有熔断器恢复为 Closed 状态
func TestRegistry_ResetAll(t *testing.T) {
	registry := NewRegistry(func(name string) *Config {
		return NewConfig(name, WithFailureThreshold(1))
	})

	ctx := context.Background()

	// 触发多个熔断器熔断
	for _, name := range []string{"service-a", "service-b"} {
		cb := registry.Get(name)
		_ = cb.Execute(ctx, func() error {
			return errors.New("error")
		})
	}

	// 确认都处于打开状态
	for _, name := range []string{"service-a", "service-b"} {
		if registry.Get(name).State() != StateOpen {
			t.Fatalf("%s 应处于 Open 状态", name)
		}
	}

	// 重置所有
	registry.ResetAll()

	// 确认都处于关闭状态
	for _, name := range []string{"service-a", "service-b"} {
		if registry.Get(name).State() != StateClosed {
			t.Errorf("%s 应处于 Closed 状态", name)
		}
	}
}

// ==================== 便捷函数测试 ====================
// 测试全局便捷函数

// TestExecuteFunction 测试全局 Execute 函数
//
// 【功能点】验证使用全局 Execute 函数执行请求
// 【测试流程】
//  1. 重置全局注册中心
//  2. 调用全局 Execute 函数执行请求
//  3. 验证请求执行成功
func TestExecuteFunction(t *testing.T) {
	// 重置全局注册中心
	registryOnce = sync.Once{}
	defaultRegistry = nil

	ctx := context.Background()

	err := Execute(ctx, "test-service", func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Execute 应成功，错误: %v", err)
	}
}

// ==================== 基准测试 ====================
// 测试熔断器的性能表现

// BenchmarkExecute_Success 基准测试成功执行
//
// 【功能点】测量成功请求的执行性能
// 【测试方法】循环执行成功请求，统计 ops/sec
func BenchmarkExecute_Success(b *testing.B) {
	cb := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(ctx, func() error {
			return nil
		})
	}
}

// BenchmarkExecute_Failure 基准测试失败执行
//
// 【功能点】测量失败请求的执行性能
// 【测试方法】循环执行失败请求，统计 ops/sec
func BenchmarkExecute_Failure(b *testing.B) {
	config := NewConfig("test", WithFailureThreshold(uint32(b.N+1)))
	cb := New(config)
	ctx := context.Background()
	err := errors.New("error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Execute(ctx, func() error {
			return err
		})
	}
}

// BenchmarkExecute_Concurrent 基准测试并发执行
//
// 【功能点】测量并发场景下的执行性能
// 【测试方法】使用 RunParallel 并发执行请求，统计吞吐量
func BenchmarkExecute_Concurrent(b *testing.B) {
	cb := New(nil)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = cb.Execute(ctx, func() error {
				return nil
			})
		}
	})
}
