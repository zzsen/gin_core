// Package app 提供框架的全局状态管理，包括数据库连接、缓存客户端、配置和日志等核心组件。
//
// 所有导出变量在框架启动阶段由 core 包自动初始化，业务层可直接引用。
// 对于多实例场景（DBList、RedisList），应使用 GetDbByName / GetRedisByName 等线程安全的访问方法。
package app

import (
	"sync"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
	"github.com/sirupsen/logrus"

	"github.com/zzsen/gin_core/model/config"

	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

var (
	// Env 当前运行环境标识（如 dev、test、prod），由配置文件或环境变量决定
	Env string
	// DB 主数据库连接实例（单库模式）
	DB *gorm.DB
	// DBResolver 数据库读写分离解析器实例，通过 gorm 的 DBResolver 插件实现读写分离
	DBResolver *gorm.DB
	// ES Elasticsearch 类型化客户端实例
	ES *elasticsearch.TypedClient
	// Etcd Etcd 客户端实例，用于服务发现或分布式配置
	Etcd *clientv3.Client
	// DBList 多数据库连接池，按别名索引。并发访问需通过 GetDbByName 方法
	DBList map[string]*gorm.DB
	// Redis 主 Redis 客户端实例（支持单机/哨兵/集群模式）
	Redis redis.UniversalClient
	// RedisList 多 Redis 连接池，按别名索引。并发访问需通过 GetRedisByName 方法
	RedisList map[string]redis.UniversalClient
	// BaseConfig 框架基础配置（解析后的结构体），包含 System、MySQL、Redis 等子配置
	BaseConfig config.BaseConfig
	// RabbitMQProducerList 使用 sync.Map 存储 RabbitMQ 生产者
	// key: queueInfo (string), value: *config.MessageQueue
	// 使用 sync.Map 替代 map + mutex，提供更好的并发读写性能
	RabbitMQProducerList sync.Map
	// Config 用户自定义配置指针，默认指向 BaseConfig，可在启动前替换为包含业务字段的扩展配置结构体
	Config any = new(config.BaseConfig)
	// Logger 全局日志实例（logrus），在框架启动时初始化
	Logger *logrus.Logger
	// lock 用于保护 DBList、RedisList 等 map 类型全局变量的并发访问
	lock sync.RWMutex
)
