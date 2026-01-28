package app

import (
	"sync"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
	"github.com/sirupsen/logrus"

	"github.com/zzsen/gin_core/model/config"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"gorm.io/gorm"
)

var (
	Env        string
	DB         *gorm.DB
	DBResolver *gorm.DB
	ES         *elasticsearch.TypedClient
	Etcd       *clientv3.Client
	DBList     map[string]*gorm.DB
	Redis      redis.UniversalClient
	RedisList  map[string]redis.UniversalClient
	BaseConfig config.BaseConfig
	// RabbitMQProducerList 使用 sync.Map 存储 RabbitMQ 生产者
	// key: queueInfo (string), value: *config.MessageQueue
	// 使用 sync.Map 替代 map + mutex，提供更好的并发读写性能
	RabbitMQProducerList sync.Map
	Config               any = new(config.BaseConfig)
	Logger               *logrus.Logger
	GVA_VP               *viper.Viper
	// lock 用于保护 DBList、RedisList 等 map 类型全局变量的并发访问
	lock sync.RWMutex
)
