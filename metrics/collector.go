// Package metrics 提供 Prometheus 指标监控功能
package metrics

import (
	"context"
	"time"

	"github.com/zzsen/gin_core/app"
)

// StartCollector 启动指标收集器
// 定期收集数据库和 Redis 连接池指标
func StartCollector(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				collectDBStats()
				collectRedisStats()
			}
		}
	}()
}

// collectDBStats 收集数据库连接池统计
func collectDBStats() {
	if app.DB == nil {
		return
	}

	sqlDB, err := app.DB.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()
	DbPoolOpenConnections.Set(float64(stats.OpenConnections))
	DbPoolIdleConnections.Set(float64(stats.Idle))
	DbPoolInUseConnections.Set(float64(stats.InUse))
	DbPoolWaitCount.Set(float64(stats.WaitCount))
}

// collectRedisStats 收集 Redis 连接池统计
func collectRedisStats() {
	if app.Redis == nil {
		return
	}

	stats := app.Redis.PoolStats()
	RedisPoolHits.Set(float64(stats.Hits))
	RedisPoolMisses.Set(float64(stats.Misses))
	RedisPoolTotalConns.Set(float64(stats.TotalConns))
	RedisPoolIdleConns.Set(float64(stats.IdleConns))
}
