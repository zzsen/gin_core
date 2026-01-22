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

// TestNew 测试创建熔断器
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

// TestClosedToOpen_ConsecutiveFailures 测试连续失败触发熔断
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

// TestOpenState_RejectsRequests 测试打开状态拒绝请求
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

// TestCounts 测试计数统计
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

// TestOnStateChange 测试状态变更回调
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

// TestReset 测试手动重置
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

// TestContextCancellation 测试上下文取消
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

// TestConcurrentExecution 测试并发执行
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

// TestRegistry_Get 测试从注册中心获取熔断器
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

// TestExecuteFunction 测试全局 Execute 函数
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

// BenchmarkExecute_Success 基准测试成功执行
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
