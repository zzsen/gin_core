package initialize

import (
	"context"
	"fmt"

	"github.com/zzsen/gin_core/app"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/zzsen/gin_core/logger"
)

func InitElasticsearch() {
	if app.BaseConfig.Es == nil {
		logger.Error("[es] single es has no config, please check config")
		return
	}
	esConfig := elasticsearch.Config{
		Addresses: app.BaseConfig.Es.Addresses,
		Password:  app.BaseConfig.Es.Password,
		Username:  app.BaseConfig.Es.Username,
	}

	es, err := elasticsearch.NewTypedClient(esConfig)

	if err != nil {
		panic(fmt.Errorf("[es] 初始化es client失败: %v", err))
	} else {
		app.ES = es
	}

	if err = info(); err != nil {
		panic(fmt.Errorf("[es] 获取es信息失败: %v", err))
	}
}

func info() error {
	res, err := app.ES.Info().Do(context.Background())

	if err != nil {
		return err
	}

	// Print client and server version numbers.
	logger.Info("[es] Client: %s", elasticsearch.Version)
	logger.Info("[es] Server: %s", res.Version.Int)
	return nil
}
