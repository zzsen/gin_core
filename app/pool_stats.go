package app

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zzsen/gin_core/logger"
	"gorm.io/gorm"
)

// DBInstanceStats 单个数据库实例的连接池统计信息
type DBInstanceStats struct {
	MaxOpenConns      int   `json:"max_open_conns"`      // 最大打开连接数
	OpenConns         int   `json:"open_conns"`          // 当前打开连接数
	InUse             int   `json:"in_use"`              // 使用中的连接数
	Idle              int   `json:"idle"`                // 空闲连接数
	WaitCount         int64 `json:"wait_count"`          // 等待连接的总次数
	WaitDurationMs    int64 `json:"wait_duration_ms"`    // 等待连接的总时间(毫秒)
	MaxIdleClosed     int64 `json:"max_idle_closed"`     // 因超过空闲连接数被关闭的连接数
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"` // 因超过生命周期被关闭的连接数
}

// RedisInstanceStats 单个 Redis 实例的连接池统计信息
type RedisInstanceStats struct {
	PoolSize    int `json:"pool_size"`    // 连接池大小
	ActiveConns int `json:"active_conns"` // 活跃连接数
	IdleConns   int `json:"idle_conns"`   // 空闲连接数
}

// PoolStats 连接池统计信息
type PoolStats struct {
	// 主数据库连接池
	DBMaxOpenConns      int   `json:"db_max_open_conns"`
	DBOpenConns         int   `json:"db_open_conns"`
	DBInUse             int   `json:"db_in_use"`
	DBIdle              int   `json:"db_idle"`
	DBWaitCount         int64 `json:"db_wait_count"`
	DBWaitDurationMs    int64 `json:"db_wait_duration_ms"`
	DBMaxIdleClosed     int64 `json:"db_max_idle_closed"`
	DBMaxLifetimeClosed int64 `json:"db_max_lifetime_closed"`

	// 主 Redis 连接池
	RedisPoolSize    int `json:"redis_pool_size"`
	RedisActiveConns int `json:"redis_active_conns"`
	RedisIdleConns   int `json:"redis_idle_conns"`

	// 多数据库连接池统计（按别名索引）
	DBListStats map[string]*DBInstanceStats `json:"db_list_stats,omitempty"`
	// 数据库解析器连接池统计（读写分离）
	DBResolverStats *DBInstanceStats `json:"db_resolver_stats,omitempty"`
	// 多 Redis 连接池统计（按别名索引）
	RedisListStats map[string]*RedisInstanceStats `json:"redis_list_stats,omitempty"`
}

// HealthStatus 健康状态
type HealthStatus struct {
	Healthy bool           `json:"healthy"`
	Error   string         `json:"error,omitempty"`
	Stats   map[string]int `json:"stats,omitempty"`
}

// collectDBStats 从 gorm.DB 实例中采集连接池统计信息
func collectDBStats(db *gorm.DB) *DBInstanceStats {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil
	}
	s := sqlDB.Stats()
	return &DBInstanceStats{
		MaxOpenConns:      s.MaxOpenConnections,
		OpenConns:         s.OpenConnections,
		InUse:             s.InUse,
		Idle:              s.Idle,
		WaitCount:         s.WaitCount,
		WaitDurationMs:    s.WaitDuration.Milliseconds(),
		MaxIdleClosed:     s.MaxIdleClosed,
		MaxLifetimeClosed: s.MaxLifetimeClosed,
	}
}

// collectRedisStats 从 Redis 客户端实例中采集连接池统计信息
func collectRedisStats(client redis.UniversalClient) *RedisInstanceStats {
	if client == nil {
		return nil
	}
	ps := client.PoolStats()
	return &RedisInstanceStats{
		PoolSize:    int(ps.TotalConns),
		ActiveConns: int(ps.TotalConns - ps.IdleConns),
		IdleConns:   int(ps.IdleConns),
	}
}

// GetPoolStats 获取所有连接池统计信息
//
// 采集范围：
// 1. 主数据库（app.DB）
// 2. 数据库解析器（app.DBResolver，读写分离）
// 3. 多数据库列表（app.DBList）
// 4. 主 Redis（app.Redis）
// 5. 多 Redis 列表（app.RedisList）
func GetPoolStats() *PoolStats {
	stats := &PoolStats{}

	// 1. 主数据库连接池统计
	if dbStats := collectDBStats(DB); dbStats != nil {
		stats.DBMaxOpenConns = dbStats.MaxOpenConns
		stats.DBOpenConns = dbStats.OpenConns
		stats.DBInUse = dbStats.InUse
		stats.DBIdle = dbStats.Idle
		stats.DBWaitCount = dbStats.WaitCount
		stats.DBWaitDurationMs = dbStats.WaitDurationMs
		stats.DBMaxIdleClosed = dbStats.MaxIdleClosed
		stats.DBMaxLifetimeClosed = dbStats.MaxLifetimeClosed
	}

	// 2. 数据库解析器连接池统计
	stats.DBResolverStats = collectDBStats(DBResolver)

	// 3. 多数据库列表连接池统计
	lock.RLock()
	if len(DBList) > 0 {
		stats.DBListStats = make(map[string]*DBInstanceStats, len(DBList))
		for name, db := range DBList {
			if s := collectDBStats(db); s != nil {
				stats.DBListStats[name] = s
			}
		}
	}

	// 5. 多 Redis 列表连接池统计
	if len(RedisList) > 0 {
		stats.RedisListStats = make(map[string]*RedisInstanceStats, len(RedisList))
		for name, client := range RedisList {
			if s := collectRedisStats(client); s != nil {
				stats.RedisListStats[name] = s
			}
		}
	}
	lock.RUnlock()

	// 4. 主 Redis 连接池统计
	if rs := collectRedisStats(Redis); rs != nil {
		stats.RedisPoolSize = rs.PoolSize
		stats.RedisActiveConns = rs.ActiveConns
		stats.RedisIdleConns = rs.IdleConns
	}

	return stats
}

// checkDBHealth 检查单个 gorm.DB 实例的健康状态并记录连接池使用率告警
func checkDBHealth(db *gorm.DB, label string) HealthStatus {
	status := HealthStatus{Healthy: true}
	sqlDB, err := db.DB()
	if err != nil {
		status.Healthy = false
		status.Error = err.Error()
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		status.Healthy = false
		status.Error = err.Error()
		return status
	}

	dbStats := sqlDB.Stats()
	status.Stats = map[string]int{
		"max_open": dbStats.MaxOpenConnections,
		"open":     dbStats.OpenConnections,
		"in_use":   dbStats.InUse,
		"idle":     dbStats.Idle,
	}

	if dbStats.MaxOpenConnections > 0 && dbStats.InUse > dbStats.MaxOpenConnections*80/100 {
		logger.Warn("[连接池] %s 连接使用率过高: %d/%d (%.1f%%)",
			label, dbStats.InUse, dbStats.MaxOpenConnections,
			float64(dbStats.InUse)*100/float64(dbStats.MaxOpenConnections))
	}
	return status
}

// checkRedisHealth 检查单个 Redis 客户端实例的健康状态
func checkRedisHealth(client redis.UniversalClient) HealthStatus {
	status := HealthStatus{Healthy: true}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		status.Healthy = false
		status.Error = err.Error()
		return status
	}

	ps := client.PoolStats()
	status.Stats = map[string]int{
		"total": int(ps.TotalConns),
		"idle":  int(ps.IdleConns),
	}
	return status
}

// CheckPoolHealth 检查连接池健康状态
//
// 检查范围：
// 1. 主数据库（mysql）
// 2. 数据库解析器（mysql_resolver，读写分离）
// 3. 多数据库列表（mysql:<aliasName>）
// 4. 主 Redis（redis）
// 5. 多 Redis 列表（redis:<aliasName>）
// 6. Elasticsearch
// 7. Etcd
func CheckPoolHealth() map[string]HealthStatus {
	result := make(map[string]HealthStatus)

	// 1. 检查主数据库
	if BaseConfig.System.UseMysql && DB != nil {
		result["mysql"] = checkDBHealth(DB, "MySQL")
	}

	// 2. 检查数据库解析器（读写分离）
	if BaseConfig.System.UseMysql && DBResolver != nil {
		result["mysql_resolver"] = checkDBHealth(DBResolver, "MySQL Resolver")
	}

	// 3. 检查多数据库列表
	lock.RLock()
	for name, db := range DBList {
		if db != nil {
			result["mysql:"+name] = checkDBHealth(db, "MySQL:"+name)
		}
	}

	// 5. 检查多 Redis 列表
	for name, client := range RedisList {
		if client != nil {
			result["redis:"+name] = checkRedisHealth(client)
		}
	}
	lock.RUnlock()

	// 4. 检查主 Redis
	if BaseConfig.System.UseRedis && Redis != nil {
		result["redis"] = checkRedisHealth(Redis)
	}

	// 6. 检查 Elasticsearch
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

	// 7. 检查 Etcd
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
