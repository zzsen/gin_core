package config

type BaseConfig struct {
	System       SystemInfo       `yaml:"system"`
	Service      ServiceInfo      `yaml:"service"`
	Log          LoggersConfig    `yaml:"log"`
	Db           *DbInfo          `yaml:"db"`
	DbList       []DbInfo         `yaml:"dbList"`
	DbResolvers  DbResolvers      `yaml:"dbResolvers"`
	Redis        *RedisInfo       `yaml:"redis"`
	RedisList    []RedisInfo      `yaml:"redisList"`
	RabbitMQ     RabbitMQInfo     `yaml:"rabbitMQ"`
	RabbitMQList RabbitMqListInfo `yaml:"rabbitMQList"`
	Es           *EsInfo          `yaml:"es"`
	Smtp         SmtpInfo         `yaml:"smtp"`
}
