// Package metrics 提供 Prometheus 指标监控功能
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP 请求相关指标
var (
	// HttpRequestsTotal HTTP 请求总数
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HttpRequestDuration HTTP 请求耗时（秒）
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// HttpRequestsInFlight 当前处理中的请求数
	HttpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)
)

// 数据库连接池指标
var (
	// DbPoolOpenConnections 数据库打开连接数
	DbPoolOpenConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_open_connections",
			Help: "Number of open database connections",
		},
	)

	// DbPoolIdleConnections 数据库空闲连接数
	DbPoolIdleConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_idle_connections",
			Help: "Number of idle database connections",
		},
	)

	// DbPoolInUseConnections 数据库使用中连接数
	DbPoolInUseConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_pool_in_use_connections",
			Help: "Number of database connections currently in use",
		},
	)

	// DbPoolWaitCount 等待连接总次数
	DbPoolWaitCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "db_pool_wait_count_total",
			Help: "Total number of times waited for a connection",
		},
	)
)

// Redis 连接池指标
var (
	// RedisPoolHits Redis 连接池命中次数
	RedisPoolHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_pool_hits_total",
			Help: "Total number of times a free connection was found in the pool",
		},
	)

	// RedisPoolMisses Redis 连接池未命中次数
	RedisPoolMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_pool_misses_total",
			Help: "Total number of times a free connection was not found in the pool",
		},
	)

	// RedisPoolTotalConns Redis 连接池总连接数
	RedisPoolTotalConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_total_connections",
			Help: "Total number of connections in the Redis pool",
		},
	)

	// RedisPoolIdleConns Redis 连接池空闲连接数
	RedisPoolIdleConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_pool_idle_connections",
			Help: "Number of idle connections in the Redis pool",
		},
	)
)

// NewCounter 创建自定义计数器
func NewCounter(name, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: name,
		Help: help,
	})
}

// NewCounterVec 创建带标签的自定义计数器
func NewCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	return promauto.NewCounterVec(prometheus.CounterOpts{
		Name: name,
		Help: help,
	}, labels)
}

// NewGauge 创建自定义仪表
func NewGauge(name, help string) prometheus.Gauge {
	return promauto.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: help,
	})
}

// NewHistogram 创建自定义直方图
func NewHistogram(name, help string, buckets []float64) prometheus.Histogram {
	return promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    name,
		Help:    help,
		Buckets: buckets,
	})
}
