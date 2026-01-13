// Package initialize 提供各种服务的初始化功能
// 本文件专门负责Redis客户端的初始化，支持单实例和集群模式，以及多实例列表配置
package initialize

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/model/config"

	"github.com/redis/go-redis/v9"
	"github.com/zzsen/gin_core/logger"
)

// initRedisClient 初始化单个Redis客户端
// 该函数会：
// 1. 根据配置选择Redis模式（集群模式或单实例模式）
// 2. 创建对应的Redis客户端实例
// 3. 测试连接并返回客户端
// 参数：
//   - redisCfg: Redis配置信息
//
// 返回：
//   - redis.UniversalClient: Redis客户端实例
//   - error: 错误信息
func initRedisClient(redisCfg config.RedisInfo) (redis.UniversalClient, error) {
	var client redis.UniversalClient

	// 根据配置选择Redis模式
	if redisCfg.UseCluster {
		// 使用集群模式，支持Redis Cluster
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    redisCfg.ClusterAddrs, // 集群节点地址列表
			Password: redisCfg.Password,     // 集群访问密码
		})
	} else {
		// 使用单实例模式，连接单个Redis服务器
		client = redis.NewClient(&redis.Options{
			Addr:     redisCfg.Addr,     // Redis服务器地址
			Password: redisCfg.Password, // Redis访问密码
			DB:       redisCfg.DB,       // 数据库编号
		})
	}

	// 测试Redis连接，使用Ping命令验证连通性
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("连接失败, ping failed, err: %v", err)
	}

	// 记录连接成功日志，包含别名和Ping响应
	logger.Info("[redis] redis aliasName: %s, connect ping response: %s", redisCfg.AliasName, pong)
	return client, nil
}

// InitRedis 初始化单个Redis客户端
// 该函数会：
// 1. 检查Redis配置是否存在
// 2. 初始化Redis客户端连接
// 3. 将客户端实例存储到全局app.Redis中
func InitRedis() {
	// 检查Redis配置是否存在
	if app.BaseConfig.Redis == nil {
		panic(exception.NewInitError("redis", "检查配置", fmt.Errorf("未找到Redis配置, 请检查配置")))
	}

	// 初始化Redis客户端
	redisClient, err := initRedisClient(*app.BaseConfig.Redis)
	if err != nil {
		panic(exception.NewInitError("redis", "初始化连接", err))
	}

	// 将Redis客户端实例存储到全局变量中，供其他模块使用
	app.Redis = redisClient
}

// InitRedisList 初始化多个Redis客户端列表
// 该函数会：
// 1. 创建Redis客户端映射表
// 2. 遍历所有Redis配置并初始化连接
// 3. 将客户端实例按别名存储到全局app.RedisList中
func InitRedisList() {
	// 初始化Redis客户端映射表
	redisMap := make(map[string]redis.UniversalClient)

	// 遍历所有Redis配置并初始化连接
	for _, redisCfg := range app.BaseConfig.RedisList {
		client, err := initRedisClient(redisCfg)
		if err != nil {
			panic(exception.NewInitErrorWithConfig("redis", "初始化连接", redisCfg.AliasName, err))
		}
		// 将Redis客户端实例按别名存储到映射表中
		redisMap[redisCfg.AliasName] = client
	}

	// 将Redis客户端映射表存储到全局变量中，供其他模块使用
	app.RedisList = redisMap
}
