package config

import "github.com/zzsen/gin_core/constant"

type LoggersConfig struct {
	FilePath     string                     `yaml:"filePath"`
	MaxAge       int                        `yaml:"maxAge"`
	RotationTime int                        `yaml:"rotationTime"`
	RotationSize int                        `yaml:"rotationSize"`
	Loggers      []LoggerConfig             `yaml:"loggers"`
	PrintFile    *constant.LogPrintFileEnum `yaml:"printFile"`
	PrintFunc    *bool                      `yaml:"printFunc"`
}

type LoggerConfig struct {
	Level        string                     `yaml:"level"`
	FileName     string                     `yaml:"fileName"`
	FilePath     string                     `yaml:"filePath"`
	MaxAge       int                        `yaml:"maxAge"`
	RotationTime int                        `yaml:"rotationTime"`
	RotationSize int                        `yaml:"rotationSize"`
	PrintFile    *constant.LogPrintFileEnum `yaml:"printFile"`
	PrintFunc    *bool                      `yaml:"printFunc"`
}
