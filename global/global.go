package global

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/zzsen/gin_core/model/config"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var (
	DB         *gorm.DB
	DBResolver *gorm.DB
	DBList     map[string]*gorm.DB
	Redis      redis.UniversalClient
	RedisList  map[string]redis.UniversalClient
	BaseConfig config.BaseConfig
	Config     interface{}
	GVA_VP     *viper.Viper
	// GVA_LOG    *oplogging.Logger
	GVA_LOG *zap.Logger
	lock    sync.RWMutex
)

// GetDbByName 通过名称获取db 如果不存在则panic
func GetDbByName(dbname string) *gorm.DB {
	lock.RLock()
	defer lock.RUnlock()
	db, ok := DBList[dbname]
	if !ok || db == nil {
		panic("db no init")
	}
	return db
}

func GetRedisByName(name string) redis.UniversalClient {
	redis, ok := RedisList[name]
	if !ok || redis == nil {
		panic(fmt.Sprintf("redis `%s` no init", name))
	}
	return redis
}
