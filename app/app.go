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
	Env                  string
	DB                   *gorm.DB
	DBResolver           *gorm.DB
	ES                   *elasticsearch.TypedClient
	Etcd                 *clientv3.Client
	DBList               map[string]*gorm.DB
	Redis                redis.UniversalClient
	RedisList            map[string]redis.UniversalClient
	BaseConfig           config.BaseConfig
	RabbitMQProducerList map[string]*config.MessageQueue = make(map[string]*config.MessageQueue)
	Config               any                             = new(config.BaseConfig)
	Logger               *logrus.Logger
	GVA_VP               *viper.Viper
	lock                 sync.RWMutex
)
