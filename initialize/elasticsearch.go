package initialize

import (
	"context"

	"github.com/zzsen/gin_core/global"

	elasticsearch "github.com/elastic/go-elasticsearch/v8"
	"github.com/zzsen/gin_core/logger"
)

func InitElasticsearch() {
	if global.BaseConfig.Es == nil {
		logger.Error("[es] single es has no config, please check config")
		return
	}
	esConfig := elasticsearch.Config{
		Addresses: global.BaseConfig.Es.Addresses,
		Password:  global.BaseConfig.Es.Password,
		Username:  global.BaseConfig.Es.Username,
	}

	es, err := elasticsearch.NewTypedClient(esConfig)

	if err != nil {
		logger.Error("[es] Error occurs while creating es client: %s", err)
		return
	} else {
		global.ES = es
	}

	if err = info(); err != nil {
		logger.Error("[es] Error occurs while getting es info: %s", err)
		return
	}
}

func info() error {
	res, err := global.ES.Info().Do(context.Background())

	if err != nil {
		return err
	}

	// Print client and server version numbers.
	logger.Info("[es] Client: %s", elasticsearch.Version)
	logger.Info("[es] Server: %s", res.Version.Int)
	return nil
}
