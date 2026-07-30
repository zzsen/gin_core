package configcenter

import (
	"fmt"
	"path"
	"strings"

	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
)

// ApplyWhitelist 将 next 中白名单字段应用到 prev（就地修改 prev），并触发运行时副作用。
//
// 约定：
// - `log.level`：更新 prev.Log.Loggers[*].Level，并调用 logger.SetLevel（取 next 中第一个非空 logger level）
// - `rateLimit.*`：整体用 next.RateLimit 覆盖 prev.RateLimit（中间件每请求读 BaseConfig，故足够）
//
// 【流程】
// 1. 校验入参；whitelist 为空时使用内置默认
// 2. 按规则匹配并应用 log.level / rateLimit
// 3. 未识别的规则忽略（向前兼容扩展）
func ApplyWhitelist(prev, next *config.BaseConfig, whitelist []string) error {
	// 步骤1：入参与默认白名单
	if prev == nil || next == nil {
		return fmt.Errorf("configcenter: apply nil BaseConfig")
	}
	if len(whitelist) == 0 {
		whitelist = []string{"log.level", "rateLimit.*"}
	}

	// 步骤2–3：按规则应用
	appliedLog := false
	appliedRL := false
	for _, rule := range whitelist {
		rule = strings.TrimSpace(rule)
		switch {
		case rule == "log.level" || matchPath("log.level", rule):
			if err := applyLogLevel(prev, next); err != nil {
				return err
			}
			appliedLog = true
		case rule == "rateLimit" || strings.HasPrefix(rule, "rateLimit.") || rule == "rateLimit.*":
			prev.RateLimit = next.RateLimit
			appliedRL = true
		}
	}
	_ = appliedLog
	_ = appliedRL
	return nil
}

// applyLogLevel 将 next 中的日志级别同步到 prev，并调用 logger.SetLevel。
//
// 【流程】
// 1. 取 next 首个非空 logger level；为空则跳过
// 2. SetLevel 更新运行时全局级别
// 3. 回写 prev.Log.Loggers 各级别字段
func applyLogLevel(prev, next *config.BaseConfig) error {
	// 步骤1：解析目标级别
	level := firstLoggerLevel(next)
	if level == "" {
		return nil
	}
	// 步骤2：运行时副作用
	if err := logger.SetLevel(level); err != nil {
		return fmt.Errorf("configcenter: set log level: %w", err)
	}
	// 步骤3：写回配置模型
	if len(prev.Log.Loggers) == 0 {
		prev.Log.Loggers = []config.LoggerConfig{{Level: level}}
		return nil
	}
	for i := range prev.Log.Loggers {
		prev.Log.Loggers[i].Level = level
	}
	return nil
}

// firstLoggerLevel 返回 cfg 中第一个非空的 logger level；无则空串。
func firstLoggerLevel(cfg *config.BaseConfig) string {
	if cfg == nil {
		return ""
	}
	for _, l := range cfg.Log.Loggers {
		if strings.TrimSpace(l.Level) != "" {
			return strings.TrimSpace(l.Level)
		}
	}
	return ""
}

// matchPath 使用 path.Match 判断 field 是否匹配 pattern（如 log.*）。
func matchPath(field, pattern string) bool {
	ok, err := path.Match(pattern, field)
	return err == nil && ok
}
