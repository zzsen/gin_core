package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 配置中心相关 Prometheus 指标
var (
	// ConfigReloadTotal 配置加载/热更结果计数
	ConfigReloadTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "config_reload_total",
			Help: "Total etcd config-center reload attempts",
		},
		[]string{"result"}, // success | reject | error
	)
)
