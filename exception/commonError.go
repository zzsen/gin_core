package exception

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)

// CommonError 通用异常,msg会返回给前端
type CommonError struct {
	msg string
}

func (e CommonError) Error() string {
	return e.msg
}

func NewCommonError(msg string) CommonError {
	return CommonError{msg: msg}
}

func (e CommonError) OnException(*gin.Context) (msg string, code int) {
	return e.Error(), response.ResponseExceptionCommon.GetCode()
}
