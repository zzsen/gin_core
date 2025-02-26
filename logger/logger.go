package logger

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/constant"
	"github.com/zzsen/gin_core/model/config"
)

// Logger 是日志记录器实例
var Logger *logrus.Logger

var defaultFilePath = "./log/"
var defaultFileName = "app"

// defaultRotationTime 默认日志轮转时间
var defaultRotationTime = 60

// defaultMaxAge 默认日志最大保存时间
var defaultMaxAge = 30

// defaultRotationSize 默认日志轮转大小
var defaultRotationSize = 1024

// init 函数在包被导入时执行，用于初始化日志设置
func init() {
	// 初始化日志记录器
	Logger = logrus.New()

	// 设置日志级别为 Debug
	Logger.SetLevel(logrus.DebugLevel)

	defaultLogWriter, _ := initRotatelogs(config.LoggersConfig{}, config.LoggerConfig{}, "")

	// 配置 lfshook
	writeMap := lfshook.WriterMap{}

	for _, logLevel := range logrus.AllLevels {
		writeMap[logLevel] = defaultLogWriter
	}
	// 添加 lfshook 到 Logger
	Logger.AddHook(lfshook.NewHook(writeMap, &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	}))
}

// CustomFormatter 自定义日志格式化器
type CustomFormatter struct {
	logrus.TextFormatter
	PrintFile constant.LogPrintFileEnum
	PrintFunc bool
}

// Format 实现 logrus.Formatter 接口的 Format 方法
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// 获取调用者信息
	if pc, file, line, ok := runtime.Caller(10); ok {
		funcName := runtime.FuncForPC(pc).Name()
		// 将文件和行号信息添加到日志条目的数据中
		if f.PrintFile == constant.PrintAbsoluteFile {
			entry.Data["file"] = fmt.Sprintf("%s:%d", file, line)
		} else if f.PrintFile == constant.PrintRelativeFile {
			if pwd, err := os.Getwd(); err == nil {
				if relPath, err := filepath.Rel(pwd, file); err == nil {
					file = relPath
				}
			}
			entry.Data["file"] = fmt.Sprintf("%s:%d", file, line)
		}
		if f.PrintFunc {
			entry.Data["func"] = funcName
		}
	}
	// 调用默认的 TextFormatter 进行格式化
	return f.TextFormatter.Format(entry)
}

func InitLogger(loggersConfig config.LoggersConfig) *logrus.Logger {
	// 初始化日志记录器
	Logger = logrus.New()

	// 设置日志级别为 Debug
	Logger.SetLevel(logrus.TraceLevel)

	for _, logLevel := range logrus.AllLevels {
		// 配置 lfshook
		writeMap := lfshook.WriterMap{}
		printFile := constant.NoPrintFile
		printFunc := false

		defaultLogWriter, _ := initRotatelogs(config.LoggersConfig{}, config.LoggerConfig{}, logLevel.String())
		writeMap[logLevel] = defaultLogWriter

		for _, loggerConfig := range loggersConfig.Loggers {
			level, err := logrus.ParseLevel(loggerConfig.Level)
			if err == nil && level == logLevel {
				// 设置是否打印文件信息
				if loggersConfig.PrintFile != nil {
					printFile = *loggersConfig.PrintFile
				}
				if loggerConfig.PrintFile != nil {
					printFile = *loggerConfig.PrintFile
				}

				// 设置是否打印函数信息
				if loggersConfig.PrintFunc != nil {
					printFunc = *loggersConfig.PrintFunc
				}
				if loggerConfig.PrintFunc != nil {
					printFunc = *loggerConfig.PrintFunc
				}

				if logWriter, err := initRotatelogs(loggersConfig, loggerConfig, level.String()); err == nil {
					writeMap[level] = logWriter
					break
				}
			}
		}

		writeMap[logLevel] = defaultLogWriter
		// 添加 lfshook 到 Logger
		Logger.AddHook(lfshook.NewHook(writeMap, &CustomFormatter{
			PrintFile: printFile,
			PrintFunc: printFunc,
			TextFormatter: logrus.TextFormatter{
				TimestampFormat: "2006-01-02 15:04:05",
			},
		}))
	}

	return Logger
}

func initRotatelogs(globalConfig config.LoggersConfig,
	loggerConfig config.LoggerConfig, level string) (*rotatelogs.RotateLogs, error) {
	// 设置日志文件路径
	filePath := defaultFilePath
	if globalConfig.FilePath != "" {
		filePath = globalConfig.FilePath
	}
	if loggerConfig.FilePath != "" {
		filePath = loggerConfig.FilePath
	}

	// 设置日志文件名
	fileName := defaultFileName
	if level != "" {
		fileName = level
	}
	if loggerConfig.FileName != "" {
		fileName = loggerConfig.FileName
	}

	// 设置日志轮转时间
	rotationTime := defaultRotationTime
	if globalConfig.RotationTime != 0 {
		rotationTime = globalConfig.RotationTime
	}
	if loggerConfig.RotationTime != 0 {
		rotationTime = loggerConfig.RotationTime
	}

	// 设置日志最大保存时间
	maxAge := defaultMaxAge
	if globalConfig.MaxAge != 0 {
		maxAge = globalConfig.MaxAge
	}
	if loggerConfig.MaxAge != 0 {
		maxAge = loggerConfig.MaxAge
	}

	// 设置日志轮转大小
	rotationSize := defaultRotationSize
	if globalConfig.RotationSize != 0 {
		rotationSize = globalConfig.RotationSize
	}
	if loggerConfig.RotationSize != 0 {
		rotationSize = loggerConfig.RotationSize
	}

	// 设置输出到文件
	fullFileName := path.Join(filePath, fileName)

	filePattern := fullFileName + ".%Y%m%d.log"
	if rotationTime < 60 {
		filePattern = fullFileName + ".%Y%m%d%H%M.log"
	} else if rotationTime >= 60 && rotationTime < 24*60 {
		filePattern = fullFileName + ".%Y%m%d%H.log"
	}
	// 设置 rotatelogs，实现日志文件轮转
	logWriter, _ := rotatelogs.New(
		filePattern,
		rotatelogs.WithLinkName(fullFileName),
		rotatelogs.WithMaxAge(time.Duration(maxAge*24)*time.Hour),
		rotatelogs.WithRotationTime(time.Duration(rotationTime)*time.Minute),
		rotatelogs.WithRotationSize(int64(rotationSize)*1024),
	)
	return logWriter, nil
}

// Add 函数用于添加日志记录
func Add(requestId, info string, err error) {
	if err != nil {
		// 如果有错误，记录 Error 级别的日志
		Logger.WithFields(logrus.Fields{
			"request_id": requestId,
			"info":       info,
			"error":      err.Error(),
		}).Error()
	} else {
		// 如果没有错误，记录 Info 级别的日志
		Logger.WithFields(logrus.Fields{
			"request_id": requestId,
			"info":       info,
			"error":      "",
		}).Info()
	}
}

func Info(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		Logger.Infof(msg, arg...)
	} else {
		Logger.Info(msg)
	}
}

func Error(msg string, arg ...interface{}) {
	if len(arg) > 0 {
		Logger.Errorf(msg, arg...)
	} else {
		Logger.Error(msg)
	}
}
