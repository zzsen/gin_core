// Package middleware 提供 HTTP 中间件
// 本文件实现限流中间件，用于控制 API 请求速率
package middleware

import (
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/config"
	"github.com/zzsen/gin_core/model/response"
	"github.com/zzsen/gin_core/ratelimit"
)

var (
	limiterOnce   sync.Once
	globalLimiter ratelimit.Limiter
)

// initLimiter 初始化限流器（单例）
func initLimiter() {
	limiterOnce.Do(func() {
		cfg := app.BaseConfig.RateLimit
		store := cfg.GetStore()

		switch store {
		case "redis":
			if app.Redis != nil {
				globalLimiter = ratelimit.NewRedisLimiter(app.Redis, "ratelimit:")
				logger.Info("[限流] 使用 Redis 限流器")
			} else {
				logger.Warn("[限流] Redis 未初始化，降级为内存限流器")
				globalLimiter = ratelimit.NewMemoryLimiter(time.Duration(cfg.GetCleanupInterval()) * time.Second)
			}
		default:
			globalLimiter = ratelimit.NewMemoryLimiter(time.Duration(cfg.GetCleanupInterval()) * time.Second)
			logger.Info("[限流] 使用内存限流器")
		}
	})
}

// RateLimitHandler 限流中间件
// 根据配置的规则对请求进行限流
func RateLimitHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := app.BaseConfig.RateLimit
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 初始化限流器
		initLimiter()
		if globalLimiter == nil {
			logger.Error("[限流] 限流器初始化失败")
			c.Next()
			return
		}

		// 查找匹配的规则
		rule := findMatchingRule(c.Request.Method, c.Request.URL.Path, cfg.Rules)

		// 确定限流参数
		var rateLimit, burst int
		var keyType, message string

		if rule != nil {
			rateLimit = rule.GetRate()
			burst = rule.GetBurst()
			keyType = rule.GetKeyType()
			message = rule.Message
		}

		// 使用默认值
		if rateLimit <= 0 {
			rateLimit = cfg.GetDefaultRate()
		}
		if burst <= 0 {
			burst = cfg.GetDefaultBurst()
		}
		if keyType == "" {
			keyType = "ip"
		}
		if message == "" {
			message = cfg.GetMessage()
		}

		// 生成限流键
		key := generateRateLimitKey(c, keyType, c.Request.URL.Path)

		// 检查是否允许
		allowed, err := globalLimiter.Allow(c.Request.Context(), key, rateLimit, burst)
		if err != nil {
			logger.Error("[限流] 检查失败: %v", err)
			c.Next()
			return
		}

		if !allowed {
			logger.Warn("[限流] 请求被限流, key: %s, path: %s", key, c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusTooManyRequests, response.Response{
				Code: http.StatusTooManyRequests,
				Msg:  message,
			})
			return
		}

		c.Next()
	}
}

// findMatchingRule 查找匹配的限流规则
// 优先级：精确匹配 > 通配符匹配 > 无匹配
func findMatchingRule(method, requestPath string, rules []config.RateLimitRule) *config.RateLimitRule {
	var wildcardMatch *config.RateLimitRule

	for i := range rules {
		rule := &rules[i]

		// 检查 HTTP 方法
		if rule.Method != "" && !strings.EqualFold(rule.Method, method) {
			continue
		}

		// 精确匹配
		if rule.Path == requestPath {
			return rule
		}

		// 通配符匹配
		if strings.HasSuffix(rule.Path, "/*") {
			prefix := strings.TrimSuffix(rule.Path, "/*")
			if strings.HasPrefix(requestPath, prefix) {
				// 保留最长匹配
				if wildcardMatch == nil || len(rule.Path) > len(wildcardMatch.Path) {
					wildcardMatch = rule
				}
			}
		}

		// 路径模式匹配
		if matched, _ := path.Match(rule.Path, requestPath); matched {
			if wildcardMatch == nil || len(rule.Path) > len(wildcardMatch.Path) {
				wildcardMatch = rule
			}
		}
	}

	return wildcardMatch
}

// generateRateLimitKey 生成限流键
func generateRateLimitKey(c *gin.Context, keyType, requestPath string) string {
	switch keyType {
	case "ip":
		return "ip:" + c.ClientIP() + ":" + requestPath
	case "user":
		// 尝试从上下文获取用户 ID
		if userID, exists := c.Get("userID"); exists {
			return "user:" + toString(userID) + ":" + requestPath
		}
		if userID, exists := c.Get("user_id"); exists {
			return "user:" + toString(userID) + ":" + requestPath
		}
		// 降级为 IP 限流
		return "ip:" + c.ClientIP() + ":" + requestPath
	case "global":
		return "global:" + requestPath
	default:
		return "ip:" + c.ClientIP() + ":" + requestPath
	}
}

// toString 将任意类型转为字符串
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return string(rune(val))
	case int64:
		return string(rune(val))
	case uint:
		return string(rune(val))
	case uint64:
		return string(rune(val))
	default:
		return ""
	}
}

// GetLimiter 获取全局限流器实例
func GetLimiter() ratelimit.Limiter {
	initLimiter()
	return globalLimiter
}

// CloseLimiter 关闭限流器
func CloseLimiter() error {
	if globalLimiter != nil {
		return globalLimiter.Close()
	}
	return nil
}
