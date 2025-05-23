package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"

	"github.com/gin-gonic/gin"
)

func ExceptionHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				message := "服务端异常"
				code := response.ResponseExceptionUnknown.GetCode()
				if handler, ok := err.(exception.Handler); ok {
					message, code = handler.OnException(ctx)
				} else {
					logger.Logger.WithFields(logrus.Fields{
						"error":     err,
						"stackInfo": string(debug.Stack()),
					}).Error()
				}
				_ = ctx.Error(fmt.Errorf("%d : %s", code, message))
				ctx.JSON(http.StatusOK, gin.H{
					"code": code,
					"msg":  message,
					"data": "",
				})
				ctx.Abort()
				return
			}
		}()
		ctx.Next()
	}
}
