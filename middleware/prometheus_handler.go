// Package middleware 提供 Gin 框架的中间件功能
// 本文件实现了 Prometheus 指标采集中间件
package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/app"
	"github.com/zzsen/gin_core/metrics"
)

// excludePaths 不统计的路径
var excludePaths map[string]bool

// PrometheusHandler Prometheus 指标采集中间件
// 该中间件会：
// 1. 统计 HTTP 请求总数
// 2. 统计请求耗时分布
// 3. 统计当前处理中的请求数
func PrometheusHandler() gin.HandlerFunc {
	// 初始化排除路径
	excludePaths = make(map[string]bool)
	for _, path := range app.BaseConfig.Metrics.ExcludePaths {
		excludePaths[path] = true
	}

	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// 跳过不统计的路径
		if excludePaths[path] {
			c.Next()
			return
		}

		// 增加处理中请求数
		metrics.HttpRequestsInFlight.Inc()
		defer metrics.HttpRequestsInFlight.Dec()

		// 记录开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算耗时
		duration := time.Since(start).Seconds()

		// 记录指标
		method := c.Request.Method
		status := strconv.Itoa(c.Writer.Status())

		metrics.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
