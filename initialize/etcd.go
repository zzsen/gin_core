package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcd() {
	if app.BaseConfig.Etcd == nil {
		logger.Error("[etcd] etcd has no config, please check config")
		return
	}

	timeout := constant.DefaultEtcdTimeout
	if app.BaseConfig.Etcd.Timeout != nil {
		timeout = *app.BaseConfig.Etcd.Timeout
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   app.BaseConfig.Etcd.Addresses,
		Username:    app.BaseConfig.Etcd.Username,
		Password:    app.BaseConfig.Etcd.Password,
		DialTimeout: time.Duration(timeout) * time.Second,
	})
	if err != nil {
		panic(fmt.Errorf("[etcd] 初始化 etcd client 失败: %v", err))
	}

	app.Etcd = cli
	logger.Info("[etcd] etcd已初始化")
}
