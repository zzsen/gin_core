package middleware

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/logger"
)

func init() {
	err := core.RegisterMiddleware("logHandler", LogHandler)
	if err != nil {
		logger.Error(err.Error())
	}
}

func LogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		endTime := time.Now()

		// 计算请求执行时间
		latencyTime := endTime.Sub(startTime)

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

		// 获取 Gin 中间件中的错误信息
		var errorsStr string
		for _, err := range c.Errors.Errors() {
			errorsStr += err + "; "
		}

		logger.Logger.WithFields(logrus.Fields{
			"request_id":   requestId,
			"status_code":  statusCode,
			"latency_time": latencyTime,
			"client_ip":    clientIP,
			"req_method":   reqMethod,
			"ua_token":     header,
			"req_uri":      reqUrl,
			"body":         reqJsonStr,
			"err_str":      errorsStr,
		}).Trace()
	}
}
