// Package metrics 配置中心指标测试
//
// ==================== 测试说明 ====================
// 验证 config_reload_total Counter 可 Inc 并被 DefaultGatherer 收集。
//
// 测试覆盖内容：
// 1. success / reject / error 标签 Inc
// 2. GatherAndCount 可发现指标名
//
// 运行测试：go test ./metrics/ -count=1 -run ConfigReload -v
// ==================================================

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

// TestConfigReloadMetrics_可收集 config_reload_total 已注册
//
// 【功能点】配置重载结果计数可收集
// 【测试流程】
// 1. 分别 Inc success/reject/error
// 2. GatherAndCount 断言指标存在
func TestConfigReloadMetrics_可收集(t *testing.T) {
	ConfigReloadTotal.WithLabelValues("success").Inc()
	ConfigReloadTotal.WithLabelValues("reject").Inc()
	ConfigReloadTotal.WithLabelValues("error").Inc()

	n, err := testutil.GatherAndCount(prometheus.DefaultGatherer, "config_reload_total")
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
}
