package initialize

import (
	"context"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/model/config"

	"github.com/redis/go-redis/v9"
	"github.com/zzsen/gin_core/logger"
	"go.uber.org/zap"
)

func initRedisClient(redisCfg config.RedisInfo) (redis.UniversalClient, error) {
	var client redis.UniversalClient
	// 使用集群模式
	if redisCfg.UseCluster {
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisCfg.ClusterAddrs,
			Password: redisCfg.Password,
		})
	} else {
		// 使用单例模式
		client = redis.NewClient(&redis.Options{
			Addr:     redisCfg.Addr,
			Password: redisCfg.Password,
			DB:       redisCfg.DB,
		})
	}
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		logger.Error("redis connect ping failed, err:", zap.String("name", redisCfg.Name), zap.Error(err))
		return nil, err
	}

	logger.Info("redis connect ping response:", zap.String("name", redisCfg.Name), zap.String("pong", pong))
	return client, nil
}

func InitRedis() {
	redisClient, err := initRedisClient(global.BaseConfig.Redis)
	if err != nil {
		panic(err)
	}
	global.Redis = redisClient
}

func InitRedisList() {
	redisMap := make(map[string]redis.UniversalClient)

	for _, redisCfg := range global.BaseConfig.RedisList {
		client, err := initRedisClient(redisCfg)
		if err != nil {
			panic(err)
		}
		redisMap[redisCfg.Name] = client
	}

	global.RedisList = redisMap
}
