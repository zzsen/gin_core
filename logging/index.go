package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"strings"
)

type LoggersConfig struct {
	Loggers []LoggerConfig `yaml:"loggers"`
}

type LoggerConfig struct {
	Type       string `yaml:"type"`
	Level      string `yaml:"level"`
	FilePath   string `yaml:"filePath"`
	MaxSize    int    `yaml:"maxSize"`    //单文件最大字节 MB
	MaxAge     int    `yaml:"maxAge"`     //文件最长时间 天
	MaxBackups int    `yaml:"maxBackups"` //最多多少个历史文件
	Compress   bool   `yaml:"compress"`   //是否压缩历史文件
}

type LoggerType string

const (
	StdLogger  LoggerType = "stdLogger"
	FileLogger LoggerType = "fileLogger"
)

// CustomLogger 自定义logger，覆盖默认logger
func CustomLogger(config LoggersConfig) {
	logger := NewTeeWithRotate(config.Loggers)
	ResetDefault(logger)
}

type RotateOptions struct {
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}

//初始化默认logger
var defaultInfoFileConfig = LoggerConfig{
	Type:       string(FileLogger),
	Level:      "info",
	FilePath:   "./log/info.log",
	MaxSize:    100,
	MaxAge:     30,
	MaxBackups: 30,
	Compress:   false,
}
var defaultErrorFileConfig = LoggerConfig{
	Type:       string(FileLogger),
	Level:      "error",
	FilePath:   "./log/error.log",
	MaxSize:    100,
	MaxAge:     30,
	MaxBackups: 30,
	Compress:   false,
}
var defaultStdConfig = LoggerConfig{
	Type:       string(StdLogger),
	Level:      "info",
	FilePath:   "",
	MaxSize:    0,
	MaxAge:     0,
	MaxBackups: 0,
	Compress:   false,
}
var DefaultLoggers = LoggersConfig{Loggers: []LoggerConfig{defaultInfoFileConfig, defaultErrorFileConfig, defaultStdConfig}}
var defaultLogger = NewTeeWithRotate(DefaultLoggers.Loggers)
var (
	Info   = defaultLogger.Info
	Warn   = defaultLogger.Warn
	Error  = defaultLogger.Error
	DPanic = defaultLogger.DPanic
	Panic  = defaultLogger.Panic
	Fatal  = defaultLogger.Fatal
	Debug  = defaultLogger.Debug

	//高性能接口
	HInfo   = defaultLogger.HInfo
	HWarn   = defaultLogger.HWarn
	HError  = defaultLogger.HError
	HDPanic = defaultLogger.HDPanic
	HPanic  = defaultLogger.HPanic
	HFatal  = defaultLogger.HFatal
	HDebug  = defaultLogger.HDebug
)

func NewTeeWithRotate(configs []LoggerConfig) *Logger {
	var cores []zapcore.Core
	cfg := getZapConfig()
	for _, config := range configs {
		level := getLevel(config.Level)
		lv := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level
		})
		loggerType := getLoggerType(config.Type)
		var core zapcore.Core
		if loggerType == FileLogger {
			w := zapcore.AddSync(&lumberjack.Logger{
				Filename:   config.FilePath,
				MaxSize:    config.MaxSize,
				MaxBackups: config.MaxBackups,
				MaxAge:     config.MaxAge,
				Compress:   config.Compress,
				LocalTime:  true,
			})

			core = zapcore.NewCore(
				zapcore.NewConsoleEncoder(cfg.EncoderConfig),
				zapcore.AddSync(w),
				lv,
			)
		} else {
			core = zapcore.NewCore(
				zapcore.NewConsoleEncoder(cfg.EncoderConfig),
				zapcore.AddSync(os.Stderr),
				lv,
			)
		}
		cores = append(cores, core)
	}

	zapInstance := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := zapInstance.Sugar()
	logger := &Logger{
		zapInstance:      zapInstance,
		sugarZapInstance: sugar,
	}
	return logger
}

func getLevel(levelStr string) Level {
	lowerCase := strings.ToLower(levelStr)
	switch lowerCase {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "panic":
		return PanicLevel
	case "dpanic":
		return DPanicLevel
	case "fatal":
		return FatalLevel
	}
	panic("unknown log level:" + levelStr)
}

func getLoggerType(loggerType string) LoggerType {
	switch loggerType {
	case string(StdLogger):
		return StdLogger
	case string(FileLogger):
		return FileLogger
	}
	panic("unknown log type:" + loggerType)
}

// New create a new logger (not support log rotating).
func New(writer io.Writer, level Level) *Logger {
	if writer == nil {
		panic("the writer is nil")
	}
	cfg := getZapConfig()
	core := zapcore.NewCore(
		//zapcore.NewJSONEncoder(cfg.EncoderConfig),
		zapcore.NewConsoleEncoder(cfg.EncoderConfig),
		zapcore.AddSync(writer),
		level,
	)
	zapInstance := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := zapInstance.Sugar()
	logger := &Logger{
		zapInstance:      zapInstance,
		level:            level,
		sugarZapInstance: sugar,
	}
	return logger
}

func getZapConfig() zap.Config {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	cfg.EncoderConfig.ConsoleSeparator = " - "
	//cfg.EncoderConfig.EncodeCaller = zapcore.FullCallerEncoder
	return cfg
}

// ResetDefault not safe for concurrent use
func ResetDefault(l *Logger) {
	defaultLogger = l
	Info = defaultLogger.Info
	Warn = defaultLogger.Warn
	Error = defaultLogger.Error
	DPanic = defaultLogger.DPanic
	Panic = defaultLogger.Panic
	Fatal = defaultLogger.Fatal
	Debug = defaultLogger.Debug

	//高性能接口
	HInfo = defaultLogger.HInfo
	HWarn = defaultLogger.HWarn
	HError = defaultLogger.HError
	HDPanic = defaultLogger.HDPanic
	HPanic = defaultLogger.HPanic
	HFatal = defaultLogger.HFatal
	HDebug = defaultLogger.HDebug
}

func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}
