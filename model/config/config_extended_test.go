// Package config 配置模块测试
//
// ==================== 测试说明 ====================
// 扩展单元测试：聚焦 DbInfo.Dsn、CORSConfig、SystemInfo、各配置结构体边界，
// 以及 MessageQueue 的消息确认分支（通过 Acknowledger 桩，无需真实 MQ）。
//
// 测试覆盖内容：
// 1. DbInfo.Dsn 在多组参数下的连接字符串与默认 charset/loc 回填
// 2. CORSConfig Getter 的边界分支（MaxAge、AllowOrigins 等）
// 3. SystemInfo（system.go）字段零值与显式组合
// 4. BaseConfig、Metrics、Redis、Etcd、ES、SMTP、Tracing 等结构体的边界赋值
// 5. DbResolver.ReplicaConfigs 空从库、ServiceInfo 关闭超时负数边界
// 6. MessageQueue：handleMessage / waitForConfirm、initConn/initChannel 短路、initDeadLetterQueue（rabbitMQDeadLetterSetup 桩）
//
// 运行测试：go test -v ./model/config/... -run Test
// ==================================================

package config

import (
	"context"
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAcknowledger 用于在无真实 Channel 时验证 Ack/Nack 调用路径
type stubAcknowledger struct {
	ackCount  int
	nackCalls []struct {
		multiple bool
		requeue  bool
	}
}

func (s *stubAcknowledger) Ack(uint64, bool) error {
	s.ackCount++
	return nil
}

func (s *stubAcknowledger) Nack(_ uint64, multiple bool, requeue bool) error {
	s.nackCalls = append(s.nackCalls, struct {
		multiple bool
		requeue  bool
	}{multiple: multiple, requeue: requeue})
	return nil
}

func (s *stubAcknowledger) Reject(uint64, bool) error {
	return nil
}

// deadLetterStubChannel 实现 rabbitMQDeadLetterSetup，用于 initDeadLetterQueue 单元测试
type deadLetterStubChannel struct {
	exchangeErr error
	queueErr    error
	bindErr     error

	lastQueueArgs amqp.Table
}

func (s *deadLetterStubChannel) ExchangeDeclare(string, string, bool, bool, bool, bool, amqp.Table) error {
	return s.exchangeErr
}

func (s *deadLetterStubChannel) QueueDeclare(name string, _, _, _, _ bool, args amqp.Table) (amqp.Queue, error) {
	s.lastQueueArgs = args
	return amqp.Queue{Name: name}, s.queueErr
}

func (s *deadLetterStubChannel) QueueBind(string, string, string, bool, amqp.Table) error {
	return s.bindErr
}

// TestDbInfo_Dsn_连接字符串生成
//
// 【功能点】DbInfo.Dsn 在各种用户名、密码、主机、端口、库名、charset、loc 组合下生成预期 DSN，并校验 parseTime 固定为 True
// 【测试流程】
// 1. 表驱动构造多组 DbInfo（含空 charset/loc、仅填其一、特殊字符密码、端口与库名边界）
// 2. 调用 Dsn() 并与期望完整字符串比对
func TestDbInfo_Dsn_连接字符串生成(t *testing.T) {
	tests := []struct {
		name     string
		input    DbInfo
		expected string
	}{
		{
			name: "空 charset 与 loc 时使用 utf8mb4 与 Local",
			input: DbInfo{
				Username: "root",
				Password: "secret",
				Host:     "127.0.0.1",
				Port:     3306,
				DBName:   "app",
			},
			expected: "root:secret@tcp(127.0.0.1:3306)/app?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "仅自定义 charset",
			input: DbInfo{
				Username: "u",
				Password: "p",
				Host:     "db.internal",
				Port:     3306,
				DBName:   "db",
				Charset:  "utf8",
				Loc:      "",
			},
			expected: "u:p@tcp(db.internal:3306)/db?charset=utf8&parseTime=True&loc=Local",
		},
		{
			name: "仅自定义 loc",
			input: DbInfo{
				Username: "u",
				Password: "p",
				Host:     "db.internal",
				Port:     3306,
				DBName:   "db",
				Charset:  "",
				Loc:      "Europe/Berlin",
			},
			expected: "u:p@tcp(db.internal:3306)/db?charset=utf8mb4&parseTime=True&loc=Europe/Berlin",
		},
		{
			name: "charset 与 loc 均已配置",
			input: DbInfo{
				Username: "admin",
				Password: "x",
				Host:     "10.0.0.5",
				Port:     3307,
				DBName:   "analytics",
				Charset:  "latin1",
				Loc:      "UTC",
			},
			expected: "admin:x@tcp(10.0.0.5:3307)/analytics?charset=latin1&parseTime=True&loc=UTC",
		},
		{
			name: "空库名与零端口仍按格式化输出",
			input: DbInfo{
				Username: "user",
				Password: "pass",
				Host:     "localhost",
				Port:     0,
				DBName:   "",
			},
			expected: "user:pass@tcp(localhost:0)/?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "密码含特殊字符仍原样拼接",
			input: DbInfo{
				Username: "u",
				Password: "p:w@d",
				Host:     "h",
				Port:     3306,
				DBName:   "n",
			},
			expected: "u:p:w@d@tcp(h:3306)/n?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "用户名为空",
			input: DbInfo{
				Username: "",
				Password: "pwd",
				Host:     "mysql",
				Port:     3306,
				DBName:   "db",
			},
			expected: ":pwd@tcp(mysql:3306)/db?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 每次使用副本，避免 Dsn 对 charset/loc 的回填影响其它用例的 input 字面量预期
			db := tt.input
			assert.Equal(t, tt.expected, (&db).Dsn())
			assert.Contains(t, tt.expected, "parseTime=True")
		})
	}
}

// TestDbInfo_Dsn_默认字段写回接收者
//
// 【功能点】Dsn 在 charset/loc 为空时会写回 DbInfo 字段，后续调用应沿用已回填值
// 【测试流程】
// 1. 构造 charset、loc 均为空的 DbInfo 并调用一次 Dsn
// 2. 断言接收者 Charset、Loc 已被设为 utf8mb4、Local，再次 Dsn 结果一致
func TestDbInfo_Dsn_默认字段写回接收者(t *testing.T) {
	db := &DbInfo{
		Username: "u",
		Password: "p",
		Host:     "h",
		Port:     3306,
		DBName:   "n",
	}
	first := db.Dsn()
	assert.Equal(t, "utf8mb4", db.Charset)
	assert.Equal(t, "Local", db.Loc)
	second := db.Dsn()
	assert.Equal(t, first, second)
}

// TestCORSConfig_GetMaxAge_边界值
//
// 【功能点】GetMaxAge 在非正数时返回默认 86400，正数时返回配置值
// 【测试流程】
// 1. 表驱动传入 MaxAge：0、负数、正数
// 2. 断言返回值符合约定
func TestCORSConfig_GetMaxAge_边界值(t *testing.T) {
	tests := []struct {
		name   string
		maxAge int
		want   int
	}{
		{"零值默认", 0, 86400},
		{"负数默认", -1, 86400},
		{"较小负数默认", -86400, 86400},
		{"正数原样", 1, 1},
		{"常用缓存时长", 7200, 7200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CORSConfig{MaxAge: tt.maxAge}
			assert.Equal(t, tt.want, c.GetMaxAge())
		})
	}
}

// TestCORSConfig_GetAllowOrigins_allowMethods_allowHeaders_显式与默认
//
// 【功能点】AllowOrigins 非空时返回原切片；AllowMethods/AllowHeaders 默认列表完整性与自定义分支
// 【测试流程】
// 1. nil / 空切片 AllowOrigins → 默认 ["*"]
// 2. 非空 AllowOrigins → 与原切片相等
// 3. 默认 AllowMethods 包含文档所述六种方法且顺序一致
// 4. 默认 AllowHeaders 为四条默认头
func TestCORSConfig_GetAllowOrigins_allowMethods_allowHeaders_显式与默认(t *testing.T) {
	t.Run("AllowOrigins_nil 与空切片均默认星号", func(t *testing.T) {
		assert.Equal(t, []string{"*"}, (&CORSConfig{AllowOrigins: nil}).GetAllowOrigins())
		assert.Equal(t, []string{"*"}, (&CORSConfig{AllowOrigins: []string{}}).GetAllowOrigins())
	})
	t.Run("AllowOrigins 显式值", func(t *testing.T) {
		orig := []string{"https://a.example", "*.b.example"}
		cfg := &CORSConfig{AllowOrigins: orig}
		assert.Equal(t, orig, cfg.GetAllowOrigins())
	})
	defaultMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	assert.Equal(t, defaultMethods, (&CORSConfig{}).GetAllowMethods())
	defaultHeaders := []string{"Content-Type", "Authorization", "X-Trace-Id", "X-Request-Id"}
	assert.Equal(t, defaultHeaders, (&CORSConfig{}).GetAllowHeaders())
}

// TestSystemInfo_字段零值与组合
//
// 【功能点】SystemInfo（system.go）仅包含开关与 GcTime，验证零值与全开启组合是否符合预期赋值
// 【测试流程】
// 1. 断言零值结构各字段为 Go 零值
// 2. 构造全 true 与非零 GcTime，断言字段一一对应
func TestSystemInfo_字段零值与组合(t *testing.T) {
	t.Run("零值", func(t *testing.T) {
		var s SystemInfo
		assert.Equal(t, 0, s.GcTime)
		assert.False(t, s.UseRedis)
		assert.False(t, s.UseMysql)
		assert.False(t, s.UseEs)
		assert.False(t, s.UseEtcd)
		assert.False(t, s.UseRabbitMQ)
		assert.False(t, s.UseSchedule)
	})
	t.Run("全开与非零 GcTime", func(t *testing.T) {
		s := SystemInfo{
			GcTime:      120,
			UseRedis:    true,
			UseMysql:    true,
			UseEs:       true,
			UseEtcd:     true,
			UseRabbitMQ: true,
			UseSchedule: true,
		}
		assert.Equal(t, 120, s.GcTime)
		assert.True(t, s.UseRedis && s.UseMysql && s.UseEs && s.UseEtcd && s.UseRabbitMQ && s.UseSchedule)
	})
}

// TestBaseConfig_嵌套字段零值与指针
//
// 【功能点】BaseConfig 顶层与各嵌套字段在未赋值时的零值形态（指针为 nil、切片为 nil）
// 【测试流程】
// 1. 构建 BaseConfig 零值
// 2. 断言 Db、Etcd、Redis、Es、Tracing 指针为 nil，DbList 等切片为 nil
func TestBaseConfig_嵌套字段零值与指针(t *testing.T) {
	var b BaseConfig
	assert.Nil(t, b.Db)
	assert.Nil(t, b.Etcd)
	assert.Nil(t, b.Redis)
	assert.Nil(t, b.Es)
	assert.Nil(t, b.Tracing)
	assert.Nil(t, b.DbList)
	assert.Nil(t, b.DbResolvers)
	assert.Nil(t, b.RedisList)
}

// TestServiceInfo_GetShutdownTimeout_零与负数
//
// 【功能点】ShutdownTimeout <= 0 时返回默认 5 秒
// 【测试流程】
// 1. 传入 0、负数
// 2. 断言均为 5；正数则原样返回
func TestServiceInfo_GetShutdownTimeout_零与负数(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		want    int
	}{
		{"零", 0, 5},
		{"负数", -10, 5},
		{"正数", 30, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServiceInfo{ShutdownTimeout: tt.timeout}
			assert.Equal(t, tt.want, s.GetShutdownTimeout())
		})
	}
}

// TestDbResolver_ReplicaConfigs_空从库列表
//
// 【功能点】未配置 Replicas 时 ReplicaConfigs 返回空切片（长度为 0）
// 【测试流程】
// 1. 构造仅有 Sources 的 DbResolver
// 2. 调用 ReplicaConfigs 并断言长度为 0 且不 panic
func TestDbResolver_ReplicaConfigs_空从库列表(t *testing.T) {
	r := &DbResolver{
		Sources: []DbInfo{
			{Username: "w", Password: "p", Host: "m", Port: 3306, DBName: "d"},
		},
		Replicas: nil,
	}
	cfg := r.ReplicaConfigs()
	assert.NotNil(t, cfg)
	assert.Len(t, cfg, 0)
}

// TestRateLimitConfig_GetDefaultBurst_沿用默认速率双倍
//
// 【功能点】DefaultBurst 未设且 DefaultRate 自定义时，突发容量为速率的两倍
// 【测试流程】
// 1. DefaultRate=40、DefaultBurst=0 → GetDefaultBurst=80
// 2. DefaultRate 默认、DefaultBurst=0 → 200
func TestRateLimitConfig_GetDefaultBurst_沿用默认速率双倍(t *testing.T) {
	assert.Equal(t, 80, (&RateLimitConfig{DefaultRate: 40}).GetDefaultBurst())
	assert.Equal(t, 200, (&RateLimitConfig{}).GetDefaultBurst())
}

// TestMetricsConfig_TracingConfig_RedisInfo_边界赋值
//
// 【功能点】MetricsConfig、TracingConfig、RedisInfo 常见字段在显式赋值后可读回，用于覆盖结构体声明行
// 【测试流程】
// 1. 构造包含 Enabled、路径、采样率、Redis 连接池等字段的结构体
// 2. 断言字段与赋值一致
func TestMetricsConfig_TracingConfig_RedisInfo_边界赋值(t *testing.T) {
	m := MetricsConfig{
		Enabled:      true,
		Path:         "/custom-metrics",
		ExcludePaths: []string{"/health", "/ready"},
	}
	assert.True(t, m.Enabled)
	assert.Equal(t, "/custom-metrics", m.Path)
	assert.Equal(t, []string{"/health", "/ready"}, m.ExcludePaths)

	tr := TracingConfig{
		Enabled:                 true,
		ServiceName:             "svc",
		ExporterType:            "otlp",
		Endpoint:                "localhost:4317",
		SampleRate:              0.25,
		Insecure:                true,
		PropagatorType:          "tracecontext",
		EnableDBTracing:         true,
		EnableRedisTracing:      false,
		EnableHTTPClientTracing: true,
	}
	assert.Equal(t, 0.25, tr.SampleRate)
	assert.True(t, tr.EnableHTTPClientTracing)

	r := RedisInfo{
		AliasName:    "cache",
		Addr:         "127.0.0.1:6379",
		ClusterAddrs: []string{"r1:6379", "r2:6379"},
		UseCluster:   true,
		DB:           3,
		Password:     "redis-pwd",
		PoolSize:     32,
		MinIdleConns: 8,
	}
	assert.True(t, r.UseCluster)
	assert.Equal(t, 3, r.DB)
	assert.Equal(t, 32, r.PoolSize)
}

// TestEtcdInfo_EsInfo_SmtpInfo_字段赋值
//
// 【功能点】EtcdInfo、EsInfo、SmtpInfo 各字段赋值与零值边界（空切片、空字符串、nil 指针）
// 【测试流程】
// 1. 分别为三类配置赋值典型字段并断言
// 2. Etcd Timeout 为 nil 时字段为零值
func TestEtcdInfo_EsInfo_SmtpInfo_字段赋值(t *testing.T) {
	e := EtcdInfo{
		Addresses: []string{"etcd1:2379"},
		Username:  "etcd-user",
		Password:  "etcd-pass",
		Timeout:   nil,
	}
	assert.Equal(t, []string{"etcd1:2379"}, e.Addresses)
	assert.Nil(t, e.Timeout)

	es := EsInfo{
		Addresses: []string{"http://es:9200"},
		Username:  "elastic",
		Password:  "es-pass",
	}
	assert.Len(t, es.Addresses, 1)

	smtp := SmtpInfo{
		Host:     "smtp.mail.com",
		Username: "noreply@mail.com",
		Password: "smtp-secret",
		Sender:   "noreply@mail.com",
	}
	assert.Equal(t, "smtp.mail.com", smtp.Host)
}

// TestMessageQueue_handleMessage_无处理函数时Ack
//
// 【功能点】未配置 FunWithCtx 与 Fun 时直接 Ack，避免消息悬挂
// 【测试流程】
// 1. 构造带 stub Acknowledger 的 Delivery
// 2. 调用 handleMessage，断言 Ack 被调用一次且无 Nack
func TestMessageQueue_handleMessage_无处理函数时Ack(t *testing.T) {
	stub := &stubAcknowledger{}
	mq := MessageQueue{}
	msg := amqp.Delivery{Acknowledger: stub, Body: []byte("noop")}
	mq.handleMessage(context.Background(), msg)
	assert.Equal(t, 1, stub.ackCount)
	assert.Empty(t, stub.nackCalls)
}

// TestMessageQueue_handleMessage_FunWithCtx成功时Ack
//
// 【功能点】FunWithCtx 返回 nil 时确认消息
// 【测试流程】
// 1. 注入返回 nil 的 FunWithCtx
// 2. handleMessage 后断言 Ack 次数为 1
func TestMessageQueue_handleMessage_FunWithCtx成功时Ack(t *testing.T) {
	stub := &stubAcknowledger{}
	mq := MessageQueue{
		FunWithCtx: func(context.Context, string) error { return nil },
	}
	msg := amqp.Delivery{Acknowledger: stub, Body: []byte("ok")}
	mq.handleMessage(context.Background(), msg)
	assert.Equal(t, 1, stub.ackCount)
	assert.Empty(t, stub.nackCalls)
}

// TestMessageQueue_handleMessage_Fun成功时Ack
//
// 【功能点】在未配置 FunWithCtx 时回退到 Fun，成功则 Ack
// 【测试流程】
// 1. 仅设置 Fun 返回 nil
// 2. handleMessage 后断言 Ack
func TestMessageQueue_handleMessage_Fun成功时Ack(t *testing.T) {
	stub := &stubAcknowledger{}
	mq := MessageQueue{
		Fun: func(string) error { return nil },
	}
	msg := amqp.Delivery{Acknowledger: stub, Body: []byte("legacy")}
	mq.handleMessage(context.Background(), msg)
	assert.Equal(t, 1, stub.ackCount)
}

// TestMessageQueue_handleMessage_失败未超最大重试时Nack重入队
//
// 【功能点】处理失败且重试次数小于 MaxRetry 时 Nack(requeue=true)
// 【测试流程】
// 1. FunWithCtx 返回错误，ConsumeConfig.MaxRetry=5，x-death count=2
// 2. 断言产生一次 Nack 且 requeue 为 true
func TestMessageQueue_handleMessage_失败未超最大重试时Nack重入队(t *testing.T) {
	stub := &stubAcknowledger{}
	mq := MessageQueue{
		FunWithCtx: func(context.Context, string) error { return errors.New("fail") },
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 5,
		},
	}
	msg := amqp.Delivery{
		Acknowledger: stub,
		Body:         []byte("retry"),
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{"count": int64(2)},
			},
		},
	}
	mq.handleMessage(context.Background(), msg)
	assert.Equal(t, 0, stub.ackCount)
	requireSingleNack(t, stub, false, true)
}

// TestMessageQueue_handleMessage_失败达到默认最大重试时丢弃
//
// 【功能点】MaxRetry<=0 时使用默认 3；达到上限后 Nack(requeue=false)
// 【测试流程】
// 1. MaxRetry 置 0，x-death count=3
// 2. 断言 Nack multiple=false 且 requeue=false
func TestMessageQueue_handleMessage_失败达到默认最大重试时丢弃(t *testing.T) {
	stub := &stubAcknowledger{}
	mq := MessageQueue{
		FunWithCtx: func(context.Context, string) error { return errors.New("fail") },
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 0,
		},
	}
	msg := amqp.Delivery{
		Acknowledger: stub,
		Body:         []byte("giveup"),
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{"count": int64(3)},
			},
		},
	}
	mq.handleMessage(context.Background(), msg)
	requireSingleNack(t, stub, false, false)
}

func requireSingleNack(t *testing.T, stub *stubAcknowledger, multiple, requeue bool) {
	t.Helper()
	assert.Len(t, stub.nackCalls, 1)
	assert.Equal(t, multiple, stub.nackCalls[0].multiple)
	assert.Equal(t, requeue, stub.nackCalls[0].requeue)
}

// TestMessageQueue_waitForConfirm_确认通道未初始化
//
// 【功能点】confirmChan 为 nil 时返回明确错误
// 【测试流程】
// 1. 构造未设置 confirmChan 的 MessageQueue
// 2. 调用 waitForConfirm，断言错误包含业务语义
func TestMessageQueue_waitForConfirm_确认通道未初始化(t *testing.T) {
	mq := MessageQueue{}
	err := mq.waitForConfirm(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "确认通道未初始化")
}

// TestMessageQueue_waitForConfirm_上下文已取消
//
// 【功能点】ctx 已取消时优先返回超时类错误
// 【测试流程】
// 1. 使用已 cancel 的 context 与阻塞型 confirmChan
// 2. waitForConfirm 应立即返回错误
func TestMessageQueue_waitForConfirm_上下文已取消(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mq := MessageQueue{
		confirmChan: make(chan amqp.Confirmation),
	}
	err := mq.waitForConfirm(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "等待确认超时")
}

// TestMessageQueue_waitForConfirm_通道已关闭
//
// 【功能点】确认通道关闭且无可读确认时返回错误
// 【测试流程】
// 1. 创建并关闭 buffered confirmChan
// 2. waitForConfirm 读到 ok=false 分支
func TestMessageQueue_waitForConfirm_通道已关闭(t *testing.T) {
	ch := make(chan amqp.Confirmation)
	close(ch)
	mq := MessageQueue{confirmChan: ch}
	err := mq.waitForConfirm(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "确认通道已关闭")
}

// TestMessageQueue_waitForConfirm_服务端否定确认
//
// 【功能点】收到 Ack=false 的 Confirmation 时返回错误
// 【测试流程】
// 1. 向缓冲通道写入 Ack: false
// 2. waitForConfirm 返回包含 deliveryTag 的错误
func TestMessageQueue_waitForConfirm_服务端否定确认(t *testing.T) {
	ch := make(chan amqp.Confirmation, 1)
	ch <- amqp.Confirmation{Ack: false, DeliveryTag: 42}
	mq := MessageQueue{confirmChan: ch}
	err := mq.waitForConfirm(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "消息未被确认")
}

// TestMessageQueue_waitForConfirm_成功路径
//
// 【功能点】Ack=true 时正常返回
// 【测试流程】
// 1. 向缓冲通道写入成功确认
// 2. waitForConfirm 返回 nil
func TestMessageQueue_waitForConfirm_成功路径(t *testing.T) {
	ch := make(chan amqp.Confirmation, 1)
	ch <- amqp.Confirmation{Ack: true, DeliveryTag: 1}
	mq := MessageQueue{confirmChan: ch}
	assert.NoError(t, mq.waitForConfirm(context.Background()))
}

// TestMessageQueue_initConn_复用非关闭连接
//
// 【功能点】Conn 已存在且未关闭时不发起新的 Dial
// 【测试流程】
// 1. 将 Conn 指向零值 Connection（IsClosed 为 false）
// 2. 连续调用 initConn 均应成功且不依赖网络
func TestMessageQueue_initConn_复用非关闭连接(t *testing.T) {
	mq := MessageQueue{
		Conn: &amqp.Connection{},
	}
	assert.NoError(t, mq.initConn())
	assert.NoError(t, mq.initConn())
	assert.NotNil(t, mq.Conn)
}

// TestMessageQueue_initChannel_通道已打开则短路
//
// 【功能点】Channel 非 nil 且未关闭时跳过后续 AMQP 初始化逻辑
// 【测试流程】
// 1. 预置零值 Channel（未标记关闭）
// 2. initChannel 立即返回 nil
func TestMessageQueue_initChannel_通道已打开则短路(t *testing.T) {
	mq := MessageQueue{
		Channel: &amqp.Channel{},
	}
	assert.NoError(t, mq.initChannel())
	assert.NotNil(t, mq.Channel)
}

// TestMessageQueue_InitChannelForProducer_通道已打开则短路
//
// 【功能点】生产者初始化在 Channel 有效时不访问网络
// 【测试流程】
// 1. 预置零值 Channel
// 2. InitChannelForProducer 返回 nil
func TestMessageQueue_InitChannelForProducer_通道已打开则短路(t *testing.T) {
	mq := MessageQueue{
		Channel: &amqp.Channel{},
	}
	assert.NoError(t, mq.InitChannelForProducer())
}

// TestMessageQueue_GetFuncInfo_未设置处理函数
//
// 【功能点】Fun 非函数类型（nil）时 GetFuncInfo 返回空字符串
// 【测试流程】
// 1. 使用零值 MessageQueue
// 2. 断言 GetFuncInfo 为空
func TestMessageQueue_GetFuncInfo_未设置处理函数(t *testing.T) {
	mq := MessageQueue{}
	assert.Empty(t, mq.GetFuncInfo())
}

// TestScheduleInfo_GetFuncInfo_Cmd非函数
//
// 【功能点】Cmd 为 nil 时反射判定非函数，返回空字符串
// 【测试流程】
// 1. 构造 Cmd 为零值的 ScheduleInfo
// 2. GetFuncInfo 返回 ""
func TestScheduleInfo_GetFuncInfo_Cmd非函数(t *testing.T) {
	var s ScheduleInfo
	assert.Empty(t, s.GetFuncInfo())
}

// TestMessageQueue_handleMessage_Fun失败时按重试策略Nack
//
// 【功能点】仅配置 Fun 时失败分支与 FunWithCtx 一致
// 【测试流程】
// 1. Fun 返回错误且 x-death count 低于 MaxRetry
// 2. 断言单次 Nack 且 requeue=true
func TestMessageQueue_handleMessage_Fun失败时按重试策略Nack(t *testing.T) {
	stub := &stubAcknowledger{}
	mq := MessageQueue{
		Fun: func(string) error { return errors.New("boom") },
		ConsumeConfig: ConsumeConfig{
			MaxRetry: 4,
		},
	}
	msg := amqp.Delivery{
		Acknowledger: stub,
		Body:         []byte("x"),
		Headers: amqp.Table{
			"x-death": []interface{}{
				amqp.Table{"count": int64(1)},
			},
		},
	}
	mq.handleMessage(context.Background(), msg)
	requireSingleNack(t, stub, false, true)
}

// TestMessageQueue_initDeadLetterQueue_表驱动
//
// 【功能点】initDeadLetterQueue 在交换机/队列/绑定失败时返回包装错误；成功路径下 TTL 映射到 x-message-ttl
// 【测试流程】
// 1. 使用 deadLetterStubChannel 注入各阶段错误并断言错误文案
// 2. TTL 为 0 与大于 0 时分别断言 QueueDeclare 的 args
func TestMessageQueue_initDeadLetterQueue_表驱动(t *testing.T) {
	baseMQ := func() *MessageQueue {
		return &MessageQueue{
			MQName:       "mq",
			QueueName:    "order.q",
			ExchangeName: "order.ex",
			ExchangeType: "topic",
			RoutingKey:   "rk",
			DeadLetter: DeadLetterConfig{
				Enabled:    true,
				Exchange:   "custom.dlx",
				QueueName:  "custom.dlq",
				RoutingKey: "dead.rk",
			},
		}
	}

	t.Run("ExchangeDeclare 失败", func(t *testing.T) {
		stub := &deadLetterStubChannel{exchangeErr: errors.New("boom")}
		err := baseMQ().initDeadLetterQueue(stub)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "声明死信交换机失败")
	})

	t.Run("QueueDeclare 失败", func(t *testing.T) {
		stub := &deadLetterStubChannel{queueErr: errors.New("qe")}
		err := baseMQ().initDeadLetterQueue(stub)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建死信队列失败")
	})

	t.Run("QueueBind 失败", func(t *testing.T) {
		stub := &deadLetterStubChannel{bindErr: errors.New("be")}
		err := baseMQ().initDeadLetterQueue(stub)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "死信队列绑定失败")
	})

	t.Run("成功且无 TTL 时 args 为 nil", func(t *testing.T) {
		stub := &deadLetterStubChannel{}
		mq := baseMQ()
		mq.DeadLetter.MessageTTL = 0
		assert.NoError(t, mq.initDeadLetterQueue(stub))
		assert.Nil(t, stub.lastQueueArgs)
	})

	t.Run("MessageTTL 写入队列参数", func(t *testing.T) {
		stub := &deadLetterStubChannel{}
		mq := baseMQ()
		mq.DeadLetter.MessageTTL = 60000
		assert.NoError(t, mq.initDeadLetterQueue(stub))
		requireTTLArg(t, stub.lastQueueArgs, int64(60000))
	})
}

func requireTTLArg(t *testing.T, tab amqp.Table, want int64) {
	t.Helper()
	require.NotNil(t, tab)
	got, ok := tab["x-message-ttl"]
	require.True(t, ok)
	assert.Equal(t, want, got)
}

// TestMessageQueue_Close_连接与通道均为nil时不Panic
//
// 【功能点】验证 Close 在 Conn、Channel 均为 nil 时安全返回（短路两处分支）
// 【测试流程】
// 1. 构造零值 MessageQueue
// 2. 在 NotPanics 内调用 Close
func TestMessageQueue_Close_连接与通道均为nil时不Panic(t *testing.T) {
	mq := MessageQueue{}
	assert.NotPanics(t, func() {
		mq.Close()
	})
}

// TestMessageQueue_GetFuncInfo_类型化nil消费函数返回空串
//
// 【功能点】覆盖 Fun 为 nil func、reflect.FuncForPC(0) 返回 nil 时走 funcInfo 空分支
// 【测试流程】
// 1. var fn func(string) error 并赋给 MessageQueue.Fun
// 2. 断言 GetFuncInfo 为空
func TestMessageQueue_GetFuncInfo_类型化nil消费函数返回空串(t *testing.T) {
	var fn func(string) error
	mq := MessageQueue{Fun: fn}
	assert.Empty(t, mq.GetFuncInfo())
}

// TestScheduleInfo_GetFuncInfo_类型化nil定时任务函数返回空串
//
// 【功能点】覆盖 Cmd 为 nil func、FuncForPC(0) 为 nil 的分支
// 【测试流程】
// 1. var cmd func() 并赋给 ScheduleInfo.Cmd
// 2. 断言 GetFuncInfo 为空
func TestScheduleInfo_GetFuncInfo_类型化nil定时任务函数返回空串(t *testing.T) {
	var cmd func()
	s := ScheduleInfo{Cmd: cmd}
	assert.Empty(t, s.GetFuncInfo())
}
