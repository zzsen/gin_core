package app

import (
	"fmt"
	"sync"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
	"github.com/sirupsen/logrus"

	"github.com/zzsen/gin_core/logger"
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

// GetDbByName 通过名称获取db 如果不存在则panic
func GetDbByName(dbname string) *gorm.DB {
	lock.RLock()
	defer lock.RUnlock()
	db, ok := DBList[dbname]
	if !ok || db == nil {
		panic("db no init")
	}
	return db
}

func GetRedisByName(name string) redis.UniversalClient {
	redis, ok := RedisList[name]
	if !ok || redis == nil {
		panic(fmt.Sprintf("redis `%s` no init", name))
	}
	return redis
}

func SendRabbitMqMsg(queueName string, exchangeName string,
	exchangeType string, routingKey string, message string, mqConfigNames ...string) {
	if len(mqConfigNames) == 0 {
		mqConfigNames = []string{""}
	}
	for _, mqConfigName := range mqConfigNames {
		messageQueue := config.MessageQueue{
			MQName:       mqConfigName,
			QueueName:    queueName,
			ExchangeName: exchangeName,
			ExchangeType: exchangeType,
			RoutingKey:   routingKey,
		}
		// 获取消息队列连接字符串
		mqConnStr := BaseConfig.RabbitMQ.Url()
		// 如果配置了消息队列名称, 则使用对应的消息队列
		if messageQueue.MQName != "" {
			mqConnStr = BaseConfig.RabbitMQList.Url(messageQueue.MQName)
		}
		if mqConnStr == "" {
			logger.Error("[消息队列] 未找到对应的消息队列配置, MQName: %s", messageQueue.MQName)
			return
		}
		messageQueue.MqConnStr = mqConnStr
		sendRabbitMqMsg(messageQueue, message)
	}
}

func sendRabbitMqMsg(messageQueue config.MessageQueue, message string) {
	if _, ok := RabbitMQProducerList[messageQueue.GetInfo()]; !ok {
		RabbitMQProducerList[messageQueue.GetInfo()] = &messageQueue
	}

	err := RabbitMQProducerList[messageQueue.GetInfo()].Publish(message)

	if err != nil {
		logger.Error("[消息队列] 消息发布失败, error: %v", err)
		return
	}

	logger.Info("[消息队列] 消息发布成功, queueInfo: %s, message: %s", messageQueue.GetInfo(), message)
}
