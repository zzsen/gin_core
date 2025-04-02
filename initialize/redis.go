package initialize

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/model/config"

	"github.com/redis/go-redis/v9"
	"github.com/zzsen/gin_core/logger"
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
		return nil, fmt.Errorf("连接失败, ping failed, err: %v", err)
	}

	logger.Info("[redis] redis aliasName: %s, connect ping response: %s", redisCfg.AliasName, pong)
	return client, nil
}

func InitRedis() {
	if global.BaseConfig.Redis == nil {
		logger.Error("[redis] single redis has no config, please check config")
		return
	}
	redisClient, err := initRedisClient(*global.BaseConfig.Redis)
	if err != nil {
		panic(fmt.Errorf("[redis] 初始化redis失败, %s", err.Error()))
	}
	global.Redis = redisClient
}

func InitRedisList() {
	redisMap := make(map[string]redis.UniversalClient)

	for _, redisCfg := range global.BaseConfig.RedisList {
		client, err := initRedisClient(redisCfg)
		if err != nil {
			panic(fmt.Errorf("[redis] 初始化redis [%s]失败, %s", redisCfg.AliasName, err.Error()))
		}
		redisMap[redisCfg.AliasName] = client
	}

	global.RedisList = redisMap
}
