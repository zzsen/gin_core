package exception

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)

// CommonError 通用业务异常，msg 会原样返回给前端。
// 适用于业务逻辑中需要向用户展示具体错误信息的场景。
//
// 使用方式：panic(exception.NewCommonError("xxx 不能为空"))
type CommonError struct {
	msg string
}

// Error 实现 error 接口，返回异常消息
func (e CommonError) Error() string {
	return e.msg
}

// NewCommonError 创建通用业务异常。
// 参数 msg 将直接返回给前端用户，请确保不包含敏感信息。
func NewCommonError(msg string) CommonError {
	return CommonError{msg: msg}
}

// OnException 实现 Handler 接口，返回异常消息和通用异常状态码
func (e CommonError) OnException(*gin.Context) (msg string, code int) {
	return e.Error(), response.ResponseExceptionCommon.GetCode()
}
