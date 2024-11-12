package config

import (
	"github.com/zzsen/gin_core/logger"
)

type BaseConfig struct {
	System      SystemInfo           `yaml:"system"`
	Service     ServiceInfo          `yaml:"service"`
	Log         logger.LoggersConfig `yaml:"log"`
	Db          DbInfo               `yaml:"db"`
	DbResolvers []DbResolver         `yaml:"dbResolvers"`
	Redis       RedisInfo            `yaml:"redis"`
	RedisList   []RedisInfo          `yaml:"redisList"`
	Smtp        SmtpInfo             `yaml:"smtp"`
}
