package initialize

import (
	"fmt"
	"time"

	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/global"
	"github.com/zzsen/gin_core/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcd() {
	if global.BaseConfig.Etcd == nil {
		logger.Error("[etcd] etcd has no config, please check config")
		return
	}

	timeout := constant.DefaultEtcdTimeout
	if global.BaseConfig.Etcd.Timeout != nil {
		timeout = *global.BaseConfig.Etcd.Timeout
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   global.BaseConfig.Etcd.Addresses,
		Username:    global.BaseConfig.Etcd.Username,
		Password:    global.BaseConfig.Etcd.Password,
		DialTimeout: time.Duration(timeout) * time.Second,
	})
	if err != nil {
		panic(fmt.Errorf("[etcd] 初始化 etcd client 失败: %v", err))
	}

	global.Etcd = cli
	logger.Info("[etcd] etcd已初始化")
}
