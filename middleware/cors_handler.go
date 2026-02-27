// Package middleware 提供 HTTP 中间件
// 本文件实现 CORS（跨域资源共享）中间件
package middleware

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
)

// CORSHandler 跨域资源共享中间件
// 用于处理浏览器的跨域请求，支持预检请求（OPTIONS）
// 配置项通过 app.BaseConfig.CORS 进行设置
//
// 功能特性：
// - 支持配置允许的来源（支持通配符 *）
// - 支持配置允许的 HTTP 方法
// - 支持配置允许的请求头
// - 支持配置暴露的响应头
// - 支持配置是否允许携带凭证（Cookie）
// - 支持配置预检请求缓存时间
//
// 使用示例：
//
//	在配置文件中启用：
//	cors:
//	  enabled: true
//	  allowOrigins:
//	    - "http://localhost:3000"
//	    - "https://example.com"
//	  allowMethods:
//	    - "GET"
//	    - "POST"
//	  allowHeaders:
//	    - "Content-Type"
//	    - "Authorization"
func CORSHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := app.BaseConfig.CORS
		if !cfg.Enabled {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// 检查来源是否被允许
		if !isOriginAllowed(origin, cfg.AllowOrigins) {
			c.Next()
			return
		}

		// 设置 CORS 响应头
		if len(cfg.AllowOrigins) == 1 && cfg.AllowOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}

		// 设置允许的方法
		if len(cfg.AllowMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ", "))
		} else {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		// 设置允许的请求头
		if len(cfg.AllowHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ", "))
		} else {
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Trace-Id, X-Request-Id")
		}

		// 设置暴露的响应头
		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ", "))
		}

		// 设置是否允许携带凭证
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 设置预检请求缓存时间
		maxAge := cfg.MaxAge
		if maxAge <= 0 {
			maxAge = 86400 // 默认 24 小时
		}
		c.Header("Access-Control-Max-Age", strconv.Itoa(maxAge))

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// isOriginAllowed 检查来源是否被允许
//
// 对通配符规则（如 *.example.com），解析 origin URL 提取 hostname 后
// 在子域名边界上做精确匹配，防止 evil-example.com 绕过 *.example.com
func isOriginAllowed(origin string, allowOrigins []string) bool {
	if len(allowOrigins) == 0 {
		return true
	}

	for _, allowed := range allowOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			host := extractHost(origin)
			if host == "" {
				continue
			}
			if host == domain || strings.HasSuffix(host, "."+domain) {
				return true
			}
		}
	}
	return false
}

// extractHost 从 origin URL 中提取主机名（不含端口）
func extractHost(origin string) string {
	u, err := url.Parse(origin)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
