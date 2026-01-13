package app

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

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
