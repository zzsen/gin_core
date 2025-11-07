// Package middleware 提供Gin框架的中间件功能
// 本文件实现了全局异常处理器中间件，用于捕获和处理HTTP请求过程中的异常
package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/zzsen/gin_core/exception"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"

	"github.com/gin-gonic/gin"
)

// ExceptionHandler 全局异常处理器中间件
// 该中间件会：
// 1. 使用defer和recover机制捕获所有panic异常
// 2. 根据异常类型选择不同的处理策略
// 3. 记录异常信息和堆栈跟踪
// 4. 返回统一的错误响应格式
// 5. 中断请求处理流程
// 返回：
//   - gin.HandlerFunc: Gin中间件函数
func ExceptionHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 使用defer和recover机制捕获异常
		defer func() {
			// 捕获panic异常
			if err := recover(); err != nil {
				// 设置默认的错误消息和错误码
				message := "服务端异常"
				code := response.ResponseExceptionUnknown.GetCode()

				// 如果是error类型且是validator校验异常，转换为InvalidParam异常
				if errValue, ok := err.(error); ok {
					if validationErrors, ok := errValue.(validator.ValidationErrors); ok {
						err = exception.NewInvalidParamFromValidator(validationErrors)
					}
				}

				// 检查异常是否实现了自定义异常处理接口
				if handler, ok := err.(exception.Handler); ok {
					// 如果实现了自定义异常处理接口，调用其处理方法
					message, code = handler.OnException(ctx)
				} else {
					// 如果未实现自定义异常处理接口，记录异常信息和堆栈跟踪
					logger.Logger.WithFields(logrus.Fields{
						"error":     err,                   // 异常信息
						"stackInfo": string(debug.Stack()), // 堆栈跟踪信息
					}).Error()
				}

				// 将错误信息添加到Gin上下文的错误列表中
				_ = ctx.Error(fmt.Errorf("%d : %s", code, message))

				// 返回统一的错误响应格式
				ctx.JSON(http.StatusOK, gin.H{
					"code": code,    // 错误码
					"msg":  message, // 错误消息
					"data": "",      // 数据字段（异常时为空）
				})

				// 中断请求处理流程，不再执行后续的中间件和处理器
				ctx.Abort()
				return
			}
		}()

		// 继续执行下一个中间件或处理器
		ctx.Next()
	}
}
