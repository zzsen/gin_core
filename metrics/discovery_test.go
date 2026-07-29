package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// TestDiscoveryMetrics_可收集 discovery 相关 Counter 已注册到默认 gatherer
//
// 【功能点】keepalive rebuild / health probe / health transition 指标可 Inc 并 Collect
func TestDiscoveryMetrics_可收集(t *testing.T) {
	DiscoveryKeepaliveRebuildTotal.WithLabelValues("success").Inc()
	DiscoveryHealthProbeTotal.WithLabelValues("ok").Inc()
	DiscoveryHealthTransitionTotal.WithLabelValues("healthy", "degraded").Inc()

	n, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "discovery_keepalive_rebuild_total")
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)

	n, err = testutil.GatherAndCount(prometheus.DefaultGatherer, "discovery_health_probe_total")
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)

	n, err = testutil.GatherAndCount(prometheus.DefaultGatherer, "discovery_health_transition_total")
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
}
