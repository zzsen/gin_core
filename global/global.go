package global

import (
	"context"
	"fmt"
	"sync"
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var (
	DB         *gorm.DB
	DBResolver *gorm.DB
	ES         *elasticsearch.TypedClient
	DBList     map[string]*gorm.DB
	Redis      redis.UniversalClient
	RedisList  map[string]redis.UniversalClient
	BaseConfig config.BaseConfig
	Config     interface{}
	Logger     *logrus.Logger
	GVA_VP     *viper.Viper
	// GVA_LOG    *oplogging.Logger
	GVA_LOG *zap.Logger
	lock    sync.RWMutex
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
	exchangeType string, routingKey string, message string) {
	conn, err := amqp.Dial(BaseConfig.RabbitMQ.Url())
	if err != nil {
		logger.Error("[消息队列] 连接rabbitMQ失败, error: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Error("[消息队列] 连接rabbitMQ通道失败, error: %v", err)
		return
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		queueName,    // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		logger.Error("[消息队列] 定义exchange失败, error: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		exchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		})
	if err != nil {
		logger.Error("[消息队列] 消息发布失败, error: %v", err)
		return
	}

	logger.Info("[消息队列] 消息发布成功, queueName: %s, exchangeName: %s, exchangeType: %s, routingKey: %s, message: %s",
		queueName, exchangeName, exchangeType, routingKey, message)
}
