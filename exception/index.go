// Package exception 定义框架的异常体系。
//
// 所有业务异常均通过 panic 抛出，由框架的 recover 中间件统一捕获并转换为 HTTP 响应。
// 自定义异常需实现 Handler 接口；框架内置了 CommonError（通用异常）、AuthFailed（认证失败）、
// RpcError（RPC 调用异常）和 InvalidParam（参数校验异常）等类型。
package exception

import "github.com/gin-gonic/gin"

// Handler 异常处理器接口。
// 所有通过 panic 抛出的异常如果实现了此接口，框架将调用 OnException 获取响应消息和状态码；
// 未实现此接口的 panic 值将被视为未知异常（500）。
type Handler interface {
	// OnException 处理异常并返回响应信息。
	// 返回值 msg 将作为 HTTP 响应的错误消息，code 将作为业务状态码。
	OnException(ctx *gin.Context) (msg string, code int)
}
