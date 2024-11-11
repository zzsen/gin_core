package config

import (
	"github.com/zzsen/gin_core/logging"
)

type BaseConfig struct {
	System      SystemInfo            `yaml:"system"`
	Service     ServiceInfo           `yaml:"service"`
	Log         logging.LoggersConfig `yaml:"log"`
	Db          DbInfo                `yaml:"db"`
	DbResolvers []DbResolver          `yaml:"dbResolvers"`
	Redis       RedisInfo             `yaml:"redis"`
	RedisList   []RedisInfo           `yaml:"redisList"`
	Smtp        SmtpInfo              `yaml:"smtp"`
}
