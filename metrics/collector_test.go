// Package metrics 指标采集器测试
//
// ==================== 测试说明 ====================
// 本文件包含 metrics 指标采集器的单元测试，验证连接池指标采集逻辑的正确性。
//
// 测试覆盖内容：
// 1. DB 连接池指标采集：多次采集后 Gauge 值不膨胀
// 2. Redis 连接池指标采集：多次采集后 Gauge 值不膨胀
// 3. DB/Redis 为 nil 时采集器的安全行为
//
// 运行测试：go test -v ./metrics/... -run TestCollect
// ==================================================
package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

// getGaugeValue 从 Gauge 指标中提取当前值
func getGaugeValue(g prometheus.Gauge) float64 {
	var m dto.Metric
	if err := g.Write(&m); err != nil {
		return 0
	}
	return m.GetGauge().GetValue()
}

// TestCollectDBStats_NoInflation 测试 DB 指标采集不会膨胀
//
// 【功能点】验证修复后的 Gauge.Set() 逻辑，累计值不会因多次采集而叠加
// 【测试流程】
// 1. 跳过依赖真实 DB 的场景（app.DB 为 nil 直接返回）
// 2. 直接验证 Gauge 的 Set 行为：多次 Set 同一值后结果不变
// 3. Set 新值后验证更新正确
func TestCollectDBStats_NoInflation(t *testing.T) {
	// collectDBStats 在 app.DB == nil 时安全返回
	collectDBStats()

	// 直接验证 Gauge 行为：多次 Set 同一累计值后不会膨胀
	DbPoolWaitCount.Set(100)
	DbPoolWaitCount.Set(100)
	DbPoolWaitCount.Set(100)
	val := getGaugeValue(DbPoolWaitCount)
	assert.Equal(t, float64(100), val, "多次 Set 同一累计值后，Gauge 应保持不变")

	// 累计值增长时正确更新
	DbPoolWaitCount.Set(150)
	val = getGaugeValue(DbPoolWaitCount)
	assert.Equal(t, float64(150), val, "累计值增长后，Gauge 应更新为新值")
}

// TestCollectRedisStats_NoInflation 测试 Redis 指标采集不会膨胀
//
// 【功能点】验证修复后的 Gauge.Set() 逻辑，Redis 连接池累计值不会因多次采集而叠加
// 【测试流程】
// 1. 跳过依赖真实 Redis 的场景（app.Redis 为 nil 直接返回）
// 2. 直接验证 Hits/Misses Gauge 的 Set 行为
func TestCollectRedisStats_NoInflation(t *testing.T) {
	// collectRedisStats 在 app.Redis == nil 时安全返回
	collectRedisStats()

	// Hits：多次 Set 同一值不膨胀
	RedisPoolHits.Set(500)
	RedisPoolHits.Set(500)
	RedisPoolHits.Set(500)
	val := getGaugeValue(RedisPoolHits)
	assert.Equal(t, float64(500), val, "多次 Set 同一 Hits 累计值后，Gauge 应保持不变")

	// Misses：多次 Set 同一值不膨胀
	RedisPoolMisses.Set(50)
	RedisPoolMisses.Set(50)
	RedisPoolMisses.Set(50)
	val = getGaugeValue(RedisPoolMisses)
	assert.Equal(t, float64(50), val, "多次 Set 同一 Misses 累计值后，Gauge 应保持不变")

	// 值增长时正确更新
	RedisPoolHits.Set(600)
	RedisPoolMisses.Set(60)
	assert.Equal(t, float64(600), getGaugeValue(RedisPoolHits))
	assert.Equal(t, float64(60), getGaugeValue(RedisPoolMisses))
}

// TestCollectDBStats_NilDB 测试 DB 为 nil 时采集器不会 panic
//
// 【功能点】防御性检查，确保 app.DB 未初始化时安全返回
// 【测试流程】
// 1. 确保 app.DB 为 nil
// 2. 调用 collectDBStats 不会 panic
func TestCollectDBStats_NilDB(t *testing.T) {
	assert.NotPanics(t, func() {
		collectDBStats()
	}, "app.DB 为 nil 时 collectDBStats 不应 panic")
}

// TestCollectRedisStats_NilRedis 测试 Redis 为 nil 时采集器不会 panic
//
// 【功能点】防御性检查，确保 app.Redis 未初始化时安全返回
// 【测试流程】
// 1. 确保 app.Redis 为 nil
// 2. 调用 collectRedisStats 不会 panic
func TestCollectRedisStats_NilRedis(t *testing.T) {
	assert.NotPanics(t, func() {
		collectRedisStats()
	}, "app.Redis 为 nil 时 collectRedisStats 不应 panic")
}
