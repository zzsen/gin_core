// Package logger 提供统一的日志记录功能
// 本文件实现了基于logrus的日志系统，支持日志轮转、多级别配置和结构化日志记录
package logger

import (
	"fmt"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/model/config"
)

// Logger 是日志记录器实例，全局单例
var Logger *logrus.Logger

// printCaller 是否打印调用者信息
var printCaller bool

// 默认配置常量
var defaultFilePath = "./log/" // 默认日志文件路径
var defaultFileName = "app"    // 默认日志文件名
var defaultRotationTime = 60   // 默认日志轮转时间（分钟）
var defaultMaxAge = 30         // 默认日志最大保存时间（天）
var defaultRotationSize = 1024 // 默认日志轮转大小（KB）

// init 函数在包被导入时执行，用于初始化默认日志设置
// 该函数会：
// 1. 创建默认的logrus日志记录器
// 2. 设置默认日志级别为Debug
// 3. 配置默认的日志轮转和输出格式
func init() {
	// 初始化日志记录器
	Logger = logrus.New()

	// 设置日志级别为 Debug
	Logger.SetLevel(logrus.DebugLevel)

	// 初始化默认的日志轮转配置
	defaultLogWriter, err := initRotatelogs(config.LoggersConfig{}, config.LoggerConfig{}, "")
	if err != nil {
		Logger.Errorf("[logger] 初始化默认日志轮转失败: %v", err)
	}

	// 配置 lfshook，为所有日志级别设置相同的输出
	writeMap := lfshook.WriterMap{}

	// 为所有日志级别配置相同的输出
	for _, logLevel := range logrus.AllLevels {
		writeMap[logLevel] = defaultLogWriter
	}

	// 添加 lfshook 到 Logger，设置时间格式
	Logger.AddHook(lfshook.NewHook(writeMap, &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	}))
}

// InitLogger 根据配置初始化日志记录器
// 该函数会：
// 1. 创建新的logrus日志记录器
// 2. 设置日志级别为Trace（最高级别）
// 3. 为每个日志级别配置对应的输出
// 4. 支持自定义日志配置和默认配置回退
// 参数：
//   - loggersConfig: 日志配置信息
//
// 返回：
//   - *logrus.Logger: 配置完成的日志记录器
func InitLogger(loggersConfig config.LoggersConfig) *logrus.Logger {
	// 初始化日志记录器
	Logger := logrus.New()

	// 设置日志级别为 Trace（最高级别，记录所有日志）
	Logger.SetLevel(logrus.TraceLevel)

	// 为每个日志级别配置对应的输出
	for _, logLevel := range logrus.AllLevels {
		// 配置 lfshook
		writeMap := lfshook.WriterMap{}

		// 查找当前日志级别对应的配置
		for _, loggerConfig := range loggersConfig.Loggers {
			level, err := logrus.ParseLevel(loggerConfig.Level)
			if err == nil && level == logLevel {
				// 如果找到匹配的配置，使用该配置初始化日志轮转
				if logWriter, err := initRotatelogs(loggersConfig, loggerConfig, level.String()); err == nil {
					writeMap[level] = logWriter
					break
				}
			}
		}

		// 如果没有找到匹配的配置，使用默认配置
		if writeMap[logLevel] == nil {
			defaultLogWriter, err := initRotatelogs(loggersConfig, config.LoggerConfig{}, logLevel.String())
			if err != nil {
				Logger.Errorf("[logger] 初始化日志轮转失败 [%s]: %v", logLevel.String(), err)
			}
			writeMap[logLevel] = defaultLogWriter
		}

		// 添加 lfshook 到 Logger，配置输出格式
		Logger.AddHook(lfshook.NewHook(writeMap, &logrus.TextFormatter{
			FullTimestamp:          true,                  // 显示完整时间戳
			TimestampFormat:        "2006-01-02 15:04:05", // 时间格式
			DisableLevelTruncation: true,                  // 禁用级别截断
		}))
	}

	// 保存是否打印调用者信息的配置（由包装函数使用）
	printCaller = loggersConfig.PrintCaller
	return Logger
}

// initRotatelogs 初始化日志轮转配置
// 该函数会：
// 1. 设置日志文件路径和文件名
// 2. 配置日志轮转时间、最大保存时间和轮转大小
// 3. 根据轮转时间生成不同的文件命名模式
// 4. 创建支持轮转的日志写入器
// 参数：
//   - globalConfig: 全局日志配置
//   - loggerConfig: 特定日志级别配置
//   - level: 日志级别字符串
//
// 返回：
//   - *rotatelogs.RotateLogs: 日志轮转写入器
//   - error: 错误信息
func initRotatelogs(globalConfig config.LoggersConfig,
	loggerConfig config.LoggerConfig, level string) (*rotatelogs.RotateLogs, error) {
	// 设置日志文件路径，优先级：loggerConfig > globalConfig > default
	filePath := defaultFilePath
	if globalConfig.FilePath != "" {
		filePath = globalConfig.FilePath
	}
	if loggerConfig.FilePath != "" {
		filePath = loggerConfig.FilePath
	}

	// 设置日志文件名，优先级：loggerConfig > level > default
	fileName := defaultFileName
	if level != "" {
		fileName = level
	}
	if loggerConfig.FileName != "" {
		fileName = loggerConfig.FileName
	}

	// 设置日志轮转时间，优先级：loggerConfig > globalConfig > default
	rotationTime := defaultRotationTime
	if globalConfig.RotationTime != 0 {
		rotationTime = globalConfig.RotationTime
	}
	if loggerConfig.RotationTime != 0 {
		rotationTime = loggerConfig.RotationTime
	}

	// 设置日志最大保存时间，优先级：loggerConfig > globalConfig > default
	maxAge := defaultMaxAge
	if globalConfig.MaxAge != 0 {
		maxAge = globalConfig.MaxAge
	}
	if loggerConfig.MaxAge != 0 {
		maxAge = loggerConfig.MaxAge
	}

	// 设置日志轮转大小，优先级：loggerConfig > globalConfig > default
	rotationSize := defaultRotationSize
	if globalConfig.RotationSize != 0 {
		rotationSize = globalConfig.RotationSize
	}
	if loggerConfig.RotationSize != 0 {
		rotationSize = loggerConfig.RotationSize
	}

	// 构建完整的日志文件路径
	fullFileName := path.Join(filePath, fileName)

	// 根据轮转时间生成不同的文件命名模式
	filePattern := fullFileName + ".%Y%m%d.log"
	if rotationTime <= 60 {
		// 轮转时间小于等于1小时，精确到分钟
		filePattern = fullFileName + ".%Y%m%d%H%M.log"
	} else if rotationTime > 60 && rotationTime < 24*60 {
		// 轮转时间在1小时到1天之间，精确到小时
		filePattern = fullFileName + ".%Y%m%d%H.log"
	}

	// 创建支持轮转的日志写入器
	logWriter, err := rotatelogs.New(
		filePattern,                           // 文件命名模式
		rotatelogs.WithLinkName(fullFileName), // 软链接名称
		rotatelogs.WithMaxAge(time.Duration(maxAge*24)*time.Hour),            // 最大保存时间
		rotatelogs.WithRotationTime(time.Duration(rotationTime)*time.Minute), // 轮转时间间隔
		rotatelogs.WithRotationSize(int64(rotationSize)*1024),                // 轮转大小限制
	)
	return logWriter, err
}

// Add 函数用于添加带请求ID的结构化日志记录
// 该函数会：
// 1. 根据是否有错误选择日志级别
// 2. 记录包含请求ID、信息和错误的结构化日志
// 参数：
//   - requestId: 请求标识符
//   - info: 日志信息
//   - err: 错误信息（可为nil）
func Add(requestId, info string, err error) {
	entry := withCallerFields(2)
	if err != nil {
		// 如果有错误，记录 Error 级别的日志
		entry.WithFields(logrus.Fields{
			"request_id": requestId,
			"info":       info,
			"error":      err.Error(),
		}).Error()
	} else {
		// 如果没有错误，记录 Info 级别的日志
		entry.WithFields(logrus.Fields{
			"request_id": requestId,
			"info":       info,
			"error":      "",
		}).Info()
	}
}

// callerInfo 调用者信息结构
type callerInfo struct {
	File string // 文件路径（含行号）
	Func string // 函数名
}

// getCaller 获取调用者信息
// 参数：
//   - skip: 跳过的调用栈层数
//
// 返回：
//   - callerInfo: 包含文件路径和函数名的调用者信息
func getCaller(skip int) callerInfo {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return callerInfo{File: "unknown:0", Func: "unknown"}
	}

	// 获取函数名
	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		funcName = fn.Name()
	}

	return callerInfo{
		File: fmt.Sprintf("%s:%d", file, line),
		Func: funcName,
	}
}

// withCallerFields 为日志条目添加调用者信息
// 参数：
//   - skip: 跳过的调用栈层数
//
// 返回：
//   - *logrus.Entry: 带调用者信息的日志条目
func withCallerFields(skip int) *logrus.Entry {
	if !printCaller {
		return logrus.NewEntry(Logger)
	}
	caller := getCaller(skip)
	return Logger.WithFields(logrus.Fields{
		"func": caller.Func,
		"file": caller.File,
	})
}

// sanitizeLog 对日志消息进行脱敏处理
func sanitizeLog(msg string, arg ...any) string {
	var result string
	if len(arg) > 0 {
		result = fmt.Sprintf(msg, arg...)
	} else {
		result = msg
	}
	return SanitizeMessage(result)
}

// Info 记录Info级别的日志，支持格式化字符串（自动脱敏）
func Info(msg string, arg ...any) {
	entry := withCallerFields(3)
	entry.Info(sanitizeLog(msg, arg...))
}

// Error 记录Error级别的日志，支持格式化字符串（自动脱敏）
func Error(msg string, arg ...any) {
	entry := withCallerFields(3)
	entry.Error(sanitizeLog(msg, arg...))
}

// Warn 记录Warn级别的日志，支持格式化字符串（自动脱敏）
func Warn(msg string, arg ...any) {
	entry := withCallerFields(3)
	entry.Warn(sanitizeLog(msg, arg...))
}

// Debug 记录Debug级别的日志，支持格式化字符串（自动脱敏）
func Debug(msg string, arg ...any) {
	entry := withCallerFields(3)
	entry.Debug(sanitizeLog(msg, arg...))
}

// --- 结构化日志和脱敏功能 ---

// sensitiveKeywords 敏感字段关键词列表（小写）
var sensitiveKeywords = []string{
	"password", "pwd", "passwd",
	"token", "accesstoken", "refreshtoken",
	"secret", "apikey", "api_key",
	"authorization", "auth",
	"credential", "private",
}

// sensitivePatterns 敏感信息正则模式（用于消息内容脱敏）
// 匹配格式：key=value, key:value, "key":"value", key: value 等
var sensitivePatterns []*regexp.Regexp

func init() {
	// 构建敏感信息匹配正则
	for _, keyword := range sensitiveKeywords {
		// 匹配 key=value 格式（URL参数、配置等）
		sensitivePatterns = append(sensitivePatterns,
			regexp.MustCompile(`(?i)(`+keyword+`)\s*[=:]\s*["']?([^"'\s&,;}\]]+)["']?`))
		// 匹配 JSON 格式 "key": "value"
		sensitivePatterns = append(sensitivePatterns,
			regexp.MustCompile(`(?i)"(`+keyword+`)"\s*:\s*"([^"]+)"`))
	}
}

// MaskValue 对敏感值进行脱敏处理
// 保留首尾各2个字符，中间用****替换
func MaskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}

// SanitizeMessage 对消息内容进行脱敏处理
// 自动检测并脱敏消息中的敏感信息
func SanitizeMessage(msg string) string {
	result := msg
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			submatches := pattern.FindStringSubmatch(match)
			if len(submatches) >= 3 {
				value := submatches[2]
				masked := MaskValue(value)
				// 保持原格式，只替换值
				return strings.Replace(match, value, masked, 1)
			}
			// 无法解析时，整体替换为 ****
			return "****"
		})
	}
	return result
}

// isSensitiveField 判断字段名是否为敏感字段
func isSensitiveField(key string) bool {
	keyLower := strings.ToLower(key)
	for _, keyword := range sensitiveKeywords {
		if strings.Contains(keyLower, keyword) {
			return true
		}
	}
	return false
}

// SanitizeFields 对字段进行脱敏处理
// 自动检测敏感字段并进行脱敏
func SanitizeFields(fields map[string]any) logrus.Fields {
	result := make(logrus.Fields)
	for key, value := range fields {
		if isSensitiveField(key) {
			if str, ok := value.(string); ok {
				result[key] = MaskValue(str)
			} else {
				result[key] = "****"
			}
		} else {
			result[key] = value
		}
	}
	return result
}

// InfoWithFields 带结构化字段的Info日志（字段和消息自动脱敏）
func InfoWithFields(fields map[string]any, msg string, arg ...any) {
	entry := withCallerFields(3).WithFields(SanitizeFields(fields))
	entry.Info(sanitizeLog(msg, arg...))
}

// ErrorWithFields 带结构化字段的Error日志（字段和消息自动脱敏）
func ErrorWithFields(fields map[string]any, msg string, arg ...any) {
	entry := withCallerFields(3).WithFields(SanitizeFields(fields))
	entry.Error(sanitizeLog(msg, arg...))
}

// WarnWithFields 带结构化字段的Warn日志（字段和消息自动脱敏）
func WarnWithFields(fields map[string]any, msg string, arg ...any) {
	entry := withCallerFields(3).WithFields(SanitizeFields(fields))
	entry.Warn(sanitizeLog(msg, arg...))
}

// DebugWithFields 带结构化字段的Debug日志（字段和消息自动脱敏）
func DebugWithFields(fields map[string]any, msg string, arg ...any) {
	entry := withCallerFields(3).WithFields(SanitizeFields(fields))
	entry.Debug(sanitizeLog(msg, arg...))
}

// Trace 记录Trace级别的日志（自动脱敏）
func Trace(msg string, arg ...any) {
	entry := withCallerFields(3)
	entry.Trace(sanitizeLog(msg, arg...))
}

// TraceWithFields 带结构化字段的Trace日志（字段和消息自动脱敏）
func TraceWithFields(fields map[string]any, msg string, arg ...any) {
	entry := withCallerFields(3).WithFields(SanitizeFields(fields))
	entry.Trace(sanitizeLog(msg, arg...))
}
