// Package initialize 提供各种服务的初始化功能
// 本文件专门负责Elasticsearch客户端的初始化配置
package initialize

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
	"github.com/zzsen/gin_core/logger"
)

// InitElasticsearch 初始化Elasticsearch客户端
// 该函数会：
// 1. 检查配置是否存在
// 2. 创建ES客户端连接
// 3. 测试连接并获取服务器信息
// 4. 将客户端实例存储到全局app.ES中
func InitElasticsearch() {
	// 检查ES配置是否存在，如果为空则记录错误并返回
	if app.BaseConfig.Es == nil {
		logger.Error("[es] single es has no config, please check config")
		return
	}

	// 构建ES客户端配置
	esConfig := elasticsearch.Config{
		Addresses: app.BaseConfig.Es.Addresses, // ES服务器地址列表
		Password:  app.BaseConfig.Es.Password,  // ES访问密码
		Username:  app.BaseConfig.Es.Username,  // ES访问用户名
	}

	// 创建类型化的ES客户端实例
	es, err := elasticsearch.NewTypedClient(esConfig)

	// 如果创建客户端失败，则抛出panic
	if err != nil {
		panic(fmt.Errorf("[es] 初始化es client失败: %v", err))
	} else {
		// 将ES客户端实例存储到全局变量中，供其他模块使用
		app.ES = es
	}

	// 测试ES连接并获取服务器信息，如果失败则抛出panic
	if err = info(); err != nil {
		panic(fmt.Errorf("[es] 获取es信息失败: %v", err))
	}
}

// info 获取Elasticsearch服务器信息并测试连接
// 该函数会：
// 1. 调用ES的Info API获取服务器信息
// 2. 记录客户端和服务器版本信息
// 3. 返回错误信息（如果有的话）
func info() error {
	// 调用ES Info API获取服务器信息
	res, err := app.ES.Info().Do(context.Background())

	if err != nil {
		return err
	}

	// 记录客户端和服务器版本信息，用于调试和监控
	logger.Info("[es] Client: %s", elasticsearch.Version)
	logger.Info("[es] Server: %s", res.Version.Int)
	return nil
}
