package logger

import (
	"strings"

	"github.com/sirupsen/logrus"
)

const loggerTimestampFormat = "2006-01-02 15:04:05"

// normalizeFormat 将配置值规范为 text 或 json；非法/空 → text
func normalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return "json"
	default:
		return "text"
	}
}

// isKnownFormat 判断是否为合法 format 配置值（text / json）
func isKnownFormat(format string) bool {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text", "json":
		return true
	default:
		return false
	}
}

// resolveFormat 解析生效格式：级别配置 > 全局配置 > text
// 非法级别值视为空，回退全局；非法全局回退 text
func resolveFormat(global, level string) string {
	if f := strings.TrimSpace(level); f != "" {
		if isKnownFormat(f) {
			return normalizeFormat(f)
		}
		return normalizeFormat(global)
	}
	return normalizeFormat(global)
}

// newFormatter 按 format 构造 logrus Formatter
func newFormatter(format string) logrus.Formatter {
	switch normalizeFormat(format) {
	case "json":
		return &logrus.JSONFormatter{TimestampFormat: loggerTimestampFormat}
	default:
		return &logrus.TextFormatter{
			FullTimestamp:          true,
			TimestampFormat:        loggerTimestampFormat,
			DisableLevelTruncation: true,
		}
	}
}
