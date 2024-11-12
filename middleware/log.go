package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
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
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		endTime := time.Now()
		if raw != "" {
			path = path + "?" + raw
		}
		//使用高效logger进行打印
		logger.HInfo("["+c.Request.Method+"] "+path, logger.FieldTime("TimeStamp", endTime),
			logger.FieldDuration("Latency", endTime.Sub(start)),
			logger.FieldString("ClientIP", c.ClientIP()),
			logger.FieldInt("StatusCode", c.Writer.Status()),
			logger.FieldString("ErrorMessage", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			logger.FieldInt("BodySize", c.Writer.Size()))
	}
}
