// Package initialize 提供各种服务的初始化功能
// 本文件专门负责Etcd客户端的初始化配置
package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// InitEtcd 初始化Etcd客户端
// 该函数会：
// 1. 检查Etcd配置是否存在
// 2. 设置连接超时时间（使用默认值或配置值）
// 3. 创建Etcd客户端连接
// 4. 将客户端实例存储到全局app.Etcd中
func InitEtcd() {
	// 检查Etcd配置是否存在，如果为空则记录错误并返回
	if app.BaseConfig.Etcd == nil {
		logger.Error("[etcd] etcd has no config, please check config")
		return
	}

	// 设置连接超时时间，优先使用配置中的超时值，否则使用默认值
	timeout := constant.DefaultEtcdTimeout
	if app.BaseConfig.Etcd.Timeout != nil {
		timeout = *app.BaseConfig.Etcd.Timeout
	}

	// 创建Etcd v3客户端实例，配置连接参数
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   app.BaseConfig.Etcd.Addresses,        // Etcd服务器地址列表
		Username:    app.BaseConfig.Etcd.Username,         // Etcd访问用户名
		Password:    app.BaseConfig.Etcd.Password,         // Etcd访问密码
		DialTimeout: time.Duration(timeout) * time.Second, // 连接超时时间
	})

	// 如果创建客户端失败，则抛出panic
	if err != nil {
		panic(fmt.Errorf("[etcd] 初始化 etcd client 失败: %v", err))
	}

	// 将Etcd客户端实例存储到全局变量中，供其他模块使用
	app.Etcd = cli
	// 记录初始化成功日志
	logger.Info("[etcd] etcd已初始化")
}
