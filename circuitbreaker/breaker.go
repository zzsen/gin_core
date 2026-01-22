// Package circuitbreaker 提供服务熔断功能
// 用于在下游服务异常时自动降级，防止级联故障
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State 熔断器状态
type State int

const (
	// StateClosed 关闭状态（正常运行）
	// 所有请求正常通过，同时统计成功/失败次数
	StateClosed State = iota

	// StateOpen 打开状态（熔断中）
	// 所有请求直接失败，不会调用下游服务
	StateOpen

	// StateHalfOpen 半开状态（探测中）
	// 允许部分请求通过，用于探测服务是否恢复
	StateHalfOpen
)

// String 返回状态的字符串表示
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// 预定义错误
var (
	// ErrCircuitOpen 熔断器处于打开状态
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrTooManyRequests 半开状态下请求过多
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// Counts 请求计数统计
type Counts struct {
	Requests             uint32 // 总请求数
	TotalSuccesses       uint32 // 成功总数
	TotalFailures        uint32 // 失败总数
	ConsecutiveSuccesses uint32 // 连续成功数
	ConsecutiveFailures  uint32 // 连续失败数
}

// CircuitBreaker 熔断器
// 实现了断路器模式，用于保护下游服务
//
// 状态转换：
//   - Closed -> Open: 连续失败次数达到阈值，或失败率达到阈值
//   - Open -> HalfOpen: 超时时间到达后自动转换
//   - HalfOpen -> Closed: 探测请求连续成功
//   - HalfOpen -> Open: 探测请求失败
type CircuitBreaker struct {
	name          string
	config        *Config
	mu            sync.Mutex
	state         State
	counts        Counts
	expiry        time.Time // 状态过期时间
	halfOpenCount uint32    // 半开状态下的请求数
}

// New 创建熔断器
// 参数：
//   - config: 熔断器配置，为 nil 时使用默认配置
//
// 返回：
//   - *CircuitBreaker: 熔断器实例
func New(config *Config) *CircuitBreaker {
	if config == nil {
		config = DefaultConfig("default")
	}
	return &CircuitBreaker{
		name:   config.Name,
		config: config,
		state:  StateClosed,
		expiry: time.Now().Add(config.Interval),
	}
}

// Execute 执行受熔断器保护的函数
// 参数：
//   - ctx: 上下文，用于取消控制
//   - fn: 要执行的函数，返回 error 表示失败
//
// 返回：
//   - error: 执行错误或熔断错误
//
// 使用示例：
//
//	err := breaker.Execute(ctx, func() error {
//	    resp, err := http.Get("http://service/api")
//	    if err != nil {
//	        return err
//	    }
//	    defer resp.Body.Close()
//	    return nil
//	})
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 检查是否允许执行
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	// 执行业务逻辑
	err := fn()

	// 记录结果
	cb.afterRequest(err == nil)

	return err
}

// beforeRequest 请求前检查
func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	switch state {
	case StateClosed:
		return nil
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.config.MaxRequests {
			return ErrTooManyRequests
		}
		cb.halfOpenCount++
		return nil
	}
	return nil
}

// afterRequest 请求后记录结果
func (cb *CircuitBreaker) afterRequest(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess 成功处理
func (cb *CircuitBreaker) onSuccess(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.Requests++
		cb.counts.TotalSuccesses++
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
	case StateHalfOpen:
		cb.counts.ConsecutiveSuccesses++
		cb.counts.ConsecutiveFailures = 0
		// 连续成功达到阈值，关闭熔断器
		if cb.counts.ConsecutiveSuccesses >= cb.config.MaxRequests {
			cb.setState(StateClosed, now)
		}
	}
}

// onFailure 失败处理
func (cb *CircuitBreaker) onFailure(state State, now time.Time) {
	switch state {
	case StateClosed:
		cb.counts.Requests++
		cb.counts.TotalFailures++
		cb.counts.ConsecutiveFailures++
		cb.counts.ConsecutiveSuccesses = 0

		// 检查是否需要打开熔断器
		if cb.shouldTrip() {
			cb.setState(StateOpen, now)
		}
	case StateHalfOpen:
		// 半开状态下失败，重新打开熔断器
		cb.setState(StateOpen, now)
	}
}

// shouldTrip 判断是否应该触发熔断
func (cb *CircuitBreaker) shouldTrip() bool {
	// 连续失败次数达到阈值
	if cb.counts.ConsecutiveFailures >= cb.config.FailureThreshold {
		return true
	}

	// 失败率达到阈值
	if cb.counts.Requests >= cb.config.MinRequests {
		ratio := float64(cb.counts.TotalFailures) / float64(cb.counts.Requests)
		if ratio >= cb.config.FailureRatio {
			return true
		}
	}

	return false
}

// currentState 获取当前状态（处理状态转换）
func (cb *CircuitBreaker) currentState(now time.Time) State {
	switch cb.state {
	case StateClosed:
		// 检查统计周期是否过期
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.reset(now)
		}
	case StateOpen:
		// 检查是否可以进入半开状态
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state
}

// setState 设置状态
func (cb *CircuitBreaker) setState(state State, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state
	cb.reset(now)

	// 触发回调
	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.name, prev, state)
	}
}

// reset 重置计数器
func (cb *CircuitBreaker) reset(now time.Time) {
	cb.counts = Counts{}
	cb.halfOpenCount = 0

	switch cb.state {
	case StateClosed:
		cb.expiry = now.Add(cb.config.Interval)
	case StateOpen:
		cb.expiry = now.Add(cb.config.Timeout)
	case StateHalfOpen:
		cb.expiry = time.Time{}
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.currentState(time.Now())
}

// Counts 获取当前计数
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.counts
}

// Name 获取熔断器名称
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// Reset 手动重置熔断器到关闭状态
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = StateClosed
	cb.reset(time.Now())
}
