package exception

import (
	"github.com/gin-gonic/gin"
	"github.com/zzsen/gin_core/model/response"
)

type InvalidParam struct {
	msg string
}

func (e InvalidParam) Error() string {
	if e.msg != "" {
		return e.msg
	}
	return response.ResponseParamInvalid.GetMsg()
}

func NewInvalidParam(msg string) InvalidParam {
	return InvalidParam{msg: msg}
}

func (e InvalidParam) OnException(*gin.Context) (msg string, code int) {
	return e.Error(), response.ResponseParamInvalid.GetCode()
}
