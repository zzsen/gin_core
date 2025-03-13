package config

type LoggersConfig struct {
	FilePath     string         `yaml:"filePath"`
	MaxAge       int            `yaml:"maxAge"`
	RotationTime int            `yaml:"rotationTime"`
	RotationSize int            `yaml:"rotationSize"`
	Loggers      []LoggerConfig `yaml:"loggers"`
	PrintCaller  bool           `yaml:"printCaller"`
}

type LoggerConfig struct {
	Level        string `yaml:"level"`
	FileName     string `yaml:"fileName"`
	FilePath     string `yaml:"filePath"`
	MaxAge       int    `yaml:"maxAge"`
	RotationTime int    `yaml:"rotationTime"`
	RotationSize int    `yaml:"rotationSize"`
}

func (loggersConfig LoggersConfig) ToDbLoggerConfig() LoggersConfig {
	logSubfix := "DB"
	if loggersConfig.FilePath != "" {
		loggersConfig.FilePath = loggersConfig.FilePath + logSubfix
	}
	for i := range loggersConfig.Loggers {
		if loggersConfig.Loggers[i].FileName != "" {
			loggersConfig.Loggers[i].FileName = loggersConfig.Loggers[i].FileName + logSubfix
		}
		if loggersConfig.Loggers[i].FilePath != "" {
			loggersConfig.Loggers[i].FilePath = loggersConfig.Loggers[i].FilePath + logSubfix
		}
	}
	return loggersConfig
}
