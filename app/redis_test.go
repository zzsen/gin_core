// Package app Redis 访问函数测试
//
// ==================== 测试说明 ====================
// 本文件包含 app 包中 Redis 相关函数的单元测试。
//
// 测试覆盖内容：
// 1. GetRedisByName 通过名称获取 Redis 客户端实例
// 2. 不存在或 nil 值的错误处理
//
// 运行测试：go test -v ./app/... -run TestRedis
// ==================================================
package app

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestGetRedisByName_Exists 测试获取已存在的 Redis 实例
//
// 【功能点】验证 GetRedisByName 在 Redis 已注册时正确返回实例
// 【测试流程】
// 1. 在 RedisList 中注册一个 Redis 客户端
// 2. 调用 GetRedisByName 获取
// 3. 验证返回实例不为 nil 且无错误
func TestGetRedisByName_Exists(t *testing.T) {
	origRedisList := RedisList
	defer func() { RedisList = origRedisList }()

	mockClient := redis.NewClient(&redis.Options{})
	RedisList = map[string]redis.UniversalClient{
		"test_redis": mockClient,
	}

	client, err := GetRedisByName("test_redis")
	assert.NoError(t, err)
	assert.Equal(t, mockClient, client)
}

// TestGetRedisByName_NotExists 测试获取不存在的 Redis 实例
//
// 【功能点】验证 GetRedisByName 在 Redis 未注册时返回错误
// 【测试流程】
// 1. 确保 RedisList 中无目标名称
// 2. 调用 GetRedisByName
// 3. 验证返回 nil 和错误
func TestGetRedisByName_NotExists(t *testing.T) {
	origRedisList := RedisList
	defer func() { RedisList = origRedisList }()

	RedisList = map[string]redis.UniversalClient{}

	client, err := GetRedisByName("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "nonexistent")
}

// TestGetRedisByName_NilValue 测试获取值为 nil 的 Redis 实例
//
// 【功能点】验证 GetRedisByName 在实例为 nil 时返回错误
// 【测试流程】
// 1. 在 RedisList 中注册一个 nil 值
// 2. 调用 GetRedisByName
// 3. 验证返回错误
func TestGetRedisByName_NilValue(t *testing.T) {
	origRedisList := RedisList
	defer func() { RedisList = origRedisList }()

	RedisList = map[string]redis.UniversalClient{
		"nil_redis": nil,
	}

	client, err := GetRedisByName("nil_redis")
	assert.Error(t, err)
	assert.Nil(t, client)
}
