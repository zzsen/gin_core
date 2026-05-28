// Package circuitbreaker 熔断器 Prometheus 指标测试
//
// ==================== 测试说明 ====================
// 本文件包含熔断器 Prometheus 指标采集的单元测试。
//
// 测试覆盖内容：
// 1. BreakerCollector Describe/Collect 正确输出指标
// 2. RecordSuccess/RecordFailure Counter 递增
// 3. RecordRejected Counter 递增
// 4. RecordStateTransition Counter 递增
// 5. state gauge 反映当前熔断器状态
//
// 运行测试：go test -v ./circuitbreaker/... -run TestMetrics
// ==================================================
package circuitbreaker

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBreakerCollector_Describe 测试 Describe 输出
//
// 【功能点】验证 Collector 正确声明所有指标描述
// 【测试流程】
// 1. 创建 Collector
// 2. 调用 Describe 收集所有描述
// 3. 验证包含 state gauge 和 counter 指标
func TestBreakerCollector_Describe(t *testing.T) {
	registry := NewRegistry(nil)
	collector := NewBreakerCollector(registry)

	ch := make(chan *prometheus.Desc, 20)
	go func() {
		collector.Describe(ch)
		close(ch)
	}()

	var descs []*prometheus.Desc
	for desc := range ch {
		descs = append(descs, desc)
	}

	assert.True(t, len(descs) >= 4, "should have at least 4 descriptors")
}

// TestBreakerCollector_Collect_StateGauge 测试状态 Gauge 采集
//
// 【功能点】验证 Collect 输出正确的熔断器状态值
// 【测试流程】
// 1. 注册一个 Closed 状态的熔断器
// 2. Collect 应输出 state=0（Closed）
func TestBreakerCollector_Collect_StateGauge(t *testing.T) {
	registry := NewRegistry(nil)
	registry.Get("test-service")
	collector := NewBreakerCollector(registry)

	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(collector)

	metrics, err := reg.Gather()
	require.NoError(t, err)

	found := false
	for _, mf := range metrics {
		if mf.GetName() == "circuit_breaker_state" {
			found = true
			require.Len(t, mf.GetMetric(), 1)
			assert.Equal(t, float64(StateClosed), mf.GetMetric()[0].GetGauge().GetValue())
		}
	}
	assert.True(t, found, "circuit_breaker_state metric should exist")
}

// TestBreakerCollector_RecordSuccess 测试成功请求计数
//
// 【功能点】RecordSuccess 后 requests_total{result=success} 应递增
func TestBreakerCollector_RecordSuccess(t *testing.T) {
	registry := NewRegistry(nil)
	collector := NewBreakerCollector(registry)

	collector.RecordSuccess("svc-a")
	collector.RecordSuccess("svc-a")

	expected := `
# HELP circuit_breaker_requests_total Total number of requests processed by circuit breakers
# TYPE circuit_breaker_requests_total counter
circuit_breaker_requests_total{name="svc-a",result="success"} 2
`
	err := testutil.CollectAndCompare(collector.metrics.requestsTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}

// TestBreakerCollector_RecordFailure 测试失败请求计数
//
// 【功能点】RecordFailure 后 requests_total{result=failure} 应递增
func TestBreakerCollector_RecordFailure(t *testing.T) {
	registry := NewRegistry(nil)
	collector := NewBreakerCollector(registry)

	collector.RecordFailure("svc-b")

	expected := `
# HELP circuit_breaker_requests_total Total number of requests processed by circuit breakers
# TYPE circuit_breaker_requests_total counter
circuit_breaker_requests_total{name="svc-b",result="failure"} 1
`
	err := testutil.CollectAndCompare(collector.metrics.requestsTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}

// TestBreakerCollector_RecordRejected 测试被拒绝请求计数
//
// 【功能点】RecordRejected 后 rejected_total 应递增
func TestBreakerCollector_RecordRejected(t *testing.T) {
	registry := NewRegistry(nil)
	collector := NewBreakerCollector(registry)

	collector.RecordRejected("svc-c")
	collector.RecordRejected("svc-c")
	collector.RecordRejected("svc-c")

	expected := `
# HELP circuit_breaker_rejected_total Total number of requests rejected by open circuit breakers
# TYPE circuit_breaker_rejected_total counter
circuit_breaker_rejected_total{name="svc-c"} 3
`
	err := testutil.CollectAndCompare(collector.metrics.rejectedTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}

// TestBreakerCollector_RecordStateTransition 测试状态转换计数
//
// 【功能点】RecordStateTransition 后 state_transitions_total 应递增
func TestBreakerCollector_RecordStateTransition(t *testing.T) {
	registry := NewRegistry(nil)
	collector := NewBreakerCollector(registry)

	collector.RecordStateTransition("svc-d", StateClosed, StateOpen)

	expected := `
# HELP circuit_breaker_state_transitions_total Total number of state transitions
# TYPE circuit_breaker_state_transitions_total counter
circuit_breaker_state_transitions_total{from="closed",name="svc-d",to="open"} 1
`
	err := testutil.CollectAndCompare(collector.metrics.transitionsTotal, strings.NewReader(expected))
	assert.NoError(t, err)
}

// TestEnableMetrics_IsMetricsEnabled 中文描述：启用指标开关与全局标志
//
// 【功能点】验证 EnableMetrics 将 IsMetricsEnabled 置为 true 并返回 Collector
// 【测试流程】
// 1. 若进程内已启用则 Skip（避免 prometheus.MustRegister 重复panic）
// 2. 调用 EnableMetrics(NewRegistry(nil))
// 3. 断言返回值非空且 IsMetricsEnabled() 为 true
func TestEnableMetrics_IsMetricsEnabled(t *testing.T) {
	if IsMetricsEnabled() {
		t.Skip("EnableMetrics 仅首次注册到默认 Prometheus Registry；已启用则跳过以防 MustRegister 冲突")
	}

	reg := NewRegistry(nil)
	col := EnableMetrics(reg)
	require.NotNil(t, col)
	assert.True(t, IsMetricsEnabled())
}
