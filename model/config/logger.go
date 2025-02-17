package config

type LoggersConfig struct {
	FilePath     string         `yaml:"filePath"`
	MaxAge       int            `yaml:"maxAge"`
	RotationTime int            `yaml:"rotationTime"`
	RotationSize int            `yaml:"rotationSize"`
	Loggers      []LoggerConfig `yaml:"loggers"`
}

type LoggerConfig struct {
	Level        string `yaml:"level"`
	FileName     string `yaml:"fileName"`
	FilePath     string `yaml:"filePath"`
	MaxAge       int    `yaml:"maxAge"`
	RotationTime int    `yaml:"rotationTime"`
	RotationSize int    `yaml:"rotationSize"`
}
