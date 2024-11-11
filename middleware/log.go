package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/logging"
)

func init() {
	err := core.RegisterMiddleware("logHandler", LogHandler)
	if err != nil {
		logging.Error(err.Error())
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
		logging.HInfo("["+c.Request.Method+"] "+path, logging.FieldTime("TimeStamp", endTime),
			logging.FieldDuration("Latency", endTime.Sub(start)),
			logging.FieldString("ClientIP", c.ClientIP()),
			logging.FieldInt("StatusCode", c.Writer.Status()),
			logging.FieldString("ErrorMessage", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			logging.FieldInt("BodySize", c.Writer.Size()))
	}
}
