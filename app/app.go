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

// GetDbByName 通过名称获取db，如果不存在则返回错误
// 参数：
//   - dbname: 数据库别名
//
// 返回：
//   - *gorm.DB: 数据库实例
//   - error: 如果数据库不存在或未初始化则返回错误
func GetDbByName(dbname string) (*gorm.DB, error) {
	lock.RLock()
	defer lock.RUnlock()
	db, ok := DBList[dbname]
	if !ok || db == nil {
		return nil, fmt.Errorf("[db] 数据库 `%s` 未初始化或不可用", dbname)
	}
	return db, nil
}

// GetRedisByName 通过名称获取Redis客户端，如果不存在则返回错误
// 参数：
//   - name: Redis别名
//
// 返回：
//   - redis.UniversalClient: Redis客户端实例
//   - error: 如果Redis不存在或未初始化则返回错误
func GetRedisByName(name string) (redis.UniversalClient, error) {
	redisClient, ok := RedisList[name]
	if !ok || redisClient == nil {
		return nil, fmt.Errorf("[redis] Redis `%s` 未初始化或不可用", name)
	}
	return redisClient, nil
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
	queueInfo := messageQueue.GetInfo()

	// 优先使用已初始化的发送者
	producer, ok := RabbitMQProducerList[queueInfo]
	if !ok {
		// 如果发送者未初始化，则创建并初始化
		producer = &messageQueue
		// 初始化连接和通道
		err := producer.InitChannelForProducer()
		if err != nil {
			logger.Error("[消息队列] 初始化发送者失败, queueInfo: %s, error: %v", queueInfo, err)
			return
		}
		RabbitMQProducerList[queueInfo] = producer
		logger.Info("[消息队列] 动态初始化发送者成功, queueInfo: %s", queueInfo)
	}

	// 检查通道是否已关闭，如果关闭则重新初始化
	if producer.Channel == nil || producer.Channel.IsClosed() {
		err := producer.InitChannelForProducer()
		if err != nil {
			logger.Error("[消息队列] 重新初始化发送者失败, queueInfo: %s, error: %v", queueInfo, err)
			return
		}
	}

	err := producer.Publish(message)
	if err != nil {
		logger.Error("[消息队列] 消息发布失败, queueInfo: %s, error: %v", queueInfo, err)
		return
	}

	logger.Info("[消息队列] 消息发布成功, queueInfo: %s, message: %s", queueInfo, message)
}
