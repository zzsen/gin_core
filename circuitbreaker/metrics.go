// Package circuitbreaker Prometheus 指标采集
package circuitbreaker

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricsEnabled bool
	metricsOnce    sync.Once
)

// IsMetricsEnabled 返回 Prometheus 指标是否已启用
func IsMetricsEnabled() bool {
	return metricsEnabled
}

// breakerMetrics 熔断器 Prometheus 指标集
type breakerMetrics struct {
	requestsTotal    *prometheus.CounterVec
	rejectedTotal    *prometheus.CounterVec
	transitionsTotal *prometheus.CounterVec
}

func newBreakerMetrics() *breakerMetrics {
	return &breakerMetrics{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "circuit_breaker_requests_total",
			Help: "Total number of requests processed by circuit breakers",
		}, []string{"name", "result"}),
		rejectedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "circuit_breaker_rejected_total",
			Help: "Total number of requests rejected by open circuit breakers",
		}, []string{"name"}),
		transitionsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "circuit_breaker_state_transitions_total",
			Help: "Total number of state transitions",
		}, []string{"name", "from", "to"}),
	}
}

// BreakerCollector 实现 prometheus.Collector 接口
// 在 Describe/Collect 时从 Registry 实时采集状态
type BreakerCollector struct {
	registry   *Registry
	stateGauge *prometheus.Desc
	metrics    *breakerMetrics
}

// NewBreakerCollector 创建熔断器指标采集器
func NewBreakerCollector(registry *Registry) *BreakerCollector {
	return &BreakerCollector{
		registry: registry,
		stateGauge: prometheus.NewDesc(
			"circuit_breaker_state",
			"Current state of the circuit breaker (0=closed, 1=open, 2=half-open)",
			[]string{"name"}, nil,
		),
		metrics: newBreakerMetrics(),
	}
}

// Describe 实现 prometheus.Collector
func (c *BreakerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.stateGauge
	c.metrics.requestsTotal.Describe(ch)
	c.metrics.rejectedTotal.Describe(ch)
	c.metrics.transitionsTotal.Describe(ch)
}

// Collect 实现 prometheus.Collector
func (c *BreakerCollector) Collect(ch chan<- prometheus.Metric) {
	for name, stats := range c.registry.Stats() {
		ch <- prometheus.MustNewConstMetric(
			c.stateGauge, prometheus.GaugeValue,
			float64(stats.State), name,
		)
	}
	c.metrics.requestsTotal.Collect(ch)
	c.metrics.rejectedTotal.Collect(ch)
	c.metrics.transitionsTotal.Collect(ch)
}

// RecordSuccess 记录成功请求
func (c *BreakerCollector) RecordSuccess(name string) {
	c.metrics.requestsTotal.WithLabelValues(name, "success").Inc()
}

// RecordFailure 记录失败请求
func (c *BreakerCollector) RecordFailure(name string) {
	c.metrics.requestsTotal.WithLabelValues(name, "failure").Inc()
}

// RecordRejected 记录被拒绝的请求
func (c *BreakerCollector) RecordRejected(name string) {
	c.metrics.rejectedTotal.WithLabelValues(name).Inc()
}

// RecordStateTransition 记录状态转换
func (c *BreakerCollector) RecordStateTransition(name string, from, to State) {
	c.metrics.transitionsTotal.WithLabelValues(name, from.String(), to.String()).Inc()
}

// EnableMetrics 启用 Prometheus 指标采集（幂等）
func EnableMetrics(registry *Registry) *BreakerCollector {
	var collector *BreakerCollector
	metricsOnce.Do(func() {
		metricsEnabled = true
		collector = NewBreakerCollector(registry)
		prometheus.MustRegister(collector)
	})
	return collector
}
