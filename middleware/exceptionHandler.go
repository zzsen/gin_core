package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/zzsen/gin_core/core"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/logging"
	"github.com/zzsen/gin_core/model/response"

	"github.com/gin-gonic/gin"
)

func init() {
	err := core.RegisterMiddleware("exceptionHandler", ExceptionHandler)
	if err != nil {
		logging.Error(err.Error())
	}
}

func ExceptionHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				message := "服务端异常"
				code := response.ResponseExceptionUnknown.GetCode()
				if handler, ok := err.(exception.Handler); ok {
					message, code = handler.OnException(ctx)
				} else {
					logging.Error("%v", err)
					logging.Error(string(debug.Stack()))
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
