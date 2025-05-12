package middleware

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/logger"
)

func TraceLogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		endTime := time.Now()

		// 计算请求执行时间
		responseTime := endTime.Sub(startTime)

		// 获取请求相关信息
		reqMethod := c.Request.Method                                     // 请求方式
		reqUrl := c.Request.RequestURI                                    // 请求路由
		statusCode := c.Writer.Status()                                   // 状态码
		clientIP := c.ClientIP()                                          // 请求IP
		header := c.GetHeader("User-Agent") + "@@" + c.GetHeader("token") // 请求头
		_ = c.Request.ParseMultipartForm(128)
		reqForm := c.Request.Form
		var reqJsonStr string

		// 将请求数据转为 JSON 字符串
		if len(reqForm) > 0 {
			reqJsonByte, _ := json.Marshal(reqForm)
			reqJsonStr = string(reqJsonByte)
		}

		// 获取请求中的 requestId
		requestId := c.GetString("requestId")

		// 获取请求中的 traceId
		traceId, exists := c.Get("traceId")
		if !exists {
			traceId = ""
		}

		// 获取 Gin 中间件中的错误信息
		var errorsStr string
		for _, err := range c.Errors.Errors() {
			errorsStr += err + "; "
		}

		logger.Logger.WithFields(logrus.Fields{
			"traceId":      traceId,
			"requestId":    requestId,
			"statusCode":   statusCode,
			"responseTime": responseTime,
			"clientIp":     clientIP,
			"reqMethod":    reqMethod,
			"uaToken":      header,
			"reqUri":       reqUrl,
			"body":         reqJsonStr,
			"errStr":       errorsStr,
		}).Trace()
	}
}
