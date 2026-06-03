// Package initialize 初始化模块扩展单元测试
//
// ==================== 测试说明 ====================
// 本文件覆盖配置校验、Redis(miniredis)、MySQL 连接失败路径、GORM 日志与命名策略、
// Etcd/Elasticsearch 错误分支、链路追踪启停等场景，以提升无外部依赖时的可测性。
//
// 测试覆盖内容：
// 1. MysqlDsn 默认字符集与时区
// 2. InitSingleDB 连接被拒绝时返回错误
// 3. InitGormLoggerConfig 默认值与指针覆盖
// 4. InitGormConfig 表命名策略
// 5. Migrate 非法迁移模式直接返回
// 6. InitDB / InitDBList 缺失配置或连接失败时 Panic
// 7. InitRedis / InitRedisList 缺失配置、Ping 失败、Miniredis 成功初始化
// 8. InitRedisClient 连接池默认值与 Ping 失败
// 9. InitEtcd 缺失配置与空 Endpoints 失败
// 10. InitElasticsearch 缺失配置与 Info 请求失败
// 11. InitTracing 未配置/禁用/Stdout 导出/不支持导出类型
// 12. ShutdownTracing 未初始化时不报错
// 13. StartMqConsumeWithContext 连接串为空时直接返回
// 14. RegisterTable 追加迁移实体
// 15. InitDBConnConfig SQLite 内存库连接池配置
// 16. InitDBCallbacks 创建与更新自动填充时间字段
// 17. InitDBResolver DbResolvers 为空时不 Panic
//
// 运行测试：go test -v ./initialize/... -run "Test"
// ==================================================
package initialize

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	elasticsearch "github.com/elastic/go-elasticsearch/v9"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/model/config"
	"github.com/zzsen/gin_core/tracing"
	gormLogger "gorm.io/gorm/logger"
)

// initializeGlobalsSnapshot 测试中备份 initialize/app 相关全局状态。
type initializeGlobalsSnapshot struct {
	baseConfig     config.BaseConfig
	tableEntity    []any
	tracerShutdown func(context.Context) error
	db             *gorm.DB
	redis          redis.UniversalClient
	redisList      map[string]redis.UniversalClient
	etcd           *clientv3.Client
	es             *elasticsearch.TypedClient
}

func saveInitializeGlobals() initializeGlobalsSnapshot {
	rlCopy := make(map[string]redis.UniversalClient)
	for k, v := range app.RedisList {
		rlCopy[k] = v
	}
	return initializeGlobalsSnapshot{
		baseConfig:     app.BaseConfig,
		tableEntity:    append([]any(nil), tableEntity...),
		tracerShutdown: tracerShutdown,
		db:             app.DB,
		redis:          app.Redis,
		redisList:      rlCopy,
		etcd:           app.Etcd,
		es:             app.ES,
	}
}

func restoreInitializeGlobals(t *testing.T, snap initializeGlobalsSnapshot) {
	t.Helper()

	if app.Redis != nil && app.Redis != snap.redis {
		_ = app.Redis.Close()
	}
	for k, c := range app.RedisList {
		if c == nil {
			continue
		}
		prev := snap.redisList[k]
		if prev != c {
			_ = c.Close()
		}
	}
	if app.Etcd != nil && app.Etcd != snap.etcd {
		_ = app.Etcd.Close()
	}

	app.DB = snap.db
	app.Redis = snap.redis
	app.RedisList = snap.redisList
	app.Etcd = snap.etcd
	app.ES = snap.es
	app.BaseConfig = snap.baseConfig
	tableEntity = append([]any(nil), snap.tableEntity...)
	tracerShutdown = snap.tracerShutdown
}

// TestMysqlDsn_默认字符集与时区
//
// 【功能点】验证 DbInfo.Dsn 在 charset/loc 为空时补齐默认值，保证 mysql.go 使用的连接串参数一致
// 【测试流程】构造 Host/Port 等字段，Charset 与 Loc 留空；调用 Dsn；断言包含 utf8mb4 与 Local
func TestMysqlDsn_默认字符集与时区(t *testing.T) {
	db := config.DbInfo{
		Host:     "127.0.0.1",
		Port:     3306,
		DBName:   "demo",
		Username: "u",
		Password: "p",
	}
	out := db.Dsn()
	assert.Contains(t, out, "charset=utf8mb4")
	assert.Contains(t, out, "loc=Local")
	assert.Contains(t, out, "u:p@tcp(127.0.0.1:3306)/demo")
}

// TestInitSingleDB_连接被拒绝时返回错误
//
// 【功能点】验证 initSingleDB 在无法连接 MySQL 时返回包装错误（不 panic）
// 【测试流程】使用 127.0.0.1:1 作为端口构造 DbInfo；调用 initSingleDB；断言返回错误且包含上下文信息
func TestInitSingleDB_连接被拒绝时返回错误(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Log: config.LoggersConfig{},
	}

	dbCfg := config.DbInfo{
		AliasName: "main",
		Host:      "127.0.0.1",
		Port:      1,
		DBName:    "test_gin_core",
		Username:  "root",
		Password:  "x",
		Migrate:   "ignore",
	}

	_, err := initSingleDB(dbCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "初始化 mysql client")
}

// TestInitGormLoggerConfig_默认值与指针覆盖
//
// 【功能点】验证 initGormLoggerConfig 对 IgnoreRecordNotFoundError、LogLevel、SlowThreshold 的默认与覆盖逻辑
// 【测试流程】分别使用零值、显式指针构造 DbInfo；比对返回的 gormLogger.Config 字段
func TestInitGormLoggerConfig_默认值与指针覆盖(t *testing.T) {
	t.Run("默认值", func(t *testing.T) {
		cfg := initGormLoggerConfig(config.DbInfo{})
		assert.True(t, cfg.IgnoreRecordNotFoundError)
		assert.Equal(t, gormLogger.Warn, cfg.LogLevel)
		assert.Equal(t, time.Duration(constant.DefaultDBSlowThreshold)*time.Millisecond, cfg.SlowThreshold)
	})

	t.Run("指针覆盖", func(t *testing.T) {
		ignore := false
		level := int(gormLogger.Error)
		slow := 777
		cfg := initGormLoggerConfig(config.DbInfo{
			IgnoreRecordNotFoundError: &ignore,
			LogLevel:                  &level,
			SlowThreshold:             &slow,
		})
		assert.False(t, cfg.IgnoreRecordNotFoundError)
		assert.Equal(t, gormLogger.Error, cfg.LogLevel)
		assert.Equal(t, 777*time.Millisecond, cfg.SlowThreshold)
	})
}

// TestInitGormConfig_表命名策略
//
// 【功能点】验证 initGormConfig 根据 DbInfo 设置 SingularTable 与 TablePrefix
// 【测试流程】备份 BaseConfig；设置最小 Log 配置；构造不同 DbInfo；检查 NamingStrategy 字段
func TestInitGormConfig_表命名策略(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{Log: config.LoggersConfig{}}

	stFalse := false
	cfgFalse := initGormConfig(config.DbInfo{
		SingularTable: &stFalse,
		TablePrefix:   "t_",
	})
	require.NotNil(t, cfgFalse)
	nsFalse, ok := cfgFalse.NamingStrategy.(schema.NamingStrategy)
	require.True(t, ok)
	assert.False(t, nsFalse.SingularTable)
	assert.Equal(t, "t_", nsFalse.TablePrefix)

	cfgDefault := initGormConfig(config.DbInfo{})
	require.NotNil(t, cfgDefault)
	nsDefault, ok := cfgDefault.NamingStrategy.(schema.NamingStrategy)
	require.True(t, ok)
	assert.True(t, nsDefault.SingularTable)
}

// TestMigrate_非法迁移模式直接返回
//
// 【功能点】验证 Migrate 对非 create/update 模式不访问数据库
// 【测试流程】传入 nil DB 与任意非法 migrateMode；调用 Migrate；不应 panic
func TestMigrate_非法迁移模式直接返回(t *testing.T) {
	assert.NotPanics(t, func() {
		Migrate(nil, "none")
		Migrate(nil, "")
	})
}

// TestInitDB_缺失配置时Panic
//
// 【功能点】验证 InitDB 在 app.BaseConfig.Db 为 nil 时抛出 InitError
// 【测试流程】备份配置后将 Db 置 nil；recover 校验 panic 类型与服务名
func TestInitDB_缺失配置时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{Db: nil}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok, "recover type %T", r)
		assert.Equal(t, "db", err.Service)
	}()

	InitDB()
}

// TestInitDBList_连接失败时Panic带别名
//
// 【功能点】验证 InitDBList 在某一别名数据库连接失败时 panic，且 InitError 携带配置名
// 【测试流程】DbList 配置无效端口；recover 并校验 Config 字段为别名
func TestInitDBList_连接失败时Panic带别名(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Log: config.LoggersConfig{},
		DbList: []config.DbInfo{
			{AliasName: "shard-a", Host: "127.0.0.1", Port: 1, DBName: "db", Username: "u", Password: "p"},
		},
	}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok, "recover type %T", r)
		assert.Equal(t, "db", err.Service)
		assert.Equal(t, "shard-a", err.Config)
	}()

	InitDBList()
}

// TestInitRedis_缺失配置时Panic
//
// 【功能点】验证 InitRedis 在 Redis 配置指针为空时 panic
// 【测试流程】BaseConfig.Redis=nil；recover 校验 InitError
func TestInitRedis_缺失配置时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{Redis: nil}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "redis", err.Service)
	}()

	InitRedis()
}

// TestInitRedis_Ping失败时Panic
//
// 【功能点】验证 InitRedis 在无法连通 Redis 时将底层错误包装后 panic
// 【测试流程】配置无效地址；recover 并校验 InitError
func TestInitRedis_Ping失败时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Redis: &config.RedisInfo{
			AliasName:  "solo",
			Addr:       "127.0.0.1:1",
			UseCluster: false,
		},
	}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "redis", err.Service)
	}()

	InitRedis()
}

// TestInitRedis_Miniredis成功初始化
//
// 【功能点】验证 InitRedis 在使用 miniredis 时能完成 Ping 并写入 app.Redis
// 【测试流程】启动 miniredis；写入 BaseConfig.Redis.Addr；调用 InitRedis；Ping；defer 恢复全局并关闭连接
func TestInitRedis_Miniredis成功初始化(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	app.BaseConfig = config.BaseConfig{
		Log: config.LoggersConfig{},
		Redis: &config.RedisInfo{
			AliasName: "unit",
			Addr:      mr.Addr(),
			Password:  "",
			DB:        0,
		},
	}

	assert.NotPanics(t, func() {
		InitRedis()
	})
	require.NotNil(t, app.Redis)
	pong, err := app.Redis.Ping(context.Background()).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)
}

// TestInitRedisList_Miniredis多实例
//
// 【功能点】验证 InitRedisList 可为多个别名分别建立客户端
// 【测试流程】启动两个 miniredis 实例；写入 RedisList；调用 InitRedisList；分别 Ping
func TestInitRedisList_Miniredis多实例(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	mr1, err := miniredis.Run()
	require.NoError(t, err)
	defer mr1.Close()
	mr2, err := miniredis.Run()
	require.NoError(t, err)
	defer mr2.Close()

	app.BaseConfig = config.BaseConfig{
		Log: config.LoggersConfig{},
		RedisList: []config.RedisInfo{
			{AliasName: "cache-a", Addr: mr1.Addr()},
			{AliasName: "cache-b", Addr: mr2.Addr()},
		},
	}

	assert.NotPanics(t, func() {
		InitRedisList()
	})
	require.Len(t, app.RedisList, 2)
	for alias, cli := range app.RedisList {
		pong, e := cli.Ping(context.Background()).Result()
		require.NoError(t, e, "alias=%s", alias)
		assert.Equal(t, "PONG", pong)
	}
}

// TestInitRedisClient_连接池默认值与Ping失败
//
// 【功能点】验证 poolSize/minIdleConns≤0 时回落到常量默认值；集群模式与单机模式在地址无效时 Ping 失败返回错误
// 【测试流程】miniredis 成功路径使用 PoolSize=0；集群与单机指向无效端口断言返回错误
func TestInitRedisClient_连接池默认值与Ping失败(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cli, err := initRedisClient(config.RedisInfo{
		AliasName:    "pool-default",
		Addr:         mr.Addr(),
		PoolSize:     0,
		MinIdleConns: 0,
	})
	require.NoError(t, err)
	require.NotNil(t, cli)
	defer cli.Close()

	_, err = initRedisClient(config.RedisInfo{
		AliasName:    "cluster-fail",
		UseCluster:   true,
		ClusterAddrs: []string{"127.0.0.1:1"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")

	_, err = initRedisClient(config.RedisInfo{
		AliasName:  "standalone-fail",
		UseCluster: false,
		Addr:       "127.0.0.1:1",
	})
	require.Error(t, err)
}

// TestInitRedisList_某一实例失败时Panic带别名
//
// 【功能点】验证 InitRedisList 在列表中某项 Ping 失败时 panic，InitError.Config 为别名
// 【测试流程】第一项合法 miniredis，第二项非法端口；recover 校验 Config
func TestInitRedisList_某一实例失败时Panic带别名(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	app.BaseConfig = config.BaseConfig{
		Log: config.LoggersConfig{},
		RedisList: []config.RedisInfo{
			{AliasName: "ok", Addr: mr.Addr()},
			{AliasName: "bad", Addr: "127.0.0.1:1"},
		},
	}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		initErr, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "redis", initErr.Service)
		assert.Equal(t, "bad", initErr.Config)
	}()

	InitRedisList()
}

// TestInitEtcd_缺失配置时Panic
//
// 【功能点】验证 InitEtcd 在 Etcd 配置缺失时 panic
// 【测试流程】Etcd=nil；recover 校验 InitError
func TestInitEtcd_缺失配置时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{Etcd: nil}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "etcd", err.Service)
	}()

	InitEtcd()
}

// TestInitEtcd_空Endpoints创建客户端失败时Panic
//
// 【功能点】验证 InitEtcd 在 clientv3.New 返回错误时的 panic 路径
// 【测试流程】Addresses 为空切片；recover 校验 InitError 服务名为 etcd
func TestInitEtcd_空Endpoints创建客户端失败时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Etcd: &config.EtcdInfo{
			Addresses: []string{},
		},
	}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "etcd", err.Service)
	}()

	InitEtcd()
}

// TestInitElasticsearch_缺失配置时Panic
//
// 【功能点】验证 InitElasticsearch 在 Es 配置缺失时 panic
// 【测试流程】Es=nil；recover 校验 InitError
func TestInitElasticsearch_缺失配置时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{Es: nil}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "es", err.Service)
	}()

	InitElasticsearch()
}

// TestInitElasticsearch_Info请求失败时Panic
//
// 【功能点】验证创建客户端成功但 Info 请求失败时的 panic 路径
// 【测试流程】Addresses 指向本机拒绝端口；recover 校验 InitError 服务名为 es
func TestInitElasticsearch_Info请求失败时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Es: &config.EsInfo{
			Addresses: []string{"http://127.0.0.1:1"},
		},
	}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		err, ok := r.(*exception.InitError)
		require.True(t, ok)
		assert.Equal(t, "es", err.Service)
	}()

	InitElasticsearch()
}

// TestInitTracing_未配置时跳过
//
// 【功能点】验证 Tracing 配置为 nil 时不 panic、不调用采集端
// 【测试流程】Tracing=nil；调用 InitTracing；断言无 panic
func TestInitTracing_未配置时跳过(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{Tracing: nil}
	assert.NotPanics(t, func() {
		InitTracing()
	})
}

// TestInitTracing_禁用时跳过
//
// 【功能点】验证 Enabled=false 时不初始化导出器
// 【测试流程】Tracing.Enabled=false；InitTracing；无 panic
func TestInitTracing_禁用时跳过(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Tracing: &config.TracingConfig{
			Enabled: false,
		},
	}
	assert.NotPanics(t, func() {
		InitTracing()
	})
}

// TestInitTracing_Stdout导出成功并可关闭
//
// 【功能点】验证 InitTracing 在使用 stdout 导出器时能完成初始化，ShutdownTracing 可正常调用
// 【测试流程】Enabled=true、ExporterType=stdout；InitTracing；ShutdownTracing；再由 defer 恢复全局快照
func TestInitTracing_Stdout导出成功并可关闭(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Tracing: &config.TracingConfig{
			Enabled:        true,
			ServiceName:    "gin-core-unit",
			ExporterType:   "stdout",
			SampleRate:     1,
			PropagatorType: "tracecontext",
		},
	}

	assert.NotPanics(t, func() {
		InitTracing()
	})
	require.NotNil(t, tracerShutdown)
	ShutdownTracing()
}

// TestShutdownTracing_未初始化时不报错
//
// 【功能点】验证 tracerShutdown 为空时 ShutdownTracing 迅速返回
// 【测试流程】确保 tracerShutdown 为 nil；调用 ShutdownTracing；无 panic
func TestShutdownTracing_未初始化时不报错(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	tracerShutdown = nil
	assert.NotPanics(t, func() {
		ShutdownTracing()
	})
}

// TestIsTracingEnabled_委托tracing包
//
// 【功能点】验证 IsTracingEnabled 与 tracing.IsEnabled 在当前全局状态下返回值一致
// 【测试流程】读取两边返回值并断言相等
func TestIsTracingEnabled_委托tracing包(t *testing.T) {
	assert.Equal(t, tracing.IsEnabled(), IsTracingEnabled())
}

// TestStartMqConsumeWithContext_连接串为空时直接返回
//
// 【功能点】验证 MQName 指向不存在实例且列表未命中时连接串为空，函数提前返回且不设置 MqConnStr
// 【测试流程】构造 BaseConfig 与 MessageQueue；同步调用 startMqConsumeWithContext；断言 MqConnStr 仍为空
func TestStartMqConsumeWithContext_连接串为空时直接返回(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		RabbitMQ:     config.RabbitMQInfo{Host: "127.0.0.1", Port: 5672, Username: "guest", Password: "guest"},
		RabbitMQList: config.RabbitMqListInfo{},
	}

	mq := &config.MessageQueue{
		MQName:       "not-found-alias",
		QueueName:    "q",
		ExchangeName: "ex",
		ExchangeType: "direct",
		RoutingKey:   "k",
	}

	startMqConsumeWithContext(context.Background(), mq)
	assert.Empty(t, mq.MqConnStr)
}

// TestInitTracing_不支持的导出类型时记录错误且不设置shutdown
//
// 【功能点】验证 InitTracing 在 InitTracer 返回错误时不赋值 tracerShutdown（初始化失败路径）
// 【测试流程】ExporterType 非法；InitTracing；断言 tracerShutdown 仍为 nil（注意 tracing 包全局配置可能被部分写入）
func TestInitTracing_不支持的导出类型时记录错误且不设置shutdown(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Tracing: &config.TracingConfig{
			Enabled:      true,
			ServiceName:  "bad-exporter",
			ExporterType: "not-a-real-exporter",
			SampleRate:   1,
		},
	}

	InitTracing()
	assert.Nil(t, tracerShutdown)
}

// TestRegisterTable_追加迁移实体
//
// 【功能点】验证 RegisterTable 向 package 级 tableEntity 追加模型且在 restore 后恢复原切片
// 【测试流程】记录当前 tableEntity 长度；调用 RegisterTable；断言长度+1；defer restoreInitializeGlobals 校验恢复
func TestRegisterTable_追加迁移实体(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	before := len(tableEntity)
	RegisterTable(struct{ ID int }{})
	assert.Equal(t, before+1, len(tableEntity))
}

// TestInitDBConnConfig_Sqlite内存库连接池配置成功
//
// 【功能点】验证 initDBConnConfig 能对底层 *sql.DB 写入连接池参数且无错误（使用内存 SQLite，无需外部 MySQL）
// 【测试流程】gorm.Open(sqlite memory)；调用 initDBConnConfig；断言返回 nil
func TestInitDBConnConfig_Sqlite内存库连接池配置成功(t *testing.T) {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = initDBConnConfig(gormDB, config.DbInfo{
		MaxIdleConns:    8,
		MaxOpenConns:    88,
		ConnMaxIdleTime: 90,
		ConnMaxLifetime: 120,
	})
	require.NoError(t, err)
}

// TestInitDB_Mysql连接失败时Panic
//
// 【功能点】验证 InitDB 在 Db 配置存在但无法连接 MySQL 时 panic 且 InitError 服务名为 db
// 【测试流程】设置 Db 指向本地关闭端口；recover 校验 exception.InitError
func TestInitDB_Mysql连接失败时Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Log: config.LoggersConfig{},
		Db: &config.DbInfo{
			Host:     "127.0.0.1",
			Port:     1,
			DBName:   "x",
			Username: "u",
			Password: "p",
			Migrate:  "invalid-mode-no-db-touch",
		},
	}

	defer func() {
		r := recover()
		require.NotNil(t, r)
		initErr, ok := r.(*exception.InitError)
		require.True(t, ok, "recover type %T", r)
		assert.Equal(t, "db", initErr.Service)
	}()

	InitDB()
}

// TestInitDBResolver_DbResolvers为空时不Panic
//
// 【功能点】验证 len(DbResolvers)==0 时 InitDBResolver 直接返回且不 panic
// 【测试流程】DbResolvers 置 nil；调用 InitDBResolver；断言无 panic
func TestInitDBResolver_DbResolvers为空时不Panic(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{DbResolvers: nil}
	assert.NotPanics(t, func() {
		InitDBResolver()
	})
}

// TestInitDBList_DbList为空时初始化空映射
//
// 【功能点】验证 DbList 为空时 InitDBList 仍构建空的 app.DBList 映射并完成日志分支
// 【测试流程】DbList=[]；InitDBList；断言 DBList 非 nil 且长度为 0
func TestInitDBList_DbList为空时初始化空映射(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Log:    config.LoggersConfig{},
		DbList: []config.DbInfo{},
	}

	assert.NotPanics(t, func() {
		InitDBList()
	})
	require.NotNil(t, app.DBList)
	assert.Len(t, app.DBList, 0)
}

// TestInitRedisList_RedisList为空时映射为空
//
// 【功能点】验证 RedisList 为空时 InitRedisList 写入空的 app.RedisList
// 【测试流程】RedisList=[]；InitRedisList；断言 RedisList 长度为 0
func TestInitRedisList_RedisList为空时映射为空(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	app.BaseConfig = config.BaseConfig{
		Log:       config.LoggersConfig{},
		RedisList: []config.RedisInfo{},
	}

	assert.NotPanics(t, func() {
		InitRedisList()
	})
	require.NotNil(t, app.RedisList)
	assert.Len(t, app.RedisList, 0)
}

// TestInitialRabbitMqWithContext_无队列且上下文已取消
//
// 【功能点】验证仅启动故障恢复协程、无消费者时若 ctx 已取消则监听协程立即退出（可测分支）
// 【测试流程】先 cancel 再调用 InitialRabbitMqWithContext；短睡眠等待 goroutine 收敛；无 RabbitMQ 依赖
func TestInitialRabbitMqWithContext_无队列且上下文已取消(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.NotPanics(t, func() {
		InitialRabbitMqWithContext(ctx)
	})
	time.Sleep(30 * time.Millisecond)
}

// testInitDBCallbackTimeModel 用于验证 initDBCallbacks 对 CreateTime/UpdateTime 的填充（SQLite 内存库）
type testInitDBCallbackTimeModel struct {
	ID         uint `gorm:"primarykey"`
	Name       string
	CreateTime time.Time
	UpdateTime time.Time
}

// TestInitDBCallbacks_SQLite创建与更新自动填充时间字段
//
// 【功能点】验证 initDBCallbacks 注册的 Create/Update 前钩子能为 CreateTime、UpdateTime 及批量创建写入合理时间戳
// 【测试流程】
// 1. 打开 SQLite 内存库并注册回调后 AutoMigrate 测试模型
// 2. 单条创建、批量创建校验时间非零且批量记录均被填充
// 3. Update 名称后校验 UpdateTime 相对创建时刻单调演进
func TestInitDBCallbacks_SQLite创建与更新自动填充时间字段(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	initDBCallbacks(db)
	require.NoError(t, db.AutoMigrate(&testInitDBCallbackTimeModel{}))

	var row testInitDBCallbackTimeModel
	row.Name = "single"
	require.NoError(t, db.Create(&row).Error)
	assert.False(t, row.CreateTime.IsZero())
	assert.False(t, row.UpdateTime.IsZero())

	batch := []testInitDBCallbackTimeModel{{Name: "a"}, {Name: "b"}}
	require.NoError(t, db.Create(&batch).Error)

	var batchRows []testInitDBCallbackTimeModel
	require.NoError(t, db.Where("name IN ?", []string{"a", "b"}).Order("id").Find(&batchRows).Error)
	require.Len(t, batchRows, 2)
	for _, r := range batchRows {
		assert.False(t, r.CreateTime.IsZero())
		assert.False(t, r.UpdateTime.IsZero())
	}

	var loaded testInitDBCallbackTimeModel
	require.NoError(t, db.First(&loaded, row.ID).Error)
	createdAt := loaded.CreateTime
	time.Sleep(15 * time.Millisecond)
	loaded.Name = "renamed"
	require.NoError(t, db.Save(&loaded).Error)
	require.NoError(t, db.First(&loaded, row.ID).Error)
	assert.Equal(t, "renamed", loaded.Name)
	assert.False(t, loaded.UpdateTime.Before(createdAt), "UpdateTime 应被更新回调刷新")
}

// TestMigrate_SQLite_create与update模式执行AutoMigrate
//
// 【功能点】验证 Migrate 在 create 模式下先 Drop 再建表，在 update 模式下再次 AutoMigrate 仍可执行
// 【测试流程】
// 1. save/restore package 级 tableEntity
// 2. RegisterTable 注册测试模型并用 SQLite 内存库执行 Migrate(create)
// 3. 断言表存在后执行 Migrate(update)
func TestMigrate_SQLite_create与update模式执行AutoMigrate(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	type migrateProbeModel struct {
		ID   uint `gorm:"primarykey"`
		Name string
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	before := len(tableEntity)
	RegisterTable(&migrateProbeModel{})
	assert.Equal(t, before+1, len(tableEntity))

	Migrate(db, "create")
	assert.True(t, db.Migrator().HasTable(&migrateProbeModel{}))

	Migrate(db, "update")
	assert.True(t, db.Migrator().HasTable(&migrateProbeModel{}))
}

// TestInitialRabbitMq_空消费者列表等价于无参WithContext
//
// 【功能点】验证 InitialRabbitMq 将调用委托给 InitialRabbitMqWithContext（空列表时不启动消费协程）
// 【测试流程】
// 1. 直接调用 InitialRabbitMq 不传队列配置
// 2. 短暂等待并发路径收敛，断言无 panic（后台故障恢复协程依赖 Background，测试结束后仍常驻，与生产行为一致）
func TestInitialRabbitMq_空消费者列表等价于无参WithContext(t *testing.T) {
	assert.NotPanics(t, func() {
		InitialRabbitMq()
	})
	time.Sleep(30 * time.Millisecond)
}

// TestInitRedisClient_集群地址为空或单机地址为空时Ping失败
//
// 【功能点】验证 initRedisClient 在集群地址列表为空或单机 Addr 为空时无法完成 Ping 并返回错误
// 【测试流程】
// 1. UseCluster=true 且 ClusterAddrs 为空切片调用 initRedisClient
// 2. UseCluster=false 且 Addr 为空字符串再次调用
func TestInitRedisClient_集群地址为空或单机地址为空时Ping失败(t *testing.T) {
	_, err := initRedisClient(config.RedisInfo{
		AliasName:    "cluster-no-addr",
		UseCluster:   true,
		ClusterAddrs: []string{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")

	_, err = initRedisClient(config.RedisInfo{
		AliasName:  "standalone-no-addr",
		UseCluster: false,
		Addr:       "",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ping failed")
}

// TestInitRedisClient_启用Redis链路追踪时单机客户端成功
//
// 【功能点】验证 tracing.IsRedisTracingEnabled 为 true 时单机 redis.NewClient 分支执行 AddHook 后仍能 Ping 成功
// 【测试流程】
// 1. tracing.InitTracer 启用 stdout 与 EnableRedisTracing
// 2. defer 中 Shutdown 并 InitTracer(nil) 复位全局 tracingConfig
// 3. miniredis 地址初始化客户端并 Ping
func TestInitRedisClient_启用Redis链路追踪时单机客户端成功(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	shutdown, err := tracing.InitTracer(&config.TracingConfig{
		Enabled:            true,
		EnableRedisTracing: true,
		ServiceName:        "ut-redis-hook",
		ExporterType:       "stdout",
		SampleRate:         1,
		PropagatorType:     "tracecontext",
	})
	require.NoError(t, err)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(ctx)
		_, _ = tracing.InitTracer(nil)
	}()

	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	cli, err := initRedisClient(config.RedisInfo{
		AliasName:    "traced-single",
		Addr:         mr.Addr(),
		UseCluster:   false,
		PoolSize:     2,
		MinIdleConns: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, cli)
	defer cli.Close()

	pong, err := cli.Ping(context.Background()).Result()
	require.NoError(t, err)
	assert.Equal(t, "PONG", pong)
}

// TestInfo_ES连通成功时返回nil
//
// 【功能点】验证 info 在 Info API 返回合法 JSON 时记录版本并返回 nil（覆盖 initialize.elasticsearch.info 成功路径）
// 【测试流程】
// 1. save/restore app.ES
// 2. httptest 返回包含 version.number 的根响应
// 3. NewTypedClient 指向该地址并调用 info()
func TestInfo_ES连通成功时返回nil(t *testing.T) {
	snap := saveInitializeGlobals()
	defer restoreInitializeGlobals(t, snap)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"ut-node","cluster_name":"ut","cluster_uuid":"x","tagline":"You Know, for Search","version":{"number":"9.2.1"}}`))
	}))
	defer srv.Close()

	es, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{srv.URL},
	})
	require.NoError(t, err)
	app.ES = es

	assert.NoError(t, info())
}
