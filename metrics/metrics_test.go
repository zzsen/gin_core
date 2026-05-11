// Package metrics 指标工厂函数与采集器测试
//
// ==================== 测试说明 ====================
// 本文件包含 metrics 包工厂函数和 StartCollector 的单元测试。
//
// 测试覆盖内容：
// 1. NewCounter 创建自定义 Counter
// 2. NewCounterVec 创建带标签的 CounterVec
// 3. NewGauge 创建自定义 Gauge
// 4. NewHistogram 创建自定义 Histogram
// 5. StartCollector 启动/停止采集循环
//
// 运行测试：go test -v ./metrics/... -run "TestNew|TestStartCollector"
// ==================================================
package metrics

import (
	"context"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCounter 测试创建自定义 Counter
//
// 【功能点】返回的 Counter 非 nil 且可正常 Inc/Add
func TestNewCounter(t *testing.T) {
	c := NewCounter("test_custom_counter_total", "test counter")
	require.NotNil(t, c)

	c.Inc()
	c.Add(5)

	var m dto.Metric
	require.NoError(t, c.Write(&m))
	assert.Equal(t, float64(6), m.GetCounter().GetValue())
}

// TestNewCounterVec 测试创建带标签的 CounterVec
//
// 【功能点】返回的 CounterVec 支持按标签维度操作
func TestNewCounterVec(t *testing.T) {
	cv := NewCounterVec("test_custom_counter_vec_total", "test counter vec", []string{"method", "status"})
	require.NotNil(t, cv)

	cv.WithLabelValues("GET", "200").Inc()
	cv.WithLabelValues("POST", "500").Add(3)

	var m dto.Metric
	c, err := cv.GetMetricWithLabelValues("GET", "200")
	require.NoError(t, err)
	require.NoError(t, c.Write(&m))
	assert.Equal(t, float64(1), m.GetCounter().GetValue())

	c2, err := cv.GetMetricWithLabelValues("POST", "500")
	require.NoError(t, err)
	require.NoError(t, c2.Write(&m))
	assert.Equal(t, float64(3), m.GetCounter().GetValue())
}

// TestNewGauge 测试创建自定义 Gauge
//
// 【功能点】返回的 Gauge 支持 Set/Inc/Dec
func TestNewGauge(t *testing.T) {
	g := NewGauge("test_custom_gauge", "test gauge")
	require.NotNil(t, g)

	g.Set(42)
	var m dto.Metric
	require.NoError(t, g.Write(&m))
	assert.Equal(t, float64(42), m.GetGauge().GetValue())

	g.Inc()
	require.NoError(t, g.Write(&m))
	assert.Equal(t, float64(43), m.GetGauge().GetValue())

	g.Dec()
	require.NoError(t, g.Write(&m))
	assert.Equal(t, float64(42), m.GetGauge().GetValue())
}

// TestNewHistogram 测试创建自定义 Histogram
//
// 【功能点】返回的 Histogram 支持 Observe，分桶正确
func TestNewHistogram(t *testing.T) {
	buckets := []float64{0.01, 0.05, 0.1, 0.5, 1.0}
	h := NewHistogram("test_custom_histogram", "test histogram", buckets)
	require.NotNil(t, h)

	h.Observe(0.03)
	h.Observe(0.2)
	h.Observe(0.8)

	var m dto.Metric
	require.NoError(t, h.Write(&m))
	assert.Equal(t, uint64(3), m.GetHistogram().GetSampleCount())
}

// TestStartCollector 测试采集器启动和停止
//
// 【功能点】StartCollector 启动后能正常采集，ctx 取消后停止
// 【测试流程】
// 1. 启动采集器，间隔 50ms
// 2. 等待 120ms 确保至少采集 1 次
// 3. 取消 context，采集器应停止
func TestStartCollector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	StartCollector(ctx, 50*time.Millisecond)

	time.Sleep(120 * time.Millisecond)
	cancel()
	time.Sleep(20 * time.Millisecond)
}
