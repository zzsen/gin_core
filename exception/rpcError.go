package exception

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/logging"
	"github.com/zzsen/gin_core/model/response"
)

type RpcError struct {
	msg string
}

func (e RpcError) Error() string {
	return e.msg
}

func NewRpcError(msg string) RpcError {
	return RpcError{msg: msg}
}

func (e RpcError) OnException(*gin.Context) (msg string, code int) {
	logging.Error("%v", e)
	logging.Error(string(debug.Stack()))
	return e.Error(), response.ResponseExceptionRpc.GetCode()
}
