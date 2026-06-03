// Package app 连接池统计函数测试
//
// ==================== 测试说明 ====================
// 本文件包含 app 包中连接池统计相关函数的单元测试。
//
// 测试覆盖内容：
// 1. collectDBStats 采集数据库连接池统计信息（含 SQLite 内存库）
// 2. collectRedisStats 采集 Redis 连接池统计信息（含 miniredis）
// 3. GetPoolStats 获取全部连接池统计（含 SQLite + miniredis 完整聚合）
// 4. checkDBHealth / checkRedisHealth / CheckPoolHealth / IsAllHealthy 成功路径
// 5. IsAllHealthy 在无服务时的行为
// 6. HealthStatus / DBInstanceStats / RedisInstanceStats 结构体基本操作
//
// 运行测试：go test -v ./app/... -run TestPoolStats
// ==================================================
package app

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zzsen/gin_core/model/config"
	"gorm.io/gorm"
)

// TestCollectDBStats_Nil 测试 collectDBStats 对 nil 入参的处理
//
// 【功能点】验证 collectDBStats 对 nil 安全返回
// 【测试流程】
// 1. 传入 nil
// 2. 验证返回 nil
func TestCollectDBStats_Nil(t *testing.T) {
	result := collectDBStats(nil)
	assert.Nil(t, result)
}

// TestCollectRedisStats_Nil 测试 collectRedisStats 对 nil 入参的处理
//
// 【功能点】验证 collectRedisStats 对 nil 安全返回
// 【测试流程】
// 1. 传入 nil
// 2. 验证返回 nil
func TestCollectRedisStats_Nil(t *testing.T) {
	result := collectRedisStats(nil)
	assert.Nil(t, result)
}

// TestGetPoolStats_AllNil 测试所有连接实例为 nil 时的 GetPoolStats
//
// 【功能点】验证 GetPoolStats 在无任何连接时安全返回空统计
// 【测试流程】
// 1. 确保所有全局连接变量为 nil
// 2. 调用 GetPoolStats
// 3. 验证返回非 nil 的空 PoolStats
func TestGetPoolStats_AllNil(t *testing.T) {
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	origRedis := Redis
	origRedisList := RedisList
	defer func() {
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
		Redis = origRedis
		RedisList = origRedisList
	}()

	DB = nil
	DBResolver = nil
	DBList = nil
	Redis = nil
	RedisList = nil

	stats := GetPoolStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats.DBMaxOpenConns)
	assert.Equal(t, 0, stats.RedisPoolSize)
	assert.Nil(t, stats.DBListStats)
	assert.Nil(t, stats.RedisListStats)
	assert.Nil(t, stats.DBResolverStats)
}

// TestGetPoolStats_WithEmptyLists 测试有空列表时的 GetPoolStats
//
// 【功能点】验证空的 DBList/RedisList 不产生统计条目
// 【测试流程】
// 1. 设置 DBList 和 RedisList 为空 map
// 2. 调用 GetPoolStats
// 3. 验证列表统计为空
func TestGetPoolStats_WithEmptyLists(t *testing.T) {
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	origRedis := Redis
	origRedisList := RedisList
	defer func() {
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
		Redis = origRedis
		RedisList = origRedisList
	}()

	DB = nil
	DBResolver = nil
	DBList = map[string]*gorm.DB{}
	Redis = nil
	RedisList = map[string]redis.UniversalClient{}

	stats := GetPoolStats()
	assert.NotNil(t, stats)
	assert.Nil(t, stats.DBListStats)
	assert.Nil(t, stats.RedisListStats)
}

// TestIsAllHealthy_NoServices 测试无服务时 IsAllHealthy 返回 true
//
// 【功能点】验证 IsAllHealthy 在无任何服务配置时返回 true
// 【测试流程】
// 1. 确保 BaseConfig 所有 Use* 为 false
// 2. 调用 IsAllHealthy
// 3. 验证返回 true
func TestIsAllHealthy_NoServices(t *testing.T) {
	origBaseConfig := BaseConfig
	origDB := DB
	origRedis := Redis
	origDBList := DBList
	origRedisList := RedisList
	defer func() {
		BaseConfig = origBaseConfig
		DB = origDB
		Redis = origRedis
		DBList = origDBList
		RedisList = origRedisList
	}()

	BaseConfig = config.BaseConfig{}
	DB = nil
	Redis = nil
	DBList = nil
	RedisList = nil

	assert.True(t, IsAllHealthy())
}

// TestHealthStatusStruct 测试 HealthStatus 结构体基本操作
//
// 【功能点】验证 HealthStatus 字段赋值和读取
// 【测试流程】
// 1. 构造健康和不健康的状态
// 2. 验证字段值
func TestHealthStatusStruct(t *testing.T) {
	healthy := HealthStatus{Healthy: true, Stats: map[string]int{"open": 5}}
	assert.True(t, healthy.Healthy)
	assert.Equal(t, 5, healthy.Stats["open"])

	unhealthy := HealthStatus{Healthy: false, Error: "connection refused"}
	assert.False(t, unhealthy.Healthy)
	assert.Equal(t, "connection refused", unhealthy.Error)
}

// TestDBInstanceStatsStruct 测试 DBInstanceStats 结构体
//
// 【功能点】验证 DBInstanceStats 字段赋值
// 【测试流程】
// 1. 构造实例
// 2. 验证字段值
func TestDBInstanceStatsStruct(t *testing.T) {
	stats := DBInstanceStats{
		MaxOpenConns:      100,
		OpenConns:         50,
		InUse:             30,
		Idle:              20,
		WaitCount:         5,
		WaitDurationMs:    150,
		MaxIdleClosed:     2,
		MaxLifetimeClosed: 1,
	}

	assert.Equal(t, 100, stats.MaxOpenConns)
	assert.Equal(t, 50, stats.OpenConns)
	assert.Equal(t, 30, stats.InUse)
	assert.Equal(t, 20, stats.Idle)
}

// TestRedisInstanceStatsStruct 测试 RedisInstanceStats 结构体
//
// 【功能点】验证 RedisInstanceStats 字段赋值
// 【测试流程】
// 1. 构造实例
// 2. 验证字段值
func TestRedisInstanceStatsStruct(t *testing.T) {
	stats := RedisInstanceStats{
		PoolSize:    10,
		ActiveConns: 5,
		IdleConns:   5,
	}

	assert.Equal(t, 10, stats.PoolSize)
	assert.Equal(t, 5, stats.ActiveConns)
	assert.Equal(t, 5, stats.IdleConns)
}

// TestCollectDBStats_RealSQLite 测试从真实 gorm.DB 采集连接池统计
//
// 【功能点】验证 collectDBStats 返回与 sql.DB.Stats 一致的统计字段
// 【测试流程】
// 1. 打开 SQLite 内存库并设置 SetMaxOpenConns
// 2. 调用 collectDBStats
// 3. 校验 MaxOpenConns 等与配置一致且计数非负
func TestCollectDBStats_RealSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(8)

	st := collectDBStats(db)
	require.NotNil(t, st)
	assert.Equal(t, 8, st.MaxOpenConns)
	assert.GreaterOrEqual(t, st.OpenConns, 0)
	assert.GreaterOrEqual(t, st.InUse, 0)
	assert.GreaterOrEqual(t, st.Idle, 0)
	assert.GreaterOrEqual(t, st.WaitCount, int64(0))
	assert.GreaterOrEqual(t, st.WaitDurationMs, int64(0))
	_ = sqlDB.Close()
}

// TestCollectRedisStats_Miniredis 测试从 miniredis 客户端采集连接池统计
//
// 【功能点】验证 collectRedisStats 基于 PoolStats 填充 RedisInstanceStats
// 【测试流程】
// 1. 启动 miniredis 并创建 go-redis Client，执行 Ping 触发连接池
// 2. 调用 collectRedisStats
// 3. 校验返回非 nil 且各计数字段合理
func TestCollectRedisStats_Miniredis(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())

	rs := collectRedisStats(client)
	require.NotNil(t, rs)
	assert.GreaterOrEqual(t, rs.PoolSize, 0)
	assert.GreaterOrEqual(t, rs.ActiveConns, 0)
	assert.GreaterOrEqual(t, rs.IdleConns, 0)
	assert.Equal(t, rs.PoolSize, rs.ActiveConns+rs.IdleConns)
}

// TestGetPoolStats_RealSQLiteAndMiniredis 测试全局 DB/Redis 下的完整采集流程
//
// 【功能点】验证 GetPoolStats 聚合主库、解析器、DBList、主 Redis、RedisList 的统计
// 【测试流程】
// 1. 保存并重置 DB、DBResolver、DBList、Redis、RedisList 为 SQLite 与 miniredis 实例
// 2. 调用 GetPoolStats
// 3. 校验主库字段、DBResolverStats、DBListStats、Redis 与 RedisList 条目存在且一致
func TestGetPoolStats_RealSQLiteAndMiniredis(t *testing.T) {
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	origRedis := Redis
	origRedisList := RedisList
	defer func() {
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
		Redis = origRedis
		RedisList = origRedisList
	}()

	dbMain, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlMain, err := dbMain.DB()
	require.NoError(t, err)
	sqlMain.SetMaxOpenConns(5)

	dbResolver, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlRes, err := dbResolver.DB()
	require.NoError(t, err)
	sqlRes.SetMaxOpenConns(6)

	dbExtra, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlExtra, err := dbExtra.DB()
	require.NoError(t, err)
	sqlExtra.SetMaxOpenConns(7)

	mr1 := miniredis.RunT(t)
	clientMain := redis.NewClient(&redis.Options{Addr: mr1.Addr()})
	defer func() { _ = clientMain.Close() }()
	ctx := context.Background()
	require.NoError(t, clientMain.Ping(ctx).Err())

	mr2 := miniredis.RunT(t)
	clientList := redis.NewClient(&redis.Options{Addr: mr2.Addr()})
	defer func() { _ = clientList.Close() }()
	require.NoError(t, clientList.Ping(ctx).Err())

	DB = dbMain
	DBResolver = dbResolver
	DBList = map[string]*gorm.DB{"extra": dbExtra}
	Redis = clientMain
	RedisList = map[string]redis.UniversalClient{"cache": clientList}

	stats := GetPoolStats()
	require.NotNil(t, stats)
	assert.Equal(t, 5, stats.DBMaxOpenConns)
	require.NotNil(t, stats.DBResolverStats)
	assert.Equal(t, 6, stats.DBResolverStats.MaxOpenConns)
	require.Contains(t, stats.DBListStats, "extra")
	assert.Equal(t, 7, stats.DBListStats["extra"].MaxOpenConns)

	rsMain := collectRedisStats(clientMain)
	require.NotNil(t, rsMain)
	assert.Equal(t, rsMain.PoolSize, stats.RedisPoolSize)
	assert.Equal(t, rsMain.ActiveConns, stats.RedisActiveConns)
	assert.Equal(t, rsMain.IdleConns, stats.RedisIdleConns)

	require.Contains(t, stats.RedisListStats, "cache")
	assert.Equal(t, collectRedisStats(clientList).PoolSize, stats.RedisListStats["cache"].PoolSize)

	_ = sqlMain.Close()
	_ = sqlRes.Close()
	_ = sqlExtra.Close()
}

// TestCheckDBHealth_RealSQLite 测试 SQLite 数据库健康检查成功路径
//
// 【功能点】验证 checkDBHealth 在 Ping 成功时 Healthy 为 true 且 Stats 含连接池字段
// 【测试流程】
// 1. 打开 SQLite 内存库并设置 MaxOpenConns
// 2. 调用 checkDBHealth
// 3. 断言 Healthy、Stats 中的 max_open/open/in_use/idle
func TestCheckDBHealth_RealSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)
	defer func() { _ = sqlDB.Close() }()

	status := checkDBHealth(db, "sqlite-test")
	assert.True(t, status.Healthy)
	assert.Empty(t, status.Error)
	require.NotNil(t, status.Stats)
	assert.Equal(t, 4, status.Stats["max_open"])
	assert.Contains(t, status.Stats, "open")
	assert.Contains(t, status.Stats, "in_use")
	assert.Contains(t, status.Stats, "idle")
}

// TestCheckRedisHealth_Miniredis 测试 Redis 健康检查成功路径
//
// 【功能点】验证 checkRedisHealth 在 Ping 成功时返回 Healthy 与连接池统计
// 【测试流程】
// 1. 启动 miniredis 并创建客户端后 Ping
// 2. 调用 checkRedisHealth
// 3. 断言 Healthy 及 Stats 中的 total、idle
func TestCheckRedisHealth_Miniredis(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()
	require.NoError(t, client.Ping(context.Background()).Err())

	status := checkRedisHealth(client)
	assert.True(t, status.Healthy)
	assert.Empty(t, status.Error)
	require.NotNil(t, status.Stats)
	assert.Contains(t, status.Stats, "total")
	assert.Contains(t, status.Stats, "idle")
}

// TestCheckPoolHealth_MysqlRedisResolverAndLists 测试开启 MySQL/Redis 与多实例分支
//
// 【功能点】验证 CheckPoolHealth 覆盖 mysql、mysql_resolver、DBList、redis、RedisList 检查键
// 【测试流程】
// 1. 保存 BaseConfig、DB、DBResolver、DBList、Redis、RedisList、ES、Etcd
// 2. 设置 UseMysql/UseRedis，关闭 ES/Etcd，注入 SQLite 与 miniredis 客户端
// 3. 调用 CheckPoolHealth，断言各键存在且 Healthy
func TestCheckPoolHealth_MysqlRedisResolverAndLists(t *testing.T) {
	origBase := BaseConfig
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	origRedis := Redis
	origRedisList := RedisList
	origES := ES
	origEtcd := Etcd
	defer func() {
		BaseConfig = origBase
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
		Redis = origRedis
		RedisList = origRedisList
		ES = origES
		Etcd = origEtcd
	}()

	dbMain, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlMain, err := dbMain.DB()
	require.NoError(t, err)
	defer func() { _ = sqlMain.Close() }()

	dbResolver, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlRes, err := dbResolver.DB()
	require.NoError(t, err)
	defer func() { _ = sqlRes.Close() }()

	dbShard, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlShard, err := dbShard.DB()
	require.NoError(t, err)
	defer func() { _ = sqlShard.Close() }()

	mrMain := miniredis.RunT(t)
	clientMain := redis.NewClient(&redis.Options{Addr: mrMain.Addr()})
	defer func() { _ = clientMain.Close() }()

	mrAlt := miniredis.RunT(t)
	clientAlt := redis.NewClient(&redis.Options{Addr: mrAlt.Addr()})
	defer func() { _ = clientAlt.Close() }()

	ctx := context.Background()
	require.NoError(t, clientMain.Ping(ctx).Err())
	require.NoError(t, clientAlt.Ping(ctx).Err())

	BaseConfig = config.BaseConfig{
		System: config.SystemInfo{
			UseMysql: true,
			UseRedis: true,
			UseEs:    false,
			UseEtcd:  false,
		},
	}
	DB = dbMain
	DBResolver = dbResolver
	DBList = map[string]*gorm.DB{"shard": dbShard}
	Redis = clientMain
	RedisList = map[string]redis.UniversalClient{"sessions": clientAlt}
	ES = nil
	Etcd = nil

	result := CheckPoolHealth()
	require.True(t, result["mysql"].Healthy)
	require.True(t, result["mysql_resolver"].Healthy)
	require.True(t, result["mysql:shard"].Healthy)
	require.True(t, result["redis"].Healthy)
	require.True(t, result["redis:sessions"].Healthy)
	assert.NotContains(t, result, "elasticsearch")
	assert.NotContains(t, result, "etcd")
}

// TestIsAllHealthy_AllSQLiteRedisHealthy 测试全部连接健康时 IsAllHealthy 为 true
//
// 【功能点】验证 IsAllHealthy 在 CheckPoolHealth 全部 Healthy 时返回 true
// 【测试流程】
// 1. 与 CheckPoolHealth 集成测试相同方式注入 SQLite/miniredis
// 2. 调用 IsAllHealthy
// 3. 断言为 true
func TestIsAllHealthy_AllSQLiteRedisHealthy(t *testing.T) {
	origBase := BaseConfig
	origDB := DB
	origDBResolver := DBResolver
	origDBList := DBList
	origRedis := Redis
	origRedisList := RedisList
	origES := ES
	origEtcd := Etcd
	defer func() {
		BaseConfig = origBase
		DB = origDB
		DBResolver = origDBResolver
		DBList = origDBList
		Redis = origRedis
		RedisList = origRedisList
		ES = origES
		Etcd = origEtcd
	}()

	dbMain, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlMain, err := dbMain.DB()
	require.NoError(t, err)
	defer func() { _ = sqlMain.Close() }()

	dbResolver, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlRes, err := dbResolver.DB()
	require.NoError(t, err)
	defer func() { _ = sqlRes.Close() }()

	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()
	require.NoError(t, client.Ping(context.Background()).Err())

	BaseConfig = config.BaseConfig{
		System: config.SystemInfo{
			UseMysql: true,
			UseRedis: true,
			UseEs:    false,
			UseEtcd:  false,
		},
	}
	DB = dbMain
	DBResolver = dbResolver
	DBList = nil
	Redis = client
	RedisList = nil
	ES = nil
	Etcd = nil

	assert.True(t, IsAllHealthy())
}
