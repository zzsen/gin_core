package app

import (
	"context"
	"time"

	"github.com/zzsen/gin_core/logger"
)

// PoolStats 连接池统计信息
type PoolStats struct {
	// 数据库连接池
	DBMaxOpenConns      int   `json:"db_max_open_conns"`      // 最大打开连接数
	DBOpenConns         int   `json:"db_open_conns"`          // 当前打开连接数
	DBInUse             int   `json:"db_in_use"`              // 使用中的连接数
	DBIdle              int   `json:"db_idle"`                // 空闲连接数
	DBWaitCount         int64 `json:"db_wait_count"`          // 等待连接的总次数
	DBWaitDurationMs    int64 `json:"db_wait_duration_ms"`    // 等待连接的总时间(毫秒)
	DBMaxIdleClosed     int64 `json:"db_max_idle_closed"`     // 因超过空闲连接数被关闭的连接数
	DBMaxLifetimeClosed int64 `json:"db_max_lifetime_closed"` // 因超过生命周期被关闭的连接数

	// Redis连接池
	RedisPoolSize    int `json:"redis_pool_size"`    // 连接池大小
	RedisActiveConns int `json:"redis_active_conns"` // 活跃连接数
	RedisIdleConns   int `json:"redis_idle_conns"`   // 空闲连接数
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy bool           `json:"healthy"`
	Error   string         `json:"error,omitempty"`
	Stats   map[string]int `json:"stats,omitempty"`
}

// GetPoolStats 获取所有连接池统计信息
func GetPoolStats() *PoolStats {
	stats := &PoolStats{}

	// 获取数据库连接池统计
	if DB != nil {
		if sqlDB, err := DB.DB(); err == nil {
			dbStats := sqlDB.Stats()
			stats.DBMaxOpenConns = dbStats.MaxOpenConnections
			stats.DBOpenConns = dbStats.OpenConnections
			stats.DBInUse = dbStats.InUse
			stats.DBIdle = dbStats.Idle
			stats.DBWaitCount = dbStats.WaitCount
			stats.DBWaitDurationMs = dbStats.WaitDuration.Milliseconds()
			stats.DBMaxIdleClosed = dbStats.MaxIdleClosed
			stats.DBMaxLifetimeClosed = dbStats.MaxLifetimeClosed
		}
	}

	// 获取Redis连接池统计
	if Redis != nil {
		poolStats := Redis.PoolStats()
		stats.RedisPoolSize = int(poolStats.TotalConns)
		stats.RedisActiveConns = int(poolStats.TotalConns - poolStats.IdleConns)
		stats.RedisIdleConns = int(poolStats.IdleConns)
	}

	return stats
}

// CheckPoolHealth 检查连接池健康状态
func CheckPoolHealth() map[string]HealthStatus {
	result := make(map[string]HealthStatus)

	// 检查数据库
	if BaseConfig.System.UseMysql && DB != nil {
		status := HealthStatus{Healthy: true}
		sqlDB, err := DB.DB()
		if err != nil {
			status.Healthy = false
			status.Error = err.Error()
		} else {
			// Ping 检查
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := sqlDB.PingContext(ctx); err != nil {
				status.Healthy = false
				status.Error = err.Error()
			}
			// 连接池状态
			dbStats := sqlDB.Stats()
			status.Stats = map[string]int{
				"max_open": dbStats.MaxOpenConnections,
				"open":     dbStats.OpenConnections,
				"in_use":   dbStats.InUse,
				"idle":     dbStats.Idle,
			}
			// 告警：连接池使用率过高（超过80%）
			if dbStats.MaxOpenConnections > 0 && dbStats.InUse > dbStats.MaxOpenConnections*80/100 {
				logger.Warn("[连接池] MySQL连接使用率过高: %d/%d (%.1f%%)",
					dbStats.InUse, dbStats.MaxOpenConnections,
					float64(dbStats.InUse)*100/float64(dbStats.MaxOpenConnections))
			}
		}
		result["mysql"] = status
	}

	// 检查Redis
	if BaseConfig.System.UseRedis && Redis != nil {
		status := HealthStatus{Healthy: true}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := Redis.Ping(ctx).Err(); err != nil {
			status.Healthy = false
			status.Error = err.Error()
		}
		poolStats := Redis.PoolStats()
		status.Stats = map[string]int{
			"total": int(poolStats.TotalConns),
			"idle":  int(poolStats.IdleConns),
		}
		result["redis"] = status
	}

	// 检查Elasticsearch
	if BaseConfig.System.UseEs && ES != nil {
		status := HealthStatus{Healthy: true}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := ES.Info().Do(ctx)
		if err != nil {
			status.Healthy = false
			status.Error = err.Error()
		}
		result["elasticsearch"] = status
	}

	// 检查Etcd
	if BaseConfig.System.UseEtcd && Etcd != nil {
		status := HealthStatus{Healthy: true}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, err := Etcd.Status(ctx, Etcd.Endpoints()[0])
		if err != nil {
			status.Healthy = false
			status.Error = err.Error()
		}
		result["etcd"] = status
	}

	return result
}

// IsAllHealthy 检查所有服务是否健康
func IsAllHealthy() bool {
	health := CheckPoolHealth()
	for _, status := range health {
		if !status.Healthy {
			return false
		}
	}
	return true
}
