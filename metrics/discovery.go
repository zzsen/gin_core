package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Discovery 相关 Prometheus 指标（服务注册/发现增强）
var (
	// DiscoveryKeepaliveRebuildTotal KeepAlive 重建次数
	DiscoveryKeepaliveRebuildTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "discovery_keepalive_rebuild_total",
			Help: "Total discovery keepalive lease rebuild attempts",
		},
		[]string{"result"}, // success | fail
	)

	// DiscoveryHealthProbeTotal ready 探测次数
	DiscoveryHealthProbeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "discovery_health_probe_total",
			Help: "Total discovery readiness probes for health unlink",
		},
		[]string{"result"}, // ok | fail
	)

	// DiscoveryHealthTransitionTotal 健康摘除状态迁移次数
	DiscoveryHealthTransitionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "discovery_health_transition_total",
			Help: "Total discovery health unlink state transitions",
		},
		[]string{"from", "to"}, // healthy | degraded | unlinked
	)
)
