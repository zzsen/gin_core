package exception

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)

// AuthFailed 认证失败异常。
// 当请求的身份认证不通过时（如 Token 无效、过期等），通过 panic 抛出此异常，
// 框架将返回统一的认证失败响应码和消息。
//
// 使用方式：panic(exception.AuthFailed{})
type AuthFailed struct {
}

// Error 实现 error 接口，返回认证失败的默认消息
func (_ AuthFailed) Error() string {
	return response.ResponseAuthFailed.GetMsg()
}

// OnException 实现 Handler 接口，返回认证失败消息和状态码
func (authFailed AuthFailed) OnException(*gin.Context) (msg string, code int) {
	return authFailed.Error(), response.ResponseAuthFailed.GetCode()
}
