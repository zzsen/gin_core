package exception

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/logger"
	"github.com/zzsen/gin_core/model/response"
)

// RpcError RPC 调用异常。
// 当远程服务调用失败时抛出此异常，框架会记录错误日志和调用栈，并返回 RPC 异常状态码。
//
// 使用方式：panic(exception.NewRpcError("用户服务调用超时"))
type RpcError struct {
	msg string
}

// Error 实现 error 接口，返回 RPC 异常消息
func (e RpcError) Error() string {
	return e.msg
}

// NewRpcError 创建 RPC 调用异常。
// 参数 msg 为错误描述信息，将记录到日志并返回给调用方。
func NewRpcError(msg string) RpcError {
	return RpcError{msg: msg}
}

// OnException 实现 Handler 接口，记录错误日志和调用栈，返回 RPC 异常状态码
func (e RpcError) OnException(*gin.Context) (msg string, code int) {
	logger.Error("%v", e)
	logger.Error("%s", string(debug.Stack()))
	return e.Error(), response.ResponseExceptionRpc.GetCode()
}
